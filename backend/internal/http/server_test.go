package http

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"social-network/backend/internal/config"
	"social-network/backend/internal/domain"
	realtimews "social-network/backend/internal/realtime/ws"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"

	"github.com/gorilla/websocket"
)

var testNow = time.Unix(1_700_000_000, 0).UTC()

type fixedClock struct{}

func (fixedClock) Now() time.Time { return testNow }

type testPasswordHasher struct{}

func (testPasswordHasher) Hash(password string) (string, error) {
	return "test-hash:" + password, nil
}

func (testPasswordHasher) Compare(hash, password string) error {
	if hash != "test-hash:"+password {
		return fmt.Errorf("password mismatch")
	}
	return nil
}

type failingSessionRepo struct {
	getSession *domain.Session
	getErr     error
	deleteErr  error
	getCalls   int
}

func (r *failingSessionRepo) Create(context.Context, *domain.Session) error {
	return nil
}

func (r *failingSessionRepo) GetByToken(context.Context, string) (*domain.Session, error) {
	r.getCalls++
	return r.getSession, r.getErr
}

func (r *failingSessionRepo) DeleteByToken(context.Context, string) error {
	return r.deleteErr
}

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
	server       *Handler
	handler      http.Handler
	users        *sqlite.UserRepo
	sessions     *service.SessionService
	follows      *service.FollowService
	posts        *service.PostService
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
	avatarStager, err := service.NewMediaStager(ids, uploadDir, service.MaxAvatarBytes)
	if err != nil {
		t.Fatalf("new avatar stager: %v", err)
	}
	users := sqlite.NewUserRepo(db)
	transactions := sqlite.NewTransactionManager(db)
	auth := service.NewAuthService(
		users,
		transactions,
		sessions,
		testPasswordHasher{},
		fixedClock{},
		avatarStager,
	)
	profile := service.NewProfileService(transactions, fixedClock{}, avatarStager, nil)
	follows := service.NewFollowService(users, sqlite.NewFollowRepo(db), transactions, fixedClock{})
	userProfiles := service.NewUserService(transactions)
	avatarDelivery := service.NewAvatarDeliveryService(transactions, uploadDir)
	postStager, err := service.NewMediaStager(ids, uploadDir, service.MaxMediaBytes)
	if err != nil {
		t.Fatalf("new post media stager: %v", err)
	}
	posts := service.NewPostService(transactions, fixedClock{}, postStager)
	postMedia := service.NewPostMediaDeliveryService(transactions, uploadDir)
	comments := service.NewCommentService(transactions, fixedClock{}, postStager)
	commentMedia := service.NewCommentMediaDeliveryService(transactions, uploadDir)
	groups := service.NewGroupService(transactions, fixedClock{})
	groupEvents := service.NewGroupEventService(transactions, fixedClock{})
	notifications := service.NewNotificationService(transactions, fixedClock{})
	chats := service.NewChatService(transactions, fixedClock{})
	hub := realtimews.NewHubWithNow(nil, fixedClock{}.Now)
	go hub.Run()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = hub.BeginDrain(ctx)
		select {
		case <-hub.Done():
		case <-ctx.Done():
		}
	})
	handler := NewHandler(db, sessions, media, auth, profile, follows, userProfiles, avatarDelivery, posts, postMedia, comments, commentMedia, groups, groupEvents, notifications, chats, NewCookieSessionTokenExtractor(config.SessionCookieName), false, "", nil)
	handler.SetRealtimeHub(hub)

	return &testEnvironment{
		db:        db,
		server:    handler,
		handler:   handler.Routes(),
		users:     users,
		sessions:  sessions,
		follows:   follows,
		posts:     posts,
		uploadDir: uploadDir,
	}
}

func TestClosedAdmissionRejectsMutationsAndWebSocketButKeepsReadsAvailable(t *testing.T) {
	env := newTestEnvironment(t)
	_, token := env.createUserAndSession(t, "shutdown-admission@example.com")
	env.server.CloseAdmission()

	for _, testCase := range []struct {
		method string
		path   string
		token  string
		want   int
	}{
		{method: http.MethodPost, path: "/api/auth/login", want: http.StatusServiceUnavailable},
		{method: http.MethodGet, path: "/ws", token: token, want: http.StatusServiceUnavailable},
		{method: http.MethodGet, path: "/api/health", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/auth/me", token: token, want: http.StatusOK},
		{method: http.MethodGet, path: "/static/avatars/neutral.svg", want: http.StatusOK},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(testCase.method, testCase.path, nil)
		if testCase.token != "" {
			addSessionCookie(request, testCase.token)
		}
		env.handler.ServeHTTP(recorder, request)
		if recorder.Code != testCase.want {
			t.Fatalf("%s %s: status=%d want=%d body=%q", testCase.method, testCase.path, recorder.Code, testCase.want, recorder.Body.String())
		}
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

func setProfilePrivacy(t *testing.T, env *testEnvironment, token string, isPrivate bool) authUserResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/api/profile", strings.NewReader(fmt.Sprintf(`{"is_private":%t}`, isPrivate)))
	addSessionCookie(req, token)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("set profile privacy to %t: status=%d body=%q", isPrivate, rec.Code, rec.Body.String())
	}
	var response authUserResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode profile privacy response: %v", err)
	}
	return response
}

func authenticatedRequest(method, path, token string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, path, body)
	addSessionCookie(req, token)
	return req
}

func defaultRegisterFields(email string) map[string]string {
	return map[string]string{
		"email":         email,
		"password":      "correct horse battery staple",
		"first_name":    "Test",
		"last_name":     "User",
		"date_of_birth": "14-03-1992",
	}
}

func newRegisterRequest(t *testing.T, fields map[string]string, avatarName string, avatar []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("write register field %s: %v", name, err)
		}
	}
	if avatarName != "" {
		part, err := writer.CreateFormFile("avatar", avatarName)
		if err != nil {
			t.Fatalf("create avatar form file: %v", err)
		}
		if _, err := part.Write(avatar); err != nil {
			t.Fatalf("write avatar form file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close register multipart: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func newProfileAvatarRequest(t *testing.T, filename string, avatar []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("avatar", filename)
	if err != nil {
		t.Fatalf("create profile avatar form file: %v", err)
	}
	if _, err := part.Write(avatar); err != nil {
		t.Fatalf("write profile avatar: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close profile avatar multipart: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/profile/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func sessionCookieFromResponse(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == config.SessionCookieName && cookie.Value != "" {
			return cookie
		}
	}
	t.Fatalf("response did not set session cookie: %+v", rec.Result().Cookies())
	return nil
}

func assertDBRowCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("expected %d rows in %s, got %d", want, table, got)
	}
}

func newSessionFailureHandler(store *failingSessionRepo) http.Handler {
	return newSessionFailureHandlerWithFrontend(store, "")
}

func newSessionFailureHandlerWithFrontend(store *failingSessionRepo, frontendDir string) http.Handler {
	ids := &sequenceID{}
	sessions := service.NewSessionService(store, fixedClock{}, ids, 24*time.Hour)
	auth := service.NewAuthService(nil, nil, sessions, nil, nil, nil)
	return NewHandler(
		nil,
		sessions,
		nil,
		auth,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		NewCookieSessionTokenExtractor(config.SessionCookieName),
		false,
		frontendDir,
		nil,
	).Routes()
}

