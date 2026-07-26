package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidOAuthCompletionRequestMatrix(t *testing.T) {
	expectedOrigin := "https://example.com:443"
	for _, testCase := range []struct {
		name        string
		origins     []string
		fetchSites  []string
		wantAllowed bool
	}{
		{name: "matching origin without fetch metadata", origins: []string{"https://example.com"}, wantAllowed: true},
		{name: "normalized matching origin", origins: []string{"HTTPS://Example.COM."}, wantAllowed: true},
		{name: "matching origin and same origin metadata", origins: []string{"https://example.com:443"}, fetchSites: []string{"same-origin"}, wantAllowed: true},
		{name: "same origin metadata without origin", fetchSites: []string{"same-origin"}, wantAllowed: true},
		{name: "missing both headers"},
		{name: "different origin", origins: []string{"https://other.example"}},
		{name: "malformed origin", origins: []string{"://example.com"}},
		{name: "origin null", origins: []string{"null"}},
		{name: "origin with slash", origins: []string{"https://example.com/"}},
		{name: "origin list", origins: []string{"https://example.com, https://other.example"}},
		{name: "multiple origin headers", origins: []string{"https://example.com", "https://example.com"}},
		{name: "empty origin", origins: []string{""}},
		{name: "same site metadata", origins: []string{"https://example.com"}, fetchSites: []string{"same-site"}},
		{name: "cross site metadata", origins: []string{"https://example.com"}, fetchSites: []string{"cross-site"}},
		{name: "none metadata", origins: []string{"https://example.com"}, fetchSites: []string{"none"}},
		{name: "unknown metadata", origins: []string{"https://example.com"}, fetchSites: []string{"future-value"}},
		{name: "multiple fetch metadata headers", origins: []string{"https://example.com"}, fetchSites: []string{"same-origin", "same-origin"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/auth/oauth/flows/token/complete", nil)
			for _, origin := range testCase.origins {
				request.Header.Add("Origin", origin)
			}
			for _, fetchSite := range testCase.fetchSites {
				request.Header.Add("Sec-Fetch-Site", fetchSite)
			}
			if got := validOAuthCompletionRequest(request, expectedOrigin); got != testCase.wantAllowed {
				t.Fatalf("validOAuthCompletionRequest()=%v want=%v", got, testCase.wantAllowed)
			}
		})
	}
}

func TestOAuthNonceCookieSetAndClearUseMatchingSecurityAttributes(t *testing.T) {
	setRecorder := httptest.NewRecorder()
	setOAuthNonceCookie(setRecorder, "nonce-value", true)
	setCookie := oauthNonceCookieFromResponse(t, setRecorder, false)
	if !setCookie.Secure {
		t.Fatal("secure OAuth nonce cookie must have Secure")
	}

	clearRecorder := httptest.NewRecorder()
	clearOAuthNonceCookie(clearRecorder, true)
	clearCookie := oauthNonceCookieFromResponse(t, clearRecorder, true)
	if !clearCookie.Secure ||
		clearCookie.Name != setCookie.Name ||
		clearCookie.Path != setCookie.Path ||
		clearCookie.HttpOnly != setCookie.HttpOnly ||
		clearCookie.SameSite != setCookie.SameSite {
		t.Fatalf("clear cookie attributes differ: set=%+v clear=%+v", setCookie, clearCookie)
	}
}
