package service_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

type mutableChatClock struct {
	mu  sync.RWMutex
	now time.Time
}

func (c *mutableChatClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *mutableChatClock) Set(now time.Time) {
	c.mu.Lock()
	c.now = now
	c.mu.Unlock()
}

type beforeTransactionManager struct {
	inner  repo.TransactionManager
	before func()
}

func (m *beforeTransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(repo.TransactionRepositories) error,
) error {
	if m.before != nil {
		m.before()
	}
	return m.inner.WithinTransaction(ctx, fn)
}

type sequenceChatClock struct {
	mu    sync.Mutex
	times []time.Time
}

func (c *sequenceChatClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.times) == 0 {
		return time.Time{}
	}
	now := c.times[0]
	if len(c.times) > 1 {
		c.times = c.times[1:]
	}
	return now
}

type chatFixture struct {
	db       *sql.DB
	clock    *mutableChatClock
	users    *sqlite.UserRepo
	follows  *sqlite.FollowRepo
	sessions *service.SessionService
	groups   *service.GroupService
	chats    *service.ChatService
	userIDs  []int64
	tokens   []string
}

func newChatFixture(t *testing.T) *chatFixture {
	t.Helper()
	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)
	appClock := &mutableChatClock{now: now}
	users := sqlite.NewUserRepo(db)
	sessions := service.NewSessionService(sqlite.NewSessionRepo(db), appClock, &authTestIDGenerator{}, 24*time.Hour)
	fixture := &chatFixture{
		db: db, clock: appClock, users: users, follows: sqlite.NewFollowRepo(db), sessions: sessions,
		groups: service.NewGroupService(sqlite.NewTransactionManager(db), appClock),
		chats:  service.NewChatService(sqlite.NewTransactionManager(db), appClock),
	}
	for index := 0; index < 4; index++ {
		userID := createPostTestUser(t, context.Background(), users, fmt.Sprintf("chat-user-%d@example.com", index+1), now)
		session, err := sessions.Create(context.Background(), userID)
		if err != nil {
			t.Fatalf("create chat session: %v", err)
		}
		fixture.userIDs = append(fixture.userIDs, userID)
		fixture.tokens = append(fixture.tokens, session.Token)
	}
	return fixture
}

func chatClientID(sequence int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", sequence)
}

func (f *chatFixture) acceptFollow(t *testing.T, followerIndex, followedIndex int) {
	t.Helper()
	if _, err := f.follows.Upsert(
		context.Background(), f.userIDs[followerIndex], f.userIDs[followedIndex],
		domain.FollowAccepted, f.clock.Now(),
	); err != nil {
		t.Fatalf("create accepted follow: %v", err)
	}
}