func TestFrontendFilesAreServedWithoutShadowingBackendRoutes(t *testing.T) {
	frontendDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(frontendDir, "js"), 0o755); err != nil {
		t.Fatalf("create frontend js directory: %v", err)
	}
	if err := os.Mkdir(filepath.Join(frontendDir, "css"), 0o755); err != nil {
		t.Fatalf("create frontend css directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "index.html"), []byte("<h1>loop frontend</h1>\n"), 0o644); err != nil {
		t.Fatalf("write frontend index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "js", "app.js"), []byte("console.log('loop');\n"), 0o644); err != nil {
		t.Fatalf("write frontend script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "css", "app.css"), []byte("body { color: black; }\n"), 0o644); err != nil {
		t.Fatalf("write frontend stylesheet: %v", err)
	}

	store := &failingSessionRepo{getErr: errors.New("session database read failed")}
	handler := newSessionFailureHandlerWithFrontend(store, frontendDir)

	for _, test := range []struct {
		path        string
		contentType string
		body        string
	}{
		{path: "/", contentType: "text/html", body: "<h1>loop frontend</h1>"},
		{path: "/js/app.js", contentType: "text/javascript", body: "console.log('loop');"},
		{path: "/css/app.css", contentType: "text/css", body: "body { color: black; }"},
		{path: "/static/avatars/neutral.svg", contentType: "image/svg+xml", body: "<svg"},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		addSessionCookie(req, "session-token")
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d body=%q", test.path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), test.contentType) {
			t.Fatalf("%s: unexpected content type %q", test.path, rec.Header().Get("Content-Type"))
		}
		if !strings.Contains(rec.Body.String(), test.body) {
			t.Fatalf("%s: unexpected body %q", test.path, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	unknownAPIReq := httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
	addSessionCookie(unknownAPIReq, "session-token")
	handler.ServeHTTP(rec, unknownAPIReq)
	if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("API fallback was shadowed by frontend: status=%d content-type=%q body=%q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/uploads/1", nil))
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("uploads route was shadowed by frontend: status=%d content-type=%q body=%q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}

	if store.getCalls != 0 {
		t.Fatalf("public routes performed %d session lookups", store.getCalls)
	}
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
	for _, path := range []string{"/api/auth", "/api/posts/42"} {
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

func TestRegisterWithoutAvatarCreatesUserSessionAndNeutralPlaceholder(t *testing.T) {
	env := newTestEnvironment(t)
	req := newRegisterRequest(t, defaultRegisterFields("neutral@example.com"), "", nil)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%q", rec.Code, rec.Body.String())
	}
	responseBody := append([]byte(nil), rec.Body.Bytes()...)
	var response authUserResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if !bytes.Contains(responseBody, []byte(`"gender":null`)) {
		t.Fatalf("missing nullable gender in response: %s", responseBody)
	}
	if !bytes.Contains(responseBody, []byte(`"is_private":false`)) {
		t.Fatalf("missing default public privacy in response: %s", responseBody)
	}
	if response.Email != "neutral@example.com" || response.Gender != nil || response.AvatarURL != domain.NeutralAvatarPlaceholderURL {
		t.Fatalf("unexpected register response: %+v", response)
	}
	if !response.CreatedAt.Equal(testNow) || !response.UpdatedAt.Equal(testNow) {
		t.Fatalf("unexpected user timestamps: %+v", response)
	}
	cookie := sessionCookieFromResponse(t, rec)
	if !cookie.HttpOnly || cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || !cookie.Expires.Equal(testNow.Add(24*time.Hour)) {
		t.Fatalf("unexpected session cookie: %+v", cookie)
	}
	assertDBRowCount(t, env.db, "users", 1)
	assertDBRowCount(t, env.db, "media", 0)
	assertDBRowCount(t, env.db, "sessions", 1)
	assertDBRowCount(t, env.db, "notification_user_states", 1)
	var notificationRevision int64
	if err := env.db.QueryRow(`SELECT revision FROM notification_user_states WHERE user_id = ?`, response.ID).Scan(&notificationRevision); err != nil || notificationRevision != 0 {
		t.Fatalf("registered notification state: revision=%d err=%v", notificationRevision, err)
	}

	stored, err := env.users.GetByEmail(context.Background(), "neutral@example.com")
	if err != nil {
		t.Fatalf("get registered user: %v", err)
	}
	if stored.PasswordHash != "test-hash:correct horse battery staple" {
		t.Fatalf("password was not hashed: %q", stored.PasswordHash)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	addSessionCookie(meReq, cookie.Value)
	meRec := httptest.NewRecorder()
	env.handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me: expected 200, got %d body=%q", meRec.Code, meRec.Body.String())
	}
	if !bytes.Contains(meRec.Body.Bytes(), []byte(`"is_private":false`)) {
		t.Fatalf("me response is missing privacy: %s", meRec.Body.Bytes())
	}

	for _, placeholderURL := range []string{
		domain.MaleAvatarPlaceholderURL,
		domain.FemaleAvatarPlaceholderURL,
		domain.NeutralAvatarPlaceholderURL,
	} {
		placeholderRec := httptest.NewRecorder()
		env.handler.ServeHTTP(placeholderRec, httptest.NewRequest(http.MethodGet, placeholderURL, nil))
		if placeholderRec.Code != http.StatusOK || placeholderRec.Header().Get("Content-Type") != "image/svg+xml" {
			t.Fatalf("placeholder %s: status=%d content-type=%q", placeholderURL, placeholderRec.Code, placeholderRec.Header().Get("Content-Type"))
		}
	}
}

func TestRegisterWithWebPAvatarPersistsMediaInRegistration(t *testing.T) {
	env := newTestEnvironment(t)
	fields := defaultRegisterFields("avatar@example.com")
	fields["gender"] = "female"
	webp := []byte("RIFF\x10\x00\x00\x00WEBPVP8 avatar-data")
	req := newRegisterRequest(t, fields, "avatar.webp", webp)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%q", rec.Code, rec.Body.String())
	}
	var response authUserResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if response.Gender == nil || *response.Gender != domain.GenderFemale {
		t.Fatalf("unexpected avatar response: %+v", response)
	}
	assertDBRowCount(t, env.db, "users", 1)
	assertDBRowCount(t, env.db, "media", 1)
	assertDBRowCount(t, env.db, "sessions", 1)

	var mediaID int64
	var storageKey, mime string
	var size int64
	if err := env.db.QueryRow(`SELECT id, storage_key, mime, size FROM media LIMIT 1`).Scan(&mediaID, &storageKey, &mime, &size); err != nil {
		t.Fatalf("query avatar media: %v", err)
	}
	wantAvatarURL := "/api/users/" + strconv.FormatInt(response.ID, 10) + "/avatar?v=" + strconv.FormatInt(mediaID, 10)
	if response.AvatarURL != wantAvatarURL {
		t.Fatalf("unexpected avatar URL: got %q want %q", response.AvatarURL, wantAvatarURL)
	}
	if mime != "image/webp" || size != int64(len(webp)) || filepath.Ext(storageKey) != ".webp" {
		t.Fatalf("unexpected avatar metadata: key=%q mime=%q size=%d", storageKey, mime, size)
	}
	stored, err := os.ReadFile(filepath.Join(env.uploadDir, storageKey))
	if err != nil {
		t.Fatalf("read stored avatar: %v", err)
	}
	if !bytes.Equal(stored, webp) {
		t.Fatalf("stored avatar changed: %q", stored)
	}

	cookie := sessionCookieFromResponse(t, rec)
	avatarReq := httptest.NewRequest(http.MethodGet, response.AvatarURL, nil)
	addSessionCookie(avatarReq, cookie.Value)
	avatarRec := httptest.NewRecorder()
	env.handler.ServeHTTP(avatarRec, avatarReq)
	if avatarRec.Code != http.StatusOK || avatarRec.Header().Get("Content-Type") != "image/webp" || !bytes.Equal(avatarRec.Body.Bytes(), webp) {
		t.Fatalf("avatar delivery failed: status=%d type=%q body=%q", avatarRec.Code, avatarRec.Header().Get("Content-Type"), avatarRec.Body.Bytes())
	}
	if avatarRec.Header().Get("Content-Length") != strconv.Itoa(len(webp)) ||
		avatarRec.Header().Get("X-Content-Type-Options") != "nosniff" ||
		avatarRec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("unexpected avatar headers: %+v", avatarRec.Header())
	}
}

func TestUserAvatarDeliveryEnforcesCurrentPrivacyAndFollowRelation(t *testing.T) {
	env := newTestEnvironment(t)
	avatarBytes := []byte("RIFF\x10\x00\x00\x00WEBPVP8 controlled-avatar")
	registerRec := httptest.NewRecorder()
	env.handler.ServeHTTP(registerRec, newRegisterRequest(
		t,
		defaultRegisterFields("avatar-owner@example.com"),
		"avatar.webp",
		avatarBytes,
	))
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register avatar owner: status=%d body=%q", registerRec.Code, registerRec.Body.String())
	}
	var owner authUserResponse
	if err := json.NewDecoder(registerRec.Body).Decode(&owner); err != nil {
		t.Fatalf("decode avatar owner: %v", err)
	}
	ownerToken := sessionCookieFromResponse(t, registerRec).Value
	_, acceptedToken := env.createUserAndSession(t, "avatar-accepted@example.com")
	_, pendingToken := env.createUserAndSession(t, "avatar-pending@example.com")
	_, outsiderToken := env.createUserAndSession(t, "avatar-outsider@example.com")

	followPath := "/api/users/" + strconv.FormatInt(owner.ID, 10) + "/follow"
	follow := func(token string, wantStatus service.RelationshipStatus) {
		t.Helper()
		rec := httptest.NewRecorder()
		env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPut, followPath, token, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("follow avatar owner: status=%d body=%q", rec.Code, rec.Body.String())
		}
		var response followStatusResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("decode follow response: %v", err)
		}
		if response.Status != wantStatus {
			t.Fatalf("follow avatar owner: got %q want %q", response.Status, wantStatus)
		}
	}
	requestAvatar := func(label, token string, wantCode int) {
		t.Helper()
		t.Run(label, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, owner.AvatarURL, nil)
			if token != "" {
				addSessionCookie(req, token)
			}
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, req)
			if rec.Code != wantCode {
				t.Fatalf("want status %d, got %d body=%q", wantCode, rec.Code, rec.Body.String())
			}
			switch wantCode {
			case http.StatusOK:
				if !bytes.Equal(rec.Body.Bytes(), avatarBytes) {
					t.Fatalf("unexpected avatar bytes: %q", rec.Body.Bytes())
				}
				if rec.Header().Get("Content-Type") != "image/webp" ||
					rec.Header().Get("Content-Length") != strconv.Itoa(len(avatarBytes)) ||
					rec.Header().Get("X-Content-Type-Options") != "nosniff" ||
					rec.Header().Get("Cache-Control") != "private, no-store" {
					t.Fatalf("unexpected avatar headers: %+v", rec.Header())
				}
			case http.StatusForbidden:
				if rec.Body.String() != "{\"error\":\"forbidden\"}\n" {
					t.Fatalf("unexpected forbidden response: %q", rec.Body.String())
				}
			}
		})
	}

	follow(acceptedToken, service.RelationshipAccepted)
	setProfilePrivacy(t, env, ownerToken, true)
	follow(pendingToken, service.RelationshipPending)

	requestAvatar("private-owner", ownerToken, http.StatusOK)
	requestAvatar("private-accepted", acceptedToken, http.StatusOK)
	requestAvatar("private-pending", pendingToken, http.StatusForbidden)
	requestAvatar("private-outsider", outsiderToken, http.StatusForbidden)
	requestAvatar("unauthenticated", "", http.StatusUnauthorized)

	unfollowRec := httptest.NewRecorder()
	env.handler.ServeHTTP(unfollowRec, authenticatedRequest(http.MethodDelete, followPath, acceptedToken, nil))
	if unfollowRec.Code != http.StatusNoContent {
		t.Fatalf("unfollow avatar owner: status=%d body=%q", unfollowRec.Code, unfollowRec.Body.String())
	}
	requestAvatar("private-after-unfollow", acceptedToken, http.StatusForbidden)

	setProfilePrivacy(t, env, ownerToken, false)
	follow(acceptedToken, service.RelationshipAccepted)
	requestAvatar("public-owner", ownerToken, http.StatusOK)
	requestAvatar("public-accepted", acceptedToken, http.StatusOK)
	requestAvatar("public-pending", pendingToken, http.StatusOK)
	requestAvatar("public-outsider", outsiderToken, http.StatusOK)

	setProfilePrivacy(t, env, ownerToken, true)
	requestAvatar("private-restored-accepted", acceptedToken, http.StatusOK)
	requestAvatar("private-closes-pending", pendingToken, http.StatusForbidden)
	requestAvatar("private-closes-outsider", outsiderToken, http.StatusForbidden)

	var mediaID int64
	if err := env.db.QueryRow(`SELECT avatar_media_id FROM users WHERE id = ?`, owner.ID).Scan(&mediaID); err != nil {
		t.Fatalf("query avatar media ID: %v", err)
	}
	legacyPath := "/uploads/" + strconv.FormatInt(mediaID, 10)
	ownerLegacyRec := httptest.NewRecorder()
	env.handler.ServeHTTP(ownerLegacyRec, authenticatedRequest(http.MethodGet, legacyPath, ownerToken, nil))
	if ownerLegacyRec.Code != http.StatusOK || !bytes.Equal(ownerLegacyRec.Body.Bytes(), avatarBytes) {
		t.Fatalf("owner legacy avatar access failed: status=%d body=%q", ownerLegacyRec.Code, ownerLegacyRec.Body.Bytes())
	}
	outsiderLegacyRec := httptest.NewRecorder()
	env.handler.ServeHTTP(outsiderLegacyRec, authenticatedRequest(http.MethodGet, legacyPath, outsiderToken, nil))
	if outsiderLegacyRec.Code != http.StatusNotFound {
		t.Fatalf("outsider accessed owner-only upload: status=%d body=%q", outsiderLegacyRec.Code, outsiderLegacyRec.Body.String())
	}
}

