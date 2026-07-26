package service_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"social-network/backend/internal/oauth"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/repo"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

type oauthStubProvider struct {
	mu       sync.Mutex
	identity *oauth.Identity
	state    string
}

func (p *oauthStubProvider) Name() string  { return oauth.ProviderGitHub }
func (p *oauthStubProvider) Label() string { return "GitHub" }
func (p *oauthStubProvider) AuthorizationURL(state string) string {
	p.mu.Lock()
	p.state = state
	p.mu.Unlock()
	return "https://github.test/authorize?state=" + url.QueryEscape(state)
}
func (p *oauthStubProvider) Exchange(_ context.Context, code string) (string, error) {
	if code == "exchange-failure" {
		return "", errors.New("exchange failed")
	}
	return "access-token", nil
}
func (p *oauthStubProvider) Identity(_ context.Context, _ string) (*oauth.Identity, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	copy := *p.identity
	return &copy, nil
}

type oauthFixture struct {
	root     string
	db       *sql.DB
	auth     *service.AuthService
	provider *oauthStubProvider
	ids      *authTestIDGenerator
	clock    clock.Clock
	stager   *service.MediaStager
}

func newOAuthFixture(t *testing.T, transactions repo.TransactionManager) *oauthFixture {
	t.Helper()
	root := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(root, "oauth.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ids := &authTestIDGenerator{}
	appClock := clock.RealClock{}
	sessions := service.NewSessionService(sqlite.NewSessionRepo(db), appClock, ids, 24*time.Hour)
	stager, err := service.NewMediaStager(ids, filepath.Join(root, "uploads"), service.MaxAvatarBytes)
	if err != nil {
		t.Fatalf("new media stager: %v", err)
	}
	provider := &oauthStubProvider{identity: &oauth.Identity{
		ProviderUserID: "github-123",
		Email:          "github@example.com",
		EmailVerified:  true,
		Username:       "octocat",
		DisplayName:    "The Octocat",
	}}
	registry, err := oauth.NewRegistry(provider)
	if err != nil {
		t.Fatalf("new OAuth registry: %v", err)
	}
	if transactions == nil {
		transactions = sqlite.NewTransactionManager(db)
	}
	auth := service.NewAuthService(
		sqlite.NewUserRepo(db),
		transactions,
		sessions,
		authTestHasher{},
		appClock,
		stager,
		service.WithOAuth(
			registry,
			sqlite.NewAuthIdentityRepo(db),
			sqlite.NewAuthFlowRepo(db),
			ids,
		),
	)
	return &oauthFixture{
		root: root, db: db, auth: auth, provider: provider, ids: ids, clock: appClock, stager: stager,
	}
}

func TestOAuthStateIsConsumedBeforeProviderErrorOrMissingCode(t *testing.T) {
	fixture := newOAuthFixture(t, nil)
	ctx := context.Background()
	for _, testCase := range []struct {
		name          string
		providerError string
		code          string
		want          error
	}{
		{name: "provider denial", providerError: "access_denied", want: service.ErrOAuthProviderError},
		{name: "missing code", want: service.ErrOAuthCodeMissing},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			browserNonce := testOAuthBrowserNonce(testCase.name)
			if _, err := fixture.auth.StartOAuth(
				ctx,
				"github",
				"/groups",
				oauth.HashBrowserNonce(browserNonce),
			); err != nil {
				t.Fatalf("start OAuth: %v", err)
			}
			state := fixture.provider.state
			result, err := fixture.auth.HandleOAuthCallback(ctx, service.OAuthCallbackInput{
				Provider:                     "github",
				State:                        state,
				Code:                         testCase.code,
				ProviderError:                testCase.providerError,
				BrowserNonce:                 browserNonce,
				RegistrationBrowserNonceHash: oauth.HashBrowserNonce(testOAuthBrowserNonce("registration")),
			})
			if !errors.Is(err, testCase.want) {
				t.Fatalf("expected %v, got %v", testCase.want, err)
			}
			if result == nil || !result.BrowserBindingVerified {
				t.Fatalf("verified callback error must report browser binding: %+v", result)
			}
			_, err = fixture.auth.HandleOAuthCallback(ctx, service.OAuthCallbackInput{
				Provider:     "github",
				State:        state,
				Code:         "valid-code",
				BrowserNonce: browserNonce,
			})
			if !errors.Is(err, service.ErrOAuthStateInvalid) {
				t.Fatalf("expected consumed state error, got %v", err)
			}
		})
	}
}