func TestDirectChatAuthorizationIdempotencyAndHistoryAfterUnfollow(t *testing.T) {
	fixture := newChatFixture(t)
	ctx := context.Background()
	fixture.acceptFollow(t, 0, 1)
	first, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(1),
		Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]},
		Body:            "  hello direct  ",
	})
	if err != nil || !first.Created || first.Message.Body != "hello direct" {
		t.Fatalf("first direct send: result=%+v err=%v", first, err)
	}
	duplicate, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(1),
		Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]},
		Body:            "hello direct",
	})
	if err != nil || duplicate.Created || duplicate.Message.ID != first.Message.ID {
		t.Fatalf("idempotent send: result=%+v err=%v", duplicate, err)
	}
	if result, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(1),
		Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]},
		Body:            "different canonical body",
	}); !errors.Is(err, service.ErrConflict) || result != nil {
		t.Fatalf("conflicting idempotency key: result=%+v err=%v", result, err)
	}
	second, err := fixture.chats.Send(ctx, fixture.userIDs[1], fixture.tokens[1], service.ChatSendInput{
		ClientMessageID: chatClientID(2),
		Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[0]},
		Body:            "reply",
	})
	if err != nil || !second.Created {
		t.Fatalf("reverse direct send: result=%+v err=%v", second, err)
	}
	var conversations int
	if err := fixture.db.QueryRow("SELECT COUNT(*) FROM direct_conversations").Scan(&conversations); err != nil || conversations != 1 {
		t.Fatalf("normalized conversation count=%d err=%v", conversations, err)
	}

	if err := fixture.follows.Delete(ctx, fixture.userIDs[0], fixture.userIDs[1]); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	page, err := fixture.chats.DirectHistory(ctx, fixture.userIDs[0], fixture.userIDs[1], nil, 20)
	if err != nil || len(page.Messages) != 2 {
		t.Fatalf("history after unfollow: page=%+v err=%v", page, err)
	}
	if result, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(3),
		Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]},
		Body:            "must be forbidden",
	}); !errors.Is(err, service.ErrForbidden) || result != nil {
		t.Fatalf("send after unfollow: result=%+v err=%v", result, err)
	}

	if _, err := fixture.chats.DirectHistory(ctx, fixture.userIDs[2], fixture.userIDs[0], nil, 20); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("unrelated empty direct history: %v", err)
	}
	fixture.acceptFollow(t, 2, 0)
	empty, err := fixture.chats.DirectHistory(ctx, fixture.userIDs[2], fixture.userIDs[0], nil, 20)
	if err != nil || len(empty.Messages) != 0 {
		t.Fatalf("eligible empty direct history: page=%+v err=%v", empty, err)
	}
	if _, err := fixture.chats.DirectHistory(ctx, fixture.userIDs[0], fixture.userIDs[0], nil, 20); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("self direct history: %v", err)
	}
	if _, err := fixture.chats.DirectHistory(ctx, fixture.userIDs[0], 999999, nil, 20); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("unknown direct history target: %v", err)
	}
}

func TestGroupChatAccessLeaveAndRejoinHistory(t *testing.T) {
	fixture := newChatFixture(t)
	ctx := context.Background()
	group, err := fixture.groups.Create(ctx, fixture.userIDs[0], "Realtime group", "Group chat access")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := fixture.groups.RequestJoin(ctx, fixture.userIDs[1], group.ID); err != nil {
		t.Fatalf("request member join: %v", err)
	}
	if _, err := fixture.groups.AcceptJoinRequest(ctx, fixture.userIDs[0], group.ID, fixture.userIDs[1]); err != nil {
		t.Fatalf("accept member: %v", err)
	}
	if _, err := fixture.groups.RequestJoin(ctx, fixture.userIDs[2], group.ID); err != nil {
		t.Fatalf("request outsider join: %v", err)
	}

	ownerMessage, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(10),
		Chat:            domain.ChatRef{Kind: domain.ChatGroup, TargetID: group.ID},
		Body:            "owner message",
	})
	if err != nil || len(ownerMessage.RecipientUserIDs) != 1 || ownerMessage.RecipientUserIDs[0] != fixture.userIDs[1] {
		t.Fatalf("owner group send: result=%+v err=%v", ownerMessage, err)
	}
	if _, err := fixture.chats.Send(ctx, fixture.userIDs[1], fixture.tokens[1], service.ChatSendInput{
		ClientMessageID: chatClientID(11),
		Chat:            domain.ChatRef{Kind: domain.ChatGroup, TargetID: group.ID},
		Body:            "member message",
	}); err != nil {
		t.Fatalf("member group send: %v", err)
	}
	for _, userIndex := range []int{0, 1} {
		page, err := fixture.chats.GroupHistory(ctx, fixture.userIDs[userIndex], group.ID, nil, 20)
		if err != nil || len(page.Messages) != 2 {
			t.Fatalf("group history user %d: page=%+v err=%v", userIndex, page, err)
		}
	}
	if _, err := fixture.chats.GroupHistory(ctx, fixture.userIDs[2], group.ID, nil, 20); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("requested user group history: %v", err)
	}
	if result, err := fixture.chats.Send(ctx, fixture.userIDs[2], fixture.tokens[2], service.ChatSendInput{
		ClientMessageID: chatClientID(12),
		Chat:            domain.ChatRef{Kind: domain.ChatGroup, TargetID: group.ID},
		Body:            "requested user",
	}); !errors.Is(err, service.ErrForbidden) || result != nil {
		t.Fatalf("requested user group send: result=%+v err=%v", result, err)
	}
	if _, err := fixture.chats.GroupHistory(ctx, fixture.userIDs[0], 999999, nil, 20); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("unknown group history: %v", err)
	}

	if _, err := fixture.groups.Leave(ctx, fixture.userIDs[1], group.ID); err != nil {
		t.Fatalf("member leave: %v", err)
	}
	if _, err := fixture.chats.GroupHistory(ctx, fixture.userIDs[1], group.ID, nil, 20); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("history after leave: %v", err)
	}
	if _, err := fixture.groups.RequestJoin(ctx, fixture.userIDs[1], group.ID); err != nil {
		t.Fatalf("request rejoin: %v", err)
	}
	if _, err := fixture.groups.AcceptJoinRequest(ctx, fixture.userIDs[0], group.ID, fixture.userIDs[1]); err != nil {
		t.Fatalf("accept rejoin: %v", err)
	}
	page, err := fixture.chats.GroupHistory(ctx, fixture.userIDs[1], group.ID, nil, 20)
	if err != nil || len(page.Messages) != 2 {
		t.Fatalf("history after rejoin: page=%+v err=%v", page, err)
	}
}

