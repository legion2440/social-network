package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"social-network/backend/internal/domain"
)

func notificationJSONRequest(method, path, token, body string) *http.Request {
	req := authenticatedRequest(method, path, token, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func listNotificationsForTest(t *testing.T, env *testEnvironment, token string) notificationPageResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodGet, "/api/notifications?limit=20", token, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list notifications: status=%d body=%q", rec.Code, rec.Body.String())
	}
	var page notificationPageResponse
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode notifications: %v", err)
	}
	return page
}

func notificationActionForTest(
	t *testing.T,
	env *testEnvironment,
	token string,
	notificationID int64,
	action string,
	want int,
) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	path := "/api/notifications/" + strconv.FormatInt(notificationID, 10) + "/action"
	env.handler.ServeHTTP(rec, notificationJSONRequest(http.MethodPut, path, token, fmt.Sprintf(`{"action":%q}`, action)))
	if rec.Code != want {
		t.Fatalf("notification action %s: status=%d body=%q want=%d", action, rec.Code, rec.Body.String(), want)
	}
	return rec
}

func TestNotificationFollowLifecycleReadActionAndObservableFKUpdate(t *testing.T) {
	env := newTestEnvironment(t)
	targetID, targetToken := env.createUserAndSession(t, "notification-target@example.com")
	followerID, followerToken := env.createUserAndSession(t, "notification-follower@example.com")
	setProfilePrivacy(t, env, targetToken, true)

	followPath := "/api/users/" + strconv.FormatInt(targetID, 10) + "/follow"
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPut, followPath, followerToken, nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"pending"`) {
		t.Fatalf("create follow request: status=%d body=%q", rec.Code, rec.Body.String())
	}

	page := listNotificationsForTest(t, env, targetToken)
	if len(page.Notifications) != 1 || page.UnreadCount != 1 || page.Revision != 1 {
		t.Fatalf("initial notification page: %+v", page)
	}
	requestNotification := page.Notifications[0]
	if requestNotification.Type != domain.NotificationFollowRequest || requestNotification.Actor.ID != followerID || requestNotification.FollowID == nil {
		t.Fatalf("unexpected follow request notification: %+v", requestNotification)
	}

	readPath := "/api/notifications/" + strconv.FormatInt(requestNotification.ID, 10) + "/read"
	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPut, readPath, followerToken, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("foreign mark-read: status=%d body=%q", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPut, readPath, targetToken, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("mark read: status=%d body=%q", rec.Code, rec.Body.String())
	}
	var readResult notificationReadResponse
	if err := json.NewDecoder(rec.Body).Decode(&readResult); err != nil {
		t.Fatalf("decode mark read: %v", err)
	}
	if readResult.UnreadCount != 0 || readResult.Revision != 2 || readResult.Notification.ReadAt == nil {
		t.Fatalf("mark read result: %+v", readResult)
	}
	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPut, readPath, targetToken, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("repeat mark read: status=%d body=%q", rec.Code, rec.Body.String())
	}
	if err := json.NewDecoder(rec.Body).Decode(&readResult); err != nil || readResult.Revision != 2 {
		t.Fatalf("repeat mark read changed revision: %+v err=%v", readResult, err)
	}

	rec = notificationActionForTest(t, env, targetToken, requestNotification.ID, "accept", http.StatusOK)
	var actionResult notificationActionResponse
	if err := json.NewDecoder(rec.Body).Decode(&actionResult); err != nil {
		t.Fatalf("decode accept: %v", err)
	}
	if actionResult.Revision != 3 || actionResult.UnreadCount != 0 || actionResult.Notification.Resolution == nil ||
		*actionResult.Notification.Resolution != domain.NotificationAccepted || actionResult.Source == nil ||
		actionResult.Source.Relationship == nil || !actionResult.Source.Relationship.FollowsMe ||
		actionResult.Source.Relationship.Status != "none" {
		t.Fatalf("accept response: %+v", actionResult)
	}
	rec = notificationActionForTest(t, env, targetToken, requestNotification.ID, "accept", http.StatusOK)
	if err := json.NewDecoder(rec.Body).Decode(&actionResult); err != nil || actionResult.Revision != 3 {
		t.Fatalf("idempotent accept changed revision: %+v err=%v", actionResult, err)
	}
	notificationActionForTest(t, env, targetToken, requestNotification.ID, "decline", http.StatusConflict)

	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodDelete, followPath, followerToken, nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unfollow: status=%d body=%q", rec.Code, rec.Body.String())
	}
	page = listNotificationsForTest(t, env, targetToken)
	if page.Revision != 4 || len(page.Notifications) != 1 || page.Notifications[0].FollowID != nil ||
		page.Notifications[0].Resolution == nil || *page.Notifications[0].Resolution != domain.NotificationAccepted {
		t.Fatalf("follow FK update was not observable: %+v", page)
	}

	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPut, followPath, followerToken, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("new follow lifecycle: status=%d body=%q", rec.Code, rec.Body.String())
	}
	page = listNotificationsForTest(t, env, targetToken)
	if page.Revision != 5 || page.UnreadCount != 1 || len(page.Notifications) != 2 ||
		page.Notifications[0].ID == requestNotification.ID || page.Notifications[0].Resolution != nil {
		t.Fatalf("new follow lifecycle notification: %+v", page)
	}
	notificationActionForTest(t, env, targetToken, requestNotification.ID, "accept", http.StatusOK)
	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodGet, followPath, followerToken, nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"pending"`) {
		t.Fatalf("old action changed new lifecycle: status=%d body=%q", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodDelete, followPath, followerToken, nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("cancel pending follow: status=%d body=%q", rec.Code, rec.Body.String())
	}
	page = listNotificationsForTest(t, env, targetToken)
	if page.Revision != 6 || page.UnreadCount != 0 || page.Notifications[0].FollowID != nil ||
		page.Notifications[0].Resolution == nil || *page.Notifications[0].Resolution != domain.NotificationCancelled {
		t.Fatalf("pending cancellation must make one revision change: %+v", page)
	}
}

