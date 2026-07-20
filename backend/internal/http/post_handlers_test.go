package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/service"
)

type postMultipartField struct {
	name  string
	value string
}

type postMultipartFile struct {
	fieldName string
	filename  string
	contents  []byte
}

func newPostRequest(
	t *testing.T,
	token string,
	fields []postMultipartField,
	files []postMultipartFile,
) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, field := range fields {
		if err := writer.WriteField(field.name, field.value); err != nil {
			t.Fatalf("write post field %s: %v", field.name, err)
		}
	}
	for _, file := range files {
		part, err := writer.CreateFormFile(file.fieldName, file.filename)
		if err != nil {
			t.Fatalf("create post file %s: %v", file.fieldName, err)
		}
		if _, err := part.Write(file.contents); err != nil {
			t.Fatalf("write post file %s: %v", file.fieldName, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close post multipart: %v", err)
	}
	req := authenticatedRequest(http.MethodPost, "/api/posts", token, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func createPostThroughHTTP(
	t *testing.T,
	env *testEnvironment,
	token, text string,
	privacy domain.PostPrivacy,
	selected []int64,
	file *postMultipartFile,
) postResponse {
	t.Helper()
	fields := []postMultipartField{{name: "text", value: text}, {name: "privacy", value: string(privacy)}}
	for _, userID := range selected {
		fields = append(fields, postMultipartField{name: "selected_user_id", value: strconv.FormatInt(userID, 10)})
	}
	files := []postMultipartFile{}
	if file != nil {
		files = append(files, *file)
	}
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, newPostRequest(t, token, fields, files))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s post: status=%d body=%q", privacy, rec.Code, rec.Body.String())
	}
	var response postResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode create post response: %v", err)
	}
	return response
}

func getPostPage(t *testing.T, env *testEnvironment, token, path string, wantStatus int) postPageResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodGet, path, token, nil))
	if rec.Code != wantStatus {
		t.Fatalf("GET %s: want %d, got %d body=%q", path, wantStatus, rec.Code, rec.Body.String())
	}
	if wantStatus != http.StatusOK {
		return postPageResponse{}
	}
	var response postPageResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode page %s: %v", path, err)
	}
	return response
}