func TestChatMessageValidationAndSessionCheck(t *testing.T) {
	fixture := newChatFixture(t)
	ctx := context.Background()
	fixture.acceptFollow(t, 0, 1)
	valid := strings.Repeat("🙂", service.MaxChatMessageRunes)
	result, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(20),
		Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]},
		Body:            " \n" + valid + "\t ",
	})
	if err != nil || result.Message.Body != valid {
		t.Fatalf("max Unicode message: result=%+v err=%v", result, err)
	}
	for name, input := range map[string]service.ChatSendInput{
		"empty": {
			ClientMessageID: chatClientID(21), Chat: domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]}, Body: " \n ",
		},
		"too long": {
			ClientMessageID: chatClientID(22), Chat: domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]}, Body: valid + "a",
		},
		"invalid utf8": {
			ClientMessageID: chatClientID(23), Chat: domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]}, Body: string([]byte{0xff}),
		},
		"self": {
			ClientMessageID: chatClientID(25), Chat: domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[0]}, Body: "body",
		},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], input)
			if !errors.Is(err, service.ErrInvalidInput) || result != nil {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
	normalizedID, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: "47CD9266-B43F-4A89-9338-4F9C197FF12A",
		Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]},
		Body:            "normalized UUID",
	})
	if err != nil || normalizedID.Message.ClientMessageID != "47cd9266-b43f-4a89-9338-4f9c197ff12a" {
		t.Fatalf("normalize UUID: result=%+v err=%v", normalizedID, err)
	}
	if err := fixture.sessions.Delete(ctx, fixture.tokens[0]); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if result, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(26),
		Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]},
		Body:            "after logout",
	}); !errors.Is(err, service.ErrUnauthorized) || result != nil {
		t.Fatalf("send with deleted session: result=%+v err=%v", result, err)
	}
}

