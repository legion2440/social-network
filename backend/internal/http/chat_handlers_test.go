package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

func chatGET(t *testing.T, env *testEnvironment, path, token string, want int) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		addSessionCookie(request, token)
	}
	env.handler.ServeHTTP(recorder, request)
	if recorder.Code != want {
		t.Fatalf("GET %s: status=%d body=%q want=%d", path, recorder.Code, recorder.Body.String(), want)
	}
	return recorder
}

func TestChatReadHTTPContractAndAuthoritativeResponse(t *testing.T) {
	env := newTestEnvironment(t)
	firstID, firstToken := env.createUserAndSession(t, "chat-read-http-first@example.com")
	secondID, secondToken := env.createUserAndSession(t, "chat-read-http-second@example.com")
	if _, err := sqlite.NewFollowRepo(env.db).Upsert(
		context.Background(), firstID, secondID, domain.FollowAccepted, testNow,
	); err != nil {
		t.Fatalf("create accepted follow: %v", err)
	}
	chats := service.NewChatService(sqlite.NewTransactionManager(env.db), fixedClock{})
	sent, err := chats.Send(context.Background(), firstID, firstToken, service.ChatSendInput{
		ClientMessageID: "bc6f5eaa-a195-4b3c-bad4-c1e017e2c4cc",
		Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: secondID},
		Body:            "read through HTTP",
	})
	if err != nil {
		t.Fatalf("send direct message: %v", err)
	}
	path := "/api/chats/direct/" + strconv.FormatInt(firstID, 10) + "/read"
	body := `{"through_message_id":` + strconv.FormatInt(sent.Message.ID, 10) + `}`
	request := authenticatedRequest(http.MethodPut, path, secondToken, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("mark read: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	var response chatUnreadResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode mark read: %v", err)
	}
	if response.Chat.Kind != domain.ChatDirect || response.Chat.TargetID != firstID ||
		response.ChatUnreadCount != 0 || response.UnreadCount != 0 || response.Revision != 2 ||
		response.ReadThroughMessageID == nil || *response.ReadThroughMessageID != sent.Message.ID {
		t.Fatalf("authoritative mark-read response=%+v", response)
	}

	for _, test := range []struct {
		name        string
		method      string
		path        string
		contentType string
		body        string
		want        int
		allow       string
	}{
		{name: "content type", method: http.MethodPut, path: path, contentType: "text/plain", body: body, want: http.StatusUnsupportedMediaType},
		{name: "duplicate field", method: http.MethodPut, path: path, contentType: "application/json", body: `{"through_message_id":1,"through_message_id":2}`, want: http.StatusBadRequest},
		{name: "unknown field", method: http.MethodPut, path: path, contentType: "application/json", body: `{"through_message_id":1,"extra":true}`, want: http.StatusBadRequest},
		{name: "foreign marker", method: http.MethodPut, path: "/api/chats/direct/" + strconv.FormatInt(firstID, 10) + "/read", contentType: "application/json", body: `{"through_message_id":999999}`, want: http.StatusNotFound},
		{name: "method", method: http.MethodPost, path: path, contentType: "application/json", body: body, want: http.StatusMethodNotAllowed, allow: http.MethodPut},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := authenticatedRequest(test.method, test.path, secondToken, strings.NewReader(test.body))
			if test.contentType != "" {
				req.Header.Set("Content-Type", test.contentType)
			}
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, req)
			if rec.Code != test.want || (test.allow != "" && rec.Header().Get("Allow") != test.allow) {
				t.Fatalf("status=%d allow=%q body=%q", rec.Code, rec.Header().Get("Allow"), rec.Body.String())
			}
		})
	}
}