func TestNotificationActionStrictHTTPAndNonActionableConflict(t *testing.T) {
	env := newTestEnvironment(t)
	targetID, targetToken := env.createUserAndSession(t, "notification-public@example.com")
	_, followerToken := env.createUserAndSession(t, "notification-public-follower@example.com")
	followPath := "/api/users/" + strconv.FormatInt(targetID, 10) + "/follow"
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPut, followPath, followerToken, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("follow public profile: status=%d body=%q", rec.Code, rec.Body.String())
	}
	page := listNotificationsForTest(t, env, targetToken)
	if len(page.Notifications) != 1 || page.Notifications[0].Type != domain.NotificationFollowStarted {
		t.Fatalf("expected follow_started: %+v", page)
	}
	id := page.Notifications[0].ID
	path := "/api/notifications/" + strconv.FormatInt(id, 10) + "/action"

	for name, request := range map[string]*http.Request{
		"content type": authenticatedRequest(http.MethodPut, path, targetToken, strings.NewReader(`{"action":"accept"}`)),
		"duplicate":    notificationJSONRequest(http.MethodPut, path, targetToken, `{"action":"accept","action":"accept"}`),
		"unknown":      notificationJSONRequest(http.MethodPut, path, targetToken, `{"action":"accept","extra":true}`),
		"trailing":     notificationJSONRequest(http.MethodPut, path, targetToken, `{"action":"accept"}{}`),
		"bad action":   notificationJSONRequest(http.MethodPut, path, targetToken, `{"action":"maybe"}`),
		"query":        notificationJSONRequest(http.MethodPut, path+"?extra=1", targetToken, `{"action":"accept"}`),
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			env.handler.ServeHTTP(recorder, request)
			want := http.StatusBadRequest
			if name == "content type" {
				want = http.StatusUnsupportedMediaType
			}
			if recorder.Code != want {
				t.Fatalf("status=%d body=%q want=%d", recorder.Code, recorder.Body.String(), want)
			}
		})
	}
	rec = notificationActionForTest(t, env, targetToken, id, "accept", http.StatusConflict)
	if rec.Body.String() != `{"error":"conflict"}`+"\n" {
		t.Fatalf("non-actionable error: %q", rec.Body.String())
	}
	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, notificationJSONRequest(http.MethodPost, path, targetToken, `{"action":"accept"}`))
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodPut {
		t.Fatalf("method contract: status=%d allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
}

func TestNotificationGroupInvitationLifecycleIsolation(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "notification-group-owner@example.com")
	inviteeID, inviteeToken := env.createUserAndSession(t, "notification-group-invitee@example.com")
	group := createGroupForTest(t, env, ownerToken, "Notification lifecycle group")
	base := "/api/groups/" + strconv.FormatInt(group.ID, 10)
	invite := func() {
		rec := httptest.NewRecorder()
		env.handler.ServeHTTP(rec, groupJSONRequest(http.MethodPost, base+"/invitations", ownerToken, fmt.Sprintf(`{"user_id":%d}`, inviteeID)))
		if rec.Code != http.StatusOK {
			t.Fatalf("invite: status=%d body=%q", rec.Code, rec.Body.String())
		}
	}
	invite()
	page := listNotificationsForTest(t, env, inviteeToken)
	if len(page.Notifications) != 1 || page.Notifications[0].Type != domain.NotificationGroupInvitation ||
		page.Notifications[0].Actor.ID != ownerID {
		t.Fatalf("first invitation notification: %+v", page)
	}
	firstID := page.Notifications[0].ID
	notificationActionForTest(t, env, inviteeToken, firstID, "decline", http.StatusOK)
	invite()
	page = listNotificationsForTest(t, env, inviteeToken)
	if len(page.Notifications) != 2 || page.Notifications[0].ID == firstID || page.Notifications[0].Resolution != nil {
		t.Fatalf("second invitation lifecycle: %+v", page)
	}
	secondID := page.Notifications[0].ID
	notificationActionForTest(t, env, inviteeToken, firstID, "decline", http.StatusOK)
	var status string
	if err := env.db.QueryRow(`SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?`, group.ID, inviteeID).Scan(&status); err != nil || status != "invited" {
		t.Fatalf("old lifecycle action changed new invitation: status=%q err=%v", status, err)
	}
	rec := notificationActionForTest(t, env, inviteeToken, secondID, "accept", http.StatusOK)
	var actionResult notificationActionResponse
	if err := json.NewDecoder(rec.Body).Decode(&actionResult); err != nil || actionResult.Source == nil ||
		actionResult.Source.Group == nil || actionResult.Source.Group.ViewerStatus != "member" || actionResult.Source.Group.MembersCount != 2 {
		t.Fatalf("accept invitation source: %+v err=%v", actionResult, err)
	}
}