func TestOAuthBrowserBindingConsumesMismatchedStateAndProtectsRegistration(t *testing.T) {
	fixture := newOAuthFixture(t, nil)
	ctx := context.Background()
	stateNonce := testOAuthBrowserNonce("bound-state")
	stateHash := oauth.HashBrowserNonce(stateNonce)
	if _, err := fixture.auth.StartOAuth(ctx, "github", "/groups", stateHash); err != nil {
		t.Fatalf("start OAuth: %v", err)
	}
	state := fixture.provider.state
	var payload string
	if err := fixture.db.QueryRow(`SELECT payload FROM auth_flows WHERE token = ?`, state).Scan(&payload); err != nil {
		t.Fatalf("read state payload: %v", err)
	}
	if strings.Contains(payload, stateNonce) || !strings.Contains(payload, stateHash) {
		t.Fatalf("state payload must contain only nonce hash: %q", payload)
	}

	result, err := fixture.auth.HandleOAuthCallback(ctx, service.OAuthCallbackInput{
		Provider:                     "github",
		State:                        state,
		Code:                         "valid-code",
		BrowserNonce:                 testOAuthBrowserNonce("wrong-browser"),
		RegistrationBrowserNonceHash: oauth.HashBrowserNonce(testOAuthBrowserNonce("registration")),
	})
	if result != nil || !errors.Is(err, service.ErrOAuthStateInvalid) {
		t.Fatalf("mismatched browser callback: result=%+v err=%v", result, err)
	}
	if _, err := sqlite.NewAuthFlowRepo(fixture.db).GetByToken(ctx, state); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("mismatched callback did not consume state: %v", err)
	}

	stateNonce = testOAuthBrowserNonce("second-state")
	registrationNonce := testOAuthBrowserNonce("bound-registration")
	if _, err := fixture.auth.StartOAuth(
		ctx,
		"github",
		"/groups",
		oauth.HashBrowserNonce(stateNonce),
	); err != nil {
		t.Fatalf("restart OAuth: %v", err)
	}
	callback, err := fixture.auth.HandleOAuthCallback(ctx, service.OAuthCallbackInput{
		Provider:                     "github",
		State:                        fixture.provider.state,
		Code:                         "valid-code",
		BrowserNonce:                 stateNonce,
		RegistrationBrowserNonceHash: oauth.HashBrowserNonce(registrationNonce),
	})
	if err != nil {
		t.Fatalf("bound callback: %v", err)
	}
	var registrationPayload string
	if err := fixture.db.QueryRow(
		`SELECT payload FROM auth_flows WHERE token = ?`,
		callback.RegistrationToken,
	).Scan(&registrationPayload); err != nil {
		t.Fatalf("read registration payload: %v", err)
	}
	if strings.Contains(registrationPayload, registrationNonce) ||
		!strings.Contains(registrationPayload, oauth.HashBrowserNonce(registrationNonce)) {
		t.Fatalf("registration payload must contain only nonce hash: %q", registrationPayload)
	}
	if _, err := fixture.auth.OAuthRegistrationPreview(
		ctx,
		callback.RegistrationToken,
		testOAuthBrowserNonce("foreign-registration"),
	); !errors.Is(err, service.ErrOAuthFlowExpired) {
		t.Fatalf("foreign preview error=%v", err)
	}
	if _, err := fixture.auth.OAuthRegistrationPreview(
		ctx,
		callback.RegistrationToken,
		registrationNonce,
	); err != nil {
		t.Fatalf("foreign preview consumed flow: %v", err)
	}
	if _, err := fixture.auth.CompleteOAuthRegistration(
		ctx,
		callback.RegistrationToken,
		service.CompleteOAuthRegistrationInput{
			FirstName: "OAuth", LastName: "User", DateOfBirth: "01-01-1990",
		},
		testOAuthBrowserNonce("foreign-registration"),
	); !errors.Is(err, service.ErrOAuthFlowExpired) {
		t.Fatalf("foreign completion error=%v", err)
	}
	if _, err := fixture.auth.OAuthRegistrationPreview(
		ctx,
		callback.RegistrationToken,
		registrationNonce,
	); err != nil {
		t.Fatalf("foreign completion consumed flow: %v", err)
	}
}