func TestUserAvatarDeliveryRejectsMissingAndForeignMedia(t *testing.T) {
	env := newTestEnvironment(t)
	targetID, targetToken := env.createUserAndSession(t, "avatar-target@example.com")
	otherID, otherToken := env.createUserAndSession(t, "avatar-other@example.com")
	avatarPath := "/api/users/" + strconv.FormatInt(targetID, 10) + "/avatar"

	noAvatarRec := httptest.NewRecorder()
	env.handler.ServeHTTP(noAvatarRec, authenticatedRequest(http.MethodGet, avatarPath, targetToken, nil))
	if noAvatarRec.Code != http.StatusNotFound {
		t.Fatalf("missing custom avatar: status=%d body=%q", noAvatarRec.Code, noAvatarRec.Body.String())
	}
	setProfilePrivacy(t, env, targetToken, true)
	privateNoAvatarRec := httptest.NewRecorder()
	env.handler.ServeHTTP(privateNoAvatarRec, authenticatedRequest(http.MethodGet, avatarPath, otherToken, nil))
	if privateNoAvatarRec.Code != http.StatusForbidden || privateNoAvatarRec.Body.String() != "{\"error\":\"forbidden\"}\n" {
		t.Fatalf("private avatar existence leaked: status=%d body=%q", privateNoAvatarRec.Code, privateNoAvatarRec.Body.String())
	}
	setProfilePrivacy(t, env, targetToken, false)
	unknownUserRec := httptest.NewRecorder()
	env.handler.ServeHTTP(unknownUserRec, authenticatedRequest(http.MethodGet, "/api/users/999999/avatar", targetToken, nil))
	if unknownUserRec.Code != http.StatusNotFound {
		t.Fatalf("unknown avatar user: status=%d body=%q", unknownUserRec.Code, unknownUserRec.Body.String())
	}

	mediaRepo := sqlite.NewMediaRepo(env.db)
	foreignBytes := []byte("\x89PNG\r\n\x1a\nforeign-avatar")
	foreignKey := "foreign-avatar.png"
	if err := os.WriteFile(filepath.Join(env.uploadDir, foreignKey), foreignBytes, 0o600); err != nil {
		t.Fatalf("write foreign avatar: %v", err)
	}
	foreignMediaID, err := mediaRepo.Create(context.Background(), otherID, "image/png", int64(len(foreignBytes)), foreignKey, "foreign.png", testNow)
	if err != nil {
		t.Fatalf("create foreign avatar media: %v", err)
	}
	if err := env.users.SetAvatarMediaID(context.Background(), targetID, &foreignMediaID, testNow); err != nil {
		t.Fatalf("attach foreign avatar media: %v", err)
	}
	foreignRec := httptest.NewRecorder()
	env.handler.ServeHTTP(foreignRec, authenticatedRequest(http.MethodGet, avatarPath, targetToken, nil))
	if foreignRec.Code != http.StatusNotFound {
		t.Fatalf("foreign-owned avatar was delivered: status=%d body=%q", foreignRec.Code, foreignRec.Body.String())
	}

	missingMediaID, err := mediaRepo.Create(context.Background(), targetID, "image/png", 10, "missing-avatar.png", "missing.png", testNow)
	if err != nil {
		t.Fatalf("create missing avatar media: %v", err)
	}
	if err := env.users.SetAvatarMediaID(context.Background(), targetID, &missingMediaID, testNow); err != nil {
		t.Fatalf("attach missing avatar media: %v", err)
	}
	missingRec := httptest.NewRecorder()
	env.handler.ServeHTTP(missingRec, authenticatedRequest(http.MethodGet, avatarPath, targetToken, nil))
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing physical avatar file: status=%d body=%q", missingRec.Code, missingRec.Body.String())
	}

	if err := env.users.SetAvatarMediaID(context.Background(), otherID, &foreignMediaID, testNow); err != nil {
		t.Fatalf("attach avatar media to owner: %v", err)
	}
	otherAvatarURL := domain.UserAvatarURL(&domain.User{ID: otherID, AvatarMediaID: &foreignMediaID})
	otherRec := httptest.NewRecorder()
	env.handler.ServeHTTP(otherRec, authenticatedRequest(http.MethodGet, otherAvatarURL, otherToken, nil))
	if otherRec.Code != http.StatusOK || !bytes.Equal(otherRec.Body.Bytes(), foreignBytes) {
		t.Fatalf("media owner avatar delivery failed: status=%d body=%q", otherRec.Code, otherRec.Body.Bytes())
	}
}