func TestCreatePostsSupportsStrictPrivacyAudienceAndMediaContracts(t *testing.T) {
	env := newTestEnvironment(t)
	authorID, authorToken := env.createUserAndSession(t, "post-author@example.com")
	followerID, _ := env.createUserAndSession(t, "post-follower@example.com")
	pendingID, pendingToken := env.createUserAndSession(t, "post-pending@example.com")
	outsiderID, _ := env.createUserAndSession(t, "post-outsider@example.com")

	if follow, err := env.follows.Follow(context.Background(), followerID, authorID); err != nil || follow.Status != domain.FollowAccepted {
		t.Fatalf("create accepted follower: follow=%+v err=%v", follow, err)
	}
	setProfilePrivacy(t, env, authorToken, true)
	if follow, err := env.follows.Follow(context.Background(), pendingID, authorID); err != nil || follow.Status != domain.FollowPending {
		t.Fatalf("create pending follower: follow=%+v err=%v", follow, err)
	}
	setProfilePrivacy(t, env, authorToken, false)

	publicPost := createPostThroughHTTP(t, env, authorToken, "  trimmed public post  ", domain.PostPublic, nil, nil)
	if publicPost.Text != "trimmed public post" || publicPost.MediaURL != nil || publicPost.Author.ID != authorID {
		t.Fatalf("unexpected public post response: %+v", publicPost)
	}
	createPostThroughHTTP(t, env, authorToken, "followers post", domain.PostFollowers, nil, nil)
	selectedPost := createPostThroughHTTP(t, env, authorToken, "selected post", domain.PostSelected, []int64{followerID, followerID}, nil)
	var audienceCount int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM post_selected_users WHERE post_id = ?`, selectedPost.ID).Scan(&audienceCount); err != nil {
		t.Fatalf("count selected audience: %v", err)
	}
	if audienceCount != 1 {
		t.Fatalf("duplicate selected IDs were not normalized, count=%d", audienceCount)
	}

	png := []byte("\x89PNG\r\n\x1a\npost-media")
	mediaPost := createPostThroughHTTP(t, env, authorToken, "post with image", domain.PostPublic, nil, &postMultipartFile{
		fieldName: "media",
		filename:  "post.png",
		contents:  png,
	})
	if mediaPost.MediaURL == nil || *mediaPost.MediaURL != fmt.Sprintf("/api/posts/%d/media", mediaPost.ID) {
		t.Fatalf("unexpected media URL: %+v", mediaPost.MediaURL)
	}

	for name, fields := range map[string][]postMultipartField{
		"selected missing audience": {{name: "text", value: "post"}, {name: "privacy", value: "selected"}},
		"public with audience":      {{name: "text", value: "post"}, {name: "privacy", value: "public"}, {name: "selected_user_id", value: strconv.FormatInt(followerID, 10)}},
		"pending audience":          {{name: "text", value: "post"}, {name: "privacy", value: "selected"}, {name: "selected_user_id", value: strconv.FormatInt(pendingID, 10)}},
		"outsider audience":         {{name: "text", value: "post"}, {name: "privacy", value: "selected"}, {name: "selected_user_id", value: strconv.FormatInt(outsiderID, 10)}},
		"author audience":           {{name: "text", value: "post"}, {name: "privacy", value: "selected"}, {name: "selected_user_id", value: strconv.FormatInt(authorID, 10)}},
		"duplicate text":            {{name: "text", value: "one"}, {name: "text", value: "two"}, {name: "privacy", value: "public"}},
		"duplicate privacy":         {{name: "text", value: "post"}, {name: "privacy", value: "public"}, {name: "privacy", value: "followers"}},
		"unknown field":             {{name: "text", value: "post"}, {name: "privacy", value: "public"}, {name: "surprise", value: "value"}},
		"invalid privacy":           {{name: "text", value: "post"}, {name: "privacy", value: "private"}},
		"empty text":                {{name: "text", value: " \n\t "}, {name: "privacy", value: "public"}},
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, newPostRequest(t, authorToken, fields, nil))
			if rec.Code != http.StatusBadRequest || rec.Body.String() != "{\"error\":\"invalid input\"}\n" {
				t.Fatalf("expected exact 400, got %d body=%q", rec.Code, rec.Body.String())
			}
		})
	}

	for name, files := range map[string][]postMultipartFile{
		"unknown file field": {{fieldName: "avatar", filename: "image.png", contents: png}},
		"duplicate media": {
			{fieldName: "media", filename: "one.png", contents: png},
			{fieldName: "media", filename: "two.png", contents: png},
		},
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, newPostRequest(t, authorToken, []postMultipartField{{name: "text", value: "post"}, {name: "privacy", value: "public"}}, files))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%q", rec.Code, rec.Body.String())
			}
		})
	}

	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, newPostRequest(t, authorToken,
		[]postMultipartField{{name: "text", value: "bad media"}, {name: "privacy", value: "public"}},
		[]postMultipartFile{{fieldName: "media", filename: "bad.txt", contents: []byte("plain text")}},
	))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "JPEG") {
		t.Fatalf("invalid MIME: status=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, newPostRequest(t, authorToken,
		[]postMultipartField{{name: "text", value: "too large"}, {name: "privacy", value: "public"}},
		[]postMultipartFile{{fieldName: "media", filename: "large.png", contents: append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, service.MaxMediaBytes)...)}},
	))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized media: status=%d body=%q", rec.Code, rec.Body.String())
	}

	_ = pendingToken
}

func TestPostFeedAndProfileListsApplyCurrentAccessPolicyInSQL(t *testing.T) {
	env := newTestEnvironment(t)
	authorID, authorToken := env.createUserAndSession(t, "policy-author@example.com")
	followerID, followerToken := env.createUserAndSession(t, "policy-follower@example.com")
	selectedID, selectedToken := env.createUserAndSession(t, "policy-selected@example.com")
	_, outsiderToken := env.createUserAndSession(t, "policy-outsider@example.com")

	for _, followerID := range []int64{followerID, selectedID} {
		if follow, err := env.follows.Follow(context.Background(), followerID, authorID); err != nil || follow.Status != domain.FollowAccepted {
			t.Fatalf("accept follower %d: follow=%+v err=%v", followerID, follow, err)
		}
	}
	publicPost := createPostThroughHTTP(t, env, authorToken, "public policy post", domain.PostPublic, nil, nil)
	followersPost := createPostThroughHTTP(t, env, authorToken, "followers policy post", domain.PostFollowers, nil, nil)
	selectedPost := createPostThroughHTTP(t, env, authorToken, "selected policy post", domain.PostSelected, []int64{selectedID}, nil)

	profilePath := fmt.Sprintf("/api/users/%d/posts?limit=10", authorID)
	if got := getPostPage(t, env, outsiderToken, profilePath, http.StatusOK); len(got.Posts) != 1 || got.Posts[0].ID != publicPost.ID {
		t.Fatalf("public outsider profile posts: %+v", got.Posts)
	}
	if got := getPostPage(t, env, followerToken, profilePath, http.StatusOK); len(got.Posts) != 2 || got.Posts[0].ID != followersPost.ID {
		t.Fatalf("accepted follower profile posts: %+v", got.Posts)
	}
	if got := getPostPage(t, env, selectedToken, profilePath, http.StatusOK); len(got.Posts) != 3 || got.Posts[0].ID != selectedPost.ID {
		t.Fatalf("selected follower profile posts: %+v", got.Posts)
	}
	if got := getPostPage(t, env, authorToken, profilePath, http.StatusOK); len(got.Posts) != 3 {
		t.Fatalf("author must bypass post privacy: %+v", got.Posts)
	}

	if got := getPostPage(t, env, outsiderToken, "/api/posts/feed?limit=10", http.StatusOK); len(got.Posts) != 0 {
		t.Fatalf("feed must not be global public discovery: %+v", got.Posts)
	}
	if got := getPostPage(t, env, followerToken, "/api/posts/feed?limit=10", http.StatusOK); len(got.Posts) != 2 {
		t.Fatalf("accepted follower feed: %+v", got.Posts)
	}
	if got := getPostPage(t, env, selectedToken, "/api/posts/feed?limit=10", http.StatusOK); len(got.Posts) != 3 {
		t.Fatalf("selected follower feed: %+v", got.Posts)
	}

	setProfilePrivacy(t, env, authorToken, true)
	getPostPage(t, env, outsiderToken, profilePath, http.StatusForbidden)
	if got := getPostPage(t, env, followerToken, profilePath, http.StatusOK); len(got.Posts) != 2 {
		t.Fatalf("accepted follower lost private profile access: %+v", got.Posts)
	}

	if err := env.follows.Unfollow(context.Background(), selectedID, authorID); err != nil {
		t.Fatalf("unfollow selected author: %v", err)
	}
	getPostPage(t, env, selectedToken, profilePath, http.StatusForbidden)
	setProfilePrivacy(t, env, authorToken, false)
	if got := getPostPage(t, env, selectedToken, profilePath, http.StatusOK); len(got.Posts) != 1 || got.Posts[0].ID != publicPost.ID {
		t.Fatalf("unfollow did not remove followers/selected access: %+v", got.Posts)
	}
	follow, err := env.follows.Follow(context.Background(), selectedID, authorID)
	if err != nil || follow.Status != domain.FollowAccepted {
		t.Fatalf("restore accepted relation: follow=%+v err=%v", follow, err)
	}
	if got := getPostPage(t, env, selectedToken, profilePath, http.StatusOK); len(got.Posts) != 3 {
		t.Fatalf("re-follow did not restore selected access: %+v", got.Posts)
	}
	var audienceCount int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM post_selected_users WHERE post_id = ? AND user_id = ?`, selectedPost.ID, selectedID).Scan(&audienceCount); err != nil || audienceCount != 1 {
		t.Fatalf("selected audience was not preserved: count=%d err=%v", audienceCount, err)
	}

	getPostPage(t, env, outsiderToken, "/api/users/999999/posts", http.StatusNotFound)
}

