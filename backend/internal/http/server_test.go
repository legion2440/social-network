package http

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

	return &testEnvironment{
		db:        db,
		handler:   NewHandler(db, sessions, media, auth, profile, NewCookieSessionTokenExtractor(config.SessionCookieName), false, "", nil).Routes(),
		users:     users,
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
	if response.Gender == nil || *response.Gender != domain.GenderFemale || !strings.HasPrefix(response.AvatarURL, "/uploads/") {
		t.Fatalf("unexpected avatar response: %+v", response)
	}
	assertDBRowCount(t, env.db, "users", 1)
	assertDBRowCount(t, env.db, "media", 1)
	assertDBRowCount(t, env.db, "sessions", 1)

	var storageKey, mime string
	var size int64
	if err := env.db.QueryRow(`SELECT storage_key, mime, size FROM media LIMIT 1`).Scan(&storageKey, &mime, &size); err != nil {
		t.Fatalf("query avatar media: %v", err)
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
		"about_me":"   "
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
	if response.AvatarURL != domain.NeutralAvatarPlaceholderURL {
		t.Fatalf("expected neutral placeholder after clearing gender, got %q", response.AvatarURL)
	}

	stored, err := env.users.GetByEmail(context.Background(), "profile@example.com")
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}
	if stored.FirstName != response.FirstName || stored.DateOfBirth != response.DateOfBirth || stored.Gender != nil || stored.AboutMe != nil {
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
	if !strings.HasPrefix(replaced.AvatarURL, "/uploads/") || replaced.AvatarURL == domain.MediaURL(oldMediaID) {
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
	var newStorageKey string
	if err := env.db.QueryRow(`SELECT storage_key FROM media`).Scan(&newStorageKey); err != nil {
		t.Fatalf("query new avatar: %v", err)
	}
	storedAvatar, err := os.ReadFile(filepath.Join(env.uploadDir, newStorageKey))
	if err != nil || !bytes.Equal(storedAvatar, newAvatar) {
		t.Fatalf("new avatar was not stored: contents=%q err=%v", storedAvatar, err)
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
