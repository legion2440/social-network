package http

import (
	"net/http"
	"strings"
	"time"

	"social-network/backend/internal/oauth"
)

const (
	oauthNonceCookieName = "social_network_oauth_nonce"
	oauthNonceCookiePath = "/api/auth/oauth/"
	oauthNonceMaxAge     = int((30 * time.Minute) / time.Second)
)

func setOAuthNonceCookie(w http.ResponseWriter, value string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthNonceCookieName,
		Value:    value,
		Path:     oauthNonceCookiePath,
		MaxAge:   oauthNonceMaxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func readOAuthNonceCookie(r *http.Request) string {
	if r == nil {
		return ""
	}
	cookie, err := r.Cookie(oauthNonceCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func clearOAuthNonceCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthNonceCookieName,
		Value:    "",
		Path:     oauthNonceCookiePath,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0).UTC(),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func validOAuthCompletionRequest(r *http.Request, expectedOrigin string) bool {
	if r == nil || expectedOrigin == "" {
		return false
	}

	fetchSiteValues := r.Header.Values("Sec-Fetch-Site")
	fetchSite := ""
	switch len(fetchSiteValues) {
	case 0:
	case 1:
		fetchSite = fetchSiteValues[0]
		if fetchSite != "same-origin" {
			return false
		}
	default:
		return false
	}

	originValues := r.Header.Values("Origin")
	switch len(originValues) {
	case 0:
		return fetchSite == "same-origin"
	case 1:
		value := originValues[0]
		if value == "" || strings.Contains(value, ",") {
			return false
		}
		normalized, err := oauth.NormalizeRequestOrigin(value)
		return err == nil && normalized == expectedOrigin
	default:
		return false
	}
}
