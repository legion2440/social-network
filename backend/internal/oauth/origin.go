package oauth

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const GitHubCallbackPath = "/api/auth/oauth/github/callback"

var errInvalidOrigin = errors.New("invalid origin")

func NormalizePublicOrigin(value string) (string, error) {
	parsed, err := parseHTTPURL(value)
	if err != nil || parsed.User != nil || hasQueryOrFragment(value, parsed) {
		return "", errInvalidOrigin
	}
	path := parsed.EscapedPath()
	if path != "" && path != "/" {
		return "", errInvalidOrigin
	}
	return normalizedURLOrigin(parsed)
}

func NormalizeRequestOrigin(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || strings.EqualFold(value, "null") {
		return "", errInvalidOrigin
	}
	parsed, err := parseHTTPURL(value)
	if err != nil ||
		parsed.User != nil ||
		parsed.EscapedPath() != "" ||
		hasQueryOrFragment(value, parsed) {
		return "", errInvalidOrigin
	}
	return normalizedURLOrigin(parsed)
}

func ValidateGitHubRedirectURL(value string) (string, error) {
	parsed, err := parseHTTPURL(value)
	if err != nil ||
		parsed.User != nil ||
		parsed.EscapedPath() != GitHubCallbackPath ||
		hasQueryOrFragment(value, parsed) {
		return "", errors.New("invalid GitHub OAuth redirect URL")
	}
	origin, err := normalizedURLOrigin(parsed)
	if err != nil {
		return "", errors.New("invalid GitHub OAuth redirect URL")
	}
	return origin, nil
}

func parseHTTPURL(value string) (*url.URL, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return nil, errInvalidOrigin
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil {
		return nil, errInvalidOrigin
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Opaque != "" ||
		parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errInvalidOrigin
	}
	return parsed, nil
}

func normalizedURLOrigin(parsed *url.URL) (string, error) {
	hostname := strings.ToLower(parsed.Hostname())
	hostname = strings.TrimSuffix(hostname, ".")
	if hostname == "" {
		return "", errInvalidOrigin
	}

	port, err := effectivePort(parsed)
	if err != nil {
		return "", err
	}
	return parsed.Scheme + "://" + net.JoinHostPort(hostname, port), nil
}

func effectivePort(parsed *url.URL) (string, error) {
	port := parsed.Port()
	if hasExplicitEmptyPort(parsed.Host) {
		return "", errInvalidOrigin
	}
	if port == "" {
		if parsed.Scheme == "http" {
			return "80", nil
		}
		return "443", nil
	}
	for _, character := range port {
		if character < '0' || character > '9' {
			return "", errInvalidOrigin
		}
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return "", errInvalidOrigin
	}
	return strconv.Itoa(number), nil
}

func hasExplicitEmptyPort(host string) bool {
	if strings.HasPrefix(host, "[") {
		closingBracket := strings.LastIndex(host, "]")
		return closingBracket >= 0 && host[closingBracket+1:] == ":"
	}
	return strings.HasSuffix(host, ":")
}

func hasQueryOrFragment(raw string, parsed *url.URL) bool {
	return parsed.ForceQuery ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" ||
		parsed.RawFragment != "" ||
		strings.Contains(raw, "#")
}
