package ws

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"sync/atomic"
	"testing"
	"time"

	"social-network/backend/internal/domain"
)

var testHubExpiry = time.Now().UTC().Add(time.Hour)

func startTestHub(t *testing.T) *Hub {
	t.Helper()
	hub := NewHub(log.New(io.Discard, "", 0))
	go hub.Run()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = hub.BeginDrain(ctx)
		select {
		case <-hub.Done():
		case <-ctx.Done():
			t.Errorf("hub did not stop: %s", hub.diagnosticState())
		}
	})
	return hub
}

func testClient(id string, userID int64, token string, queueSize int) *Client {
	if queueSize <= 0 {
		queueSize = ClientQueueSize
	}
	return &Client{
		id: id, userID: userID, displayName: "User",
		sessionKey: HashSessionToken(token), expiresAt: testHubExpiry,
		send: make(chan []byte, queueSize), done: make(chan struct{}),
	}
}

func registerTestClient(t *testing.T, hub *Hub, client *Client, peers ...int64) {
	t.Helper()
	client.hub = hub
	generation, err := hub.BeginPresenceSync(client.userID)
	if err != nil {
		t.Fatalf("begin presence sync: %v", err)
	}
	if err := hub.Register(client, generation, peers); err != nil {
		t.Fatalf("register client: %v", err)
	}
}

func assertChatRemove(t *testing.T, client *Client, kind domain.ChatKind, targetID int64) {
	t.Helper()
	var event struct {
		Type string         `json:"type"`
		Chat domain.ChatRef `json:"chat"`
	}
	if err := json.Unmarshal(readPayload(t, client), &event); err != nil ||
		event.Type != "chat:remove" || event.Chat != (domain.ChatRef{Kind: kind, TargetID: targetID}) {
		t.Fatalf("chat remove payload=%+v err=%v", event, err)
	}
}

func readPayload(t *testing.T, client *Client) []byte {
	t.Helper()
	select {
	case payload := <-client.send:
		return payload
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for payload")
		return nil
	}
}

func assertNoPayload(t *testing.T, client *Client) {
	t.Helper()
	select {
	case payload := <-client.send:
		t.Fatalf("unexpected payload: %s", payload)
	case <-time.After(20 * time.Millisecond):
	}
}

func drainClient(client *Client) {
	for {
		select {
		case <-client.send:
		default:
			return
		}
	}
}

func hubBarrier(t *testing.T, hub *Hub, client *Client) {
	t.Helper()
	leaseID, _, err := hub.AcquireSessionOperation(client.sessionKey, client.id)
	if err != nil {
		t.Fatalf("acquire barrier lease: %v", err)
	}
	if err := hub.CompleteSessionOperation(leaseID, nil); err != nil {
		t.Fatalf("complete barrier lease: %v", err)
	}
}

func TestHubRoutesCreatedAndIdempotentDeliveriesWithoutDuplicates(t *testing.T) {
	hub := startTestHub(t)
	origin := testClient("origin", 1, "sender-session", 0)
	sibling := testClient("sibling", 1, "sender-session", 0)
	recipient := testClient("recipient", 2, "recipient-session", 0)
	registerTestClient(t, hub, origin)
	registerTestClient(t, hub, sibling)
	registerTestClient(t, hub, recipient)
	drainClient(origin)
	drainClient(sibling)
	drainClient(recipient)

	leaseID, _, err := hub.AcquireSessionOperation(origin.sessionKey, origin.id)
	if err != nil {
		t.Fatalf("acquire operation: %v", err)
	}
	if err := hub.CompleteSessionOperation(leaseID, &Delivery{
		Created: true, RecipientUserIDs: []int64{2, 2, 1},
		AckPayload: []byte("ack"), SenderBroadcastPayload: []byte("sender"),
		RecipientBroadcastPayload: []byte("recipient"),
	}); err != nil {
		t.Fatalf("complete created operation: %v", err)
	}
	if got := string(readPayload(t, origin)); got != "ack" {
		t.Fatalf("origin payload=%q", got)
	}
	if got := string(readPayload(t, sibling)); got != "sender" {
		t.Fatalf("sibling payload=%q", got)
	}
	if got := string(readPayload(t, recipient)); got != "recipient" {
		t.Fatalf("recipient payload=%q", got)
	}
	assertNoPayload(t, origin)
	assertNoPayload(t, sibling)
	assertNoPayload(t, recipient)

	leaseID, _, err = hub.AcquireSessionOperation(origin.sessionKey, origin.id)
	if err != nil {
		t.Fatalf("acquire duplicate operation: %v", err)
	}
	if err := hub.CompleteSessionOperation(leaseID, &Delivery{
		Created: false, RecipientUserIDs: []int64{2},
		AckPayload: []byte("existing"), SenderBroadcastPayload: []byte("must-not-send"),
		RecipientBroadcastPayload: []byte("must-not-send"),
	}); err != nil {
		t.Fatalf("complete duplicate operation: %v", err)
	}
	if got := string(readPayload(t, origin)); got != "existing" {
		t.Fatalf("duplicate origin payload=%q", got)
	}
	assertNoPayload(t, sibling)
	assertNoPayload(t, recipient)
}