func TestRegisterRejectsInvalidGenderWithoutWritingRows(t *testing.T) {
	for _, gender := range []string{"", "unknown", " male "} {
		t.Run(fmt.Sprintf("%q", gender), func(t *testing.T) {
			env := newTestEnvironment(t)
			fields := defaultRegisterFields("invalid-gender@example.com")
			fields["gender"] = gender
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, newRegisterRequest(t, fields, "", nil))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%q", rec.Code, rec.Body.String())
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Fatalf("invalid registration set a cookie: %+v", rec.Result().Cookies())
			}
			assertDBRowCount(t, env.db, "users", 0)
			assertDBRowCount(t, env.db, "sessions", 0)
		})
	}
}

func TestRegisterRejectsInvalidDateAndAvatarType(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		mutate     func(map[string]string)
		avatarName string
		avatar     []byte
	}{
		{name: "impossible date", mutate: func(fields map[string]string) { fields["date_of_birth"] = "31-02-1992" }},
		{name: "invalid avatar", mutate: func(map[string]string) {}, avatarName: "avatar.txt", avatar: []byte("not an image")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			env := newTestEnvironment(t)
			fields := defaultRegisterFields("invalid@example.com")
			testCase.mutate(fields)
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, newRegisterRequest(t, fields, testCase.avatarName, testCase.avatar))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%q", rec.Code, rec.Body.String())
			}
			assertDBRowCount(t, env.db, "users", 0)
			assertDBRowCount(t, env.db, "media", 0)
			assertDBRowCount(t, env.db, "sessions", 0)
		})
	}
}

func TestDuplicateRegisterRollsBackAvatarAndDoesNotSetCookie(t *testing.T) {
	env := newTestEnvironment(t)
	firstRec := httptest.NewRecorder()
	env.handler.ServeHTTP(firstRec, newRegisterRequest(t, defaultRegisterFields("duplicate@example.com"), "", nil))
	if firstRec.Code != http.StatusCreated {
		t.Fatalf("first register: status=%d body=%q", firstRec.Code, firstRec.Body.String())
	}

	fields := defaultRegisterFields("DUPLICATE@EXAMPLE.COM")
	webp := []byte("RIFF\x10\x00\x00\x00WEBPVP8 duplicate")
	duplicateRec := httptest.NewRecorder()
	env.handler.ServeHTTP(duplicateRec, newRegisterRequest(t, fields, "duplicate.webp", webp))
	if duplicateRec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%q", duplicateRec.Code, duplicateRec.Body.String())
	}
	if len(duplicateRec.Result().Cookies()) != 0 {
		t.Fatalf("duplicate registration set a cookie: %+v", duplicateRec.Result().Cookies())
	}
	assertDBRowCount(t, env.db, "users", 1)
	assertDBRowCount(t, env.db, "media", 0)
	assertDBRowCount(t, env.db, "sessions", 1)
	files, err := os.ReadDir(env.uploadDir)
	if err != nil {
		t.Fatalf("read upload directory: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("duplicate registration left files behind: %+v", files)
	}
}

func TestProfileUpdatePersistsStrictPartialFields(t *testing.T) {
	env := newTestEnvironment(t)
	fields := defaultRegisterFields("profile@example.com")
	fields["gender"] = "male"
	fields["nickname"] = "old nickname"
	fields["about_me"] = "old bio"
	registerRec := httptest.NewRecorder()
	env.handler.ServeHTTP(registerRec, newRegisterRequest(t, fields, "", nil))
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register: status=%d body=%q", registerRec.Code, registerRec.Body.String())
	}
	cookie := sessionCookieFromResponse(t, registerRec)

	patchBody := `{
		"first_name":"  Updated  ",
		"last_name":"Profile",
		"date_of_birth":"29-02-1992",
		"gender":null,
		"nickname":"  comet  ",
		"about_me":"   ",
		"is_private":true
	}`
	req := httptest.NewRequest(http.MethodPatch, "/api/profile", strings.NewReader(patchBody))
	addSessionCookie(req, cookie.Value)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update profile: expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	var response authUserResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if response.FirstName != "Updated" || response.LastName != "Profile" || response.DateOfBirth != "29-02-1992" {
		t.Fatalf("unexpected required profile fields: %+v", response)
	}
	if response.Gender != nil || response.Nickname == nil || *response.Nickname != "comet" || response.AboutMe != nil {
		t.Fatalf("unexpected optional profile fields: %+v", response)
	}
	if !response.IsPrivate {
		t.Fatal("profile privacy was not returned")
	}
	if response.AvatarURL != domain.NeutralAvatarPlaceholderURL {
		t.Fatalf("expected neutral placeholder after clearing gender, got %q", response.AvatarURL)
	}

	stored, err := env.users.GetByEmail(context.Background(), "profile@example.com")
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}
	if stored.FirstName != response.FirstName || stored.DateOfBirth != response.DateOfBirth || stored.Gender != nil || stored.AboutMe != nil || !stored.IsPrivate {
		t.Fatalf("profile update was not persisted: %+v", stored)
	}
}

