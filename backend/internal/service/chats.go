package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/repo"

	"github.com/google/uuid"
)

const (
	MaxChatMessageRunes      = 2000
	DefaultChatPageLimit     = 20
	MaxChatPageLimit         = 50
	chatMessageCursorVersion = "v1"
	chatListCursorVersion    = "v1"
)

type ChatSendInput struct {
	ClientMessageID string
	Chat            domain.ChatRef
	Body            string
}

type ChatSendResult struct {
	Message          *domain.ChatMessage
	Created          bool
	RecipientUserIDs []int64
}

type ChatMessagePage struct {
	Messages   []*domain.ChatMessage
	NextCursor *string
}

type ChatPage struct {
	Chats      []*domain.ChatSummary
	NextCursor *string
}

type ChatService struct {
	transactions repo.TransactionManager
	clock        clock.Clock
}

func NewChatService(transactions repo.TransactionManager, appClock clock.Clock) *ChatService {
	return &ChatService{transactions: transactions, clock: appClock}
}

func (s *ChatService) Send(ctx context.Context, senderUserID int64, rawSessionToken string, input ChatSendInput) (*ChatSendResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || senderUserID <= 0 {
		return nil, ErrInvalidInput
	}
	input.ClientMessageID = strings.ToLower(strings.TrimSpace(input.ClientMessageID))
	parsedID, err := uuid.Parse(input.ClientMessageID)
	if err != nil || parsedID.String() != input.ClientMessageID {
		return nil, ErrInvalidInput
	}
	input.Body = strings.TrimSpace(input.Body)
	if !input.Chat.Kind.Valid() || input.Chat.TargetID <= 0 || !utf8.ValidString(input.Body) {
		return nil, ErrInvalidInput
	}
	bodyRunes := utf8.RuneCountInString(input.Body)
	if bodyRunes < 1 || bodyRunes > MaxChatMessageRunes {
		return nil, ErrInvalidInput
	}
	rawSessionToken = strings.TrimSpace(rawSessionToken)
	if rawSessionToken == "" {
		return nil, ErrUnauthorized
	}

	result := &ChatSendResult{RecipientUserIDs: make([]int64, 0)}
	err = s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if err := authorizeChatSession(
			ctx, repositories.Sessions(), rawSessionToken, senderUserID, s.clock.Now(),
		); err != nil {
			return err
		}
		recipients, err := authorizeChatSend(ctx, repositories, senderUserID, input.Chat)
		if err != nil {
			return err
		}

		existing, err := repositories.Chats().GetMessageByClientID(ctx, senderUserID, input.ClientMessageID)
		if err == nil {
			if existing.Chat != input.Chat || existing.Body != input.Body {
				return ErrConflict
			}
			result.Message = existing
			result.Created = false
			return nil
		}
		if !errors.Is(err, repo.ErrNotFound) {
			return err
		}

		createdAt := s.clock.Now()
		message := &domain.ChatMessage{
			SenderUserID: senderUserID, ClientMessageID: input.ClientMessageID,
			Chat: input.Chat, Body: input.Body, CreatedAt: createdAt,
		}
		switch input.Chat.Kind {
		case domain.ChatDirect:
			low, high := normalizeUserPair(senderUserID, input.Chat.TargetID)
			conversation, err := repositories.Chats().EnsureDirectConversation(ctx, low, high, createdAt)
			if err != nil {
				return err
			}
			message.DirectConversationID = &conversation.ID
		case domain.ChatGroup:
			groupID := input.Chat.TargetID
			message.GroupID = &groupID
		}
		message.ID, err = repositories.Chats().CreateMessage(ctx, message)
		if errors.Is(err, repo.ErrConflict) {
			existing, lookupErr := repositories.Chats().GetMessageByClientID(ctx, senderUserID, input.ClientMessageID)
			if lookupErr != nil {
				return lookupErr
			}
			if existing.Chat != input.Chat || existing.Body != input.Body {
				return ErrConflict
			}
			result.Message = existing
			result.Created = false
			return nil
		}
		if err != nil {
			return err
		}
		message.Sender, err = repositories.Users().GetByID(ctx, senderUserID)
		if err != nil {
			return err
		}
		result.Message = message
		result.Created = true
		result.RecipientUserIDs = recipients
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func authorizeChatSession(ctx context.Context, sessions repo.SessionRepo, rawToken string, userID int64, now time.Time) error {
	session, err := sessions.GetByToken(ctx, rawToken)
	if errors.Is(err, repo.ErrNotFound) {
		return ErrUnauthorized
	}
	if err != nil {
		return err
	}
	if session.UserID != userID || !session.ExpiresAt.After(now) {
		return ErrUnauthorized
	}
	return nil
}

func authorizeChatSend(ctx context.Context, repositories repo.TransactionRepositories, senderUserID int64, chat domain.ChatRef) ([]int64, error) {
	switch chat.Kind {
	case domain.ChatDirect:
		if chat.TargetID == senderUserID {
			return nil, ErrInvalidInput
		}
		if _, err := repositories.Users().GetByID(ctx, chat.TargetID); errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		} else if err != nil {
			return nil, err
		}
		allowed, err := directChatAllowed(ctx, repositories.Follows(), senderUserID, chat.TargetID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, ErrForbidden
		}
		return []int64{chat.TargetID}, nil
	case domain.ChatGroup:
		if _, err := repositories.Groups().Get(ctx, chat.TargetID, senderUserID); errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		} else if err != nil {
			return nil, err
		}
		status, err := repositories.Groups().GetMembershipStatus(ctx, chat.TargetID, senderUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrForbidden
		}
		if err != nil {
			return nil, err
		}
		if status == nil || (*status != domain.GroupOwner && *status != domain.GroupMember) {
			return nil, ErrForbidden
		}
		ids, err := repositories.Groups().ListActiveMemberIDs(ctx, chat.TargetID)
		if err != nil {
			return nil, err
		}
		return removeUserID(ids, senderUserID), nil
	default:
		return nil, ErrInvalidInput
	}
}