func TestChatSendChecksExpiryAfterEnteringTransactionAndUsesFreshCreatedAt(t *testing.T) {
	t.Run("expiry while waiting for transaction", func(t *testing.T) {
		fixture := newChatFixture(t)
		fixture.acceptFollow(t, 0, 1)
		expiresAt := time.Date(2026, time.July, 23, 12, 0, 0, 0, time.UTC)
		fixture.clock.Set(expiresAt.Add(-time.Second))
		transactions := &beforeTransactionManager{
			inner: sqlite.NewTransactionManager(fixture.db),
			before: func() {
				fixture.clock.Set(expiresAt.Add(time.Second))
			},
		}
		chats := service.NewChatService(transactions, fixture.clock)
		result, err := chats.Send(context.Background(), fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
			ClientMessageID: chatClientID(25),
			Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]},
			Body:            "must not persist after expiry",
		})
		if result != nil || !errors.Is(err, service.ErrUnauthorized) {
			t.Fatalf("expired send: result=%+v err=%v", result, err)
		}
		var messageCount, conversationCount int
		if err := fixture.db.QueryRow(`SELECT COUNT(*) FROM chat_messages`).Scan(&messageCount); err != nil {
			t.Fatalf("count messages: %v", err)
		}
		if err := fixture.db.QueryRow(`SELECT COUNT(*) FROM direct_conversations`).Scan(&conversationCount); err != nil {
			t.Fatalf("count conversations: %v", err)
		}
		if messageCount != 0 || conversationCount != 0 {
			t.Fatalf("expired send persisted rows: messages=%d conversations=%d", messageCount, conversationCount)
		}
	})

	t.Run("created at is sampled after authorization", func(t *testing.T) {
		fixture := newChatFixture(t)
		fixture.acceptFollow(t, 0, 1)
		authorizedAt := time.Date(2026, time.July, 22, 12, 0, 1, 0, time.UTC)
		createdAt := authorizedAt.Add(5 * time.Second)
		chats := service.NewChatService(sqlite.NewTransactionManager(fixture.db), &sequenceChatClock{
			times: []time.Time{authorizedAt, createdAt},
		})
		result, err := chats.Send(context.Background(), fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
			ClientMessageID: chatClientID(26),
			Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]},
			Body:            "fresh creation time",
		})
		if err != nil || result == nil || result.Message == nil {
			t.Fatalf("send with sequence clock: result=%+v err=%v", result, err)
		}
		if !result.Message.CreatedAt.Equal(createdAt) {
			t.Fatalf("created_at=%s want=%s", result.Message.CreatedAt, createdAt)
		}
	})
}

func TestConcurrentDuplicateChatSendCreatesOneRow(t *testing.T) {
	fixture := newChatFixture(t)
	fixture.acceptFollow(t, 0, 1)
	const workers = 8
	results := make(chan *service.ChatSendResult, workers)
	errs := make(chan error, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := fixture.chats.Send(context.Background(), fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
				ClientMessageID: chatClientID(30),
				Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]},
				Body:            "one persisted message",
			})
			results <- result
			errs <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent send: %v", err)
		}
	}
	created := 0
	var messageID int64
	for result := range results {
		if result.Created {
			created++
		}
		if messageID == 0 {
			messageID = result.Message.ID
		} else if result.Message.ID != messageID {
			t.Fatalf("duplicate sends returned different IDs: %d and %d", messageID, result.Message.ID)
		}
	}
	if created != 1 {
		t.Fatalf("created results=%d, want 1", created)
	}
	var rows int
	if err := fixture.db.QueryRow("SELECT COUNT(*) FROM chat_messages").Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("chat message rows=%d err=%v", rows, err)
	}
}

