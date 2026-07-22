package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo/sqlite"
)

func createGroupEventThroughHTTP(
	t *testing.T,
	env *testEnvironment,
	token string,
	groupID int64,
	title string,
	startsAt time.Time,
	wantStatus int,
) groupEventResponse {
	t.Helper()
	body := fmt.Sprintf(`{"title":%q,"description":"A real event description","starts_at":%q}`, title, startsAt.UTC().Format(time.RFC3339))
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, groupJSONRequest(http.MethodPost, "/api/groups/"+strconv.FormatInt(groupID, 10)+"/events", token, body))
	if recorder.Code != wantStatus {
		t.Fatalf("create event: status=%d body=%q want=%d", recorder.Code, recorder.Body.String(), wantStatus)
	}
	if wantStatus != http.StatusCreated {
		return groupEventResponse{}
	}
	var response groupEventResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode created event: %v", err)
	}
	return response
}

func getGroupEventPage(t *testing.T, env *testEnvironment, token, path string, wantStatus int) groupEventPageResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, authenticatedRequest(http.MethodGet, path, token, nil))
	if recorder.Code != wantStatus {
		t.Fatalf("get events: status=%d body=%q want=%d", recorder.Code, recorder.Body.String(), wantStatus)
	}
	if wantStatus != http.StatusOK {
		return groupEventPageResponse{}
	}
	var response groupEventPageResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode event page: %v", err)
	}
	return response
}

func respondToGroupEvent(
	t *testing.T,
	env *testEnvironment,
	token string,
	groupID, eventID int64,
	response string,
	wantStatus int,
) groupEventResponse {
	t.Helper()
	path := fmt.Sprintf("/api/groups/%d/events/%d/response", groupID, eventID)
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, groupJSONRequest(http.MethodPut, path, token, fmt.Sprintf(`{"response":%q}`, response)))
	if recorder.Code != wantStatus {
		t.Fatalf("respond to event: status=%d body=%q want=%d", recorder.Code, recorder.Body.String(), wantStatus)
	}
	if wantStatus != http.StatusOK {
		return groupEventResponse{}
	}
	var event groupEventResponse
	if err := json.NewDecoder(recorder.Body).Decode(&event); err != nil {
		t.Fatalf("decode event response: %v", err)
	}
	return event
}