func TestOAuthRegistrationCompletionAndExistingIdentityLogin(t *testing.T) {
	fixture := newOAuthFixture(t, nil)
	ctx := context.Background()
	stateNonce := testOAuthBrowserNonce("state")
	registrationNonce := testOAuthBrowserNonce("registration")
	if _, err := fixture.auth.StartOAuth(ctx, "github", "/groups", oauth.HashBrowserNonce(stateNonce)); err != nil {
		t.Fatalf("start OAuth: %v", err)
	}
	callback, err := fixture.auth.HandleOAuthCallback(ctx, service.OAuthCallbackInput{
		Provider:                     "github",
		State:                        fixture.provider.state,
		Code:                         "valid-code",
		BrowserNonce:                 stateNonce,
		RegistrationBrowserNonceHash: oauth.HashBrowserNonce(registrationNonce),
	})
	if err != nil {
		t.Fatalf("OAuth callback: %v", err)
	}
	if callback.RegistrationToken == "" || callback.Auth != nil || callback.Next != "/groups" {
		t.Fatalf("unexpected callback result: %+v", callback)
	}

	preview, err := fixture.auth.OAuthRegistrationPreview(ctx, callback.RegistrationToken, registrationNonce)
	if err != nil {
		t.Fatalf("preview flow: %v", err)
	}
	if preview.Email != "github@example.com" ||
		preview.GitHubUsername != "octocat" ||
		preview.SuggestedFirstName != "The" ||
		preview.SuggestedLastName != "Octocat" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if _, err := fixture.auth.CompleteOAuthRegistration(
		ctx,
		callback.RegistrationToken,
		service.CompleteOAuthRegistrationInput{FirstName: "Missing", LastName: "Date"},
		registrationNonce,
	); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if _, err := fixture.auth.OAuthRegistrationPreview(ctx, callback.RegistrationToken, registrationNonce); err != nil {
		t.Fatalf("validation error consumed flow: %v", err)
	}

	result, err := fixture.auth.CompleteOAuthRegistration(
		ctx,
		callback.RegistrationToken,
		service.CompleteOAuthRegistrationInput{
			FirstName:   "OAuth",
			LastName:    "User",
			DateOfBirth: "01-02-1990",
		},
		registrationNonce,
	)
	if err != nil {
		t.Fatalf("complete OAuth registration: %v", err)
	}
	if result.Auth.User.Email != "github@example.com" || result.Auth.Session == nil || result.Next != "/groups" {
		t.Fatalf("unexpected auth result: %+v", result)
	}
	if _, err := fixture.auth.OAuthRegistrationPreview(ctx, callback.RegistrationToken, registrationNonce); !errors.Is(err, service.ErrOAuthFlowExpired) {
		t.Fatalf("expected consumed registration flow, got %v", err)
	}

	fixture.provider.mu.Lock()
	fixture.provider.identity.Email = "new-verified@example.com"
	fixture.provider.identity.Username = "renamed"
	fixture.provider.mu.Unlock()
	loginNonce := testOAuthBrowserNonce("login")
	if _, err := fixture.auth.StartOAuth(ctx, "github", "/notifications", oauth.HashBrowserNonce(loginNonce)); err != nil {
		t.Fatalf("restart OAuth: %v", err)
	}
	login, err := fixture.auth.HandleOAuthCallback(ctx, service.OAuthCallbackInput{
		Provider:     "github",
		State:        fixture.provider.state,
		Code:         "valid-code",
		BrowserNonce: loginNonce,
	})
	if err != nil {
		t.Fatalf("existing identity callback: %v", err)
	}
	if login.Auth == nil || login.Auth.User.ID != result.Auth.User.ID || login.RegistrationToken != "" {
		t.Fatalf("existing identity did not reuse user: %+v", login)
	}
	var providerEmail, username string
	if err := fixture.db.QueryRow(`
		SELECT provider_email, provider_username
		FROM auth_identities
		WHERE provider = 'github' AND provider_user_id = 'github-123'
	`).Scan(&providerEmail, &username); err != nil {
		t.Fatalf("read updated identity: %v", err)
	}
	if providerEmail != "new-verified@example.com" || username != "renamed" {
		t.Fatalf("identity metadata not updated: email=%q username=%q", providerEmail, username)
	}
}