func TestChatHistoryAndMixedListCursorsDoNotDuplicateRows(t *testing.T) {
	fixture := newChatFixture(t)
	ctx := context.Background()
	fixture.acceptFollow(t, 0, 1)
	fixture.acceptFollow(t, 0, 2)
	for index := 1; index <= 5; index++ {
		targetIndex := 1
		if index == 5 {
			targetIndex = 2
		}
		if _, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
			ClientMessageID: chatClientID(40 + index),
			Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[targetIndex]},
			Body:            fmt.Sprintf("message %d", index),
		}); err != nil {
			t.Fatalf("send message %d: %v", index, err)
		}
	}
	group, err := fixture.groups.Create(ctx, fixture.userIDs[0], "Cursor group", "No messages needed")
	if err != nil {
		t.Fatalf("create cursor group: %v", err)
	}

	var messageIDs []int64
	var cursor *domain.ChatMessageCursor
	for {
		page, err := fixture.chats.DirectHistory(ctx, fixture.userIDs[0], fixture.userIDs[1], cursor, 2)
		if err != nil {
			t.Fatalf("direct history page: %v", err)
		}
		for _, message := range page.Messages {
			messageIDs = append(messageIDs, message.ID)
		}
		if page.NextCursor == nil {
			break
		}
		cursor, err = service.DecodeChatMessageCursor(*page.NextCursor)
		if err != nil {
			t.Fatalf("decode message cursor: %v", err)
		}
	}
	seenMessages := make(map[int64]struct{})
	for _, id := range messageIDs {
		if _, duplicate := seenMessages[id]; duplicate {
			t.Fatalf("duplicate history message %d across pages", id)
		}
		seenMessages[id] = struct{}{}
	}
	if len(messageIDs) != 4 {
		t.Fatalf("history message count=%d want=4", len(messageIDs))
	}

	var chatKeys []string
	var listCursor *domain.ChatListCursor
	for {
		page, err := fixture.chats.List(ctx, fixture.userIDs[0], listCursor, 1)
		if err != nil {
			t.Fatalf("chat list page: %v", err)
		}
		for _, chat := range page.Chats {
			chatKeys = append(chatKeys, fmt.Sprintf("%s:%d", chat.Kind, chat.TargetID))
		}
		if page.NextCursor == nil {
			break
		}
		listCursor, err = service.DecodeChatListCursor(*page.NextCursor)
		if err != nil {
			t.Fatalf("decode list cursor: %v", err)
		}
	}
	if len(chatKeys) != 3 {
		t.Fatalf("chat list=%v, want two directs and group %d", chatKeys, group.ID)
	}
	wantChatKeys := []string{
		fmt.Sprintf("direct:%d", fixture.userIDs[2]),
		fmt.Sprintf("direct:%d", fixture.userIDs[1]),
		fmt.Sprintf("group:%d", group.ID),
	}
	for index := range wantChatKeys {
		if chatKeys[index] != wantChatKeys[index] {
			t.Fatalf("chat list order=%v want=%v", chatKeys, wantChatKeys)
		}
	}
	seenChats := make(map[string]struct{})
	for _, key := range chatKeys {
		if _, duplicate := seenChats[key]; duplicate {
			t.Fatalf("duplicate chat %s across pages", key)
		}
		seenChats[key] = struct{}{}
	}
}

