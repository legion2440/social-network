package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo/sqlite"
)

func groupJSONRequest(method, path, token, body string) *http.Request {
	req := authenticatedRequest(method, path, token, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func createGroupForTest(t *testing.T, env *testEnvironment, token, title string) groupResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	body := fmt.Sprintf(`{"title":%q,"description":"A real group description"}`, title)
	env.handler.ServeHTTP(rec, groupJSONRequest(http.MethodPost, "/api/groups", token, body))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group: status=%d body=%q", rec.Code, rec.Body.String())
	}
	var group groupResponse
	if err := json.NewDecoder(rec.Body).Decode(&group); err != nil {
		t.Fatalf("decode created group: %v", err)
	}
	return group
}

func groupMutation(t *testing.T, env *testEnvironment, method, path, token string, want int) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(method, path, token, nil))
	if rec.Code != want {
		t.Fatalf("%s %s: status=%d body=%q, want=%d", method, path, rec.Code, rec.Body.String(), want)
	}
	return rec
}

func TestGroupsCreateStrictJSONAndOwnerMembership(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "group-owner@example.com")

	for name, request := range map[string]*http.Request{
		"unsupported content type": authenticatedRequest(http.MethodPost, "/api/groups", ownerToken, strings.NewReader(`{"title":"Title","description":"Description"}`)),
		"duplicate field":          groupJSONRequest(http.MethodPost, "/api/groups", ownerToken, `{"title":"One","title":"Two","description":"Description"}`),
		"unknown field":            groupJSONRequest(http.MethodPost, "/api/groups", ownerToken, `{"title":"Title","description":"Description","extra":true}`),
		"null title":               groupJSONRequest(http.MethodPost, "/api/groups", ownerToken, `{"title":null,"description":"Description"}`),
		"trailing json":            groupJSONRequest(http.MethodPost, "/api/groups", ownerToken, `{"title":"Title","description":"Description"}{}`),
		"empty description":        groupJSONRequest(http.MethodPost, "/api/groups", ownerToken, `{"title":"Title","description":"   "}`),
		"title over rune limit":    groupJSONRequest(http.MethodPost, "/api/groups", ownerToken, fmt.Sprintf(`{"title":%q,"description":"Description"}`, strings.Repeat("я", 101))),
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, request)
			want := http.StatusBadRequest
			if name == "unsupported content type" {
				want = http.StatusUnsupportedMediaType
			}
			if rec.Code != want {
				t.Fatalf("status=%d body=%q want=%d", rec.Code, rec.Body.String(), want)
			}
		})
	}

	group := createGroupForTest(t, env, ownerToken, "Unicode Ж group")
	if group.ID <= 0 || group.ViewerStatus != "owner" || group.MembersCount != 1 || group.Owner.ID != ownerID {
		t.Fatalf("unexpected group response: %+v", group)
	}
	assertDBRowCount(t, env.db, "groups", 1)
	assertDBRowCount(t, env.db, "group_memberships", 1)
	var status string
	if err := env.db.QueryRow(`SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?`, group.ID, ownerID).Scan(&status); err != nil || status != "owner" {
		t.Fatalf("owner membership: status=%q err=%v", status, err)
	}
}

