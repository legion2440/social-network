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

func commentMultipartRequest(
	t *testing.T,
	method, path, token string,
	fields []postMultipartField,
	files []postMultipartFile,
) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, field := range fields {
		if err := writer.WriteField(field.name, field.value); err != nil {
			t.Fatalf("write comment field %s: %v", field.name, err)
		}
	}
	for _, file := range files {
		part, err := writer.CreateFormFile(file.fieldName, file.filename)
		if err != nil {
			t.Fatalf("create comment file %s: %v", file.fieldName, err)
		}
		if _, err := part.Write(file.contents); err != nil {
			t.Fatalf("write comment file %s: %v", file.fieldName, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close comment multipart: %v", err)
	}
	req := authenticatedRequest(method, path, token, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func createCommentThroughHTTP(t *testing.T, env *testEnvironment, token string, postID int64, text string, wantStatus int) *commentResponse {
	t.Helper()
	return createCommentWithMediaThroughHTTP(t, env, token, postID, text, nil, wantStatus)
}

func createCommentWithMediaThroughHTTP(
	t *testing.T,
	env *testEnvironment,
	token string,
	postID int64,
	text string,
	file *postMultipartFile,
	wantStatus int,
) *commentResponse {
	t.Helper()
	files := []postMultipartFile(nil)
	if file != nil {
		files = append(files, *file)
	}
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, commentMultipartRequest(
		t,
		http.MethodPost,
		"/api/posts/"+strconv.FormatInt(postID, 10)+"/comments",
		token,
		[]postMultipartField{{name: "text", value: text}},
		files,
	))
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

func TestCreateCommentStrictMultipartAndPostCountContract(t *testing.T) {
	env := newTestEnvironment(t)
	_, authorToken := env.createUserAndSession(t, "comment-contract-author@example.com")
	commenterID, commenterToken := env.createUserAndSession(t, "comment-contract-user@example.com")
	post := createPostThroughHTTP(t, env, authorToken, "commented post", domain.PostPublic, nil, nil)
	path := "/api/posts/" + strconv.FormatInt(post.ID, 10) + "/comments"

	created := createCommentThroughHTTP(t, env, commenterToken, post.ID, "  hello 🙂  ", http.StatusCreated)
	if created.Text != "hello 🙂" || created.PostID != post.ID || created.Author.ID != commenterID || created.MediaURL != nil {
		t.Fatalf("unexpected comment response: %+v", created)
	}

	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodGet, path, commenterToken, nil))
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

	for name, fields := range map[string][]postMultipartField{
		"missing text":   {},
		"empty text":     {{name: "text", value: ""}},
		"whitespace":     {{name: "text", value: "  \n\t "}},
		"unknown field":  {{name: "text", value: "ok"}, {name: "extra", value: "true"}},
		"duplicate text": {{name: "text", value: "one"}, {name: "text", value: "two"}},
		"too many runes": {{name: "text", value: strings.Repeat("🙂", service.MaxCommentTextRunes+1)}},
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, commentMultipartRequest(t, http.MethodPost, path, commenterToken, fields, nil))
			if rec.Code != http.StatusBadRequest || rec.Body.String() != "{\"error\":\"invalid input\"}\n" {
				t.Fatalf("expected exact 400, got %d body=%q", rec.Code, rec.Body.String())
			}
		})
	}

	for name, files := range map[string][]postMultipartFile{
		"unknown file field": {{fieldName: "extra", filename: "x.png", contents: []byte("\x89PNG\r\n\x1a\nx")}},
		"duplicate media": {
			{fieldName: "media", filename: "one.png", contents: []byte("\x89PNG\r\n\x1a\none")},
			{fieldName: "media", filename: "two.png", contents: []byte("\x89PNG\r\n\x1a\ntwo")},
		},
		"empty media":       {{fieldName: "media", filename: "empty.png", contents: nil}},
		"unsupported media": {{fieldName: "media", filename: "bad.txt", contents: []byte("plain text")}},
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, commentMultipartRequest(
				t,
				http.MethodPost,
				path,
				commenterToken,
				[]postMultipartField{{name: "text", value: "ok"}},
				files,
			))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%q", rec.Code, rec.Body.String())
			}
		})
	}

	t.Run("media only", func(t *testing.T) {
		rec := httptest.NewRecorder()
		env.handler.ServeHTTP(rec, commentMultipartRequest(
			t,
			http.MethodPost,
			path,
			commenterToken,
			nil,
			[]postMultipartFile{{fieldName: "media", filename: "image.png", contents: []byte("\x89PNG\r\n\x1a\nmedia")}},
		))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%q", rec.Code, rec.Body.String())
		}
	})

	rec = httptest.NewRecorder()
	req := authenticatedRequest(http.MethodPost, path, commenterToken, strings.NewReader(`{"text":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	env.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("JSON content type: status=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = authenticatedRequest(http.MethodPost, path, commenterToken, strings.NewReader("broken multipart body"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	env.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed multipart: status=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, commentMultipartRequest(
		t,
		http.MethodPost,
		path,
		commenterToken,
		[]postMultipartField{{name: "text", value: "oversized"}},
		[]postMultipartFile{{fieldName: "media", filename: "too-large.png", contents: bytes.Repeat([]byte{'x'}, int(service.MaxMediaBytes+1))}},
	))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized comment media: status=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, commentMultipartRequest(
		t,
		http.MethodPost,
		path,
		commenterToken,
		[]postMultipartField{{name: "text", value: "oversized body"}},
		[]postMultipartFile{{fieldName: "media", filename: "oversized-body.png", contents: bytes.Repeat([]byte{'x'}, int(service.MaxMediaBodyBytes+1))}},
	))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized multipart body: status=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, commentMultipartRequest(
		t,
		http.MethodPost,
		"/api/posts/not-a-number/comments",
		commenterToken,
		[]postMultipartField{{name: "text", value: "ok"}},
		nil,
	))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed post id: status=%d body=%q", rec.Code, rec.Body.String())
	}
	createCommentThroughHTTP(t, env, commenterToken, post.ID+9999, "missing", http.StatusNotFound)

	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, commentMultipartRequest(
		t,
		http.MethodPost,
		path,
		"",
		[]postMultipartField{{name: "text", value: "no session"}},
		nil,
	))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated comment: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestCommentMediaDeliveryUsesCurrentPostAccessPolicy(t *testing.T) {
	env := newTestEnvironment(t)
	authorID, authorToken := env.createUserAndSession(t, "comment-media-author@example.com")
	followerID, followerToken := env.createUserAndSession(t, "comment-media-follower@example.com")
	otherID, outsiderToken := env.createUserAndSession(t, "comment-media-outsider@example.com")
	if _, err := env.follows.Follow(context.Background(), followerID, authorID); err != nil {
		t.Fatalf("follower follows author: %v", err)
	}
	publicPost := createPostThroughHTTP(t, env, authorToken, "public comment media", domain.PostPublic, nil, nil)
	followersPost := createPostThroughHTTP(t, env, authorToken, "followers comment media", domain.PostFollowers, nil, nil)

	assertMedia := func(token string, commentID int64, wantStatus int, wantBody []byte) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/comments/%d/media", commentID), nil)
		if token != "" {
			addSessionCookie(req, token)
		}
		env.handler.ServeHTTP(rec, req)
		if rec.Code != wantStatus {
			t.Fatalf("comment %d media: want %d got %d body=%q", commentID, wantStatus, rec.Code, rec.Body.String())
		}
		if wantBody != nil && !bytes.Equal(rec.Body.Bytes(), wantBody) {
			t.Fatalf("comment %d media bytes=%q want=%q", commentID, rec.Body.Bytes(), wantBody)
		}
		return rec
	}

	formats := []struct {
		name     string
		filename string
		mime     string
		contents []byte
	}{
		{name: "jpeg", filename: "photo.jpg", mime: "image/jpeg", contents: []byte("\xff\xd8\xff\xe0comment-jpeg")},
		{name: "png", filename: "photo.png", mime: "image/png", contents: []byte("\x89PNG\r\n\x1a\ncomment-png")},
		{name: "gif", filename: "photo.gif", mime: "image/gif", contents: []byte("GIF89acomment-gif")},
		{name: "webp", filename: "photo.webp", mime: "image/webp", contents: []byte("RIFF\x10\x00\x00\x00WEBPVP8 comment-webp")},
	}
	var first *commentResponse
	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			comment := createCommentWithMediaThroughHTTP(t, env, authorToken, publicPost.ID, format.name, &postMultipartFile{
				fieldName: "media", filename: format.filename, contents: format.contents,
			}, http.StatusCreated)
			if comment.MediaURL == nil || *comment.MediaURL != fmt.Sprintf("/api/comments/%d/media", comment.ID) {
				t.Fatalf("unexpected comment media URL: %+v", comment.MediaURL)
			}
			rec := assertMedia(outsiderToken, comment.ID, http.StatusOK, format.contents)
			if rec.Header().Get("Content-Type") != format.mime ||
				rec.Header().Get("Content-Length") != strconv.Itoa(len(format.contents)) ||
				rec.Header().Get("X-Content-Type-Options") != "nosniff" ||
				rec.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("unexpected comment media headers: %+v", rec.Header())
			}
			if first == nil {
				first = comment
			}
		})
	}

	followerComment := createCommentWithMediaThroughHTTP(t, env, authorToken, followersPost.ID, "followers", &postMultipartFile{
		fieldName: "media", filename: "followers.png", contents: formats[1].contents,
	}, http.StatusCreated)
	noMedia := createCommentThroughHTTP(t, env, authorToken, publicPost.ID, "text only", http.StatusCreated)
	assertMedia("", first.ID, http.StatusUnauthorized, nil)
	assertMedia(outsiderToken, followerComment.ID, http.StatusForbidden, nil)
	assertMedia(followerToken, followerComment.ID, http.StatusOK, formats[1].contents)
	assertMedia(outsiderToken, noMedia.ID, http.StatusNotFound, nil)
	assertMedia(outsiderToken, 999999, http.StatusNotFound, nil)

	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPost, fmt.Sprintf("/api/comments/%d/media", first.ID), outsiderToken, nil))
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("comment media method contract: status=%d allow=%q body=%q", rec.Code, rec.Header().Get("Allow"), rec.Body.String())
	}
	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodGet, "/api/comments/not-a-number/media", outsiderToken, nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed comment media id: status=%d body=%q", rec.Code, rec.Body.String())
	}

	if err := env.follows.Unfollow(context.Background(), followerID, authorID); err != nil {
		t.Fatalf("unfollow author: %v", err)
	}
	assertMedia(followerToken, followerComment.ID, http.StatusForbidden, nil)
	if _, err := env.follows.Follow(context.Background(), followerID, authorID); err != nil {
		t.Fatalf("re-follow author: %v", err)
	}
	assertMedia(followerToken, followerComment.ID, http.StatusOK, formats[1].contents)

	setProfilePrivacy(t, env, authorToken, true)
	assertMedia(outsiderToken, first.ID, http.StatusForbidden, nil)
	assertMedia(outsiderToken, noMedia.ID, http.StatusForbidden, nil)
	assertMedia(followerToken, first.ID, http.StatusOK, formats[0].contents)
	setProfilePrivacy(t, env, authorToken, false)

	var storageKey string
	if err := env.db.QueryRow(`
		SELECT m.storage_key
		FROM post_comments c JOIN media m ON m.id = c.media_id
		WHERE c.id = ?
	`, first.ID).Scan(&storageKey); err != nil {
		t.Fatalf("get comment media storage key: %v", err)
	}
	if err := os.Remove(filepath.Join(env.uploadDir, storageKey)); err != nil {
		t.Fatalf("remove comment media file: %v", err)
	}
	assertMedia(outsiderToken, first.ID, http.StatusNotFound, nil)

	foreignMediaID, err := env.dbInsertMedia(otherID, "foreign-comment.png", formats[1].contents)
	if err != nil {
		t.Fatalf("create foreign comment media: %v", err)
	}
	if _, err := env.db.Exec(`UPDATE post_comments SET media_id = ? WHERE id = ?`, foreignMediaID, noMedia.ID); err != nil {
		t.Fatalf("attach foreign comment media: %v", err)
	}
	assertMedia(outsiderToken, noMedia.ID, http.StatusNotFound, nil)

	missingRowComment := createCommentThroughHTTP(t, env, authorToken, publicPost.ID, "missing media row", http.StatusCreated)
	if _, err := env.db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable foreign keys for corrupt-row fixture: %v", err)
	}
	if _, err := env.db.Exec(`UPDATE post_comments SET media_id = 999999 WHERE id = ?`, missingRowComment.ID); err != nil {
		t.Fatalf("attach missing media row: %v", err)
	}
	if _, err := env.db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("restore foreign keys: %v", err)
	}
	assertMedia(outsiderToken, missingRowComment.ID, http.StatusNotFound, nil)
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
