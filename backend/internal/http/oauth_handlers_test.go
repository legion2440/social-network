package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"social-network/backend/internal/oauth"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

type httpOAuthProvider struct {
	mu       sync.Mutex
	state    string
	identity oauth.Identity
}

func (p *httpOAuthProvider) Name() string  { return oauth.ProviderGitHub }
func (p *httpOAuthProvider) Label() string { return "GitHub" }
func (p *httpOAuthProvider) AuthorizationURL(state string) string {
	p.mu.Lock()
	p.state = state
	p.mu.Unlock()
	return "https://github.test/authorize?state=" + url.QueryEscape(state)
}
func (p *httpOAuthProvider) Exchange(_ context.Context, code string) (string, error) {
	if code == "exchange-failure" {
		return "", errors.New("exchange failed")
	}
	return "access-token", nil
}
func (p *httpOAuthProvider) Identity(_ context.Context, _ string) (*oauth.Identity, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	copy := p.identity
	return &copy, nil
}

func newOAuthHTTPEnvironment(t *testing.T) (*testEnvironment, *httpOAuthProvider) {
	t.Helper()
	env := newTestEnvironment(t)
	ids := &sequenceID{}
	provider := &httpOAuthProvider{identity: oauth.Identity{
		ProviderUserID: "github-http-123",
		Email:          "github-http@example.com",
		EmailVerified:  true,
		Username:       "http-octocat",
		DisplayName:    "HTTP Octocat",
	}}
	registry, err := oauth.NewRegistry(provider)
	if err != nil {
		t.Fatalf("new OAuth registry: %v", err)
	}
	stager, err := service.NewMediaStager(ids, env.uploadDir, service.MaxAvatarBytes)
	if err != nil {
		t.Fatalf("new avatar stager: %v", err)
	}
	env.server.auth = service.NewAuthService(
		env.users,
		sqlite.NewTransactionManager(env.db),
		env.sessions,
		testPasswordHasher{},
		fixedClock{},
		stager,
		service.WithOAuth(
			registry,
			sqlite.NewAuthIdentityRepo(env.db),
			sqlite.NewAuthFlowRepo(env.db),
			ids,
		),
	)
	env.server.oauthExpectedOrigin = "http://example.com:80"
	env.handler = env.server.Routes()
	return env, provider
}

