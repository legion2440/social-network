package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"social-network/backend/internal/config"
	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"

	"github.com/gorilla/websocket"
)

func dialTestWebSocket(t *testing.T, serverURL, token string) *websocket.Conn {
	t.Helper()
	header := http.Header{}
	header.Set("Origin", serverURL)
	header.Set("Cookie", (&http.Cookie{Name: config.SessionCookieName, Value: token}).String())
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/ws"
	connection, response, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("dial websocket: status=%d err=%v", status, err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	return connection
}

func readWebSocketEvent(t *testing.T, connection *websocket.Conn, eventType string) map[string]any {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set websocket deadline: %v", err)
	}
	for range 12 {
		_, payload, err := connection.ReadMessage()
		if err != nil {
			t.Fatalf("read websocket event %s: %v", eventType, err)
		}
		var event map[string]any
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Fatalf("decode websocket payload %q: %v", payload, err)
		}
		if event["type"] == eventType {
			return event
		}
	}
	t.Fatalf("websocket event %s was not received", eventType)
	return nil
}

func TestWebSocketPersistsRoutesAndRevokesOnlyLogoutSession(t *testing.T) {
	env := newTestEnvironment(t)
	firstID, firstToken := env.createUserAndSession(t, "ws-first@example.com")
	secondID, secondToken := env.createUserAndSession(t, "ws-second@example.com")
	if _, err := sqlite.NewFollowRepo(env.db).Upsert(
		context.Background(), firstID, secondID, domain.FollowAccepted, testNow,
	); err != nil {
		t.Fatalf("create accepted follow: %v", err)
	}
	server := httptest.NewServer(env.handler)
	defer server.Close()

	origin := dialTestWebSocket(t, server.URL, firstToken)
	readWebSocketEvent(t, origin, "presence:init")
	sibling := dialTestWebSocket(t, server.URL, firstToken)
	readWebSocketEvent(t, sibling, "presence:init")
	recipient := dialTestWebSocket(t, server.URL, secondToken)
	readWebSocketEvent(t, recipient, "presence:init")

	clientMessageID := "47cd9266-b43f-4a89-9338-4f9c197ff12a"
	if err := origin.WriteJSON(map[string]any{
		"type":              "chat:send",
		"client_message_id": clientMessageID,
		"chat":              map[string]any{"kind": "direct", "target_id": secondID},
		"text":              "realtime direct message",
	}); err != nil {
		t.Fatalf("send websocket message: %v", err)
	}
	originEvent := readWebSocketEvent(t, origin, "chat:message")
	siblingEvent := readWebSocketEvent(t, sibling, "chat:message")
	recipientEvent := readWebSocketEvent(t, recipient, "chat:message")
	for name, event := range map[string]map[string]any{
		"origin": originEvent, "sibling": siblingEvent, "recipient": recipientEvent,
	} {
		if event["client_message_id"] != clientMessageID {
			t.Fatalf("%s client_message_id=%v", name, event["client_message_id"])
		}
		message, ok := event["message"].(map[string]any)
		if !ok || message["body"] != "realtime direct message" {
			t.Fatalf("%s message=%+v", name, event["message"])
		}
		sender, ok := message["sender"].(map[string]any)
		if !ok || sender["avatar_url"] != domain.NeutralAvatarPlaceholderURL {
			t.Fatalf("%s websocket sender=%+v", name, sender)
		}
	}
	originTarget := originEvent["message"].(map[string]any)["chat"].(map[string]any)["target_id"]
	recipientTarget := recipientEvent["message"].(map[string]any)["chat"].(map[string]any)["target_id"]
	if originTarget != float64(secondID) || recipientTarget != float64(firstID) {
		t.Fatalf("viewer-relative direct targets: origin=%v recipient=%v", originTarget, recipientTarget)
	}
	var rows int
	if err := env.db.QueryRow("SELECT COUNT(*) FROM chat_messages WHERE client_message_id = ?", clientMessageID).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("persisted websocket rows=%d err=%v", rows, err)
	}

	otherSession, err := env.sessions.Create(context.Background(), firstID)
	if err != nil {
		t.Fatalf("create second browser session: %v", err)
	}
	otherSocket := dialTestWebSocket(t, server.URL, otherSession.Token)
	readWebSocketEvent(t, otherSocket, "presence:init")
	logoutRecorder := httptest.NewRecorder()
	env.handler.ServeHTTP(logoutRecorder, authenticatedRequest(http.MethodPost, "/api/auth/logout", firstToken, nil))
	if logoutRecorder.Code != http.StatusNoContent {
		t.Fatalf("logout: status=%d body=%q", logoutRecorder.Code, logoutRecorder.Body.String())
	}
	for name, connection := range map[string]*websocket.Conn{"origin": origin, "sibling": sibling} {
		_ = connection.SetReadDeadline(time.Now().Add(time.Second))
		if _, _, err := connection.ReadMessage(); err == nil {
			t.Fatalf("%s socket remained open after session logout", name)
		}
	}

	secondClientMessageID := "a721f6be-dabe-4688-b06c-d72039d68391"
	if err := otherSocket.WriteJSON(map[string]any{
		"type":              "chat:send",
		"client_message_id": secondClientMessageID,
		"chat":              map[string]any{"kind": "direct", "target_id": secondID},
		"text":              "other browser session remains active",
	}); err != nil {
		t.Fatalf("send from other session: %v", err)
	}
	if event := readWebSocketEvent(t, otherSocket, "chat:message"); event["client_message_id"] != secondClientMessageID {
		t.Fatalf("other session ack=%+v", event)
	}
	if event := readWebSocketEvent(t, recipient, "chat:message"); event["client_message_id"] != secondClientMessageID {
		t.Fatalf("recipient second message=%+v", event)
	}
}