func TestOAuthEmailCollisionDoesNotLinkOrCreateUser(t *testing.T) {
	fixture := newOAuthFixture(t, nil)
	ctx := context.Background()
	if _, err := fixture.auth.Register(ctx, service.RegisterInput{
		Email:       "github@example.com",
		Password:    "password",
		FirstName:   "Local",
		LastName:    "User",
		DateOfBirth: "02-02-1990",
	}); err != nil {
		t.Fatalf("register local account: %v", err)
	}
	browserNonce := testOAuthBrowserNonce("collision")
	if _, err := fixture.auth.StartOAuth(ctx, "github", "/", oauth.HashBrowserNonce(browserNonce)); err != nil {
		t.Fatalf("start OAuth: %v", err)
	}
	_, err := fixture.auth.HandleOAuthCallback(ctx, service.OAuthCallbackInput{
		Provider:     "github",
		State:        fixture.provider.state,
		Code:         "valid-code",
		BrowserNonce: browserNonce,
	})
	if !errors.Is(err, service.ErrOAuthEmailAlreadyRegistered) {
		t.Fatalf("expected email collision, got %v", err)
	}
	for _, table := range []string{"auth_identities", "auth_flows"} {
		var count int
		if err := fixture.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("email collision left %d rows in %s", count, table)
		}
	}
}

func TestOAuthCompletionRollbackRestoresFlowAndRemovesAvatar(t *testing.T) {
	fixture := newOAuthFixture(t, nil)
	ctx := context.Background()
	stateNonce := testOAuthBrowserNonce("rollback-state")
	registrationNonce := testOAuthBrowserNonce("rollback-registration")
	if _, err := fixture.auth.StartOAuth(ctx, "github", "/", oauth.HashBrowserNonce(stateNonce)); err != nil {
		t.Fatalf("start OAuth: %v", err)
	}
	callback, err := fixture.auth.HandleOAuthCallback(ctx, service.OAuthCallbackInput{
		Provider:                     "github",
		State:                        fixture.provider.state,
		Code:                         "valid-code",
		BrowserNonce:                 stateNonce,
		RegistrationBrowserNonceHash: oauth.HashBrowserNonce(registrationNonce),
	})
	if err != nil {
		t.Fatalf("OAuth callback: %v", err)
	}

	failingAuth := service.NewAuthService(
		sqlite.NewUserRepo(fixture.db),
		failAfterCallbackTransactions{delegate: sqlite.NewTransactionManager(fixture.db)},
		service.NewSessionService(sqlite.NewSessionRepo(fixture.db), fixture.clock, fixture.ids, 24*time.Hour),
		authTestHasher{},
		fixture.clock,
		fixture.stager,
		service.WithOAuth(
			mustOAuthRegistry(t, fixture.provider),
			sqlite.NewAuthIdentityRepo(fixture.db),
			sqlite.NewAuthFlowRepo(fixture.db),
			fixture.ids,
		),
	)
	png := []byte("\x89PNG\r\n\x1a\nrollback-oauth-avatar")
	result, err := failingAuth.CompleteOAuthRegistration(
		ctx,
		callback.RegistrationToken,
		service.CompleteOAuthRegistrationInput{
			FirstName:   "Rollback",
			LastName:    "OAuth",
			DateOfBirth: "03-03-1990",
			Avatar: &service.MediaUpload{
				OriginalName: "avatar.png",
				Reader:       bytes.NewReader(png),
			},
		},
		registrationNonce,
	)
	if result != nil || !errors.Is(err, errForcedCommitFailure) {
		t.Fatalf("expected forced rollback, result=%+v err=%v", result, err)
	}
	if _, err := fixture.auth.OAuthRegistrationPreview(ctx, callback.RegistrationToken, registrationNonce); err != nil {
		t.Fatalf("rollback consumed registration flow: %v", err)
	}
	files, err := os.ReadDir(filepath.Join(fixture.root, "uploads"))
	if err != nil {
		t.Fatalf("read uploads: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("rollback left avatar files: %+v", files)
	}
}

func mustOAuthRegistry(t *testing.T, provider oauth.Provider) *oauth.Registry {
	t.Helper()
	registry, err := oauth.NewRegistry(provider)
	if err != nil {
		t.Fatalf("new OAuth registry: %v", err)
	}
	return registry
}

func testOAuthBrowserNonce(seed string) string {
	value := make([]byte, oauth.BrowserNonceBytes)
	copy(value, []byte(seed))
	return base64.RawURLEncoding.EncodeToString(value)
}
