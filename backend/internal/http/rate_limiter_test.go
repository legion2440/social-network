package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestClientIPTrustProxyRules(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		remoteAddr  string
		trustProxy  bool
		forwarded   []string
		wantAddress string
	}{
		{
			name: "untrusted ignores spoofed header", remoteAddr: "192.0.2.10:1234",
			forwarded: []string{"198.51.100.1"}, wantAddress: "192.0.2.10",
		},
		{
			name: "trusted uses first client", remoteAddr: "192.0.2.10:1234", trustProxy: true,
			forwarded: []string{"198.51.100.1, 203.0.113.2"}, wantAddress: "198.51.100.1",
		},
		{
			name: "trusted validates all entries", remoteAddr: "192.0.2.10:1234", trustProxy: true,
			forwarded: []string{"198.51.100.1, malformed"}, wantAddress: "192.0.2.10",
		},
		{
			name: "trusted validates multiple header lines", remoteAddr: "192.0.2.10:1234", trustProxy: true,
			forwarded: []string{"198.51.100.1", "malformed"}, wantAddress: "192.0.2.10",
		},
		{
			name: "trusted canonicalizes IPv6", remoteAddr: "[2001:db8::9]:1234", trustProxy: true,
			forwarded: []string{"2001:0db8::1, 2001:db8::2"}, wantAddress: "2001:db8::1",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = testCase.remoteAddr
			for _, value := range testCase.forwarded {
				request.Header.Add("X-Forwarded-For", value)
			}
			if got := requestClientIP(request, testCase.trustProxy); got != testCase.wantAddress {
				t.Fatalf("requestClientIP()=%q want=%q", got, testCase.wantAddress)
			}
		})
	}
}

func TestRequestRateLimiterSeparatesClientsAndBoundsEntries(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	limiter := newRequestRateLimiter(1, time.Minute)
	if !limiter.Allow("198.51.100.1", now) || !limiter.Allow("198.51.100.2", now) {
		t.Fatal("different clients must receive independent limits")
	}
	if limiter.Allow("198.51.100.1", now) {
		t.Fatal("same client exceeded its limit")
	}

	bounded := newRequestRateLimiter(1, time.Hour)
	for index := 0; index < maxRateLimitEntries; index++ {
		if !bounded.Allow(fmt.Sprintf("192.0.2.%d", index), now) {
			t.Fatalf("entry %d unexpectedly rejected", index)
		}
	}
	if bounded.Allow("overflow", now) || len(bounded.entries) != maxRateLimitEntries {
		t.Fatalf("limiter did not enforce entry bound: %d", len(bounded.entries))
	}
}