func TestProfileUpdateRejectsInvalidAndUnknownFieldsWithoutChanges(t *testing.T) {
	for _, testCase := range []struct {
		name string
		body string
	}{
		{name: "impossible date", body: `{"date_of_birth":"31-02-1992"}`},
		{name: "wrong date format", body: `{"date_of_birth":"1992-02-01"}`},
		{name: "invalid gender", body: `{"gender":"other"}`},
		{name: "empty gender", body: `{"gender":""}`},
		{name: "null required field", body: `{"first_name":null}`},
		{name: "null privacy", body: `{"is_private":null}`},
		{name: "string privacy", body: `{"is_private":"true"}`},
		{name: "number privacy", body: `{"is_private":1}`},
		{name: "unknown field", body: `{"email":"new@example.com"}`},
		{name: "empty object", body: `{}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			env := newTestEnvironment(t)
			_, token := env.createUserAndSession(t, "unchanged@example.com")
			req := httptest.NewRequest(http.MethodPatch, "/api/profile", strings.NewReader(testCase.body))
			addSessionCookie(req, token)
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%q", rec.Code, rec.Body.String())
			}
			stored, err := env.users.GetByEmail(context.Background(), "unchanged@example.com")
			if err != nil {
				t.Fatalf("get unchanged user: %v", err)
			}
			if stored.FirstName != "Test" || stored.DateOfBirth != "02-01-1990" || stored.Gender != nil {
				t.Fatalf("invalid update changed user: %+v", stored)
			}
		})
	}
}

func TestFollowRequestsRespectPersistedPrivacyTransitions(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "owner@example.com")
	acceptedFollowerID, acceptedFollowerToken := env.createUserAndSession(t, "accepted@example.com")
	requesterID, requesterToken := env.createUserAndSession(t, "requester@example.com")
	secondRequesterID, secondRequesterToken := env.createUserAndSession(t, "second-requester@example.com")

	followURL := func(userID int64) string {
		return "/api/users/" + strconv.FormatInt(userID, 10) + "/follow"
	}
	follow := func(token string, targetID int64) followStatusResponse {
		rec := httptest.NewRecorder()
		env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPut, followURL(targetID), token, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("follow user %d: status=%d body=%q", targetID, rec.Code, rec.Body.String())
		}
		var response followStatusResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("decode follow response: %v", err)
		}
		return response
	}

	if response := follow(acceptedFollowerToken, ownerID); response.Status != service.RelationshipAccepted {
		t.Fatalf("public profile follow must be accepted, got %+v", response)
	}
	if response := setProfilePrivacy(t, env, ownerToken, true); !response.IsPrivate {
		t.Fatal("owner did not become private")
	}
	if response := follow(requesterToken, ownerID); response.Status != service.RelationshipPending {
		t.Fatalf("private profile follow must be pending, got %+v", response)
	}
	accepted, err := env.follows.IsFollower(context.Background(), requesterID, ownerID)
	if err != nil {
		t.Fatalf("check pending follower: %v", err)
	}
	if accepted {
		t.Fatal("pending relation counted as follower")
	}

	followersRec := httptest.NewRecorder()
	env.handler.ServeHTTP(followersRec, authenticatedRequest(
		http.MethodGet,
		"/api/users/"+strconv.FormatInt(ownerID, 10)+"/followers",
		ownerToken,
		nil,
	))
	if followersRec.Code != http.StatusOK {
		t.Fatalf("list followers: status=%d body=%q", followersRec.Code, followersRec.Body.String())
	}
	var followers userListResponse
	if err := json.NewDecoder(followersRec.Body).Decode(&followers); err != nil {
		t.Fatalf("decode followers: %v", err)
	}
	if len(followers.Users) != 1 || followers.Users[0].ID != acceptedFollowerID {
		t.Fatalf("pending relation leaked into followers: %+v", followers.Users)
	}

	requestsRec := httptest.NewRecorder()
	env.handler.ServeHTTP(requestsRec, authenticatedRequest(http.MethodGet, "/api/follow-requests", ownerToken, nil))
	if requestsRec.Code != http.StatusOK {
		t.Fatalf("list follow requests: status=%d body=%q", requestsRec.Code, requestsRec.Body.String())
	}
	var requests followRequestListResponse
	if err := json.NewDecoder(requestsRec.Body).Decode(&requests); err != nil {
		t.Fatalf("decode follow requests: %v", err)
	}
	if len(requests.Requests) != 1 || requests.Requests[0].User.ID != requesterID {
		t.Fatalf("unexpected pending requests: %+v", requests.Requests)
	}

	if response := setProfilePrivacy(t, env, ownerToken, false); response.IsPrivate {
		t.Fatal("owner did not become public")
	}
	relationshipRec := httptest.NewRecorder()
	env.handler.ServeHTTP(relationshipRec, authenticatedRequest(http.MethodGet, followURL(ownerID), requesterToken, nil))
	if relationshipRec.Code != http.StatusOK {
		t.Fatalf("get relationship: status=%d body=%q", relationshipRec.Code, relationshipRec.Body.String())
	}
	var relationship relationshipResponse
	if err := json.NewDecoder(relationshipRec.Body).Decode(&relationship); err != nil {
		t.Fatalf("decode relationship: %v", err)
	}
	if relationship.Status != service.RelationshipPending {
		t.Fatalf("privacy change silently accepted pending relation: %+v", relationship)
	}
	if response := follow(requesterToken, ownerID); response.Status != service.RelationshipAccepted {
		t.Fatalf("explicit repeat follow on public profile must accept pending, got %+v", response)
	}

	setProfilePrivacy(t, env, ownerToken, true)
	if response := follow(secondRequesterToken, ownerID); response.Status != service.RelationshipPending {
		t.Fatalf("second private follow must be pending, got %+v", response)
	}
	requestsRec = httptest.NewRecorder()
	env.handler.ServeHTTP(requestsRec, authenticatedRequest(http.MethodGet, "/api/follow-requests", ownerToken, nil))
	if requestsRec.Code != http.StatusOK {
		t.Fatalf("list second follow requests: status=%d body=%q", requestsRec.Code, requestsRec.Body.String())
	}
	if err := json.NewDecoder(requestsRec.Body).Decode(&requests); err != nil {
		t.Fatalf("decode second follow requests: %v", err)
	}
	if len(requests.Requests) != 1 || requests.Requests[0].User.ID != secondRequesterID {
		t.Fatalf("unexpected second pending request: %+v", requests.Requests)
	}
	secondRequestID := requests.Requests[0].ID
	setProfilePrivacy(t, env, ownerToken, false)

	acceptURL := "/api/follow-requests/" + strconv.FormatInt(secondRequestID, 10) + "/accept"
	acceptRec := httptest.NewRecorder()
	env.handler.ServeHTTP(acceptRec, authenticatedRequest(http.MethodPost, acceptURL, ownerToken, nil))
	if acceptRec.Code != http.StatusOK {
		t.Fatalf("accept old pending: status=%d body=%q", acceptRec.Code, acceptRec.Body.String())
	}
	var acceptedResponse followStatusResponse
	if err := json.NewDecoder(acceptRec.Body).Decode(&acceptedResponse); err != nil {
		t.Fatalf("decode accepted response: %v", err)
	}
	if acceptedResponse.Status != service.RelationshipAccepted {
		t.Fatalf("accepted request returned unexpected status: %+v", acceptedResponse)
	}
	repeatAcceptRec := httptest.NewRecorder()
	env.handler.ServeHTTP(repeatAcceptRec, authenticatedRequest(http.MethodPost, acceptURL, ownerToken, nil))
	if repeatAcceptRec.Code != http.StatusOK {
		t.Fatalf("repeat accept: status=%d body=%q", repeatAcceptRec.Code, repeatAcceptRec.Body.String())
	}

	for _, pair := range []struct {
		followerID int64
		label      string
	}{
		{followerID: acceptedFollowerID, label: "existing accepted"},
		{followerID: requesterID, label: "repeated follow"},
		{followerID: secondRequesterID, label: "accepted old pending"},
	} {
		accepted, err := env.follows.IsFollower(context.Background(), pair.followerID, ownerID)
		if err != nil || !accepted {
			t.Fatalf("%s relation is not accepted: accepted=%t err=%v", pair.label, accepted, err)
		}
	}

	followingRec := httptest.NewRecorder()
	env.handler.ServeHTTP(followingRec, authenticatedRequest(
		http.MethodGet,
		"/api/users/"+strconv.FormatInt(requesterID, 10)+"/following",
		requesterToken,
		nil,
	))
	if followingRec.Code != http.StatusOK {
		t.Fatalf("list following: status=%d body=%q", followingRec.Code, followingRec.Body.String())
	}
	var following userListResponse
	if err := json.NewDecoder(followingRec.Body).Decode(&following); err != nil {
		t.Fatalf("decode following: %v", err)
	}
	if len(following.Users) != 1 || following.Users[0].ID != ownerID {
		t.Fatalf("unexpected following list: %+v", following.Users)
	}
}

func TestFollowListsEnforceTargetProfilePrivacy(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "list-owner@example.com")
	acceptedFollowerID, acceptedFollowerToken := env.createUserAndSession(t, "list-accepted@example.com")
	_, pendingToken := env.createUserAndSession(t, "list-pending@example.com")
	_, outsiderToken := env.createUserAndSession(t, "list-outsider@example.com")
	followedUserID, _ := env.createUserAndSession(t, "list-followed@example.com")

	follow := func(token string, targetUserID int64, wantStatus service.RelationshipStatus) {
		t.Helper()
		path := "/api/users/" + strconv.FormatInt(targetUserID, 10) + "/follow"
		rec := httptest.NewRecorder()
		env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodPut, path, token, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("follow user %d: status=%d body=%q", targetUserID, rec.Code, rec.Body.String())
		}
		var response followStatusResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("decode follow response: %v", err)
		}
		if response.Status != wantStatus {
			t.Fatalf("follow user %d: want status %q, got %+v", targetUserID, wantStatus, response)
		}
	}

	follow(acceptedFollowerToken, ownerID, service.RelationshipAccepted)
	follow(ownerToken, followedUserID, service.RelationshipAccepted)
	relationshipPath := "/api/users/" + strconv.FormatInt(ownerID, 10) + "/follow"
	relationshipRec := httptest.NewRecorder()
	env.handler.ServeHTTP(relationshipRec, authenticatedRequest(http.MethodGet, relationshipPath, acceptedFollowerToken, nil))
	if relationshipRec.Code != http.StatusOK {
		t.Fatalf("get accepted relationship: status=%d body=%q", relationshipRec.Code, relationshipRec.Body.String())
	}
	var relationship relationshipResponse
	if err := json.NewDecoder(relationshipRec.Body).Decode(&relationship); err != nil {
		t.Fatalf("decode accepted relationship: %v", err)
	}
	if relationship.Status != service.RelationshipAccepted {
		t.Fatalf("accepted relationship returned unexpected status: %+v", relationship)
	}

	assertLists := func(label, token string, wantCode int) {
		t.Helper()
		for _, endpoint := range []struct {
			suffix     string
			wantUserID int64
		}{
			{suffix: "followers", wantUserID: acceptedFollowerID},
			{suffix: "following", wantUserID: followedUserID},
		} {
			t.Run(label+"/"+endpoint.suffix, func(t *testing.T) {
				path := "/api/users/" + strconv.FormatInt(ownerID, 10) + "/" + endpoint.suffix
				rec := httptest.NewRecorder()
				env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodGet, path, token, nil))
				if rec.Code != wantCode {
					t.Fatalf("want status %d, got %d body=%q", wantCode, rec.Code, rec.Body.String())
				}
				if wantCode == http.StatusForbidden {
					if rec.Body.String() != "{\"error\":\"forbidden\"}\n" {
						t.Fatalf("unexpected forbidden response: %q", rec.Body.String())
					}
					return
				}
				var response userListResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("decode user list: %v", err)
				}
				if len(response.Users) != 1 || response.Users[0].ID != endpoint.wantUserID {
					t.Fatalf("unexpected users: %+v", response.Users)
				}
			})
		}
	}

	assertLists("public-outsider", outsiderToken, http.StatusOK)
	setProfilePrivacy(t, env, ownerToken, true)
	follow(pendingToken, ownerID, service.RelationshipPending)

	assertLists("owner", ownerToken, http.StatusOK)
	assertLists("accepted-follower", acceptedFollowerToken, http.StatusOK)
	assertLists("pending", pendingToken, http.StatusForbidden)
	assertLists("outsider", outsiderToken, http.StatusForbidden)
}

func TestFollowRejectUnfollowAndAuthorization(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "reject-owner@example.com")
	requesterID, requesterToken := env.createUserAndSession(t, "reject-requester@example.com")
	otherID, otherToken := env.createUserAndSession(t, "reject-other@example.com")
	setProfilePrivacy(t, env, ownerToken, true)

	followURL := "/api/users/" + strconv.FormatInt(ownerID, 10) + "/follow"
	followRec := httptest.NewRecorder()
	env.handler.ServeHTTP(followRec, authenticatedRequest(http.MethodPut, followURL, requesterToken, nil))
	if followRec.Code != http.StatusOK {
		t.Fatalf("create pending request: status=%d body=%q", followRec.Code, followRec.Body.String())
	}
	requestsRec := httptest.NewRecorder()
	env.handler.ServeHTTP(requestsRec, authenticatedRequest(http.MethodGet, "/api/follow-requests", ownerToken, nil))
	var requests followRequestListResponse
	if err := json.NewDecoder(requestsRec.Body).Decode(&requests); err != nil || len(requests.Requests) != 1 {
		t.Fatalf("get pending request: requests=%+v err=%v", requests.Requests, err)
	}
	requestID := requests.Requests[0].ID
	rejectURL := "/api/follow-requests/" + strconv.FormatInt(requestID, 10)

	unauthorizedReject := httptest.NewRecorder()
	env.handler.ServeHTTP(unauthorizedReject, authenticatedRequest(http.MethodDelete, rejectURL, otherToken, nil))
	if unauthorizedReject.Code != http.StatusNotFound {
		t.Fatalf("other user rejected request: status=%d", unauthorizedReject.Code)
	}
	rejectRec := httptest.NewRecorder()
	env.handler.ServeHTTP(rejectRec, authenticatedRequest(http.MethodDelete, rejectURL, ownerToken, nil))
	if rejectRec.Code != http.StatusNoContent {
		t.Fatalf("reject request: status=%d body=%q", rejectRec.Code, rejectRec.Body.String())
	}
	accepted, err := env.follows.IsFollower(context.Background(), requesterID, ownerID)
	if err != nil || accepted {
		t.Fatalf("rejected request counted as follower: accepted=%t err=%v", accepted, err)
	}

	setProfilePrivacy(t, env, ownerToken, false)
	followRec = httptest.NewRecorder()
	env.handler.ServeHTTP(followRec, authenticatedRequest(http.MethodPut, followURL, requesterToken, nil))
	if followRec.Code != http.StatusOK {
		t.Fatalf("follow public owner: status=%d body=%q", followRec.Code, followRec.Body.String())
	}
	relationshipURL := "/api/users/" + strconv.FormatInt(requesterID, 10) + "/follow"
	relationshipRec := httptest.NewRecorder()
	env.handler.ServeHTTP(relationshipRec, authenticatedRequest(http.MethodGet, relationshipURL, ownerToken, nil))
	var relationship relationshipResponse
	if err := json.NewDecoder(relationshipRec.Body).Decode(&relationship); err != nil {
		t.Fatalf("decode reverse relationship: %v", err)
	}
	if relationship.Status != service.RelationshipNone || !relationship.FollowsMe {
		t.Fatalf("unexpected reverse relationship: %+v", relationship)
	}

	for attempt := 0; attempt < 2; attempt++ {
		unfollowRec := httptest.NewRecorder()
		env.handler.ServeHTTP(unfollowRec, authenticatedRequest(http.MethodDelete, followURL, requesterToken, nil))
		if unfollowRec.Code != http.StatusNoContent {
			t.Fatalf("unfollow attempt %d: status=%d body=%q", attempt+1, unfollowRec.Code, unfollowRec.Body.String())
		}
	}
	accepted, err = env.follows.IsFollower(context.Background(), requesterID, ownerID)
	if err != nil || accepted {
		t.Fatalf("unfollow did not remove accepted relation: accepted=%t err=%v", accepted, err)
	}

	selfFollowRec := httptest.NewRecorder()
	env.handler.ServeHTTP(selfFollowRec, authenticatedRequest(
		http.MethodPut,
		"/api/users/"+strconv.FormatInt(otherID, 10)+"/follow",
		otherToken,
		nil,
	))
	if selfFollowRec.Code != http.StatusBadRequest {
		t.Fatalf("self-follow: expected 400, got %d", selfFollowRec.Code)
	}
	withoutSessionRec := httptest.NewRecorder()
	env.handler.ServeHTTP(withoutSessionRec, httptest.NewRequest(http.MethodPut, followURL, nil))
	if withoutSessionRec.Code != http.StatusUnauthorized {
		t.Fatalf("follow without session: expected 401, got %d", withoutSessionRec.Code)
	}
}

func TestProfileAvatarReplaceAndIdempotentDelete(t *testing.T) {
	env := newTestEnvironment(t)
	fields := defaultRegisterFields("avatar-edit@example.com")
	fields["gender"] = "female"
	oldAvatar := []byte("RIFF\x10\x00\x00\x00WEBPVP8 old-avatar")
	registerRec := httptest.NewRecorder()
	env.handler.ServeHTTP(registerRec, newRegisterRequest(t, fields, "old.webp", oldAvatar))
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register: status=%d body=%q", registerRec.Code, registerRec.Body.String())
	}
	cookie := sessionCookieFromResponse(t, registerRec)
	var oldMediaID int64
	var oldStorageKey string
	if err := env.db.QueryRow(`SELECT id, storage_key FROM media`).Scan(&oldMediaID, &oldStorageKey); err != nil {
		t.Fatalf("query old avatar: %v", err)
	}

	newAvatar := []byte("\x89PNG\r\n\x1a\nnew-avatar")
	replaceReq := newProfileAvatarRequest(t, "new.png", newAvatar)
	addSessionCookie(replaceReq, cookie.Value)
	replaceRec := httptest.NewRecorder()
	env.handler.ServeHTTP(replaceRec, replaceReq)
	if replaceRec.Code != http.StatusOK {
		t.Fatalf("replace avatar: expected 200, got %d body=%q", replaceRec.Code, replaceRec.Body.String())
	}
	var replaced authUserResponse
	if err := json.NewDecoder(replaceRec.Body).Decode(&replaced); err != nil {
		t.Fatalf("decode replace response: %v", err)
	}
	oldAvatarURL := "/api/users/" + strconv.FormatInt(replaced.ID, 10) + "/avatar?v=" + strconv.FormatInt(oldMediaID, 10)
	if !strings.HasPrefix(replaced.AvatarURL, "/api/users/"+strconv.FormatInt(replaced.ID, 10)+"/avatar?v=") || replaced.AvatarURL == oldAvatarURL {
		t.Fatalf("avatar URL was not replaced: %q", replaced.AvatarURL)
	}
	if _, err := os.Stat(filepath.Join(env.uploadDir, oldStorageKey)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old avatar file still exists: %v", err)
	}
	var oldRows int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM media WHERE id = ?`, oldMediaID).Scan(&oldRows); err != nil {
		t.Fatalf("query old media row: %v", err)
	}
	if oldRows != 0 {
		t.Fatalf("old media row still exists")
	}
	var newMediaID int64
	var newStorageKey string
	if err := env.db.QueryRow(`SELECT id, storage_key FROM media`).Scan(&newMediaID, &newStorageKey); err != nil {
		t.Fatalf("query new avatar: %v", err)
	}
	wantReplacedURL := "/api/users/" + strconv.FormatInt(replaced.ID, 10) + "/avatar?v=" + strconv.FormatInt(newMediaID, 10)
	if replaced.AvatarURL != wantReplacedURL {
		t.Fatalf("unexpected replaced avatar URL: got %q want %q", replaced.AvatarURL, wantReplacedURL)
	}
	storedAvatar, err := os.ReadFile(filepath.Join(env.uploadDir, newStorageKey))
	if err != nil || !bytes.Equal(storedAvatar, newAvatar) {
		t.Fatalf("new avatar was not stored: contents=%q err=%v", storedAvatar, err)
	}
	deliveredReq := httptest.NewRequest(http.MethodGet, replaced.AvatarURL, nil)
	addSessionCookie(deliveredReq, cookie.Value)
	deliveredRec := httptest.NewRecorder()
	env.handler.ServeHTTP(deliveredRec, deliveredReq)
	if deliveredRec.Code != http.StatusOK || deliveredRec.Header().Get("Content-Type") != "image/png" || !bytes.Equal(deliveredRec.Body.Bytes(), newAvatar) {
		t.Fatalf("replaced avatar delivery failed: status=%d type=%q body=%q", deliveredRec.Code, deliveredRec.Header().Get("Content-Type"), deliveredRec.Body.Bytes())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/profile/avatar", nil)
	addSessionCookie(deleteReq, cookie.Value)
	deleteRec := httptest.NewRecorder()
	env.handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete avatar: expected 200, got %d body=%q", deleteRec.Code, deleteRec.Body.String())
	}
	var deleted authUserResponse
	if err := json.NewDecoder(deleteRec.Body).Decode(&deleted); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if deleted.AvatarURL != domain.FemaleAvatarPlaceholderURL {
		t.Fatalf("expected female placeholder, got %q", deleted.AvatarURL)
	}
	deletedRouteReq := httptest.NewRequest(http.MethodGet, replaced.AvatarURL, nil)
	addSessionCookie(deletedRouteReq, cookie.Value)
	deletedRouteRec := httptest.NewRecorder()
	env.handler.ServeHTTP(deletedRouteRec, deletedRouteReq)
	if deletedRouteRec.Code != http.StatusNotFound {
		t.Fatalf("deleted avatar route: expected 404, got %d body=%q", deletedRouteRec.Code, deletedRouteRec.Body.String())
	}
	assertDBRowCount(t, env.db, "media", 0)
	if _, err := os.Stat(filepath.Join(env.uploadDir, newStorageKey)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted avatar file still exists: %v", err)
	}

	repeatReq := httptest.NewRequest(http.MethodDelete, "/api/profile/avatar", nil)
	addSessionCookie(repeatReq, cookie.Value)
	repeatRec := httptest.NewRecorder()
	env.handler.ServeHTTP(repeatRec, repeatReq)
	if repeatRec.Code != http.StatusOK {
		t.Fatalf("repeat delete: expected 200, got %d body=%q", repeatRec.Code, repeatRec.Body.String())
	}
}

func TestLoginMeAndIdempotentCurrentSessionLogout(t *testing.T) {
	env := newTestEnvironment(t)
	registerRec := httptest.NewRecorder()
	env.handler.ServeHTTP(registerRec, newRegisterRequest(t, defaultRegisterFields("auth@example.com"), "", nil))
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register: status=%d body=%q", registerRec.Code, registerRec.Body.String())
	}
	registerCookie := sessionCookieFromResponse(t, registerRec)

	loginBody := strings.NewReader(`{"email":"auth@example.com","password":"correct horse battery staple"}`)
	loginRec := httptest.NewRecorder()
	env.handler.ServeHTTP(loginRec, httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody))
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d body=%q", loginRec.Code, loginRec.Body.String())
	}
	if !bytes.Contains(loginRec.Body.Bytes(), []byte(`"is_private":false`)) {
		t.Fatalf("login response is missing privacy: %s", loginRec.Body.Bytes())
	}
	loginCookie := sessionCookieFromResponse(t, loginRec)
	if loginCookie.Value == registerCookie.Value {
		t.Fatal("login reused the registration session")
	}
	assertDBRowCount(t, env.db, "sessions", 2)

	wrongRec := httptest.NewRecorder()
	env.handler.ServeHTTP(wrongRec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"auth@example.com","password":"wrong"}`)))
	if wrongRec.Code != http.StatusUnauthorized || len(wrongRec.Result().Cookies()) != 0 {
		t.Fatalf("wrong password: status=%d cookies=%+v body=%q", wrongRec.Code, wrongRec.Result().Cookies(), wrongRec.Body.String())
	}
	assertDBRowCount(t, env.db, "sessions", 2)

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	addSessionCookie(meReq, loginCookie.Value)
	meRec := httptest.NewRecorder()
	env.handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me: expected 200, got %d body=%q", meRec.Code, meRec.Body.String())
	}
	if !bytes.Contains(meRec.Body.Bytes(), []byte(`"is_private":false`)) {
		t.Fatalf("me response is missing privacy: %s", meRec.Body.Bytes())
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	addSessionCookie(logoutReq, loginCookie.Value)
	logoutRec := httptest.NewRecorder()
	env.handler.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout: expected 204, got %d body=%q", logoutRec.Code, logoutRec.Body.String())
	}
	assertDBRowCount(t, env.db, "sessions", 1)
	var registrationSession int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token = ?`, registerCookie.Value).Scan(&registrationSession); err != nil {
		t.Fatalf("query registration session: %v", err)
	}
	if registrationSession != 1 {
		t.Fatal("logout deleted a different device session")
	}

	repeatReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	addSessionCookie(repeatReq, loginCookie.Value)
	repeatRec := httptest.NewRecorder()
	env.handler.ServeHTTP(repeatRec, repeatReq)
	if repeatRec.Code != http.StatusNoContent {
		t.Fatalf("repeated logout: expected 204, got %d", repeatRec.Code)
	}

	withoutCookieRec := httptest.NewRecorder()
	env.handler.ServeHTTP(withoutCookieRec, httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil))
	if withoutCookieRec.Code != http.StatusNoContent {
		t.Fatalf("logout without cookie: expected 204, got %d", withoutCookieRec.Code)
	}
}