func TestHubPublishesNotificationPayloadToEveryActiveSocketOfRecipients(t *testing.T) {
	hub := startTestHub(t)
	firstTab := testClient("first-tab", 1, "first-session", 0)
	secondTab := testClient("second-tab", 1, "second-session", 0)
	otherUser := testClient("other-user", 2, "other-session", 0)
	registerTestClient(t, hub, firstTab)
	registerTestClient(t, hub, secondTab)
	registerTestClient(t, hub, otherUser)
	drainClient(firstTab)
	drainClient(secondTab)
	drainClient(otherUser)

	if err := hub.PublishUsers(map[int64][]byte{1: []byte(`{"type":"notification:upsert"}`)}); err != nil {
		t.Fatalf("publish users: %v", err)
	}
	if got := string(readPayload(t, firstTab)); got != `{"type":"notification:upsert"}` {
		t.Fatalf("first tab payload=%q", got)
	}
	if got := string(readPayload(t, secondTab)); got != `{"type":"notification:upsert"}` {
		t.Fatalf("second tab payload=%q", got)
	}
	assertNoPayload(t, otherUser)
}

func TestHubRevocationSuppressesInFlightCompletionAndOnlyTargetsOneSession(t *testing.T) {
	hub := startTestHub(t)
	revoked := testClient("revoked", 1, "first-session", 0)
	otherSession := testClient("other", 1, "second-session", 0)
	recipient := testClient("recipient", 2, "recipient-session", 0)
	registerTestClient(t, hub, revoked)
	registerTestClient(t, hub, otherSession)
	registerTestClient(t, hub, recipient)
	drainClient(revoked)
	drainClient(otherSession)
	drainClient(recipient)

	leaseID, operationContext, err := hub.AcquireSessionOperation(revoked.sessionKey, revoked.id)
	if err != nil {
		t.Fatalf("acquire operation: %v", err)
	}
	if err := hub.RevokeSession(revoked.sessionKey); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	select {
	case <-operationContext.Done():
		t.Fatal("logout must not cancel an already acquired SQL operation")
	default:
	}
	if err := hub.CompleteSessionOperation(leaseID, &Delivery{
		Created: true, RecipientUserIDs: []int64{2},
		AckPayload: []byte("ack"), SenderBroadcastPayload: []byte("sender"),
		RecipientBroadcastPayload: []byte("recipient"),
	}); err != nil {
		t.Fatalf("complete revoked operation: %v", err)
	}
	assertNoPayload(t, revoked)
	assertNoPayload(t, recipient)

	if _, _, err := hub.AcquireSessionOperation(revoked.sessionKey, revoked.id); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("revoked client acquire: %v", err)
	}
	leaseID, _, err = hub.AcquireSessionOperation(otherSession.sessionKey, otherSession.id)
	if err != nil {
		t.Fatalf("other browser session was revoked: %v", err)
	}
	if err := hub.CompleteSessionOperation(leaseID, nil); err != nil {
		t.Fatalf("complete other session operation: %v", err)
	}
}