func TestGroupEventsCreateListAccessValidationAndPagination(t *testing.T) {
	env := newTestEnvironment(t)
	_, ownerToken := env.createUserAndSession(t, "event-owner@example.com")
	memberID, memberToken := env.createUserAndSession(t, "event-member@example.com")
	invitedID, invitedToken := env.createUserAndSession(t, "event-invited@example.com")
	_, requestedToken := env.createUserAndSession(t, "event-requested@example.com")
	_, outsiderToken := env.createUserAndSession(t, "event-outsider@example.com")
	group := createGroupForTest(t, env, ownerToken, "Events group")
	base := "/api/groups/" + strconv.FormatInt(group.ID, 10)
	addGroupMemberForPostTest(t, env, group.ID, memberID, ownerToken, memberToken)

	inviteRecorder := httptest.NewRecorder()
	env.handler.ServeHTTP(inviteRecorder, groupJSONRequest(http.MethodPost, base+"/invitations", ownerToken, fmt.Sprintf(`{"user_id":%d}`, invitedID)))
	if inviteRecorder.Code != http.StatusOK {
		t.Fatalf("invite event user: status=%d body=%q", inviteRecorder.Code, inviteRecorder.Body.String())
	}
	groupMutation(t, env, http.MethodPost, base+"/join-request", requestedToken, http.StatusOK)

	later := createGroupEventThroughHTTP(t, env, ownerToken, group.ID, "Owner later", testNow.Add(3*time.Hour), http.StatusCreated)
	earlier := createGroupEventThroughHTTP(t, env, memberToken, group.ID, "Member earlier", testNow.Add(time.Hour), http.StatusCreated)
	tied := createGroupEventThroughHTTP(t, env, ownerToken, group.ID, "Owner same start", testNow.Add(time.Hour), http.StatusCreated)
	if later.Creator.ID == earlier.Creator.ID || earlier.GroupID != group.ID || earlier.Title != "Member earlier" {
		t.Fatalf("unexpected create responses: later=%+v earlier=%+v", later, earlier)
	}

	for name, token := range map[string]string{
		"invited": invitedToken, "requested": requestedToken, "outsider": outsiderToken,
	} {
		t.Run(name, func(t *testing.T) {
			getGroupEventPage(t, env, token, base+"/events", http.StatusForbidden)
			createGroupEventThroughHTTP(t, env, token, group.ID, "Forbidden", testNow.Add(2*time.Hour), http.StatusForbidden)
		})
	}
	getGroupEventPage(t, env, ownerToken, "/api/groups/999999/events", http.StatusNotFound)
	createGroupEventThroughHTTP(t, env, ownerToken, 999999, "Missing", testNow.Add(time.Hour), http.StatusNotFound)
	for _, path := range []string{"/api/groups/nope/events", base + "/events?limit=0", base + "/events?cursor=bad", base + "/events?extra=1"} {
		getGroupEventPage(t, env, ownerToken, path, http.StatusBadRequest)
	}

	first := getGroupEventPage(t, env, ownerToken, base+"/events?limit=1", http.StatusOK)
	if len(first.Events) != 1 || first.Events[0].ID != earlier.ID || first.NextCursor == nil {
		t.Fatalf("unexpected first event page: %+v", first)
	}
	second := getGroupEventPage(t, env, ownerToken, base+"/events?limit=1&cursor="+*first.NextCursor, http.StatusOK)
	if len(second.Events) != 1 || second.Events[0].ID != tied.ID || second.NextCursor == nil {
		t.Fatalf("unexpected second event page: %+v", second)
	}
	third := getGroupEventPage(t, env, ownerToken, base+"/events?limit=1&cursor="+*second.NextCursor, http.StatusOK)
	if len(third.Events) != 1 || third.Events[0].ID != later.ID || third.NextCursor != nil {
		t.Fatalf("unexpected third event page: %+v", third)
	}

	invalidBodies := map[string]string{
		"duplicate":  fmt.Sprintf(`{"title":"One","title":"Two","description":"Description","starts_at":%q}`, testNow.Add(time.Hour).Format(time.RFC3339)),
		"unknown":    fmt.Sprintf(`{"title":"Title","description":"Description","starts_at":%q,"extra":true}`, testNow.Add(time.Hour).Format(time.RFC3339)),
		"missing":    `{"title":"Title","description":"Description"}`,
		"null":       fmt.Sprintf(`{"title":null,"description":"Description","starts_at":%q}`, testNow.Add(time.Hour).Format(time.RFC3339)),
		"past":       fmt.Sprintf(`{"title":"Past","description":"Description","starts_at":%q}`, testNow.Add(-time.Second).Format(time.RFC3339)),
		"bad date":   `{"title":"Date","description":"Description","starts_at":"tomorrow"}`,
		"long title": fmt.Sprintf(`{"title":%q,"description":"Description","starts_at":%q}`, strings.Repeat("🙂", 101), testNow.Add(time.Hour).Format(time.RFC3339)),
		"blank desc": fmt.Sprintf(`{"title":"Title","description":"   ","starts_at":%q}`, testNow.Add(time.Hour).Format(time.RFC3339)),
		"trailing":   fmt.Sprintf(`{"title":"Title","description":"Description","starts_at":%q}{}`, testNow.Add(time.Hour).Format(time.RFC3339)),
	}
	for name, body := range invalidBodies {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			env.handler.ServeHTTP(recorder, groupJSONRequest(http.MethodPost, base+"/events", ownerToken, body))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
			}
		})
	}
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, authenticatedRequest(http.MethodPost, base+"/events", ownerToken, strings.NewReader(`{}`)))
	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("create content type status=%d", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, authenticatedRequest(http.MethodDelete, base+"/events", ownerToken, nil))
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("event method status=%d allow=%q", recorder.Code, recorder.Header().Get("Allow"))
	}
	oversized := fmt.Sprintf(`{"title":"Title","description":%q,"starts_at":%q}`, strings.Repeat("a", maxGroupJSONBytes), testNow.Add(time.Hour).Format(time.RFC3339))
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, groupJSONRequest(http.MethodPost, base+"/events", ownerToken, oversized))
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized event status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestGroupEventResponseRequiresActiveMembership(t *testing.T) {
	env := newTestEnvironment(t)
	_, ownerToken := env.createUserAndSession(t, "event-response-owner@example.com")
	invitedID, invitedToken := env.createUserAndSession(t, "event-response-invited@example.com")
	_, requestedToken := env.createUserAndSession(t, "event-response-requested@example.com")
	_, outsiderToken := env.createUserAndSession(t, "event-response-outsider@example.com")
	group := createGroupForTest(t, env, ownerToken, "Response access group")
	base := "/api/groups/" + strconv.FormatInt(group.ID, 10)

	inviteRecorder := httptest.NewRecorder()
	env.handler.ServeHTTP(inviteRecorder, groupJSONRequest(http.MethodPost, base+"/invitations", ownerToken, fmt.Sprintf(`{"user_id":%d}`, invitedID)))
	if inviteRecorder.Code != http.StatusOK {
		t.Fatalf("invite event user: status=%d body=%q", inviteRecorder.Code, inviteRecorder.Body.String())
	}
	groupMutation(t, env, http.MethodPost, base+"/join-request", requestedToken, http.StatusOK)
	event := createGroupEventThroughHTTP(t, env, ownerToken, group.ID, "Protected RSVP", testNow.Add(time.Hour), http.StatusCreated)

	for name, token := range map[string]string{
		"invited":   invitedToken,
		"requested": requestedToken,
		"outsider":  outsiderToken,
	} {
		t.Run(name, func(t *testing.T) {
			respondToGroupEvent(t, env, token, group.ID, event.ID, "going", http.StatusForbidden)
		})
	}
}

