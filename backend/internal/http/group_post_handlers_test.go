package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo/sqlite"
)

func newGroupPostRequest(
	t *testing.T,
	token string,
	groupID int64,
	fields []postMultipartField,
	files []postMultipartFile,
) *http.Request {
	t.Helper()
	request := newPostRequest(t, token, fields, files)
	request.URL.Path = "/api/groups/" + strconv.FormatInt(groupID, 10) + "/posts"
	return request
}

func createGroupPostThroughHTTP(
	t *testing.T,
	env *testEnvironment,
	token string,
	groupID int64,
	text string,
	file *postMultipartFile,
	wantStatus int,
) postResponse {
	t.Helper()
	files := []postMultipartFile{}
	if file != nil {
		files = append(files, *file)
	}
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, newGroupPostRequest(
		t, token, groupID, []postMultipartField{{name: "text", value: text}}, files,
	))
	if recorder.Code != wantStatus {
		t.Fatalf("create group post: status=%d body=%q want=%d", recorder.Code, recorder.Body.String(), wantStatus)
	}
	if wantStatus != http.StatusCreated {
		return postResponse{}
	}
	var response postResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode group post: %v", err)
	}
	return response
}

func addGroupMemberForPostTest(t *testing.T, env *testEnvironment, groupID, userID int64, ownerToken, userToken string) {
	t.Helper()
	base := "/api/groups/" + strconv.FormatInt(groupID, 10)
	groupMutation(t, env, http.MethodPost, base+"/join-request", userToken, http.StatusOK)
	groupMutation(t, env, http.MethodPost, base+"/join-requests/"+strconv.FormatInt(userID, 10)+"/accept", ownerToken, http.StatusOK)
}