func TestDeliverySurvivesOriginDisconnectAndSkipsRevokedRecipientSession(t *testing.T) {
	hub := startTestHub(t)
	origin := testClient("origin", 1, "sender-session", 0)
	sibling := testClient("sibling", 1, "sender-session", 0)
	revokedRecipient := testClient("revoked-recipient", 2, "revoked-recipient-session", 0)
	activeRecipient := testClient("active-recipient", 3, "active-recipient-session", 0)
	registerTestClient(t, hub, origin)
	registerTestClient(t, hub, sibling)
	registerTestClient(t, hub, revokedRecipient)
	registerTestClient(t, hub, activeRecipient)
	drainClient(origin)
	drainClient(sibling)
	drainClient(revokedRecipient)
	drainClient(activeRecipient)

	leaseID, _, err := hub.AcquireSessionOperation(origin.sessionKey, origin.id)
	if err != nil {
		t.Fatalf("acquire operation: %v", err)
	}
	hub.Unregister(origin.id)
	hubBarrier(t, hub, sibling)
	if err := hub.RevokeSession(revokedRecipient.sessionKey); err != nil {
		t.Fatalf("revoke recipient session: %v", err)
	}
	if err := hub.CompleteSessionOperation(leaseID, &Delivery{
		Created: true, RecipientUserIDs: []int64{2, 3}, AckPayload: []byte("lost-origin-ack"),
		SenderBroadcastPayload: []byte("sender"), RecipientBroadcastPayload: []byte("recipient"),
	}); err != nil {
		t.Fatalf("complete operation: %v", err)
	}
	if got := string(readPayload(t, sibling)); got != "sender" {
		t.Fatalf("sibling payload=%q", got)
	}
	if got := string(readPayload(t, activeRecipient)); got != "recipient" {
		t.Fatalf("active recipient payload=%q", got)
	}
	assertNoPayload(t, origin)
	assertNoPayload(t, revokedRecipient)
}

func TestAcquireValidatesOriginSessionAndCompletionConsumesLeaseOnce(t *testing.T) {
	hub := startTestHub(t)
	first := testClient("first", 1, "first-session", 0)
	second := testClient("second", 1, "second-session", 0)
	registerTestClient(t, hub, first)
	registerTestClient(t, hub, second)
	drainClient(first)
	drainClient(second)

	if _, _, err := hub.AcquireSessionOperation(first.sessionKey, second.id); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("mismatched session/origin error=%v", err)
	}
	if _, _, err := hub.AcquireSessionOperation(first.sessionKey, "missing-client"); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("missing origin error=%v", err)
	}
	leaseID, _, err := hub.AcquireSessionOperation(first.sessionKey, first.id)
	if err != nil {
		t.Fatalf("valid acquire: %v", err)
	}
	if err := hub.CompleteSessionOperation(leaseID, nil); err != nil {
		t.Fatalf("first completion: %v", err)
	}
	if err := hub.CompleteSessionOperation(leaseID, &Delivery{Created: true, AckPayload: []byte("must-not-send")}); err == nil {
		t.Fatal("second completion unexpectedly succeeded")
	}
	assertNoPayload(t, first)
}

func TestOrdinaryDisconnectKeepsSiblingSocketSessionActive(t *testing.T) {
	hub := startTestHub(t)
	first := testClient("first-tab", 1, "shared-session", 0)
	second := testClient("second-tab", 1, "shared-session", 0)
	registerTestClient(t, hub, first)
	registerTestClient(t, hub, second)
	drainClient(first)
	drainClient(second)

	hub.Unregister(first.id)
	leaseID, _, err := hub.AcquireSessionOperation(second.sessionKey, second.id)
	if err != nil {
		t.Fatalf("sibling acquire after ordinary disconnect: %v", err)
	}
	if err := hub.CompleteSessionOperation(leaseID, nil); err != nil {
		t.Fatalf("complete sibling operation: %v", err)
	}
	select {
	case <-first.done:
	default:
		t.Fatal("disconnected client was not closed")
	}
	select {
	case <-second.done:
		t.Fatal("ordinary disconnect closed sibling socket")
	default:
	}
}