func TestSessionReadFailureReturnsInternalServerError(t *testing.T) {
	handler := newSessionFailureHandler(&failingSessionRepo{
		getErr: errors.New("session database read failed"),
	})
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	addSessionCookie(req, "session-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError || rec.Body.String() != "{\"error\":\"internal server error\"}\n" {
		t.Fatalf("expected session read failure to return 500, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestSessionDeleteFailureReturnsInternalServerErrorWithoutClearingCookie(t *testing.T) {
	handler := newSessionFailureHandler(&failingSessionRepo{
		getSession: &domain.Session{
			Token:     "session-token",
			UserID:    42,
			ExpiresAt: testNow.Add(time.Hour),
			CreatedAt: testNow,
		},
		deleteErr: errors.New("session database delete failed"),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	addSessionCookie(req, "session-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError || rec.Body.String() != "{\"error\":\"internal server error\"}\n" {
		t.Fatalf("expected session delete failure to return 500, got %d body=%q", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("failed logout cleared the cookie: %+v", rec.Result().Cookies())
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
		Type          string  `json:"type"`
		OnlineUserIDs []int64 `json:"online_user_ids"`
	}
	if err := conn.ReadJSON(&ready); err != nil {
		t.Fatalf("read websocket ready message: %v", err)
	}
	if ready.Type != "presence:init" || len(ready.OnlineUserIDs) != 0 || userID <= 0 {
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

func TestUserProfileReadUsesCurrentPrivacyAndAccessiblePostCounts(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, ownerToken := env.createUserAndSession(t, "profile-read-owner@example.com")
	acceptedID, acceptedToken := env.createUserAndSession(t, "profile-read-accepted@example.com")
	_, pendingToken := env.createUserAndSession(t, "profile-read-pending@example.com")
	_, outsiderToken := env.createUserAndSession(t, "profile-read-outsider@example.com")

	if _, err := env.follows.Follow(context.Background(), acceptedID, ownerID); err != nil {
		t.Fatalf("create accepted follow: %v", err)
	}
	setProfilePrivacy(t, env, ownerToken, true)
	pendingID, err := env.users.GetByEmail(context.Background(), "profile-read-pending@example.com")
	if err != nil {
		t.Fatalf("get pending user: %v", err)
	}
	if _, err := env.follows.Follow(context.Background(), pendingID.ID, ownerID); err != nil {
		t.Fatalf("create pending follow: %v", err)
	}

	for _, input := range []service.CreatePostInput{
		{Text: "public profile post", Privacy: domain.PostPublic},
		{Text: "followers profile post", Privacy: domain.PostFollowers},
		{Text: "selected profile post", Privacy: domain.PostSelected, SelectedUserIDs: []int64{acceptedID}},
	} {
		if _, err := env.posts.Create(context.Background(), ownerID, input); err != nil {
			t.Fatalf("create profile post: %v", err)
		}
	}

	path := "/api/users/" + strconv.FormatInt(ownerID, 10)
	readProfile := func(label, token string, wantCode int) (*userProfileResponse, map[string]json.RawMessage) {
		t.Helper()
		rec := httptest.NewRecorder()
		env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodGet, path, token, nil))
		if rec.Code != wantCode {
			t.Fatalf("%s: want status %d, got %d body=%q", label, wantCode, rec.Code, rec.Body.String())
		}
		if wantCode != http.StatusOK {
			return nil, nil
		}
		var response userProfileResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("%s: decode profile: %v", label, err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &fields); err != nil {
			t.Fatalf("%s: decode profile fields: %v", label, err)
		}
		if _, leaked := fields["password_hash"]; leaked {
			t.Fatalf("%s: profile leaked password hash: %s", label, rec.Body.Bytes())
		}
		return &response, fields
	}

	for _, viewer := range []struct {
		label string
		token string
	}{
		{label: "owner", token: ownerToken},
		{label: "accepted", token: acceptedToken},
	} {
		response, fields := readProfile(viewer.label, viewer.token, http.StatusOK)
		if !response.CanViewProfile || response.Email == nil || *response.Email != "profile-read-owner@example.com" ||
			response.DateOfBirth == nil || response.PostsCount == nil || *response.PostsCount != 3 {
			t.Fatalf("%s: unexpected full profile: %+v", viewer.label, response)
		}
		for _, field := range []string{"email", "gender", "about_me", "followers_count", "following_count"} {
			if _, exists := fields[field]; !exists {
				t.Fatalf("%s: visible profile omitted %q: %s", viewer.label, field, fields)
			}
		}
	}

	for _, viewer := range []struct {
		label string
		token string
	}{
		{label: "pending", token: pendingToken},
		{label: "outsider", token: outsiderToken},
	} {
		response, fields := readProfile(viewer.label, viewer.token, http.StatusOK)
		if response.CanViewProfile {
			t.Fatalf("%s unexpectedly received full private profile", viewer.label)
		}
		for _, field := range []string{"email", "date_of_birth", "gender", "about_me", "posts_count", "followers_count", "following_count"} {
			if _, exists := fields[field]; exists {
				t.Fatalf("%s: locked profile leaked %q: %s", viewer.label, field, fields)
			}
		}
	}

	setProfilePrivacy(t, env, ownerToken, false)
	publicOutsider, _ := readProfile("public outsider", outsiderToken, http.StatusOK)
	if !publicOutsider.CanViewProfile || publicOutsider.Email == nil ||
		*publicOutsider.Email != "profile-read-owner@example.com" ||
		publicOutsider.PostsCount == nil || *publicOutsider.PostsCount != 1 {
		t.Fatalf("public outsider count must use post access policy: %+v", publicOutsider)
	}
	publicAccepted, _ := readProfile("public accepted", acceptedToken, http.StatusOK)
	if publicAccepted.PostsCount == nil || *publicAccepted.PostsCount != 3 {
		t.Fatalf("accepted follower lost accessible posts: %+v", publicAccepted)
	}

	missing := httptest.NewRecorder()
	env.handler.ServeHTTP(missing, authenticatedRequest(http.MethodGet, "/api/users/999999", outsiderToken, nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("unknown profile: want 404, got %d body=%q", missing.Code, missing.Body.String())
	}
}

func TestUserDirectoryPaginatesAndReturnsViewerRelationships(t *testing.T) {
	env := newTestEnvironment(t)
	viewerID, viewerToken := env.createUserAndSession(t, "directory-viewer@example.com")
	acceptedID, _ := env.createUserAndSession(t, "directory-accepted@example.com")
	pendingID, pendingToken := env.createUserAndSession(t, "directory-pending@example.com")
	followsMeID, _ := env.createUserAndSession(t, "directory-follows-me@example.com")
	noneID, _ := env.createUserAndSession(t, "directory-none@example.com")

	if _, err := env.follows.Follow(context.Background(), viewerID, acceptedID); err != nil {
		t.Fatalf("create directory accepted relation: %v", err)
	}
	setProfilePrivacy(t, env, pendingToken, true)
	if _, err := env.follows.Follow(context.Background(), viewerID, pendingID); err != nil {
		t.Fatalf("create directory pending relation: %v", err)
	}
	if _, err := env.follows.Follow(context.Background(), followsMeID, viewerID); err != nil {
		t.Fatalf("create reverse directory relation: %v", err)
	}

	requestPage := func(path string, wantCode int) userDirectoryResponse {
		t.Helper()
		rec := httptest.NewRecorder()
		env.handler.ServeHTTP(rec, authenticatedRequest(http.MethodGet, path, viewerToken, nil))
		if rec.Code != wantCode {
			t.Fatalf("%s: want %d, got %d body=%q", path, wantCode, rec.Code, rec.Body.String())
		}
		var response userDirectoryResponse
		if wantCode == http.StatusOK {
			if bytes.Contains(rec.Body.Bytes(), []byte(`"email"`)) ||
				bytes.Contains(rec.Body.Bytes(), []byte(`"password_hash"`)) {
				t.Fatalf("directory leaked private credentials: %s", rec.Body.Bytes())
			}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("decode directory page: %v", err)
			}
		}
		return response
	}

	first := requestPage("/api/users?limit=2", http.StatusOK)
	if len(first.Users) != 2 || first.Users[0].ID != noneID || first.Users[1].ID != followsMeID || first.NextCursor == nil {
		t.Fatalf("unexpected first directory page: %+v", first)
	}
	if first.Users[0].Relationship.Status != service.RelationshipNone || first.Users[0].Relationship.FollowsMe {
		t.Fatalf("unexpected none relationship: %+v", first.Users[0].Relationship)
	}
	if !first.Users[1].Relationship.FollowsMe {
		t.Fatalf("reverse accepted relation missing: %+v", first.Users[1].Relationship)
	}

	second := requestPage("/api/users?limit=2&cursor="+url.QueryEscape(*first.NextCursor), http.StatusOK)
	if len(second.Users) != 2 || second.Users[0].ID != pendingID || second.Users[1].ID != acceptedID || second.NextCursor != nil {
		t.Fatalf("unexpected second directory page: %+v", second)
	}
	if second.Users[0].Relationship.Status != service.RelationshipPending || second.Users[1].Relationship.Status != service.RelationshipAccepted {
		t.Fatalf("directory returned wrong relationship vocabulary: %+v", second.Users)
	}
	for _, page := range []userDirectoryResponse{first, second} {
		for _, item := range page.Users {
			if item.ID == viewerID {
				t.Fatal("directory included current user")
			}
		}
	}

	for _, path := range []string{
		"/api/users?limit=0",
		"/api/users?limit=51",
		"/api/users?limit=1&limit=2",
		"/api/users?cursor=broken",
		"/api/users?unknown=1",
	} {
		requestPage(path, http.StatusBadRequest)
	}
}

func TestFollowListsEmbedViewerRelationshipWithoutExtraRequests(t *testing.T) {
	env := newTestEnvironment(t)
	ownerID, _ := env.createUserAndSession(t, "related-list-owner@example.com")
	listedID, _ := env.createUserAndSession(t, "related-list-user@example.com")
	viewerID, viewerToken := env.createUserAndSession(t, "related-list-viewer@example.com")

	for _, relation := range [][2]int64{
		{listedID, ownerID},
		{viewerID, listedID},
		{listedID, viewerID},
	} {
		if _, err := env.follows.Follow(context.Background(), relation[0], relation[1]); err != nil {
			t.Fatalf("create list relation %v: %v", relation, err)
		}
	}

	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, authenticatedRequest(
		http.MethodGet,
		"/api/users/"+strconv.FormatInt(ownerID, 10)+"/followers",
		viewerToken,
		nil,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("list related followers: status=%d body=%q", rec.Code, rec.Body.String())
	}
	var response userListResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode related followers: %v", err)
	}
	if len(response.Users) != 1 || response.Users[0].ID != listedID {
		t.Fatalf("unexpected related follower list: %+v", response.Users)
	}
	if response.Users[0].Relationship.Status != service.RelationshipAccepted || !response.Users[0].Relationship.FollowsMe {
		t.Fatalf("viewer relationship missing from list row: %+v", response.Users[0].Relationship)
	}
}