func TestGroupPostsAccessStrictMultipartCommentsMediaAndRejoin(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "group-post-owner@example.com")
	memberID, memberToken := env.createUserAndSession(t, "group-post-member@example.com")
	invitedID, invitedToken := env.createUserAndSession(t, "group-post-invited@example.com")
	requestedID, requestedToken := env.createUserAndSession(t, "group-post-requested@example.com")
	_, outsiderToken := env.createUserAndSession(t, "group-post-outsider@example.com")
	group := createGroupForTest(t, env, ownerToken, "Group content")
	base := "/api/groups/" + strconv.FormatInt(group.ID, 10)
	addGroupMemberForPostTest(t, env, group.ID, memberID, ownerToken, memberToken)

	env.handler.ServeHTTP(httptest.NewRecorder(), groupJSONRequest(
		http.MethodPost, base+"/invitations", ownerToken, fmt.Sprintf(`{"user_id":%d}`, invitedID),
	))
	groupMutation(t, env, http.MethodPost, base+"/join-request", requestedToken, http.StatusOK)

	ownerPost := createGroupPostThroughHTTP(t, env, ownerToken, group.ID, "  owner group post  ", nil, http.StatusCreated)
	if ownerPost.GroupID == nil || *ownerPost.GroupID != group.ID || ownerPost.Privacy != nil || ownerPost.Text != "owner group post" {
		t.Fatalf("unexpected owner group post: %+v", ownerPost)
	}
	png := []byte("\x89PNG\r\n\x1a\ngroup-post-image")
	memberPost := createGroupPostThroughHTTP(t, env, memberToken, group.ID, "member image post", &postMultipartFile{
		fieldName: "media", filename: "group.png", contents: png,
	}, http.StatusCreated)
	if memberPost.MediaURL == nil {
		t.Fatal("group media URL is missing")
	}

	for name, token := range map[string]string{
		"invited": invitedToken, "requested": requestedToken, "outsider": outsiderToken,
	} {
		t.Run(name, func(t *testing.T) {
			createGroupPostThroughHTTP(t, env, token, group.ID, "forbidden", nil, http.StatusForbidden)
			getPostPage(t, env, token, base+"/posts", http.StatusForbidden)
		})
	}
	createGroupPostThroughHTTP(t, env, ownerToken, 999999, "missing group", nil, http.StatusNotFound)
	for _, path := range []string{"/api/groups/not-a-number/posts", "/api/groups/0/posts"} {
		getPostPage(t, env, ownerToken, path, http.StatusBadRequest)
	}

	firstPage := getPostPage(t, env, ownerToken, base+"/posts?limit=1", http.StatusOK)
	if len(firstPage.Posts) != 1 || firstPage.Posts[0].ID != memberPost.ID || firstPage.NextCursor == nil {
		t.Fatalf("unexpected first group post page: %+v", firstPage)
	}
	secondPage := getPostPage(t, env, ownerToken, base+"/posts?limit=1&cursor="+*firstPage.NextCursor, http.StatusOK)
	if len(secondPage.Posts) != 1 || secondPage.Posts[0].ID != ownerPost.ID || secondPage.NextCursor != nil {
		t.Fatalf("unexpected second group post page: %+v", secondPage)
	}

	for name, fields := range map[string][]postMultipartField{
		"duplicate text": {{name: "text", value: "one"}, {name: "text", value: "two"}},
		"privacy":        {{name: "text", value: "post"}, {name: "privacy", value: "public"}},
		"audience":       {{name: "text", value: "post"}, {name: "selected_user_id", value: strconv.FormatInt(memberID, 10)}},
		"unknown":        {{name: "text", value: "post"}, {name: "surprise", value: "value"}},
		"blank":          {{name: "text", value: "   "}},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			env.handler.ServeHTTP(recorder, newGroupPostRequest(t, ownerToken, group.ID, fields, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
			}
		})
	}
	for name, files := range map[string][]postMultipartFile{
		"unknown file": {{fieldName: "avatar", filename: "image.png", contents: png}},
		"duplicate media": {
			{fieldName: "media", filename: "one.png", contents: png},
			{fieldName: "media", filename: "two.png", contents: png},
		},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			env.handler.ServeHTTP(recorder, newGroupPostRequest(
				t, ownerToken, group.ID, []postMultipartField{{name: "text", value: "post"}}, files,
			))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
			}
		})
	}
	nonMultipart := httptest.NewRecorder()
	env.handler.ServeHTTP(nonMultipart, authenticatedRequest(http.MethodPost, base+"/posts", ownerToken, nil))
	if nonMultipart.Code != http.StatusBadRequest {
		t.Fatalf("non-multipart status=%d body=%q", nonMultipart.Code, nonMultipart.Body.String())
	}
	methodRecorder := httptest.NewRecorder()
	env.handler.ServeHTTP(methodRecorder, authenticatedRequest(http.MethodDelete, base+"/posts", ownerToken, nil))
	if methodRecorder.Code != http.StatusMethodNotAllowed || methodRecorder.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("group posts method contract: status=%d allow=%q", methodRecorder.Code, methodRecorder.Header().Get("Allow"))
	}

	comment := createCommentThroughHTTP(t, env, memberToken, ownerPost.ID, "member comment", http.StatusCreated)
	if comment.PostID != ownerPost.ID {
		t.Fatalf("unexpected group comment: %+v", comment)
	}
	countedPage := getPostPage(t, env, ownerToken, base+"/posts?limit=10", http.StatusOK)
	var counted bool
	for _, post := range countedPage.Posts {
		if post.ID == ownerPost.ID {
			counted = true
			if post.CommentsCount != 1 {
				t.Fatalf("group comments_count=%d, want 1", post.CommentsCount)
			}
		}
	}
	if !counted {
		t.Fatal("commented group post missing from list")
	}
	mediaPath := "/api/posts/" + strconv.FormatInt(memberPost.ID, 10) + "/media"
	mediaRecorder := httptest.NewRecorder()
	env.handler.ServeHTTP(mediaRecorder, authenticatedRequest(http.MethodGet, mediaPath, memberToken, nil))
	if mediaRecorder.Code != http.StatusOK || mediaRecorder.Body.String() != string(png) {
		t.Fatalf("group media before leave: status=%d body=%q", mediaRecorder.Code, mediaRecorder.Body.String())
	}

	groupMutation(t, env, http.MethodDelete, base+"/membership", memberToken, http.StatusOK)
	getPostPage(t, env, memberToken, base+"/posts", http.StatusForbidden)
	getCommentPage(t, env, memberToken, "/api/posts/"+strconv.FormatInt(ownerPost.ID, 10)+"/comments", http.StatusForbidden)
	createCommentThroughHTTP(t, env, memberToken, ownerPost.ID, "after leave", http.StatusForbidden)
	mediaRecorder = httptest.NewRecorder()
	env.handler.ServeHTTP(mediaRecorder, authenticatedRequest(http.MethodGet, mediaPath, memberToken, nil))
	if mediaRecorder.Code != http.StatusForbidden {
		t.Fatalf("group media after leave: status=%d body=%q", mediaRecorder.Code, mediaRecorder.Body.String())
	}

	addGroupMemberForPostTest(t, env, group.ID, memberID, ownerToken, memberToken)
	if restored := getPostPage(t, env, memberToken, base+"/posts", http.StatusOK); len(restored.Posts) != 2 {
		t.Fatalf("group history not restored after rejoin: %+v", restored.Posts)
	}
	comments := getCommentPage(t, env, memberToken, "/api/posts/"+strconv.FormatInt(ownerPost.ID, 10)+"/comments", http.StatusOK)
	if len(comments.Comments) != 1 || comments.Comments[0].ID != comment.ID {
		t.Fatalf("group comments not restored after rejoin: %+v", comments.Comments)
	}

	if ownerID == requestedID {
		t.Fatal("test users unexpectedly share ids")
	}
}