func TestPresenceUsesFirstAndLastConnectionTransitions(t *testing.T) {
	hub := startTestHub(t)
	first := testClient("first-user", 1, "first-session", 0)
	secondTabOne := testClient("second-one", 2, "second-session", 0)
	secondTabTwo := testClient("second-two", 2, "second-session", 0)
	registerTestClient(t, hub, first, 2)
	drainClient(first)
	registerTestClient(t, hub, secondTabOne, 1)
	drainClient(secondTabOne)

	var update presenceUpdateEnvelope
	if err := json.Unmarshal(readPayload(t, first), &update); err != nil || update.UserID != 2 || !update.Online {
		t.Fatalf("first online transition: payload=%+v err=%v", update, err)
	}

	registerTestClient(t, hub, secondTabTwo, 1)
	drainClient(secondTabTwo)
	assertNoPayload(t, first)
	hub.Unregister(secondTabOne.id)
	hubBarrier(t, hub, first)
	assertNoPayload(t, first)
	hub.Unregister(secondTabTwo.id)
	hubBarrier(t, hub, first)
	if err := json.Unmarshal(readPayload(t, first), &update); err != nil || update.UserID != 2 || update.Online {
		t.Fatalf("last offline transition: payload=%+v err=%v", update, err)
	}
}

func TestTypingAggregatesMultipleTabsAndStopsAfterLastTab(t *testing.T) {
	hub := startTestHub(t)
	firstTab := testClient("typing-one", 1, "typing-session", 0)
	secondTab := testClient("typing-two", 1, "typing-session", 0)
	recipient := testClient("typing-recipient", 2, "recipient-session", 0)
	registerTestClient(t, hub, firstTab, 2)
	registerTestClient(t, hub, secondTab, 2)
	registerTestClient(t, hub, recipient, 1)
	drainClient(firstTab)
	drainClient(secondTab)
	drainClient(recipient)
	chat := domain.ChatRef{Kind: domain.ChatDirect, TargetID: 2}

	if err := hub.QueueTyping(firstTab.id, chat, []int64{2}, TypingStart); err != nil {
		t.Fatalf("first typing start: %v", err)
	}
	hubBarrier(t, hub, recipient)
	var event typingEnvelope
	if err := json.Unmarshal(readPayload(t, recipient), &event); err != nil || !event.Typing || event.User.ID != 1 {
		t.Fatalf("typing start: payload=%+v err=%v", event, err)
	}
	if err := hub.QueueTyping(secondTab.id, chat, []int64{2}, TypingStart); err != nil {
		t.Fatalf("second typing start: %v", err)
	}
	hubBarrier(t, hub, recipient)
	assertNoPayload(t, recipient)

	if err := hub.QueueTyping(firstTab.id, chat, nil, TypingStop); err != nil {
		t.Fatalf("first typing stop: %v", err)
	}
	hubBarrier(t, hub, recipient)
	assertNoPayload(t, recipient)
	if err := hub.QueueTyping(secondTab.id, chat, nil, TypingStop); err != nil {
		t.Fatalf("second typing stop: %v", err)
	}
	hubBarrier(t, hub, recipient)
	if err := json.Unmarshal(readPayload(t, recipient), &event); err != nil || event.Typing {
		t.Fatalf("typing stop: payload=%+v err=%v", event, err)
	}
}

func TestRelationshipRevocationSuppressesAlreadyAuthorizedDeliveryAndTyping(t *testing.T) {
	hub := startTestHub(t)
	sender := testClient("sender", 1, "sender-session", 0)
	recipient := testClient("recipient", 2, "recipient-session", 0)
	registerTestClient(t, hub, sender, 2)
	registerTestClient(t, hub, recipient, 1)
	drainClient(sender)
	drainClient(recipient)

	leaseID, _, err := hub.AcquireSessionOperation(sender.sessionKey, sender.id)
	if err != nil {
		t.Fatalf("acquire operation: %v", err)
	}
	hub.RelationshipChanged(1, 2, false)
	drainClient(sender)
	drainClient(recipient)
	if err := hub.CompleteSessionOperation(leaseID, &Delivery{
		Created: true, Chat: domain.ChatRef{Kind: domain.ChatDirect, TargetID: 2},
		RecipientUserIDs: []int64{2}, AckPayload: []byte("ack"),
		SenderBroadcastPayload: []byte("sender"), RecipientBroadcastPayload: []byte("recipient"),
	}); err != nil {
		t.Fatalf("complete stale direct delivery: %v", err)
	}
	assertNoPayload(t, sender)
	assertNoPayload(t, recipient)

	if err := hub.QueueTyping(sender.id, domain.ChatRef{Kind: domain.ChatDirect, TargetID: 2}, []int64{2}, TypingStart); err != nil {
		t.Fatalf("queue stale direct typing: %v", err)
	}
	hubBarrier(t, hub, sender)
	assertNoPayload(t, recipient)
}