func TestGroupMembershipLifecycleAuthorizationAndInbox(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "groups-owner@example.com")
	memberID, memberToken := env.createUserAndSession(t, "groups-member@example.com")
	invitedID, invitedToken := env.createUserAndSession(t, "groups-invited@example.com")
	outsiderID, outsiderToken := env.createUserAndSession(t, "groups-outsider@example.com")
	group := createGroupForTest(t, env, ownerToken, "Lifecycle group")
	base := "/api/groups/" + strconv.FormatInt(group.ID, 10)

	for _, token := range []string{ownerToken, memberToken, invitedToken, outsiderToken} {
		for _, path := range []string{base, base + "/members"} {
			rec := groupMutation(t, env, http.MethodGet, path, token, http.StatusOK)
			if bytes.Contains(rec.Body.Bytes(), []byte("@example.com")) {
				t.Fatalf("safe group response leaked email: %s", rec.Body.String())
			}
		}
	}

	rec := groupMutation(t, env, http.MethodPost, base+"/join-request", memberToken, http.StatusOK)
	var requested groupResponse
	_ = json.NewDecoder(rec.Body).Decode(&requested)
	if requested.ViewerStatus != "requested" || requested.MembersCount != 1 {
		t.Fatalf("unexpected requested state: %+v", requested)
	}
	groupMutation(t, env, http.MethodPost, base+"/join-request", memberToken, http.StatusConflict)
	groupMutation(t, env, http.MethodGet, base+"/join-requests", outsiderToken, http.StatusForbidden)

	rec = groupMutation(t, env, http.MethodGet, base+"/join-requests", ownerToken, http.StatusOK)
	var requests groupStatePageResponse
	if err := json.NewDecoder(rec.Body).Decode(&requests); err != nil || len(requests.Requests) != 1 || requests.Requests[0].User.ID != memberID {
		t.Fatalf("unexpected requests response: %+v err=%v", requests, err)
	}
	acceptPath := base + "/join-requests/" + strconv.FormatInt(memberID, 10) + "/accept"
	rec = groupMutation(t, env, http.MethodPost, acceptPath, ownerToken, http.StatusOK)
	var accepted groupResponse
	_ = json.NewDecoder(rec.Body).Decode(&accepted)
	if accepted.MembersCount != 2 || accepted.ViewerStatus != "owner" {
		t.Fatalf("unexpected accept response: %+v", accepted)
	}
	groupMutation(t, env, http.MethodPost, acceptPath, ownerToken, http.StatusConflict)
	groupMutation(t, env, http.MethodDelete, base+"/membership", memberToken, http.StatusOK)
	groupMutation(t, env, http.MethodDelete, base+"/membership", memberToken, http.StatusConflict)
	groupMutation(t, env, http.MethodDelete, base+"/membership", ownerToken, http.StatusConflict)

	invite := func(userID int64, want int) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		body := fmt.Sprintf(`{"user_id":%d}`, userID)
		env.handler.ServeHTTP(recorder, groupJSONRequest(http.MethodPost, base+"/invitations", ownerToken, body))
		if recorder.Code != want {
			t.Fatalf("invite user %d: status=%d body=%q want=%d", userID, recorder.Code, recorder.Body.String(), want)
		}
		return recorder
	}
	invite(invitedID, http.StatusOK)
	invite(invitedID, http.StatusConflict)
	invite(999999, http.StatusNotFound)
	groupMutation(t, env, http.MethodPost, base+"/invitations", outsiderToken, http.StatusUnsupportedMediaType)

	rec = groupMutation(t, env, http.MethodGet, "/api/group-invitations", invitedToken, http.StatusOK)
	var inbox groupInvitationInboxResponse
	if err := json.NewDecoder(rec.Body).Decode(&inbox); err != nil || len(inbox.Invitations) != 1 || inbox.Invitations[0].Group.ID != group.ID {
		t.Fatalf("unexpected inbox: %+v err=%v", inbox, err)
	}
	groupMutation(t, env, http.MethodPost, base+"/invitation/accept", invitedToken, http.StatusOK)
	rec = groupMutation(t, env, http.MethodGet, "/api/group-invitations", invitedToken, http.StatusOK)
	if err := json.NewDecoder(rec.Body).Decode(&inbox); err != nil || len(inbox.Invitations) != 0 {
		t.Fatalf("accepted invitation remained in inbox: %+v err=%v", inbox, err)
	}

	invite(outsiderID, http.StatusOK)
	groupMutation(t, env, http.MethodDelete, base+"/invitation", outsiderToken, http.StatusOK)
	groupMutation(t, env, http.MethodDelete, base+"/invitation", outsiderToken, http.StatusConflict)
	groupMutation(t, env, http.MethodDelete, base+"/join-request", memberToken, http.StatusConflict)

	groupMutation(t, env, http.MethodPost, base+"/join-request", memberToken, http.StatusOK)
	rejectPath := base + "/join-requests/" + strconv.FormatInt(memberID, 10)
	groupMutation(t, env, http.MethodDelete, rejectPath, ownerToken, http.StatusOK)
	groupMutation(t, env, http.MethodDelete, rejectPath, ownerToken, http.StatusConflict)
	groupMutation(t, env, http.MethodDelete, base+"/join-requests/999999", ownerToken, http.StatusNotFound)

	if ownerID == memberID {
		t.Fatal("test users unexpectedly share an ID")
	}
}