func TestGroupPostsStayOutOfPersonalSurfacesAndUseViewerAwareAvatars(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "group-isolation-owner@example.com")
	memberID, memberToken := env.createUserAndSession(t, "group-isolation-member@example.com")
	group := createGroupForTest(t, env, ownerToken, "Isolation group")
	addGroupMemberForPostTest(t, env, group.ID, memberID, ownerToken, memberToken)

	avatarBytes := []byte("\x89PNG\r\n\x1a\nprivate-member-avatar")
	mediaID, err := env.dbInsertMedia(memberID, "private-member.png", avatarBytes)
	if err != nil {
		t.Fatalf("create member avatar: %v", err)
	}
	if err := env.users.SetAvatarMediaID(context.Background(), memberID, &mediaID, testNow); err != nil {
		t.Fatalf("set member avatar: %v", err)
	}
	if _, err := env.db.Exec(`UPDATE users SET is_private = 1, gender = 'female' WHERE id = ?`, memberID); err != nil {
		t.Fatalf("make member private: %v", err)
	}

	groupPost := createGroupPostThroughHTTP(t, env, memberToken, group.ID, "private author group post", nil, http.StatusCreated)
	createCommentThroughHTTP(t, env, memberToken, groupPost.ID, "private commenter", http.StatusCreated)
	base := "/api/groups/" + strconv.FormatInt(group.ID, 10) + "/posts"
	ownerPage := getPostPage(t, env, ownerToken, base, http.StatusOK)
	if len(ownerPage.Posts) != 1 || ownerPage.Posts[0].Author.AvatarURL != domain.FemaleAvatarPlaceholderURL {
		t.Fatalf("inaccessible group author avatar leaked: %+v", ownerPage.Posts)
	}
	ownerComments := getCommentPage(t, env, ownerToken, "/api/posts/"+strconv.FormatInt(groupPost.ID, 10)+"/comments", http.StatusOK)
	if len(ownerComments.Comments) != 1 || ownerComments.Comments[0].Author.AvatarURL != domain.FemaleAvatarPlaceholderURL {
		t.Fatalf("inaccessible comment author avatar leaked: %+v", ownerComments.Comments)
	}
	memberPage := getPostPage(t, env, memberToken, base, http.StatusOK)
	wantAvatar := fmt.Sprintf("/api/users/%d/avatar?v=%d", memberID, mediaID)
	if memberPage.Posts[0].Author.AvatarURL != wantAvatar {
		t.Fatalf("own group author avatar missing: got %q want %q", memberPage.Posts[0].Author.AvatarURL, wantAvatar)
	}
	if _, err := sqlite.NewFollowRepo(env.db).Upsert(context.Background(), ownerID, memberID, domain.FollowAccepted, testNow); err != nil {
		t.Fatalf("create accepted avatar relation: %v", err)
	}
	acceptedPage := getPostPage(t, env, ownerToken, base, http.StatusOK)
	if acceptedPage.Posts[0].Author.AvatarURL != wantAvatar {
		t.Fatalf("accepted follower did not receive custom avatar: %q", acceptedPage.Posts[0].Author.AvatarURL)
	}

	if feed := getPostPage(t, env, memberToken, "/api/posts/feed?limit=10", http.StatusOK); len(feed.Posts) != 0 {
		t.Fatalf("group post leaked into personal feed: %+v", feed.Posts)
	}
	if profile := getPostPage(t, env, memberToken, "/api/users/"+strconv.FormatInt(memberID, 10)+"/posts?limit=10", http.StatusOK); len(profile.Posts) != 0 {
		t.Fatalf("group post leaked into personal profile: %+v", profile.Posts)
	}
	profileRecorder := httptest.NewRecorder()
	env.handler.ServeHTTP(profileRecorder, authenticatedRequest(http.MethodGet, "/api/users/"+strconv.FormatInt(memberID, 10), memberToken, nil))
	if profileRecorder.Code != http.StatusOK {
		t.Fatalf("read member profile: status=%d body=%q", profileRecorder.Code, profileRecorder.Body.String())
	}
	var profile map[string]any
	if err := json.NewDecoder(profileRecorder.Body).Decode(&profile); err != nil {
		t.Fatalf("decode member profile: %v", err)
	}
	if postsCount, ok := profile["posts_count"].(float64); !ok || postsCount != 0 {
		t.Fatalf("group post affected personal posts_count: %+v", profile["posts_count"])
	}
}