func TestPostPaginationIsStableAtEqualTimestampsAndFiltersBeforeLimit(t *testing.T) {
	env := newTestEnvironment(t)
	userID, token := env.createUserAndSession(t, "pagination@example.com")
	createdIDs := make([]int64, 0, 5)
	for index := 0; index < 5; index++ {
		post := createPostThroughHTTP(t, env, token, fmt.Sprintf("page post %d", index), domain.PostPublic, nil, nil)
		createdIDs = append(createdIDs, post.ID)
	}

	seen := make([]int64, 0, 5)
	path := "/api/posts/feed?limit=2"
	for {
		page := getPostPage(t, env, token, path, http.StatusOK)
		for _, post := range page.Posts {
			seen = append(seen, post.ID)
		}
		if page.NextCursor == nil {
			break
		}
		path = "/api/posts/feed?limit=2&cursor=" + *page.NextCursor
	}
	if len(seen) != len(createdIDs) {
		t.Fatalf("pagination returned %d IDs, want %d: %v", len(seen), len(createdIDs), seen)
	}
	unique := map[int64]bool{}
	for index, id := range seen {
		if unique[id] {
			t.Fatalf("duplicate post %d across pages: %v", id, seen)
		}
		unique[id] = true
		want := createdIDs[len(createdIDs)-1-index]
		if id != want {
			t.Fatalf("unstable equal-timestamp order at %d: got %d want %d", index, id, want)
		}
	}

	for _, invalidPath := range []string{
		"/api/posts/feed?cursor=bad",
		"/api/posts/feed?cursor=",
		"/api/posts/feed?limit=0",
		"/api/posts/feed?limit=51",
		"/api/posts/feed?limit=2&limit=3",
		"/api/posts/feed?unknown=1",
	} {
		getPostPage(t, env, token, invalidPath, http.StatusBadRequest)
	}

	authorID, authorToken := env.createUserAndSession(t, "filter-before-limit-author@example.com")
	viewerID, viewerToken := env.createUserAndSession(t, "filter-before-limit-viewer@example.com")
	otherID, _ := env.createUserAndSession(t, "filter-before-limit-other@example.com")
	if _, err := env.follows.Follow(context.Background(), viewerID, authorID); err != nil {
		t.Fatalf("viewer follow author: %v", err)
	}
	if _, err := env.follows.Follow(context.Background(), otherID, authorID); err != nil {
		t.Fatalf("other follow author: %v", err)
	}
	accessible := createPostThroughHTTP(t, env, authorToken, "accessible older post", domain.PostPublic, nil, nil)
	for index := 0; index < 3; index++ {
		createPostThroughHTTP(t, env, authorToken, fmt.Sprintf("inaccessible selected %d", index), domain.PostSelected, []int64{otherID}, nil)
	}
	page := getPostPage(t, env, viewerToken, fmt.Sprintf("/api/users/%d/posts?limit=1", authorID), http.StatusOK)
	if len(page.Posts) != 1 || page.Posts[0].ID != accessible.ID {
		t.Fatalf("privacy was filtered after LIMIT: %+v", page.Posts)
	}

	_ = userID
}