func TestGroupPaginationMemberOrderAndViewerAwareAvatar(t *testing.T) {
	env := newTestEnvironment(t)
	_, ownerToken := env.createUserAndSession(t, "group-page-owner@example.com")
	privateID, privateToken := env.createUserAndSession(t, "group-page-private@example.com")
	group1 := createGroupForTest(t, env, ownerToken, "First group")
	group2 := createGroupForTest(t, env, ownerToken, "Second group")

	rec := groupMutation(t, env, http.MethodGet, "/api/groups?limit=1", privateToken, http.StatusOK)
	var firstPage groupPageResponse
	if err := json.NewDecoder(rec.Body).Decode(&firstPage); err != nil || len(firstPage.Groups) != 1 || firstPage.Groups[0].ID != group2.ID || firstPage.NextCursor == nil {
		t.Fatalf("unexpected first group page: %+v err=%v", firstPage, err)
	}
	rec = groupMutation(t, env, http.MethodGet, "/api/groups?limit=1&cursor="+*firstPage.NextCursor, privateToken, http.StatusOK)
	var secondPage groupPageResponse
	if err := json.NewDecoder(rec.Body).Decode(&secondPage); err != nil || len(secondPage.Groups) != 1 || secondPage.Groups[0].ID != group1.ID {
		t.Fatalf("unexpected second group page: %+v err=%v", secondPage, err)
	}
	groupMutation(t, env, http.MethodGet, "/api/groups?limit=0", privateToken, http.StatusBadRequest)
	groupMutation(t, env, http.MethodGet, "/api/groups?cursor=bad", privateToken, http.StatusBadRequest)

	if _, err := env.db.Exec(`UPDATE users SET is_private = 1 WHERE id = ?`, privateID); err != nil {
		t.Fatalf("make invited user private: %v", err)
	}
	mediaID, err := sqlite.NewMediaRepo(env.db).Create(context.Background(), privateID, "image/png", 8, "private-avatar.png", "avatar.png", testNow)
	if err != nil {
		t.Fatalf("create private avatar media: %v", err)
	}
	if err := env.users.SetAvatarMediaID(context.Background(), privateID, &mediaID, testNow); err != nil {
		t.Fatalf("set private avatar: %v", err)
	}
	base := "/api/groups/" + strconv.FormatInt(group1.ID, 10)
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, groupJSONRequest(http.MethodPost, base+"/invitations", ownerToken, fmt.Sprintf(`{"user_id":%d}`, privateID)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("invite private user: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	groupMutation(t, env, http.MethodPost, base+"/invitation/accept", privateToken, http.StatusOK)

	rec = groupMutation(t, env, http.MethodGet, base+"/members?limit=1", ownerToken, http.StatusOK)
	var members1 groupMemberPageResponse
	if err := json.NewDecoder(rec.Body).Decode(&members1); err != nil || len(members1.Members) != 1 || members1.Members[0].Status != domain.GroupOwner || members1.NextCursor == nil {
		t.Fatalf("unexpected owner page: %+v err=%v", members1, err)
	}
	rec = groupMutation(t, env, http.MethodGet, base+"/members?limit=1&cursor="+*members1.NextCursor, ownerToken, http.StatusOK)
	var members2 groupMemberPageResponse
	if err := json.NewDecoder(rec.Body).Decode(&members2); err != nil || len(members2.Members) != 1 || members2.Members[0].User.ID != privateID {
		t.Fatalf("unexpected member page: %+v err=%v", members2, err)
	}
	if members2.Members[0].User.AvatarURL != domain.NeutralAvatarPlaceholderURL {
		t.Fatalf("group membership exposed inaccessible custom avatar: %q", members2.Members[0].User.AvatarURL)
	}
}

func TestConcurrentGroupRequestAcceptHasOneWinner(t *testing.T) {
	env := newTestEnvironment(t)
	_, ownerToken := env.createUserAndSession(t, "group-race-owner@example.com")
	memberID, memberToken := env.createUserAndSession(t, "group-race-member@example.com")
	group := createGroupForTest(t, env, ownerToken, "Race group")
	base := "/api/groups/" + strconv.FormatInt(group.ID, 10)
	groupMutation(t, env, http.MethodPost, base+"/join-request", memberToken, http.StatusOK)
	path := base + "/join-requests/" + strconv.FormatInt(memberID, 10) + "/accept"

	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPost, path, ownerToken, nil))
			statuses <- rec.Code
		}()
	}
	wg.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("expected one 200 and one 409, got %+v", counts)
	}
}