func TestPresenceSyncRejectsSnapshotsOlderThanRelationshipMutation(t *testing.T) {
	t.Run("accept wins over stale empty snapshot", func(t *testing.T) {
		hub := startTestHub(t)
		recipient := testClient("recipient", 2, "recipient-session", 0)
		registerTestClient(t, hub, recipient)
		drainClient(recipient)

		generation, err := hub.BeginPresenceSync(1)
		if err != nil {
			t.Fatalf("begin stale presence sync: %v", err)
		}
		hub.RelationshipChanged(1, 2, true)
		drainClient(recipient)
		sender := testClient("sender", 1, "sender-session", 0)
		sender.hub = hub
		if err := hub.Register(sender, generation, nil); err != nil {
			t.Fatalf("register with stale snapshot: %v", err)
		}
		var initial presenceInitEnvelope
		if err := json.Unmarshal(readPayload(t, sender), &initial); err != nil ||
			len(initial.OnlineUserIDs) != 1 || initial.OnlineUserIDs[0] != 2 {
			t.Fatalf("presence init=%+v err=%v", initial, err)
		}
		drainClient(recipient)

		leaseID, _, err := hub.AcquireSessionOperation(sender.sessionKey, sender.id)
		if err != nil {
			t.Fatalf("acquire direct operation: %v", err)
		}
		if err := hub.CompleteSessionOperation(leaseID, &Delivery{
			Created: true, Chat: domain.ChatRef{Kind: domain.ChatDirect, TargetID: 2},
			RecipientUserIDs: []int64{2}, AckPayload: []byte("ack"),
			RecipientBroadcastPayload: []byte("recipient"),
		}); err != nil {
			t.Fatalf("complete direct operation: %v", err)
		}
		if got := string(readPayload(t, sender)); got != "ack" {
			t.Fatalf("sender ack=%q", got)
		}
		if got := string(readPayload(t, recipient)); got != "recipient" {
			t.Fatalf("recipient payload=%q", got)
		}
	})

	t.Run("unfollow wins over stale eligible snapshot", func(t *testing.T) {
		hub := startTestHub(t)
		hub.RelationshipChanged(1, 2, true)
		generation, err := hub.BeginPresenceSync(1)
		if err != nil {
			t.Fatalf("begin stale presence sync: %v", err)
		}
		hub.RelationshipChanged(1, 2, false)
		sender := testClient("sender", 1, "sender-session", 0)
		recipient := testClient("recipient", 2, "recipient-session", 0)
		sender.hub = hub
		if err := hub.Register(sender, generation, []int64{2}); err != nil {
			t.Fatalf("register with stale snapshot: %v", err)
		}
		registerTestClient(t, hub, recipient)
		drainClient(sender)
		drainClient(recipient)

		leaseID, _, err := hub.AcquireSessionOperation(sender.sessionKey, sender.id)
		if err != nil {
			t.Fatalf("acquire direct operation: %v", err)
		}
		if err := hub.CompleteSessionOperation(leaseID, &Delivery{
			Created: true, Chat: domain.ChatRef{Kind: domain.ChatDirect, TargetID: 2},
			RecipientUserIDs: []int64{2}, AckPayload: []byte("must-not-send"),
			RecipientBroadcastPayload: []byte("must-not-send"),
		}); err != nil {
			t.Fatalf("complete stale direct operation: %v", err)
		}
		assertNoPayload(t, sender)
		assertNoPayload(t, recipient)
	})
}