func TestPostMediaDeliveryUsesCurrentPostAccessPolicy(t *testing.T) {
	env := newTestEnvironment(t)
	authorID, authorToken := env.createUserAndSession(t, "media-author@example.com")
	followerID, followerToken := env.createUserAndSession(t, "media-follower@example.com")
	otherID, outsiderToken := env.createUserAndSession(t, "media-outsider@example.com")
	if _, err := env.follows.Follow(context.Background(), followerID, authorID); err != nil {
		t.Fatalf("follower follow author: %v", err)
	}

	png := []byte("\x89PNG\r\n\x1a\ncontrolled-post-media")
	publicPost := createPostThroughHTTP(t, env, authorToken, "public image", domain.PostPublic, nil, &postMultipartFile{fieldName: "media", filename: "public.png", contents: png})
	followersPost := createPostThroughHTTP(t, env, authorToken, "followers image", domain.PostFollowers, nil, &postMultipartFile{fieldName: "media", filename: "followers.png", contents: png})
	textPost := createPostThroughHTTP(t, env, authorToken, "text only", domain.PostPublic, nil, nil)

	assertMedia := func(token string, postID int64, wantStatus int, wantBody []byte) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/posts/%d/media", postID), nil)
		if token != "" {
			addSessionCookie(req, token)
		}
		env.handler.ServeHTTP(rec, req)
		if rec.Code != wantStatus {
			t.Fatalf("post %d media: want %d got %d body=%q", postID, wantStatus, rec.Code, rec.Body.String())
		}
		if wantBody != nil && !bytes.Equal(rec.Body.Bytes(), wantBody) {
			t.Fatalf("post %d media bytes=%q want=%q", postID, rec.Body.Bytes(), wantBody)
		}
		return rec
	}

	publicRec := assertMedia(outsiderToken, publicPost.ID, http.StatusOK, png)
	if publicRec.Header().Get("Content-Type") != "image/png" ||
		publicRec.Header().Get("Content-Length") != strconv.Itoa(len(png)) ||
		publicRec.Header().Get("X-Content-Type-Options") != "nosniff" ||
		publicRec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("unexpected post media headers: %+v", publicRec.Header())
	}
	assertMedia("", publicPost.ID, http.StatusUnauthorized, nil)
	assertMedia(outsiderToken, followersPost.ID, http.StatusForbidden, nil)
	assertMedia(followerToken, followersPost.ID, http.StatusOK, png)
	assertMedia(authorToken, followersPost.ID, http.StatusOK, png)
	assertMedia(outsiderToken, textPost.ID, http.StatusNotFound, nil)
	assertMedia(outsiderToken, 999999, http.StatusNotFound, nil)

	if err := env.follows.Unfollow(context.Background(), followerID, authorID); err != nil {
		t.Fatalf("unfollow author: %v", err)
	}
	assertMedia(followerToken, followersPost.ID, http.StatusForbidden, nil)
	if _, err := env.follows.Follow(context.Background(), followerID, authorID); err != nil {
		t.Fatalf("re-follow author: %v", err)
	}
	assertMedia(followerToken, followersPost.ID, http.StatusOK, png)

	setProfilePrivacy(t, env, authorToken, true)
	assertMedia(outsiderToken, publicPost.ID, http.StatusForbidden, nil)
	assertMedia(followerToken, publicPost.ID, http.StatusOK, png)
	setProfilePrivacy(t, env, authorToken, false)

	var storageKey string
	if err := env.db.QueryRow(`
		SELECT m.storage_key
		FROM posts p JOIN media m ON m.id = p.media_id
		WHERE p.id = ?
	`, publicPost.ID).Scan(&storageKey); err != nil {
		t.Fatalf("get post storage key: %v", err)
	}
	if err := os.Remove(filepath.Join(env.uploadDir, storageKey)); err != nil {
		t.Fatalf("remove post media file: %v", err)
	}
	assertMedia(outsiderToken, publicPost.ID, http.StatusNotFound, nil)

	foreignMediaID, err := env.dbInsertMedia(otherID, "foreign.png", png)
	if err != nil {
		t.Fatalf("create foreign media: %v", err)
	}
	if _, err := env.db.Exec(`UPDATE posts SET media_id = ? WHERE id = ?`, foreignMediaID, textPost.ID); err != nil {
		t.Fatalf("attach foreign media: %v", err)
	}
	assertMedia(outsiderToken, textPost.ID, http.StatusNotFound, nil)
}

func (e *testEnvironment) dbInsertMedia(ownerID int64, storageKey string, contents []byte) (int64, error) {
	if err := os.WriteFile(filepath.Join(e.uploadDir, storageKey), contents, 0o600); err != nil {
		return 0, err
	}
	result, err := e.db.Exec(`
		INSERT INTO media (owner_user_id, mime, size, storage_key, original_name, created_at)
		VALUES (?, 'image/png', ?, ?, ?, ?)
	`, ownerID, len(contents), storageKey, storageKey, testNow.Unix())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