func TestPersistedDirectUnreadIsIdempotentMonotonicAndSurvivesUnfollow(t *testing.T) {
	fixture := newChatFixture(t)
	ctx := context.Background()
	fixture.acceptFollow(t, 0, 1)
	chat := domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[1]}

	first, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(8001), Chat: chat, Body: "first unread",
	})
	if err != nil || !first.Created {
		t.Fatalf("first send: result=%+v err=%v", first, err)
	}
	state := first.UnreadEffects.StatesByUser[fixture.userIDs[1]]
	if state == nil || state.ChatUnreadCount != 1 || state.UnreadCount != 1 || state.Revision != 1 {
		t.Fatalf("first unread effect=%+v", state)
	}
	duplicate, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(8001), Chat: chat, Body: "first unread",
	})
	if err != nil || duplicate.Created || len(duplicate.UnreadEffects.StatesByUser) != 0 {
		t.Fatalf("duplicate send: result=%+v err=%v", duplicate, err)
	}
	var revision, unread int64
	if err := fixture.db.QueryRow(`
		SELECT user_state.revision, read_state.unread_count
		FROM chat_user_states user_state
		JOIN direct_chat_read_states read_state ON read_state.user_id = user_state.user_id
		WHERE user_state.user_id = ?
	`, fixture.userIDs[1]).Scan(&revision, &unread); err != nil || revision != 1 || unread != 1 {
		t.Fatalf("state after duplicate: revision=%d unread=%d err=%v", revision, unread, err)
	}

	read, err := fixture.chats.MarkRead(
		ctx, fixture.userIDs[1],
		domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[0]},
		first.Message.ID,
	)
	if err != nil || !read.Changed || read.State.ChatUnreadCount != 0 ||
		read.State.UnreadCount != 0 || read.State.Revision != 2 ||
		read.State.ReadThroughMessageID == nil || *read.State.ReadThroughMessageID != first.Message.ID {
		t.Fatalf("first read=%+v err=%v", read, err)
	}
	repeated, err := fixture.chats.MarkRead(
		ctx, fixture.userIDs[1],
		domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[0]},
		first.Message.ID,
	)
	if err != nil || repeated.Changed || repeated.State.Revision != 2 {
		t.Fatalf("idempotent read=%+v err=%v", repeated, err)
	}

	fixture.clock.Set(fixture.clock.Now().Add(time.Second))
	second, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(8002), Chat: chat, Body: "second unread",
	})
	if err != nil || second.UnreadEffects.StatesByUser[fixture.userIDs[1]].Revision != 3 {
		t.Fatalf("second send=%+v err=%v", second, err)
	}
	older, err := fixture.chats.MarkRead(
		ctx, fixture.userIDs[1],
		domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[0]},
		first.Message.ID,
	)
	if err != nil || older.Changed || older.State.ChatUnreadCount != 1 || older.State.Revision != 3 {
		t.Fatalf("older marker=%+v err=%v", older, err)
	}
	if err := fixture.follows.Delete(ctx, fixture.userIDs[0], fixture.userIDs[1]); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	afterUnfollow, err := fixture.chats.MarkRead(
		ctx, fixture.userIDs[1],
		domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[0]},
		second.Message.ID,
	)
	if err != nil || !afterUnfollow.Changed || afterUnfollow.State.ChatUnreadCount != 0 ||
		afterUnfollow.State.Revision != 4 {
		t.Fatalf("read after unfollow=%+v err=%v", afterUnfollow, err)
	}
	if _, err := fixture.chats.MarkRead(
		ctx, fixture.userIDs[1],
		domain.ChatRef{Kind: domain.ChatDirect, TargetID: fixture.userIDs[2]},
		second.Message.ID,
	); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("foreign conversation marker error=%v", err)
	}
}