func TestGroupAccessRevocationSuppressesStaleSenderAndRecipientDelivery(t *testing.T) {
	hub := startTestHub(t)
	sender := testClient("group-sender", 1, "sender-session", 0)
	recipient := testClient("group-recipient", 2, "recipient-session", 0)
	recipientSibling := testClient("group-recipient-sibling", 2, "recipient-second-session", 0)
	registerTestClient(t, hub, sender)
	registerTestClient(t, hub, recipient)
	registerTestClient(t, hub, recipientSibling)
	drainClient(sender)
	drainClient(recipient)
	drainClient(recipientSibling)
	chat := domain.ChatRef{Kind: domain.ChatGroup, TargetID: 7}

	leaseID, _, err := hub.AcquireSessionOperation(sender.sessionKey, sender.id)
	if err != nil {
		t.Fatalf("acquire operation: %v", err)
	}
	hub.GroupAccessChanged(7, 2, false)
	assertChatRemove(t, recipient, domain.ChatGroup, 7)
	assertChatRemove(t, recipientSibling, domain.ChatGroup, 7)
	if err := hub.CompleteSessionOperation(leaseID, &Delivery{
		Created: true, Chat: chat, RecipientUserIDs: []int64{2},
		AckPayload: []byte("ack"), RecipientBroadcastPayload: []byte("recipient"),
	}); err != nil {
		t.Fatalf("complete after recipient leave: %v", err)
	}
	if got := string(readPayload(t, sender)); got != "ack" {
		t.Fatalf("active sender ack=%q", got)
	}
	assertNoPayload(t, recipient)
	assertNoPayload(t, recipientSibling)

	hub.GroupAccessChanged(7, 2, true)
	leaseID, _, err = hub.AcquireSessionOperation(sender.sessionKey, sender.id)
	if err != nil {
		t.Fatalf("acquire second operation: %v", err)
	}
	hub.GroupAccessChanged(7, 1, false)
	assertChatRemove(t, sender, domain.ChatGroup, 7)
	if err := hub.CompleteSessionOperation(leaseID, &Delivery{
		Created: true, Chat: chat, RecipientUserIDs: []int64{2},
		AckPayload: []byte("must-not-send"), RecipientBroadcastPayload: []byte("must-not-send"),
	}); err != nil {
		t.Fatalf("complete after sender leave: %v", err)
	}
	assertNoPayload(t, sender)
	assertNoPayload(t, recipient)

	if err := hub.QueueTyping(sender.id, chat, []int64{2}, TypingStart); err != nil {
		t.Fatalf("queue stale group typing: %v", err)
	}
	hubBarrier(t, hub, recipient)
	assertNoPayload(t, recipient)
}

func TestSlowClientIsEvictedWithoutBlockingHub(t *testing.T) {
	hub := startTestHub(t)
	slow := testClient("slow", 1, "slow-session", 1)
	registerTestClient(t, hub, slow)
	// The presence:init payload intentionally fills the one-slot queue.
	if err := hub.SendClient(slow.id, []byte("next")); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("slow client delivery error=%v", err)
	}
	select {
	case <-slow.done:
	case <-time.After(time.Second):
		t.Fatal("slow client was not closed")
	}
}

func TestConnectionLimitRejectsNinthSocketWithoutClosingExistingClients(t *testing.T) {
	hub := startTestHub(t)
	clients := make([]*Client, 0, MaxConnectionsPerUser)
	for index := 0; index < MaxConnectionsPerUser; index++ {
		client := testClient("client-"+string(rune('a'+index)), 1, "session", 0)
		registerTestClient(t, hub, client)
		clients = append(clients, client)
	}
	ninth := testClient("ninth", 1, "session", 0)
	ninth.hub = hub
	generation, err := hub.BeginPresenceSync(ninth.userID)
	if err != nil {
		t.Fatalf("begin ninth presence sync: %v", err)
	}
	if err := hub.Register(ninth, generation, nil); !errors.Is(err, ErrConnectionLimit) {
		t.Fatalf("ninth registration error=%v", err)
	}
	for _, client := range clients {
		select {
		case <-client.done:
			t.Fatal("connection limit closed an existing client")
		default:
		}
	}
}

