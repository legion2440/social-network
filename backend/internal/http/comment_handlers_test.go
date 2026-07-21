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
	"social-network/backend/internal/service"
)

func commentJSONRequest(method, path, token, body string) *http.Request {
	req := authenticatedRequest(method, path, token, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func createCommentThroughHTTP(t *testing.T, env *testEnvironment, token string, postID int64, text string, wantStatus int) *commentResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, commentJSONRequest(http.MethodPost, "/api/posts/"+strconv.FormatInt(postID, 10)+"/comments", token, `{"text":`+strconv.Quote(text)+`}`))
	if rec.Code != wantStatus {
		t.Fatalf("create comment: want %d, got %d body=%q", wantStatus, rec.Code, rec.Body.String())
	}
	if wantStatus != http.StatusCreated {
		return nil
	}
	var response commentResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode comment: %v", err)
	}
	return &response
}

func getCommentPage(t *testing.T, env *testEnvironment, token, path string, wantStatus int) commentPageResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodGet, path, token, nil))
	if rec.Code != wantStatus {
		t.Fatalf("GET %s: want %d, got %d body=%q", path, wantStatus, rec.Code, rec.Body.String())
	}
	if wantStatus != http.StatusOK {
		return commentPageResponse{}
	}
	var response commentPageResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode comment page: %v", err)
	}
	return response
}