func TestGroupUnreadUsesPhysicalMembershipLifecycle(t *testing.T) {
	fixture := newChatFixture(t)
	ctx := context.Background()
	group, err := fixture.groups.Create(ctx, fixture.userIDs[0], "Unread lifecycle", "Physical membership state")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := fixture.groups.Invite(ctx, fixture.userIDs[0], group.ID, fixture.userIDs[1]); err != nil {
		t.Fatalf("invite member: %v", err)
	}
	if _, err := fixture.groups.AcceptInvitation(ctx, fixture.userIDs[1], group.ID); err != nil {
		t.Fatalf("accept invitation: %v", err)
	}
	var firstMembershipID int64
	if err := fixture.db.QueryRow(`
		SELECT id FROM group_memberships WHERE group_id = ? AND user_id = ?
	`, group.ID, fixture.userIDs[1]).Scan(&firstMembershipID); err != nil {
		t.Fatalf("first membership id: %v", err)
	}

	message, err := fixture.chats.Send(ctx, fixture.userIDs[0], fixture.tokens[0], service.ChatSendInput{
		ClientMessageID: chatClientID(8101),
		Chat:            domain.ChatRef{Kind: domain.ChatGroup, TargetID: group.ID},
		Body:            "group unread",
	})
	if err != nil || message.UnreadEffects.StatesByUser[fixture.userIDs[1]].ChatUnreadCount != 1 {
		t.Fatalf("group send=%+v err=%v", message, err)
	}
	leave, err := fixture.groups.LeaveWithEffects(ctx, fixture.userIDs[1], group.ID)
	if err != nil {
		t.Fatalf("leave group: %v", err)
	}
	leaveState := leave.ChatUnreadEffects.StatesByUser[fixture.userIDs[1]]
	if leaveState == nil || leaveState.ChatUnreadCount != 0 || leaveState.UnreadCount != 0 || leaveState.Revision != 2 {
		t.Fatalf("leave unread effect=%+v", leaveState)
	}
	var oldStates int
	if err := fixture.db.QueryRow(`
		SELECT COUNT(*) FROM group_chat_read_states WHERE membership_id = ?
	`, firstMembershipID).Scan(&oldStates); err != nil || oldStates != 0 {
		t.Fatalf("old lifecycle state: count=%d err=%v", oldStates, err)
	}

	fixture.clock.Set(fixture.clock.Now().Add(time.Second))
	if _, err := fixture.groups.Invite(ctx, fixture.userIDs[0], group.ID, fixture.userIDs[1]); err != nil {
		t.Fatalf("reinvite member: %v", err)
	}
	if _, err := fixture.groups.AcceptInvitation(ctx, fixture.userIDs[1], group.ID); err != nil {
		t.Fatalf("reaccept invitation: %v", err)
	}
	var newMembershipID, markerID, unread int64
	if err := fixture.db.QueryRow(`
		SELECT membership.id, state.last_read_message_id, state.unread_count
		FROM group_memberships membership
		JOIN group_chat_read_states state ON state.membership_id = membership.id
		WHERE membership.group_id = ? AND membership.user_id = ?
	`, group.ID, fixture.userIDs[1]).Scan(&newMembershipID, &markerID, &unread); err != nil {
		t.Fatalf("new lifecycle state: %v", err)
	}
	if newMembershipID == firstMembershipID || markerID != message.Message.ID || unread != 0 {
		t.Fatalf("new lifecycle: old=%d new=%d marker=%d unread=%d", firstMembershipID, newMembershipID, markerID, unread)
	}
	history, err := fixture.chats.GroupHistory(ctx, fixture.userIDs[1], group.ID, nil, 20)
	if err != nil || len(history.Messages) != 1 || history.Messages[0].ID != message.Message.ID {
		t.Fatalf("history after rejoin=%+v err=%v", history, err)
	}
}

func TestChatCursorsRoundTripAndRejectMalformedValues(t *testing.T) {
	messageCursor := domain.ChatMessageCursor{CreatedAt: time.Unix(1_700_000_000, 0).UTC(), ID: 42}
	decodedMessage, err := service.DecodeChatMessageCursor(service.EncodeChatMessageCursor(messageCursor))
	if err != nil || !decodedMessage.CreatedAt.Equal(messageCursor.CreatedAt) || decodedMessage.ID != messageCursor.ID {
		t.Fatalf("message cursor round trip: got=%+v err=%v", decodedMessage, err)
	}
	listCursor := domain.ChatListCursor{
		ActivityAt: time.Unix(1_700_000_000, 0).UTC(), KindRank: 1, EntityID: 7,
	}
	decodedList, err := service.DecodeChatListCursor(service.EncodeChatListCursor(listCursor))
	if err != nil || !decodedList.ActivityAt.Equal(listCursor.ActivityAt) ||
		decodedList.KindRank != listCursor.KindRank || decodedList.EntityID != listCursor.EntityID {
		t.Fatalf("list cursor round trip: got=%+v err=%v", decodedList, err)
	}
	for _, invalid := range []string{"", "not-base64", "djI6MTcwMDAwMDAwMDo0Mg", "djE6MDox", "djE6MTcwMDAwMDAwMDow"} {
		if _, err := service.DecodeChatMessageCursor(invalid); !errors.Is(err, service.ErrInvalidInput) {
			t.Fatalf("message cursor %q: %v", invalid, err)
		}
		if _, err := service.DecodeChatListCursor(invalid); !errors.Is(err, service.ErrInvalidInput) {
			t.Fatalf("list cursor %q: %v", invalid, err)
		}
	}
}