func TestDrainWaitsForLeaseCompletionAndRejectsNewOperations(t *testing.T) {
	hub := startTestHub(t)
	client := testClient("drain", 1, "drain-session", 0)
	registerTestClient(t, hub, client)
	drainClient(client)
	leaseID, operationContext, err := hub.AcquireSessionOperation(client.sessionKey, client.id)
	if err != nil {
		t.Fatalf("acquire operation: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := hub.BeginDrain(ctx); err != nil {
		t.Fatalf("begin drain: %v", err)
	}
	select {
	case <-operationContext.Done():
	default:
		t.Fatal("drain did not cancel in-flight service context")
	}
	if _, _, err := hub.AcquireSessionOperation(client.sessionKey, client.id); !errors.Is(err, ErrHubDraining) {
		t.Fatalf("new operation during drain: %v", err)
	}
	select {
	case <-hub.Done():
		t.Fatal("hub stopped before lease completion")
	default:
	}
	if err := hub.CompleteSessionOperation(leaseID, nil); err != nil {
		t.Fatalf("complete draining operation: %v", err)
	}
	select {
	case <-hub.Done():
	case <-time.After(time.Second):
		t.Fatal("hub did not stop after final lease completion")
	}
}

func TestExpiryRevokesSessionAndTypingTTLStopsIndicator(t *testing.T) {
	base := time.Now().UTC()
	var nowUnix atomic.Int64
	nowUnix.Store(base.UnixNano())
	hub := NewHubWithNow(log.New(io.Discard, "", 0), func() time.Time {
		return time.Unix(0, nowUnix.Load()).UTC()
	})
	go hub.Run()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = hub.BeginDrain(ctx)
	})

	typingClient := testClient("typing-expiry", 1, "typing-expiry-session", 0)
	typingClient.expiresAt = base.Add(time.Hour)
	recipient := testClient("typing-expiry-recipient", 2, "recipient-session", 0)
	recipient.expiresAt = base.Add(time.Hour)
	registerTestClient(t, hub, typingClient, 2)
	registerTestClient(t, hub, recipient, 1)
	drainClient(typingClient)
	drainClient(recipient)
	chat := domain.ChatRef{Kind: domain.ChatDirect, TargetID: 2}
	if err := hub.QueueTyping(typingClient.id, chat, []int64{2}, TypingStart); err != nil {
		t.Fatalf("typing start: %v", err)
	}
	hubBarrier(t, hub, recipient)
	var typing typingEnvelope
	if err := json.Unmarshal(readPayload(t, recipient), &typing); err != nil || !typing.Typing {
		t.Fatalf("typing start payload=%+v err=%v", typing, err)
	}

	nowUnix.Store(base.Add(typingTTL + time.Second).UnixNano())
	deadline := time.After(2 * time.Second)
	for {
		select {
		case payload := <-recipient.send:
			if json.Unmarshal(payload, &typing) == nil && typing.Type == "typing:update" && !typing.Typing {
				goto typingExpired
			}
		case <-deadline:
			t.Fatal("typing indicator did not expire")
		}
	}

typingExpired:
	nowUnix.Store(base.Add(2 * time.Hour).UnixNano())
	if _, _, err := hub.AcquireSessionOperation(typingClient.sessionKey, typingClient.id); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("expired session acquire error=%v", err)
	}
	select {
	case <-typingClient.done:
	case <-time.After(time.Second):
		t.Fatal("expired session client was not closed")
	}
}

func TestHubPublicMethodsReturnImmediatelyAfterStop(t *testing.T) {
	hub := NewHub(log.New(io.Discard, "", 0))
	go hub.Run()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := hub.BeginDrain(ctx); err != nil {
		t.Fatalf("stop empty hub: %v", err)
	}
	select {
	case <-hub.Done():
	case <-ctx.Done():
		t.Fatal("empty hub did not stop")
	}

	client := testClient("post-stop", 1, "post-stop-session", 0)
	client.hub = hub
	if err := hub.Register(client, 0, nil); !errors.Is(err, ErrHubStopped) {
		t.Fatalf("post-stop register error=%v", err)
	}
	if _, _, err := hub.AcquireSessionOperation(client.sessionKey, client.id); !errors.Is(err, ErrHubStopped) {
		t.Fatalf("post-stop acquire error=%v", err)
	}
	if err := hub.CompleteSessionOperation(1, nil); !errors.Is(err, ErrHubStopped) {
		t.Fatalf("post-stop completion error=%v", err)
	}
	if err := hub.RevokeSession(client.sessionKey); !errors.Is(err, ErrHubStopped) {
		t.Fatalf("post-stop revoke error=%v", err)
	}
	client.unregister()
	select {
	case <-client.done:
	default:
		t.Fatal("post-stop unregister did not close local client resources")
	}
}