func directChatAllowed(ctx context.Context, follows repo.FollowRepo, firstUserID, secondUserID int64) (bool, error) {
	first, err := follows.IsAccepted(ctx, firstUserID, secondUserID)
	if err != nil {
		return false, err
	}
	second, err := follows.IsAccepted(ctx, secondUserID, firstUserID)
	return first || second, err
}

func (s *ChatService) DirectHistory(ctx context.Context, viewerUserID, targetUserID int64, cursor *domain.ChatMessageCursor, limit int) (*ChatMessagePage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || targetUserID <= 0 || viewerUserID == targetUserID || !validChatMessagePage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var messages []*domain.ChatMessage
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := repositories.Users().GetByID(ctx, targetUserID); errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		low, high := normalizeUserPair(viewerUserID, targetUserID)
		_, conversationErr := repositories.Chats().GetDirectConversation(ctx, low, high)
		if errors.Is(conversationErr, repo.ErrNotFound) {
			allowed, err := directChatAllowed(ctx, repositories.Follows(), viewerUserID, targetUserID)
			if err != nil {
				return err
			}
			if !allowed {
				return ErrForbidden
			}
			messages = []*domain.ChatMessage{}
			return nil
		}
		if conversationErr != nil {
			return conversationErr
		}
		var err error
		messages, err = repositories.Chats().ListDirectMessages(ctx, viewerUserID, targetUserID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildChatMessagePage(messages, limit), nil
}

func (s *ChatService) GroupHistory(ctx context.Context, viewerUserID, groupID int64, cursor *domain.ChatMessageCursor, limit int) (*ChatMessagePage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || groupID <= 0 || !validChatMessagePage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var messages []*domain.ChatMessage
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := repositories.Groups().Get(ctx, groupID, viewerUserID); errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		status, err := repositories.Groups().GetMembershipStatus(ctx, groupID, viewerUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		if status == nil || (*status != domain.GroupOwner && *status != domain.GroupMember) {
			return ErrForbidden
		}
		messages, err = repositories.Chats().ListGroupMessages(ctx, viewerUserID, groupID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildChatMessagePage(messages, limit), nil
}

func (s *ChatService) List(ctx context.Context, viewerUserID int64, cursor *domain.ChatListCursor, limit int) (*ChatPage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || !validChatListPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var chats []*domain.ChatSummary
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		chats, err = repositories.Chats().ListChats(ctx, viewerUserID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	page := &ChatPage{Chats: chats}
	if len(chats) > limit {
		page.Chats = chats[:limit]
		last := page.Chats[len(page.Chats)-1]
		encoded := EncodeChatListCursor(domain.ChatListCursor{
			ActivityAt: last.ActivityAt, KindRank: chatKindRank(last.Kind), EntityID: last.EntityID,
		})
		page.NextCursor = &encoded
	}
	return page, nil
}

func (s *ChatService) DirectPeerIDs(ctx context.Context, userID int64) ([]int64, error) {
	if s == nil || s.transactions == nil || userID <= 0 {
		return nil, ErrInvalidInput
	}
	var ids []int64
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		ids, err = repositories.Chats().ListDirectPeerIDs(ctx, userID)
		return err
	})
	return ids, err
}

func (s *ChatService) AuthorizeTyping(ctx context.Context, userID int64, rawSessionToken string, chat domain.ChatRef) ([]int64, error) {
	if s == nil || s.transactions == nil || s.clock == nil || userID <= 0 || !chat.Kind.Valid() || chat.TargetID <= 0 {
		return nil, ErrInvalidInput
	}
	var recipients []int64
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if err := authorizeChatSession(ctx, repositories.Sessions(), strings.TrimSpace(rawSessionToken), userID, s.clock.Now()); err != nil {
			return err
		}
		var err error
		recipients, err = authorizeChatSend(ctx, repositories, userID, chat)
		return err
	})
	return recipients, err
}