func TestChatHTTPEdgeCasesAndAccessMatrix(t *testing.T) {
	env := newTestEnvironment(t)
	firstID, firstToken := env.createUserAndSession(t, "chat-http-first@example.com")
	secondID, secondToken := env.createUserAndSession(t, "chat-http-second@example.com")
	outsiderID, outsiderToken := env.createUserAndSession(t, "chat-http-outsider@example.com")

	chatGET(t, env, "/api/chats", "", http.StatusUnauthorized)
	chatGET(t, env, "/api/chats/direct/not-a-number/messages", firstToken, http.StatusBadRequest)
	chatGET(t, env, "/api/chats/direct/"+strconv.FormatInt(firstID, 10)+"/messages", firstToken, http.StatusBadRequest)
	chatGET(t, env, "/api/chats/direct/999999/messages", firstToken, http.StatusNotFound)
	chatGET(t, env, "/api/chats/direct/"+strconv.FormatInt(outsiderID, 10)+"/messages", firstToken, http.StatusForbidden)

	if _, err := sqlite.NewFollowRepo(env.db).Upsert(
		context.Background(), firstID, secondID, domain.FollowAccepted, testNow,
	); err != nil {
		t.Fatalf("create accepted follow: %v", err)
	}
	for _, test := range []struct {
		path  string
		token string
	}{
		{path: "/api/chats/direct/" + strconv.FormatInt(secondID, 10) + "/messages", token: firstToken},
		{path: "/api/chats/direct/" + strconv.FormatInt(firstID, 10) + "/messages", token: secondToken},
	} {
		recorder := chatGET(t, env, test.path, test.token, http.StatusOK)
		var response chatMessagePageResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil || len(response.Messages) != 0 || response.NextCursor != nil {
			t.Fatalf("eligible empty history: response=%+v err=%v", response, err)
		}
	}

	group := createGroupForTest(t, env, firstToken, "HTTP chat group")
	if _, err := env.db.Exec(
		"INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at) VALUES (?, ?, 'member', ?, ?)",
		group.ID, secondID, testNow.Unix(), testNow.Unix(),
	); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := env.db.Exec(
		"INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at) VALUES (?, ?, 'requested', ?, ?)",
		group.ID, outsiderID, testNow.Unix(), testNow.Unix(),
	); err != nil {
		t.Fatalf("insert requested user: %v", err)
	}
	groupPath := "/api/groups/" + strconv.FormatInt(group.ID, 10) + "/chat/messages"
	chatGET(t, env, groupPath, firstToken, http.StatusOK)
	chatGET(t, env, groupPath, secondToken, http.StatusOK)
	chatGET(t, env, groupPath, outsiderToken, http.StatusForbidden)
	chatGET(t, env, "/api/groups/not-a-number/chat/messages", firstToken, http.StatusBadRequest)
	chatGET(t, env, "/api/groups/999999/chat/messages", firstToken, http.StatusNotFound)

	for _, path := range []string{
		"/api/chats?limit=0",
		"/api/chats?limit=51",
		"/api/chats?limit=20&limit=21",
		"/api/chats?cursor=",
		"/api/chats?cursor=bad",
		"/api/chats?unknown=1",
		"/api/chats/direct/" + strconv.FormatInt(secondID, 10) + "/messages?cursor=bad",
		groupPath + "?limit=nope",
	} {
		chatGET(t, env, path, firstToken, http.StatusBadRequest)
	}

	postRecorder := httptest.NewRecorder()
	postRequest := authenticatedRequest(http.MethodPost, "/api/chats", firstToken, nil)
	env.handler.ServeHTTP(postRecorder, postRequest)
	if postRecorder.Code != http.StatusMethodNotAllowed || postRecorder.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("chat list method contract: status=%d allow=%q", postRecorder.Code, postRecorder.Header().Get("Allow"))
	}
}

func TestChatHTTPListAndHistoryReturnRealMessages(t *testing.T) {
	env := newTestEnvironment(t)
	firstID, firstToken := env.createUserAndSession(t, "chat-list-first@example.com")
	secondID, secondToken := env.createUserAndSession(t, "chat-list-second@example.com")
	if _, err := sqlite.NewFollowRepo(env.db).Upsert(
		context.Background(), firstID, secondID, domain.FollowAccepted, testNow,
	); err != nil {
		t.Fatalf("create accepted follow: %v", err)
	}
	group := createGroupForTest(t, env, firstToken, "Listed chat group")
	chats := service.NewChatService(sqlite.NewTransactionManager(env.db), fixedClock{})
	sendResult, err := chats.Send(context.Background(), firstID, firstToken, service.ChatSendInput{
		ClientMessageID: "47cd9266-b43f-4a89-9338-4f9c197ff12a",
		Chat:            domain.ChatRef{Kind: domain.ChatDirect, TargetID: secondID},
		Body:            "persisted HTTP history",
	})
	if err != nil {
		t.Fatalf("send direct message: %v", err)
	}

	recorder := chatGET(t, env, "/api/chats", firstToken, http.StatusOK)
	var list chatPageResponse
	if err := json.NewDecoder(recorder.Body).Decode(&list); err != nil {
		t.Fatalf("decode chat list: %v", err)
	}
	if len(list.Chats) != 2 || list.Chats[0].Kind != domain.ChatDirect || list.Chats[1].Kind != domain.ChatGroup {
		t.Fatalf("unexpected mixed chat list: %+v (group=%d)", list, group.ID)
	}
	if list.Chats[0].TargetID != secondID || list.Chats[0].User == nil || list.Chats[0].User.ID != secondID {
		t.Fatalf("unexpected direct summary: %+v", list.Chats[0])
	}
	if list.Chats[0].LastMessage == nil || list.Chats[0].LastMessage.ID != sendResult.Message.ID {
		t.Fatalf("missing authoritative last message: %+v", list.Chats[0].LastMessage)
	}
	if list.Chats[1].Group == nil || list.Chats[1].Group.ID != group.ID || list.Chats[1].LastMessage != nil {
		t.Fatalf("unexpected group summary: %+v", list.Chats[1])
	}

	historyPath := "/api/chats/direct/" + strconv.FormatInt(secondID, 10) + "/messages"
	recorder = chatGET(t, env, historyPath, firstToken, http.StatusOK)
	var history chatMessagePageResponse
	if err := json.NewDecoder(recorder.Body).Decode(&history); err != nil || len(history.Messages) != 1 {
		t.Fatalf("decode direct history: response=%+v err=%v", history, err)
	}
	message := history.Messages[0]
	if message.ID != sendResult.Message.ID || message.ClientMessageID != sendResult.Message.ClientMessageID ||
		message.Body != "persisted HTTP history" || message.Chat.Kind != domain.ChatDirect ||
		message.Chat.TargetID != secondID || message.Sender.ID != firstID {
		t.Fatalf("unexpected history message: %+v", message)
	}

	recorder = chatGET(t, env, "/api/chats", secondToken, http.StatusOK)
	if err := json.NewDecoder(recorder.Body).Decode(&list); err != nil || len(list.Chats) != 1 ||
		list.Chats[0].TargetID != firstID {
		t.Fatalf("recipient chat list: response=%+v err=%v", list, err)
	}
}
