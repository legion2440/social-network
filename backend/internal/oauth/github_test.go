package oauth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestGitHubProvider(t *testing.T, handler http.Handler) (*GitHubProvider, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	provider, err := NewGitHubProvider(GitHubConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://social.test/api/auth/oauth/github/callback",
	}, server.Client())
	if err != nil {
		t.Fatalf("new GitHub provider: %v", err)
	}
	provider.authorizeURL = server.URL + "/authorize"
	provider.tokenURL = server.URL + "/token"
	provider.userURL = server.URL + "/user"
	provider.emailsURL = server.URL + "/user/emails"
	return provider, server
}

func TestGitHubAuthorizationURLContainsStateAndNoSecret(t *testing.T) {
	provider, _ := newTestGitHubProvider(t, http.NotFoundHandler())
	target, err := url.Parse(provider.AuthorizationURL("state-value"))
	if err != nil {
		t.Fatalf("parse authorize URL: %v", err)
	}
	if target.Query().Get("state") != "state-value" {
		t.Fatalf("unexpected state %q", target.Query().Get("state"))
	}
	if target.Query().Get("client_id") != "client-id" {
		t.Fatalf("unexpected client id %q", target.Query().Get("client_id"))
	}
	if strings.Contains(target.RawQuery, "client-secret") {
		t.Fatal("authorization URL exposed the client secret")
	}
}

func TestGitHubProviderRejectsNonHTTPRedirect(t *testing.T) {
	if _, err := NewGitHubProvider(GitHubConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "javascript://callback",
	}, nil); err == nil {
		t.Fatal("expected non-HTTP redirect URL error")
	}
}

func TestGitHubExchangeAndIdentityAlwaysUseVerifiedEmailsEndpoint(t *testing.T) {
	emailsCalled := false
	provider, _ := newTestGitHubProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if r.Method != http.MethodPost {
				t.Errorf("token method = %s", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse token form: %v", err)
			}
			if r.Form.Get("client_secret") != "client-secret" || r.Form.Get("code") != "code-value" {
				t.Errorf("unexpected token form: %v", r.Form)
			}
			_, _ = io.WriteString(w, `{"access_token":"access-token"}`)
		case "/user":
			if r.Header.Get("Authorization") != "Bearer access-token" {
				t.Errorf("unexpected authorization header %q", r.Header.Get("Authorization"))
			}
			_, _ = io.WriteString(w, `{
				"id":123,
				"login":"octocat",
				"name":"The Octocat",
				"email":"unverified-from-user@example.com"
			}`)
		case "/user/emails":
			emailsCalled = true
			_, _ = io.WriteString(w, `[
				{"email":"secondary@example.com","primary":false,"verified":true},
				{"email":"primary@example.com","primary":true,"verified":true}
			]`)
		default:
			http.NotFound(w, r)
		}
	}))

	token, err := provider.Exchange(context.Background(), "code-value")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	identity, err := provider.Identity(context.Background(), token)
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	if !emailsCalled {
		t.Fatal("expected /user/emails to be called")
	}
	if identity.ProviderUserID != "123" ||
		identity.Email != "primary@example.com" ||
		identity.Username != "octocat" ||
		identity.DisplayName != "The Octocat" ||
		!identity.EmailVerified {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestGitHubIdentityFallsBackToFirstVerifiedEmail(t *testing.T) {
	provider, _ := newTestGitHubProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = io.WriteString(w, `{"id":456,"login":"verified-user","name":""}`)
		case "/user/emails":
			_, _ = io.WriteString(w, `[
				{"email":"primary-unverified@example.com","primary":true,"verified":false},
				{"email":"verified@example.com","primary":false,"verified":true}
			]`)
		default:
			http.NotFound(w, r)
		}
	}))

	identity, err := provider.Identity(context.Background(), "access-token")
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	if identity.Email != "verified@example.com" {
		t.Fatalf("unexpected email %q", identity.Email)
	}
}

func TestGitHubIdentityRejectsMissingVerifiedEmail(t *testing.T) {
	provider, _ := newTestGitHubProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = io.WriteString(w, `{"id":789,"login":"no-email","name":""}`)
		case "/user/emails":
			_, _ = io.WriteString(w, `[
				{"email":"unverified@example.com","primary":true,"verified":false}
			]`)
		default:
			http.NotFound(w, r)
		}
	}))

	if _, err := provider.Identity(context.Background(), "access-token"); err != ErrVerifiedEmailUnavailable {
		t.Fatalf("expected verified email error, got %v", err)
	}
}

func TestGitHubProviderRejectsOversizedResponse(t *testing.T) {
	provider, _ := newTestGitHubProvider(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", int(maxProviderResponseBytes)+1))
	}))
	if _, err := provider.Exchange(context.Background(), "code"); err == nil {
		t.Fatal("expected oversized response error")
	}
}