func TestCreateCommentStrictJSONAndPostCountContract(t *testing.T) {
	env := newTestEnvironment(t)
	_, authorToken := env.createUserAndSession(t, "comment-contract-author@example.com")
	commenterID, commenterToken := env.createUserAndSession(t, "comment-contract-user@example.com")
	post := createPostThroughHTTP(t, env, authorToken, "commented post", domain.PostPublic, nil, nil)

	created := createCommentThroughHTTP(t, env, commenterToken, post.ID, "  hello 🙂  ", http.StatusCreated)
	if created.Text != "hello 🙂" || created.PostID != post.ID || created.Author.ID != commenterID {
		t.Fatalf("unexpected comment response: %+v", created)
	}

	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodGet, "/api/posts/"+strconv.FormatInt(post.ID, 10)+"/comments", commenterToken, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list comments for response contract: status=%d body=%q", rec.Code, rec.Body.String())
	}
	var rawPage struct {
		Comments []map[string]any `json:"comments"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&rawPage); err != nil || len(rawPage.Comments) != 1 {
		t.Fatalf("decode raw comment response: comments=%+v err=%v", rawPage.Comments, err)
	}
	rawAuthor, ok := rawPage.Comments[0]["author"].(map[string]any)
	if !ok {
		t.Fatalf("comment author is not an object: %#v", rawPage.Comments[0]["author"])
	}
	wantAuthorKeys := map[string]bool{
		"id": true, "first_name": true, "last_name": true,
		"nickname": true, "avatar_url": true, "is_private": true,
	}
	if len(rawAuthor) != len(wantAuthorKeys) {
		t.Fatalf("comment author exposed unexpected fields: %#v", rawAuthor)
	}
	for key := range rawAuthor {
		if !wantAuthorKeys[key] {
			t.Fatalf("comment author exposed unexpected field %q: %#v", key, rawAuthor)
		}
	}

	page := getPostPage(t, env, authorToken, "/api/posts/feed?limit=20", http.StatusOK)
	if len(page.Posts) != 1 || page.Posts[0].CommentsCount != 1 {
		t.Fatalf("post response did not expose comments_count: %+v", page.Posts)
	}
	var storedCount int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM post_comments WHERE post_id = ?`, post.ID).Scan(&storedCount); err != nil || storedCount != 1 {
		t.Fatalf("unexpected stored comment count=%d err=%v", storedCount, err)
	}

	for name, body := range map[string]string{
		"missing text":   `{}`,
		"empty text":     `{"text":""}`,
		"whitespace":     `{"text":"  \n\t "}`,
		"null text":      `{"text":null}`,
		"numeric text":   `{"text":42}`,
		"unknown field":  `{"text":"ok","extra":true}`,
		"duplicate text": `{"text":"one","text":"two"}`,
		"trailing JSON":  `{"text":"ok"}{}`,
		"malformed JSON": `{"text":`,
		"too many runes": `{"text":` + strconv.Quote(strings.Repeat("🙂", service.MaxCommentTextRunes+1)) + `}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, commentJSONRequest(http.MethodPost, "/api/posts/"+strconv.FormatInt(post.ID, 10)+"/comments", commenterToken, body))
			if rec.Code != http.StatusBadRequest || rec.Body.String() != "{\"error\":\"invalid input\"}\n" {
				t.Fatalf("expected exact 400, got %d body=%q", rec.Code, rec.Body.String())
			}
		})
	}

	rec = httptest.NewRecorder()
	req := authenticatedRequest(http.MethodPost, "/api/posts/"+strconv.FormatInt(post.ID, 10)+"/comments", commenterToken, strings.NewReader(`{"text":"ok"}`))
	req.Header.Set("Content-Type", "text/plain")
	env.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("wrong content type: status=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, commentJSONRequest(http.MethodPost, "/api/posts/"+strconv.FormatInt(post.ID, 10)+"/comments", commenterToken, strings.Repeat(" ", maxCommentJSONBytes+1)))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized comment body: status=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, commentJSONRequest(http.MethodPost, "/api/posts/not-a-number/comments", commenterToken, `{"text":"ok"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed post id: status=%d body=%q", rec.Code, rec.Body.String())
	}
	createCommentThroughHTTP(t, env, commenterToken, post.ID+9999, "missing", http.StatusNotFound)

	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, commentJSONRequest(http.MethodPost, "/api/posts/"+strconv.FormatInt(post.ID, 10)+"/comments", "", `{"text":"no session"}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated comment: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestCommentsInheritCurrentPostAccessPolicy(t *testing.T) {
	env := newTestEnvironment(t)
	authorID, authorToken := env.createUserAndSession(t, "comment-policy-author@example.com")
	acceptedID, acceptedToken := env.createUserAndSession(t, "comment-policy-accepted@example.com")
	selectedID, selectedToken := env.createUserAndSession(t, "comment-policy-selected@example.com")
	pendingID, pendingToken := env.createUserAndSession(t, "comment-policy-pending@example.com")
	_, outsiderToken := env.createUserAndSession(t, "comment-policy-outsider@example.com")

	for _, followerID := range []int64{acceptedID, selectedID} {
		if follow, err := env.follows.Follow(context.Background(), followerID, authorID); err != nil || follow.Status != domain.FollowAccepted {
			t.Fatalf("accept follower %d: follow=%+v err=%v", followerID, follow, err)
		}
	}
	setProfilePrivacy(t, env, authorToken, true)
	if follow, err := env.follows.Follow(context.Background(), pendingID, authorID); err != nil || follow.Status != domain.FollowPending {
		t.Fatalf("create pending follower: follow=%+v err=%v", follow, err)
	}
	setProfilePrivacy(t, env, authorToken, false)

	publicPost := createPostThroughHTTP(t, env, authorToken, "public comments", domain.PostPublic, nil, nil)
	followersPost := createPostThroughHTTP(t, env, authorToken, "followers comments", domain.PostFollowers, nil, nil)
	selectedPost := createPostThroughHTTP(t, env, authorToken, "selected comments", domain.PostSelected, []int64{selectedID}, nil)

	createCommentThroughHTTP(t, env, outsiderToken, publicPost.ID, "public outsider", http.StatusCreated)
	createCommentThroughHTTP(t, env, acceptedToken, followersPost.ID, "accepted follower", http.StatusCreated)
	createCommentThroughHTTP(t, env, pendingToken, followersPost.ID, "pending denied", http.StatusForbidden)
	createCommentThroughHTTP(t, env, outsiderToken, followersPost.ID, "outsider denied", http.StatusForbidden)
	createCommentThroughHTTP(t, env, selectedToken, selectedPost.ID, "selected follower", http.StatusCreated)
	createCommentThroughHTTP(t, env, acceptedToken, selectedPost.ID, "accepted but unselected", http.StatusForbidden)
	createCommentThroughHTTP(t, env, authorToken, selectedPost.ID, "author bypass", http.StatusCreated)

	setProfilePrivacy(t, env, authorToken, true)
	getCommentPage(t, env, acceptedToken, "/api/posts/"+strconv.FormatInt(publicPost.ID, 10)+"/comments", http.StatusOK)
	getCommentPage(t, env, outsiderToken, "/api/posts/"+strconv.FormatInt(publicPost.ID, 10)+"/comments", http.StatusForbidden)

	if err := env.follows.Unfollow(context.Background(), selectedID, authorID); err != nil {
		t.Fatalf("unfollow selected user: %v", err)
	}
	getCommentPage(t, env, selectedToken, "/api/posts/"+strconv.FormatInt(selectedPost.ID, 10)+"/comments", http.StatusForbidden)
	setProfilePrivacy(t, env, authorToken, false)
	if follow, err := env.follows.Follow(context.Background(), selectedID, authorID); err != nil || follow.Status != domain.FollowAccepted {
		t.Fatalf("restore selected follower: follow=%+v err=%v", follow, err)
	}
	setProfilePrivacy(t, env, authorToken, true)
	getCommentPage(t, env, selectedToken, "/api/posts/"+strconv.FormatInt(selectedPost.ID, 10)+"/comments", http.StatusOK)
}

func TestCommentPaginationIsAscendingStableAndStrict(t *testing.T) {
	env := newTestEnvironment(t)
	_, authorToken := env.createUserAndSession(t, "comment-page-author@example.com")
	_, commenterToken := env.createUserAndSession(t, "comment-page-user@example.com")
	post := createPostThroughHTTP(t, env, authorToken, "paginated comments", domain.PostPublic, nil, nil)
	for _, text := range []string{"first", "second", "third"} {
		createCommentThroughHTTP(t, env, commenterToken, post.ID, text, http.StatusCreated)
	}

	basePath := "/api/posts/" + strconv.FormatInt(post.ID, 10) + "/comments"
	first := getCommentPage(t, env, authorToken, basePath+"?limit=2", http.StatusOK)
	if len(first.Comments) != 2 || first.Comments[0].Text != "first" || first.Comments[1].Text != "second" || first.NextCursor == nil {
		t.Fatalf("unexpected first comment page: %+v", first)
	}
	second := getCommentPage(t, env, authorToken, basePath+"?limit=2&cursor="+*first.NextCursor, http.StatusOK)
	if len(second.Comments) != 1 || second.Comments[0].Text != "third" || second.NextCursor != nil {
		t.Fatalf("unexpected second comment page: %+v", second)
	}
	if first.Comments[0].ID == first.Comments[1].ID || first.Comments[1].ID == second.Comments[0].ID {
		t.Fatal("comment pages contained duplicate ids")
	}

	for _, suffix := range []string{"?limit=0", "?limit=51", "?limit=2&limit=3", "?cursor=bad", "?unknown=1"} {
		getCommentPage(t, env, authorToken, basePath+suffix, http.StatusBadRequest)
	}
}
