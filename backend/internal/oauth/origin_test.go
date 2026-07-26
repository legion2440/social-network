package oauth

import "testing"

func TestNormalizePublicOrigin(t *testing.T) {
	for _, testCase := range []struct {
		input string
		want  string
	}{
		{input: "HTTP://Example.COM", want: "http://example.com:80"},
		{input: "https://example.com/", want: "https://example.com:443"},
		{input: "http://example.com.:8080", want: "http://example.com:8080"},
		{input: "http://[::1]", want: "http://[::1]:80"},
		{input: "https://[2001:db8::1]:8443", want: "https://[2001:db8::1]:8443"},
	} {
		got, err := NormalizePublicOrigin(testCase.input)
		if err != nil || got != testCase.want {
			t.Fatalf("NormalizePublicOrigin(%q) = %q, %v; want %q", testCase.input, got, err, testCase.want)
		}
	}
}

func TestNormalizePublicOriginRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{
		"",
		" https://example.com",
		"ftp://example.com",
		"https://",
		"https://.",
		"https://user@example.com",
		"https://example.com:",
		"https://[::1]:",
		"https://example.com:0",
		"https://example.com:65536",
		"https://example.com/path",
		"https://example.com/%2F",
		"https://example.com?",
		"https://example.com#",
	} {
		if got, err := NormalizePublicOrigin(value); err == nil {
			t.Fatalf("NormalizePublicOrigin(%q) = %q, want error", value, got)
		}
	}
}

func TestNormalizeRequestOriginIsStricterThanPublicOrigin(t *testing.T) {
	for _, testCase := range []struct {
		input string
		want  string
	}{
		{input: "HTTP://Example.COM", want: "http://example.com:80"},
		{input: "https://example.com:443", want: "https://example.com:443"},
	} {
		got, err := NormalizeRequestOrigin(testCase.input)
		if err != nil || got != testCase.want {
			t.Fatalf("NormalizeRequestOrigin(%q) = %q, %v; want %q", testCase.input, got, err, testCase.want)
		}
	}
	for _, value := range []string{
		"",
		"null",
		"https://example.com/",
		"https://example.com/path",
		"https://example.com, https://other.example",
		"https://example.com?",
		"https://example.com#",
	} {
		if got, err := NormalizeRequestOrigin(value); err == nil {
			t.Fatalf("NormalizeRequestOrigin(%q) = %q, want error", value, got)
		}
	}
}

func TestValidateGitHubRedirectURLRequiresExactEscapedCallbackPath(t *testing.T) {
	got, err := ValidateGitHubRedirectURL("HTTPS://Example.COM/api/auth/oauth/github/callback")
	if err != nil || got != "https://example.com:443" {
		t.Fatalf("valid redirect origin = %q, %v", got, err)
	}
	for _, value := range []string{
		"https://example.com/api/auth/oauth/github/%63allback",
		"https://example.com/api/auth/oauth/github/callback/",
		"https://example.com/api/auth/oauth//github/callback",
		"https://user@example.com/api/auth/oauth/github/callback",
		"https://example.com:/api/auth/oauth/github/callback",
		"https://example.com/api/auth/oauth/github/callback?",
		"https://example.com/api/auth/oauth/github/callback#",
	} {
		if got, err := ValidateGitHubRedirectURL(value); err == nil {
			t.Fatalf("ValidateGitHubRedirectURL(%q) = %q, want error", value, got)
		}
	}
}
