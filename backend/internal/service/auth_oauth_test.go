package service

import "testing"

func TestNormalizeOAuthNextAllowsOnlySameOriginRelativePaths(t *testing.T) {
	for _, testCase := range []struct {
		input string
		want  string
	}{
		{input: "/", want: "/"},
		{input: "/groups", want: "/groups"},
		{input: "/users/12", want: "/users/12"},
		{input: "/messages/direct/5", want: "/messages/direct/5"},
		{input: "https://example.com", want: "/"},
		{input: "//example.com", want: "/"},
		{input: "javascript:alert(1)", want: "/"},
		{input: `\example.com`, want: "/"},
		{input: `/\example.com`, want: "/"},
		{input: "/groups\nLocation: https://example.com", want: "/"},
	} {
		if got := normalizeOAuthNext(testCase.input); got != testCase.want {
			t.Fatalf("normalizeOAuthNext(%q) = %q, want %q", testCase.input, got, testCase.want)
		}
	}
}
