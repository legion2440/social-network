package http

import (
	"net/http"
	"strings"
)

type SessionTokenExtractor interface {
	Extract(r *http.Request) (string, bool)
}

type CookieSessionTokenExtractor struct {
	name string
}

func NewCookieSessionTokenExtractor(name string) CookieSessionTokenExtractor {
	return CookieSessionTokenExtractor{name: strings.TrimSpace(name)}
}

func (e CookieSessionTokenExtractor) Extract(r *http.Request) (string, bool) {
	if r == nil || e.name == "" {
		return "", false
	}
	cookie, err := r.Cookie(e.name)
	if err != nil {
		return "", false
	}
	token := strings.TrimSpace(cookie.Value)
	return token, token != ""
}