func TestOAuthHTTPRegistrationAndExistingIdentityLogin(t *testing.T) {
	env, provider := newOAuthHTTPEnvironment(t)

	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/auth/oauth/providers", nil))
	if recorder.Code != http.StatusOK ||
		!strings.Contains(recorder.Body.String(), `"name":"github"`) {
		t.Fatalf("providers: status=%d body=%q", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	start := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start?next=%2Fgroups", nil)
	start.RemoteAddr = "192.0.2.10:1234"
	env.handler.ServeHTTP(recorder, start)
	if recorder.Code != http.StatusFound ||
		!strings.HasPrefix(recorder.Header().Get("Location"), "https://github.test/authorize") {
		t.Fatalf("start: status=%d location=%q body=%q", recorder.Code, recorder.Header().Get("Location"), recorder.Body.String())
	}
	stateCookie := oauthNonceCookieFromResponse(t, recorder, false)

	provider.mu.Lock()
	state := provider.state
	provider.mu.Unlock()
	recorder = httptest.NewRecorder()
	callbackPath := "/api/auth/oauth/github/callback?state=" + url.QueryEscape(state) + "&code=valid"
	callbackRequest := httptest.NewRequest(http.MethodGet, callbackPath, nil)
	callbackRequest.AddCookie(stateCookie)
	env.handler.ServeHTTP(recorder, callbackRequest)
	if recorder.Code != http.StatusFound {
		t.Fatalf("callback: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	registrationCookie := oauthNonceCookieFromResponse(t, recorder, false)
	if registrationCookie.Value == stateCookie.Value {
		t.Fatal("callback must rotate the browser nonce for registration")
	}
	completionURL := recorder.Header().Get("Location")
	parsedCompletion, err := url.Parse(completionURL)
	if err != nil {
		t.Fatalf("parse completion URL: %v", err)
	}
	flowToken := parsedCompletion.Query().Get("flow")
	if parsedCompletion.Path != "/oauth/complete" || flowToken == "" {
		t.Fatalf("unexpected completion URL %q", completionURL)
	}

	for attempt := 0; attempt < 2; attempt++ {
		recorder = httptest.NewRecorder()
		previewRequest := httptest.NewRequest(
			http.MethodGet,
			"/api/auth/oauth/flows/"+url.PathEscape(flowToken),
			nil,
		)
		previewRequest.AddCookie(registrationCookie)
		env.handler.ServeHTTP(recorder, previewRequest)
		if recorder.Code != http.StatusOK {
			t.Fatalf("preview %d: status=%d body=%q", attempt, recorder.Code, recorder.Body.String())
		}
		var preview service.OAuthRegistrationPreview
		if err := json.NewDecoder(recorder.Body).Decode(&preview); err != nil {
			t.Fatalf("decode preview: %v", err)
		}
		if preview.Email != "github-http@example.com" || preview.GitHubUsername != "http-octocat" {
			t.Fatalf("unsafe or incomplete preview: %+v", preview)
		}
	}

	invalid := newOAuthCompleteRequest(t, flowToken, map[string]string{
		"first_name": "OAuth", "last_name": "HTTP", "date_of_birth": "01-01-1990",
		"email": "attacker@example.com",
	}, "", nil, registrationCookie)
	invalid.RemoteAddr = "192.0.2.11:1234"
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, invalid)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("email field must be rejected: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	assertNoOAuthNonceCookie(t, recorder)

	valid := newOAuthCompleteRequest(t, flowToken, map[string]string{
		"first_name": "OAuth", "last_name": "HTTP", "date_of_birth": "01-01-1990",
		"nickname": "oauth-http",
	}, "avatar.png", []byte("\x89PNG\r\n\x1a\nhttp-avatar"), registrationCookie)
	valid.RemoteAddr = "192.0.2.11:1234"
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, valid)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("complete: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	var completion struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
		Next string `json:"next"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&completion); err != nil ||
		completion.User.ID == 0 ||
		completion.Next != "/groups" {
		t.Fatalf("completion response=%+v err=%v", completion, err)
	}
	sessionCookieFromResponse(t, recorder)
	oauthNonceCookieFromResponse(t, recorder, true)
	assertDBRowCount(t, env.db, "users", 1)
	assertDBRowCount(t, env.db, "auth_identities", 1)
	assertDBRowCount(t, env.db, "auth_flows", 0)

	recorder = httptest.NewRecorder()
	restart := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start?next=%2Fnotifications", nil)
	restart.RemoteAddr = "192.0.2.12:1234"
	env.handler.ServeHTTP(recorder, restart)
	loginStateCookie := oauthNonceCookieFromResponse(t, recorder, false)
	provider.mu.Lock()
	state = provider.state
	provider.mu.Unlock()
	recorder = httptest.NewRecorder()
	loginCallback := httptest.NewRequest(
		http.MethodGet,
		"/api/auth/oauth/github/callback?state="+url.QueryEscape(state)+"&code=valid",
		nil,
	)
	loginCallback.AddCookie(loginStateCookie)
	env.handler.ServeHTTP(recorder, loginCallback)
	if recorder.Code != http.StatusFound || recorder.Header().Get("Location") != "/notifications" {
		t.Fatalf("existing login: status=%d location=%q body=%q", recorder.Code, recorder.Header().Get("Location"), recorder.Body.String())
	}
	sessionCookieFromResponse(t, recorder)
	oauthNonceCookieFromResponse(t, recorder, true)
	assertDBRowCount(t, env.db, "users", 1)
	assertDBRowCount(t, env.db, "auth_identities", 1)
}

func TestOAuthHTTPCallbackConsumesStateAndUsesStableErrors(t *testing.T) {
	env, provider := newOAuthHTTPEnvironment(t)
	start := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start", nil)
	start.RemoteAddr = "192.0.2.20:1234"
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, start)
	stateCookie := oauthNonceCookieFromResponse(t, recorder, false)
	provider.mu.Lock()
	state := provider.state
	provider.mu.Unlock()

	recorder = httptest.NewRecorder()
	denial := httptest.NewRequest(
		http.MethodGet,
		"/api/auth/oauth/github/callback?state="+url.QueryEscape(state)+"&error=access_denied",
		nil,
	)
	denial.AddCookie(stateCookie)
	env.handler.ServeHTTP(recorder, denial)
	if recorder.Code != http.StatusFound ||
		recorder.Header().Get("Location") != "/login?oauth_error=oauth_provider_error" {
		t.Fatalf("provider error redirect: status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
	oauthNonceCookieFromResponse(t, recorder, true)

	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet,
		"/api/auth/oauth/github/callback?state="+url.QueryEscape(state)+"&code=valid",
		nil,
	))
	if recorder.Code != http.StatusFound ||
		recorder.Header().Get("Location") != "/login?oauth_error=oauth_state_invalid" {
		t.Fatalf("consumed state redirect: status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
}

func TestOAuthHTTPMissingCodeAfterBrowserBindingClearsCookie(t *testing.T) {
	env, provider := newOAuthHTTPEnvironment(t)
	start := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start", nil)
	start.RemoteAddr = "192.0.2.21:1234"
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, start)
	stateCookie := oauthNonceCookieFromResponse(t, recorder, false)
	provider.mu.Lock()
	state := provider.state
	provider.mu.Unlock()

	callback := httptest.NewRequest(
		http.MethodGet,
		"/api/auth/oauth/github/callback?state="+url.QueryEscape(state),
		nil,
	)
	callback.AddCookie(stateCookie)
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, callback)
	if recorder.Code != http.StatusFound ||
		recorder.Header().Get("Location") != "/login?oauth_error=oauth_code_missing" {
		t.Fatalf("missing code callback: status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
	oauthNonceCookieFromResponse(t, recorder, true)
}

func TestOAuthHTTPBrowserBindingAllowsOnlyOneActiveBrowserFlow(t *testing.T) {
	env, provider := newOAuthHTTPEnvironment(t)

	firstStart := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start", nil)
	firstStart.RemoteAddr = "192.0.2.50:1234"
	firstRecorder := httptest.NewRecorder()
	env.handler.ServeHTTP(firstRecorder, firstStart)
	firstCookie := oauthNonceCookieFromResponse(t, firstRecorder, false)
	provider.mu.Lock()
	firstState := provider.state
	provider.mu.Unlock()
	var firstPayload string
	if err := env.db.QueryRow(`SELECT payload FROM auth_flows WHERE token = ?`, firstState).Scan(&firstPayload); err != nil {
		t.Fatalf("read first state payload: %v", err)
	}
	if strings.Contains(firstPayload, firstCookie.Value) {
		t.Fatal("raw browser nonce was persisted in OAuth state")
	}

	secondStart := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start", nil)
	secondStart.RemoteAddr = "192.0.2.50:1234"
	secondRecorder := httptest.NewRecorder()
	env.handler.ServeHTTP(secondRecorder, secondStart)
	secondCookie := oauthNonceCookieFromResponse(t, secondRecorder, false)
	provider.mu.Lock()
	secondState := provider.state
	provider.mu.Unlock()
	if firstCookie.Value == secondCookie.Value {
		t.Fatal("each OAuth start must rotate the browser nonce")
	}

	mismatchedCallback := httptest.NewRequest(
		http.MethodGet,
		"/api/auth/oauth/github/callback?state="+url.QueryEscape(firstState)+"&code=valid",
		nil,
	)
	mismatchedCallback.AddCookie(secondCookie)
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, mismatchedCallback)
	if recorder.Code != http.StatusFound ||
		recorder.Header().Get("Location") != "/login?oauth_error=oauth_state_invalid" {
		t.Fatalf("mismatched callback: status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
	assertNoOAuthNonceCookie(t, recorder)

	retry := httptest.NewRequest(
		http.MethodGet,
		"/api/auth/oauth/github/callback?state="+url.QueryEscape(firstState)+"&code=valid",
		nil,
	)
	retry.AddCookie(firstCookie)
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, retry)
	if recorder.Header().Get("Location") != "/login?oauth_error=oauth_state_invalid" {
		t.Fatalf("mismatched state was not consumed: %q", recorder.Header().Get("Location"))
	}

	validCallback := httptest.NewRequest(
		http.MethodGet,
		"/api/auth/oauth/github/callback?state="+url.QueryEscape(secondState)+"&code=valid",
		nil,
	)
	validCallback.AddCookie(secondCookie)
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, validCallback)
	if recorder.Code != http.StatusFound {
		t.Fatalf("second callback: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	registrationCookie := oauthNonceCookieFromResponse(t, recorder, false)
	completionURL, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse completion URL: %v", err)
	}
	flowToken := completionURL.Query().Get("flow")
	if flowToken == "" {
		t.Fatalf("missing registration flow in %q", recorder.Header().Get("Location"))
	}

	for _, cookie := range []*http.Cookie{nil, firstCookie} {
		preview := httptest.NewRequest(
			http.MethodGet,
			"/api/auth/oauth/flows/"+url.PathEscape(flowToken),
			nil,
		)
		if cookie != nil {
			preview.AddCookie(cookie)
		}
		recorder = httptest.NewRecorder()
		env.handler.ServeHTTP(recorder, preview)
		if recorder.Code != http.StatusGone {
			t.Fatalf("foreign preview: status=%d body=%q", recorder.Code, recorder.Body.String())
		}
	}

	preview := httptest.NewRequest(
		http.MethodGet,
		"/api/auth/oauth/flows/"+url.PathEscape(flowToken),
		nil,
	)
	preview.AddCookie(registrationCookie)
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, preview)
	if recorder.Code != http.StatusOK {
		t.Fatalf("valid preview after foreign attempts: status=%d body=%q", recorder.Code, recorder.Body.String())
	}

	crossSite := newOAuthCompleteRequest(t, flowToken, map[string]string{
		"first_name": "OAuth", "last_name": "Browser", "date_of_birth": "01-01-1990",
	}, "", nil, registrationCookie)
	crossSite.Header.Set("Origin", "https://attacker.example")
	crossSite.Header.Set("Sec-Fetch-Site", "cross-site")
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, crossSite)
	if recorder.Code != http.StatusForbidden || !strings.Contains(recorder.Body.String(), `"forbidden"`) {
		t.Fatalf("cross-site completion: status=%d body=%q", recorder.Code, recorder.Body.String())
	}

	preview = httptest.NewRequest(
		http.MethodGet,
		"/api/auth/oauth/flows/"+url.PathEscape(flowToken),
		nil,
	)
	preview.AddCookie(registrationCookie)
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, preview)
	if recorder.Code != http.StatusOK {
		t.Fatalf("cross-site request consumed flow: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestOAuthHTTPCallbackWithoutCookieConsumesState(t *testing.T) {
	env, provider := newOAuthHTTPEnvironment(t)
	start := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start", nil)
	start.RemoteAddr = "192.0.2.51:1234"
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, start)
	stateCookie := oauthNonceCookieFromResponse(t, recorder, false)
	provider.mu.Lock()
	state := provider.state
	provider.mu.Unlock()

	callbackPath := "/api/auth/oauth/github/callback?state=" + url.QueryEscape(state) + "&code=valid"
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, callbackPath, nil))
	if recorder.Code != http.StatusFound ||
		recorder.Header().Get("Location") != "/login?oauth_error=oauth_state_invalid" {
		t.Fatalf("cookieless callback: status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
	assertNoOAuthNonceCookie(t, recorder)

	retry := httptest.NewRequest(http.MethodGet, callbackPath, nil)
	retry.AddCookie(stateCookie)
	recorder = httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, retry)
	if recorder.Header().Get("Location") != "/login?oauth_error=oauth_state_invalid" {
		t.Fatalf("cookieless callback did not consume state: %q", recorder.Header().Get("Location"))
	}
}

func TestOAuthHTTPEmailCollisionReturnsStableErrorWithoutLinking(t *testing.T) {
	env, provider := newOAuthHTTPEnvironment(t)
	if _, err := env.server.auth.Register(context.Background(), service.RegisterInput{
		Email:       "github-http@example.com",
		Password:    "password",
		FirstName:   "Local",
		LastName:    "Account",
		DateOfBirth: "02-02-1990",
	}); err != nil {
		t.Fatalf("create local account: %v", err)
	}
	start := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start", nil)
	start.RemoteAddr = "192.0.2.30:1234"
	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, start)
	stateCookie := oauthNonceCookieFromResponse(t, recorder, false)
	provider.mu.Lock()
	state := provider.state
	provider.mu.Unlock()

	recorder = httptest.NewRecorder()
	callback := httptest.NewRequest(
		http.MethodGet,
		"/api/auth/oauth/github/callback?state="+url.QueryEscape(state)+"&code=valid",
		nil,
	)
	callback.AddCookie(stateCookie)
	env.handler.ServeHTTP(recorder, callback)
	if recorder.Code != http.StatusFound ||
		recorder.Header().Get("Location") != "/login?oauth_error=oauth_email_already_registered" {
		t.Fatalf("collision redirect: status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
	oauthNonceCookieFromResponse(t, recorder, true)
	assertDBRowCount(t, env.db, "users", 1)
	assertDBRowCount(t, env.db, "auth_identities", 0)
}

func TestOAuthStartRateLimit(t *testing.T) {
	env, _ := newOAuthHTTPEnvironment(t)
	for attempt := 0; attempt < 21; attempt++ {
		request := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start", nil)
		request.RemoteAddr = "192.0.2.40:1234"
		recorder := httptest.NewRecorder()
		env.handler.ServeHTTP(recorder, request)
		want := http.StatusFound
		if attempt == 20 {
			want = http.StatusTooManyRequests
		}
		if recorder.Code != want {
			t.Fatalf("attempt %d: status=%d want=%d body=%q", attempt, recorder.Code, want, recorder.Body.String())
		}
	}
}

func newOAuthCompleteRequest(
	t *testing.T,
	token string,
	fields map[string]string,
	avatarName string,
	avatar []byte,
	nonceCookie *http.Cookie,
) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("write OAuth field %s: %v", name, err)
		}
	}
	if avatarName != "" {
		part, err := writer.CreateFormFile("avatar", filepath.Base(avatarName))
		if err != nil {
			t.Fatalf("create OAuth avatar: %v", err)
		}
		if _, err := part.Write(avatar); err != nil {
			t.Fatalf("write OAuth avatar: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close OAuth multipart: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/oauth/flows/"+url.PathEscape(token)+"/complete",
		body,
	)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Origin", "http://example.com")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	if nonceCookie != nil {
		request.AddCookie(nonceCookie)
	}
	return request
}

func oauthNonceCookieFromResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	cleared bool,
) *http.Cookie {
	t.Helper()
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name != oauthNonceCookieName {
			continue
		}
		if cookie.Path != oauthNonceCookiePath ||
			!cookie.HttpOnly ||
			cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("unexpected OAuth nonce cookie attributes: %+v", cookie)
		}
		if cleared {
			if cookie.Value != "" || cookie.MaxAge != -1 || !cookie.Expires.Before(time.Now()) {
				t.Fatalf("OAuth nonce cookie was not cleared: %+v", cookie)
			}
		} else {
			if cookie.Value == "" || cookie.MaxAge != oauthNonceMaxAge {
				t.Fatalf("OAuth nonce cookie was not set: %+v", cookie)
			}
		}
		return cookie
	}
	t.Fatalf("response did not set OAuth nonce cookie: %+v", recorder.Result().Cookies())
	return nil
}

func assertNoOAuthNonceCookie(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == oauthNonceCookieName {
			t.Fatalf("response unexpectedly changed OAuth nonce cookie: %+v", cookie)
		}
	}
}