func TestNotificationJoinRequestActionAndGroupEventFanout(t *testing.T) {
	env := newTestEnvironment(t)
	_, ownerToken := env.createUserAndSession(t, "notification-event-owner@example.com")
	memberID, memberToken := env.createUserAndSession(t, "notification-event-member@example.com")
	invitedID, invitedToken := env.createUserAndSession(t, "notification-event-invited@example.com")
	group := createGroupForTest(t, env, ownerToken, "Notification event group")
	base := "/api/groups/" + strconv.FormatInt(group.ID, 10)

	groupMutation(t, env, http.MethodPost, base+"/join-request", memberToken, http.StatusOK)
	ownerPage := listNotificationsForTest(t, env, ownerToken)
	if len(ownerPage.Notifications) != 1 || ownerPage.Notifications[0].Type != domain.NotificationGroupJoinRequest ||
		ownerPage.Notifications[0].Actor.ID != memberID {
		t.Fatalf("join request notification: %+v", ownerPage)
	}
	rec := notificationActionForTest(t, env, ownerToken, ownerPage.Notifications[0].ID, "accept", http.StatusOK)
	var joinAction notificationActionResponse
	if err := json.NewDecoder(rec.Body).Decode(&joinAction); err != nil || joinAction.Source == nil ||
		joinAction.Source.Group == nil || joinAction.Source.Group.ViewerStatus != "owner" ||
		joinAction.Source.Group.MembersCount != 2 {
		t.Fatalf("join request action source: %+v err=%v", joinAction, err)
	}

	inviteRecorder := httptest.NewRecorder()
	env.handler.ServeHTTP(inviteRecorder, groupJSONRequest(http.MethodPost, base+"/invitations", ownerToken, fmt.Sprintf(`{"user_id":%d}`, invitedID)))
	if inviteRecorder.Code != http.StatusOK {
		t.Fatalf("invite non-member: status=%d body=%q", inviteRecorder.Code, inviteRecorder.Body.String())
	}

	eventBody := fmt.Sprintf(`{"title":"Planning","description":"Plan the next meeting","starts_at":%q}`, testNow.Add(2*time.Hour).Format(time.RFC3339))
	eventRecorder := httptest.NewRecorder()
	env.handler.ServeHTTP(eventRecorder, groupJSONRequest(http.MethodPost, base+"/events", memberToken, eventBody))
	if eventRecorder.Code != http.StatusCreated {
		t.Fatalf("create event: status=%d body=%q", eventRecorder.Code, eventRecorder.Body.String())
	}
	ownerPage = listNotificationsForTest(t, env, ownerToken)
	if len(ownerPage.Notifications) != 2 || ownerPage.Notifications[0].Type != domain.NotificationGroupEvent ||
		ownerPage.Notifications[0].Event == nil || ownerPage.Notifications[0].Group == nil ||
		ownerPage.Revision != 3 {
		t.Fatalf("group event notification: %+v", ownerPage)
	}
	memberPage := listNotificationsForTest(t, env, memberToken)
	if len(memberPage.Notifications) != 0 {
		t.Fatalf("event creator must not receive own notification: %+v", memberPage)
	}
	invitedPage := listNotificationsForTest(t, env, invitedToken)
	if len(invitedPage.Notifications) != 1 || invitedPage.Notifications[0].Type != domain.NotificationGroupInvitation {
		t.Fatalf("invited non-member received event notification: %+v", invitedPage)
	}
}
