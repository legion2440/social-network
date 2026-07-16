package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCookieSessionTokenExtractor(t *testing.T) {
	extractor := NewCookieSessionTokenExtractor("session")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if token, ok := extractor.Extract(req); ok || token != "" {
		t.Fatalf("unexpected token without cookie: token=%q ok=%t", token, ok)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "  token-value  "})
	if token, ok := extractor.Extract(req); !ok || token != "token-value" {
		t.Fatalf("unexpected extracted token: token=%q ok=%t", token, ok)
	}
}
