package http

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"social-network/backend/internal/config"
	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"

	"github.com/gorilla/websocket"
)

var testNow = time.Unix(1_700_000_000, 0).UTC()

type fixedClock struct{}

func (fixedClock) Now() time.Time { return testNow }

type sequenceID struct {
	mu sync.Mutex
	n  int
}

func (g *sequenceID) New() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return fmt.Sprintf("test-id-%d", g.n), nil
}

type testEnvironment struct {
	db           *sql.DB
	handler      http.Handler
	users        *sqlite.UserRepo
	sessions     *service.SessionService
	uploadDir    string
	cookieSecure bool
}

func newTestEnvironment(t *testing.T) *testEnvironment {
	t.Helper()
	root := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(root, "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ids := &sequenceID{}
	sessions := service.NewSessionService(sqlite.NewSessionRepo(db), fixedClock{}, ids, 24*time.Hour)
	uploadDir := filepath.Join(root, "uploads")
	media, err := service.NewMediaService(sqlite.NewMediaRepo(db), fixedClock{}, ids, uploadDir)
	if err != nil {
		t.Fatalf("new media service: %v", err)
	}

	return &testEnvironment{
		db:        db,
		handler:   NewHandler(db, sessions, media, false, nil).Routes(),
		users:     sqlite.NewUserRepo(db),
		sessions:  sessions,
		uploadDir: uploadDir,
	}
}

func (e *testEnvironment) createUserAndSession(t *testing.T, email string) (int64, string) {
	t.Helper()
	userID, err := e.users.Create(context.Background(), &domain.User{
		Email:        email,
		PasswordHash: "test-password-hash",
		FirstName:    "Test",
		LastName:     "User",
		DateOfBirth:  "02-01-1990",
		CreatedAt:    testNow,
		UpdatedAt:    testNow,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	session, err := e.sessions.Create(context.Background(), userID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return userID, session.Token
}

func addSessionCookie(req *http.Request, token string) {
	req.AddCookie(&http.Cookie{Name: config.SessionCookieName, Value: token, Path: "/"})
}

func TestHealthIsPublicAndChecksDatabase(t *testing.T) {
	env := newTestEnvironment(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected content type %q", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != "{\"ok\":true}\n" {
		t.Fatalf("unexpected health body %q", rec.Body.String())
	}

	if err := env.db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}
	rec = httptest.NewRecorder()
	env.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusServiceUnavailable || rec.Body.String() != "{\"ok\":false}\n" {
		t.Fatalf("expected unavailable health, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestReservedAPIGroupsReturnJSONNotImplemented(t *testing.T) {
	env := newTestEnvironment(t)
	for _, path := range []string{"/api/auth", "/api/posts/42", "/api/notifications/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		env.handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotImplemented {
			t.Fatalf("%s: expected 501, got %d", path, rec.Code)
		}
		if rec.Header().Get("Content-Type") != "application/json" || rec.Body.String() != "{\"error\":\"not implemented\"}\n" {
			t.Fatalf("%s: unexpected response %q", path, rec.Body.String())
		}
	}
}

func TestMediaUploadAndOwnerOnlyDelivery(t *testing.T) {
	env := newTestEnvironment(t)
	_, ownerToken := env.createUserAndSession(t, "owner@example.com")
	_, otherToken := env.createUserAndSession(t, "other@example.com")
	png := append([]byte(nil), []byte("\x89PNG\r\n\x1a\nbootstrap-media")...)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(png); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/media", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	addSessionCookie(uploadReq, ownerToken)
	uploadRec := httptest.NewRecorder()
	env.handler.ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d body=%q", uploadRec.Code, uploadRec.Body.String())
	}
	var uploaded mediaResponse
	if err := json.NewDecoder(uploadRec.Body).Decode(&uploaded); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if uploaded.ID == "" || uploaded.MIME != "image/png" || uploaded.URL != "/uploads/"+uploaded.ID {
		t.Fatalf("unexpected upload response: %+v", uploaded)
	}

	withoutSession := httptest.NewRecorder()
	env.handler.ServeHTTP(withoutSession, httptest.NewRequest(http.MethodGet, uploaded.URL, nil))
	if withoutSession.Code != http.StatusUnauthorized {
		t.Fatalf("without session: expected 401, got %d", withoutSession.Code)
	}

	otherReq := httptest.NewRequest(http.MethodGet, uploaded.URL, nil)
	addSessionCookie(otherReq, otherToken)
	otherRec := httptest.NewRecorder()
	env.handler.ServeHTTP(otherRec, otherReq)
	if otherRec.Code != http.StatusNotFound {
		t.Fatalf("other owner: expected 404, got %d body=%q", otherRec.Code, otherRec.Body.String())
	}

	ownerReq := httptest.NewRequest(http.MethodGet, uploaded.URL, nil)
	addSessionCookie(ownerReq, ownerToken)
	ownerRec := httptest.NewRecorder()
	env.handler.ServeHTTP(ownerRec, ownerReq)
	if ownerRec.Code != http.StatusOK {
		t.Fatalf("owner: expected 200, got %d body=%q", ownerRec.Code, ownerRec.Body.String())
	}
	if ownerRec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("owner: unexpected content type %q", ownerRec.Header().Get("Content-Type"))
	}
	if !bytes.Equal(ownerRec.Body.Bytes(), png) {
		t.Fatalf("owner: media body changed: %q", ownerRec.Body.Bytes())
	}
}

func TestWebSocketRequiresSessionBeforeUpgrade(t *testing.T) {
	env := newTestEnvironment(t)
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestAuthenticatedWebSocketUpgrade(t *testing.T) {
	env := newTestEnvironment(t)
	userID, token := env.createUserAndSession(t, "ws@example.com")
	server := httptest.NewServer(env.handler)
	defer server.Close()

	httpURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	header := http.Header{}
	header.Set("Origin", server.URL)
	header.Set("Cookie", (&http.Cookie{Name: config.SessionCookieName, Value: token, Path: "/"}).String())
	wsURL := "ws://" + httpURL.Host + "/ws"
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("dial websocket (status %d): %v", status, err)
	}
	defer conn.Close()

	var ready struct {
		Type   string `json:"type"`
		UserID int64  `json:"userId"`
	}
	if err := conn.ReadJSON(&ready); err != nil {
		t.Fatalf("read websocket ready message: %v", err)
	}
	if ready.Type != "ready" || ready.UserID != userID {
		t.Fatalf("unexpected ready message: %+v", ready)
	}
	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "done")); err != nil {
		t.Fatalf("close websocket: %v", err)
	}
}

func TestSessionCookieDefaults(t *testing.T) {
	rec := httptest.NewRecorder()
	SetSessionCookie(rec, "token", testNow.Add(time.Hour), false)
	result := rec.Result()
	cookies := result.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != config.SessionCookieName || cookie.Value != "token" || cookie.Path != "/" {
		t.Fatalf("unexpected cookie: %+v", cookie)
	}
	if !cookie.HttpOnly || cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected cookie security attributes: %+v", cookie)
	}
}

func TestExpiredSessionIsDeletedAndCookieCleared(t *testing.T) {
	env := newTestEnvironment(t)
	_, token := env.createUserAndSession(t, "expired@example.com")
	if _, err := env.db.Exec(`UPDATE sessions SET expires_at = ? WHERE token = ?`, testNow.Add(-time.Minute).Unix(), token); err != nil {
		t.Fatalf("expire session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/uploads/1", nil)
	addSessionCookie(req, token)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%q", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != config.SessionCookieName || cookies[0].Value != "" {
		t.Fatalf("expected cleared session cookie, got %+v", cookies)
	}

	var remaining int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token = ?`, token).Scan(&remaining); err != nil {
		t.Fatalf("query expired session: %v", err)
	}
	if remaining != 0 {
		t.Fatal("expired session was not deleted")
	}
}

func TestMalformedAndMissingMediaIDsReturnNotFoundForOwner(t *testing.T) {
	env := newTestEnvironment(t)
	_, token := env.createUserAndSession(t, "missing@example.com")
	for _, path := range []string{"/uploads/not-a-number", "/uploads/" + strconv.FormatInt(999, 10), "/uploads/1/nested"} {
		req := httptest.NewRequest(http.MethodGet, path, strings.NewReader(""))
		addSessionCookie(req, token)
		rec := httptest.NewRecorder()
		env.handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: expected 404, got %d", path, rec.Code)
		}
	}
}
