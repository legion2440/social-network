package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsSameOriginRequiresExactHTTPHost(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{name: "http", origin: "http://example.test", host: "example.test", want: true},
		{name: "https with port", origin: "https://example.test:8443", host: "example.test:8443", want: true},
		{name: "case insensitive", origin: "HTTPS://EXAMPLE.TEST", host: "example.test", want: true},
		{name: "missing origin", origin: "", host: "example.test"},
		{name: "foreign host", origin: "https://attacker.test", host: "example.test"},
		{name: "foreign port", origin: "https://example.test:444", host: "example.test"},
		{name: "unsupported scheme", origin: "file://example.test", host: "example.test"},
		{name: "path is not an origin", origin: "https://example.test/path", host: "example.test"},
		{name: "userinfo is not an origin", origin: "https://user@example.test", host: "example.test"},
		{name: "malformed", origin: "://example.test", host: "example.test"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://"+test.host+"/ws", nil)
			request.Host = test.host
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if got := IsSameOrigin(request); got != test.want {
				t.Fatalf("IsSameOrigin()=%v want=%v", got, test.want)
			}
		})
	}
}

func TestDecodeStrictJSONRejectsDuplicateNestedFields(t *testing.T) {
	var request chatSendRequest
	err := decodeStrictJSON([]byte(`{
		"type":"chat:send",
		"client_message_id":"47cd9266-b43f-4a89-9338-4f9c197ff12a",
		"chat":{"kind":"direct","target_id":2,"target_id":3},
		"text":"message"
	}`), &request, true)
	if err == nil {
		t.Fatal("duplicate nested field was accepted")
	}
}