func TestGroupLeaveBroadcastsChatRemoveToEveryActiveUserSocket(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, _ := env.createUserAndSession(t, "ws-group-owner@example.com")
	memberID, memberToken := env.createUserAndSession(t, "ws-group-member@example.com")
	groups := service.NewGroupService(sqlite.NewTransactionManager(env.db), fixedClock{})
	group, err := groups.Create(context.Background(), ownerID, "Realtime removal", "Cross-tab access revoke")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := groups.Invite(context.Background(), ownerID, group.ID, memberID); err != nil {
		t.Fatalf("invite member: %v", err)
	}
	if _, err := groups.AcceptInvitation(context.Background(), memberID, group.ID); err != nil {
		t.Fatalf("accept invitation: %v", err)
	}

	server := httptest.NewServer(env.handler)
	defer server.Close()
	first := dialTestWebSocket(t, server.URL, memberToken)
	readWebSocketEvent(t, first, "presence:init")
	second := dialTestWebSocket(t, server.URL, memberToken)
	readWebSocketEvent(t, second, "presence:init")

	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, authenticatedRequest(
		http.MethodDelete, "/api/groups/"+strconv.FormatInt(group.ID, 10)+"/membership", memberToken, nil,
	))
	if recorder.Code != http.StatusOK {
		t.Fatalf("leave group: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	for name, connection := range map[string]*websocket.Conn{"first": first, "second": second} {
		event := readWebSocketEvent(t, connection, "chat:remove")
		chat, ok := event["chat"].(map[string]any)
		if !ok || chat["kind"] != "group" || chat["target_id"] != float64(group.ID) {
			t.Fatalf("%s chat remove=%+v", name, event)
		}
	}
}

func TestWebSocketNinthConnectionClosesWithPolicyViolation(t *testing.T) {
	env := newTestEnvironment(t)
	_, token := env.createUserAndSession(t, "ws-limit@example.com")
	server := httptest.NewServer(env.handler)
	defer server.Close()

	for range 8 {
		connection := dialTestWebSocket(t, server.URL, token)
		readWebSocketEvent(t, connection, "presence:init")
	}
	ninth := dialTestWebSocket(t, server.URL, token)
	if err := ninth.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set ninth deadline: %v", err)
	}
	_, _, err := ninth.ReadMessage()
	closeError, ok := err.(*websocket.CloseError)
	if !ok || closeError.Code != websocket.ClosePolicyViolation {
		t.Fatalf("ninth close error=%v", err)
	}
}

func TestWebSocketMalformedEventReturnsErrorWithoutClosingConnection(t *testing.T) {
	env := newTestEnvironment(t)
	_, token := env.createUserAndSession(t, "ws-malformed@example.com")
	server := httptest.NewServer(env.handler)
	defer server.Close()
	connection := dialTestWebSocket(t, server.URL, token)
	readWebSocketEvent(t, connection, "presence:init")

	if err := connection.WriteMessage(websocket.TextMessage, []byte("{\"type\":\"chat:send\",\"extra\":true}")); err != nil {
		t.Fatalf("write malformed event: %v", err)
	}
	if event := readWebSocketEvent(t, connection, "chat:error"); event["code"] != "invalid_event" {
		t.Fatalf("malformed event response=%+v", event)
	}
	if err := connection.WriteMessage(websocket.TextMessage, []byte("{\"type\":\"unknown\"}")); err != nil {
		t.Fatalf("write unknown event: %v", err)
	}
	if event := readWebSocketEvent(t, connection, "chat:error"); event["code"] != "invalid_event" {
		t.Fatalf("unknown event response=%+v", event)
	}
	if err := connection.WriteMessage(websocket.TextMessage, []byte(`{
		"type":"typing:start","chat":{"kind":"direct","target_id":2,"target_id":3}
	}`)); err != nil {
		t.Fatalf("write duplicate event: %v", err)
	}
	if event := readWebSocketEvent(t, connection, "chat:error"); event["code"] != "invalid_event" {
		t.Fatalf("duplicate event response=%+v", event)
	}
}

func TestWebSocketRejectsOversizedFrame(t *testing.T) {
	env := newTestEnvironment(t)
	_, token := env.createUserAndSession(t, "ws-oversized@example.com")
	server := httptest.NewServer(env.handler)
	defer server.Close()
	connection := dialTestWebSocket(t, server.URL, token)
	readWebSocketEvent(t, connection, "presence:init")

	if err := connection.WriteMessage(websocket.TextMessage, []byte(strings.Repeat("x", 17<<10))); err != nil {
		t.Fatalf("write oversized frame: %v", err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, _, err := connection.ReadMessage()
	closeError, ok := err.(*websocket.CloseError)
	if !ok || closeError.Code != websocket.CloseMessageTooBig {
		t.Fatalf("oversized close error=%v", err)
	}
}