func TestGroupEventResponsesCountsLeaveRejoinAndCrossGroup(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "event-rsvp-owner@example.com")
	memberID, memberToken := env.createUserAndSession(t, "event-rsvp-member@example.com")
	group := createGroupForTest(t, env, ownerToken, "RSVP group")
	addGroupMemberForPostTest(t, env, group.ID, memberID, ownerToken, memberToken)
	event := createGroupEventThroughHTTP(t, env, memberToken, group.ID, "Member event", testNow.Add(time.Hour), http.StatusCreated)

	going := respondToGroupEvent(t, env, ownerToken, group.ID, event.ID, "going", http.StatusOK)
	if going.GoingCount != 1 || going.NotGoingCount != 0 || going.ViewerResponse == nil || *going.ViewerResponse != domain.GroupEventGoing {
		t.Fatalf("unexpected going event: %+v", going)
	}
	going = respondToGroupEvent(t, env, ownerToken, group.ID, event.ID, "going", http.StatusOK)
	if going.GoingCount != 1 {
		t.Fatalf("idempotent going count=%d", going.GoingCount)
	}
	notGoing := respondToGroupEvent(t, env, ownerToken, group.ID, event.ID, "not_going", http.StatusOK)
	if notGoing.GoingCount != 0 || notGoing.NotGoingCount != 1 || notGoing.ViewerResponse == nil || *notGoing.ViewerResponse != domain.GroupEventNotGoing {
		t.Fatalf("unexpected not-going event: %+v", notGoing)
	}
	memberGoing := respondToGroupEvent(t, env, memberToken, group.ID, event.ID, "going", http.StatusOK)
	if memberGoing.GoingCount != 1 || memberGoing.NotGoingCount != 1 {
		t.Fatalf("combined counts: %+v", memberGoing)
	}
	respondToGroupEvent(t, env, ownerToken, group.ID, event.ID, "maybe", http.StatusBadRequest)
	responsePath := fmt.Sprintf("/api/groups/%d/events/%d/response", group.ID, event.ID)
	for name, body := range map[string]string{
		"duplicate": `{"response":"going","response":"not_going"}`,
		"unknown":   `{"response":"going","extra":true}`,
		"missing":   `{}`,
		"null":      `{"response":null}`,
	} {
		t.Run("response "+name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			env.handler.ServeHTTP(recorder, groupJSONRequest(http.MethodPut, responsePath, ownerToken, body))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
			}
		})
	}
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, authenticatedRequest(http.MethodPut, responsePath, ownerToken, strings.NewReader(`{"response":"going"}`)))
	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("response content type status=%d", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, authenticatedRequest(http.MethodPost, responsePath, ownerToken, nil))
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != http.MethodPut {
		t.Fatalf("response method status=%d allow=%q", recorder.Code, recorder.Header().Get("Allow"))
	}

	otherGroup := createGroupForTest(t, env, ownerToken, "Other RSVP group")
	respondToGroupEvent(t, env, ownerToken, otherGroup.ID, event.ID, "going", http.StatusNotFound)
	respondToGroupEvent(t, env, ownerToken, group.ID, 999999, "going", http.StatusNotFound)

	base := "/api/groups/" + strconv.FormatInt(group.ID, 10)
	groupMutation(t, env, http.MethodDelete, base+"/membership", memberToken, http.StatusOK)
	ownerPage := getGroupEventPage(t, env, ownerToken, base+"/events", http.StatusOK)
	if len(ownerPage.Events) != 1 || ownerPage.Events[0].GoingCount != 0 || ownerPage.Events[0].NotGoingCount != 1 {
		t.Fatalf("leave did not remove RSVP from counts or event disappeared: %+v", ownerPage.Events)
	}
	getGroupEventPage(t, env, memberToken, base+"/events", http.StatusForbidden)
	addGroupMemberForPostTest(t, env, group.ID, memberID, ownerToken, memberToken)
	rejoined := getGroupEventPage(t, env, memberToken, base+"/events", http.StatusOK)
	if len(rejoined.Events) != 1 || rejoined.Events[0].ViewerResponse == nil || *rejoined.Events[0].ViewerResponse != domain.GroupEventGoing || rejoined.Events[0].GoingCount != 1 {
		t.Fatalf("rejoin did not restore event response: %+v", rejoined.Events)
	}

	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			path := fmt.Sprintf("/api/groups/%d/events/%d/response", group.ID, event.ID)
			recorder := httptest.NewRecorder()
			env.handler.ServeHTTP(recorder, groupJSONRequest(http.MethodPut, path, memberToken, `{"response":"going"}`))
			statuses <- recorder.Code
		}()
	}
	wait.Wait()
	close(statuses)
	for status := range statuses {
		if status != http.StatusOK {
			t.Fatalf("parallel RSVP status=%d", status)
		}
	}
	var responseRows int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM group_event_responses WHERE event_id = ? AND user_id = ?`, event.ID, memberID).Scan(&responseRows); err != nil || responseRows != 1 {
		t.Fatalf("parallel RSVP rows=%d err=%v", responseRows, err)
	}
	if ownerID == memberID {
		t.Fatal("test users unexpectedly share id")
	}
}

func TestGroupEventCreatorAvatarIsViewerAware(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "event-avatar-owner@example.com")
	memberID, memberToken := env.createUserAndSession(t, "event-avatar-member@example.com")
	group := createGroupForTest(t, env, ownerToken, "Avatar events")
	addGroupMemberForPostTest(t, env, group.ID, memberID, ownerToken, memberToken)
	mediaID, err := sqlite.NewMediaRepo(env.db).Create(context.Background(), memberID, "image/png", 8, "event-avatar.png", "avatar.png", testNow)
	if err != nil {
		t.Fatalf("create avatar media: %v", err)
	}
	if err := env.users.SetAvatarMediaID(context.Background(), memberID, &mediaID, testNow); err != nil {
		t.Fatalf("set avatar: %v", err)
	}
	if _, err := env.db.Exec(`UPDATE users SET is_private = 1, gender = 'female' WHERE id = ?`, memberID); err != nil {
		t.Fatalf("make event creator private: %v", err)
	}
	event := createGroupEventThroughHTTP(t, env, memberToken, group.ID, "Private creator", testNow.Add(time.Hour), http.StatusCreated)
	wantCustom := fmt.Sprintf("/api/users/%d/avatar?v=%d", memberID, mediaID)
	if event.Creator.AvatarURL != wantCustom {
		t.Fatalf("creator did not receive own avatar: %q", event.Creator.AvatarURL)
	}
	page := getGroupEventPage(t, env, ownerToken, fmt.Sprintf("/api/groups/%d/events", group.ID), http.StatusOK)
	if len(page.Events) != 1 || page.Events[0].Creator.AvatarURL != domain.FemaleAvatarPlaceholderURL {
		t.Fatalf("private creator avatar leaked: %+v", page.Events)
	}
	if _, err := sqlite.NewFollowRepo(env.db).Upsert(context.Background(), ownerID, memberID, domain.FollowAccepted, testNow); err != nil {
		t.Fatalf("create accepted follow: %v", err)
	}
	page = getGroupEventPage(t, env, ownerToken, fmt.Sprintf("/api/groups/%d/events", group.ID), http.StatusOK)
	if page.Events[0].Creator.AvatarURL != wantCustom {
		t.Fatalf("accepted follower did not receive event creator avatar: %q", page.Events[0].Creator.AvatarURL)
	}
}