func buildChatMessagePage(messages []*domain.ChatMessage, limit int) *ChatMessagePage {
	page := &ChatMessagePage{Messages: messages}
	if len(messages) > limit {
		page.Messages = messages[:limit]
		oldest := page.Messages[len(page.Messages)-1]
		encoded := EncodeChatMessageCursor(domain.ChatMessageCursor{CreatedAt: oldest.CreatedAt, ID: oldest.ID})
		page.NextCursor = &encoded
	}
	for left, right := 0, len(page.Messages)-1; left < right; left, right = left+1, right-1 {
		page.Messages[left], page.Messages[right] = page.Messages[right], page.Messages[left]
	}
	return page
}

func validChatMessagePage(cursor *domain.ChatMessageCursor, limit int) bool {
	return limit >= 1 && limit <= MaxChatPageLimit && (cursor == nil || (!cursor.CreatedAt.IsZero() && cursor.ID > 0))
}

func validChatListPage(cursor *domain.ChatListCursor, limit int) bool {
	return limit >= 1 && limit <= MaxChatPageLimit && (cursor == nil || (!cursor.ActivityAt.IsZero() && cursor.EntityID > 0 && (cursor.KindRank == 0 || cursor.KindRank == 1)))
}

func EncodeChatMessageCursor(cursor domain.ChatMessageCursor) string {
	payload := chatMessageCursorVersion + ":" + strconv.FormatInt(cursor.CreatedAt.UTC().Unix(), 10) + ":" + strconv.FormatInt(cursor.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeChatMessageCursor(value string) (*domain.ChatMessageCursor, error) {
	parts, err := decodeChatCursor(value, chatMessageCursorVersion, 3)
	if err != nil {
		return nil, err
	}
	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || timestamp <= 0 {
		return nil, ErrInvalidInput
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || id <= 0 {
		return nil, ErrInvalidInput
	}
	return &domain.ChatMessageCursor{CreatedAt: time.Unix(timestamp, 0).UTC(), ID: id}, nil
}

func EncodeChatListCursor(cursor domain.ChatListCursor) string {
	payload := strings.Join([]string{
		chatListCursorVersion,
		strconv.FormatInt(cursor.ActivityAt.UTC().Unix(), 10),
		strconv.Itoa(cursor.KindRank),
		strconv.FormatInt(cursor.EntityID, 10),
	}, ":")
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeChatListCursor(value string) (*domain.ChatListCursor, error) {
	parts, err := decodeChatCursor(value, chatListCursorVersion, 4)
	if err != nil {
		return nil, err
	}
	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || timestamp <= 0 {
		return nil, ErrInvalidInput
	}
	kindRank, err := strconv.Atoi(parts[2])
	if err != nil || (kindRank != 0 && kindRank != 1) {
		return nil, ErrInvalidInput
	}
	entityID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || entityID <= 0 {
		return nil, ErrInvalidInput
	}
	return &domain.ChatListCursor{ActivityAt: time.Unix(timestamp, 0).UTC(), KindRank: kindRank, EntityID: entityID}, nil
}

func decodeChatCursor(value, version string, count int) ([]string, error) {
	value = strings.TrimSpace(value)
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || value == "" {
		return nil, ErrInvalidInput
	}
	parts := strings.Split(string(decoded), ":")
	if len(parts) != count || parts[0] != version {
		return nil, ErrInvalidInput
	}
	return parts, nil
}

func normalizeUserPair(first, second int64) (int64, int64) {
	if first < second {
		return first, second
	}
	return second, first
}

func removeUserID(ids []int64, userID int64) []int64 {
	result := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 || id == userID {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func chatKindRank(kind domain.ChatKind) int {
	if kind == domain.ChatGroup {
		return 1
	}
	return 0
}
