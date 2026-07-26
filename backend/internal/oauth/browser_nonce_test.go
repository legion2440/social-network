package oauth

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBrowserNonceGenerationAndValidation(t *testing.T) {
	nonce, err := NewBrowserNonce()
	if err != nil {
		t.Fatalf("new browser nonce: %v", err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(nonce)
	if err != nil || len(decoded) != BrowserNonceBytes || strings.Contains(nonce, "=") {
		t.Fatalf("unexpected browser nonce %q: bytes=%d err=%v", nonce, len(decoded), err)
	}
	hash := HashBrowserNonce(nonce)
	if !ValidBrowserNonceHash(hash) || !ValidBrowserNonce(nonce, hash) {
		t.Fatal("generated browser nonce must validate against its hash")
	}
	if ValidBrowserNonce(nonce+"x", hash) || ValidBrowserNonce(nonce, HashBrowserNonce(nonce+"x")) {
		t.Fatal("changed browser nonce or hash must not validate")
	}
}

func TestBrowserNonceValidationRejectsNonCanonicalOrWrongLengthValues(t *testing.T) {
	nonce := base64.RawURLEncoding.EncodeToString(make([]byte, BrowserNonceBytes))
	hash := HashBrowserNonce(nonce)
	for _, value := range []string{"", "not-base64!", nonce + "=", base64.RawURLEncoding.EncodeToString(make([]byte, 31))} {
		if ValidBrowserNonce(value, hash) {
			t.Fatalf("nonce %q must be rejected", value)
		}
	}
	for _, value := range []string{"", "not-base64!", hash + "=", base64.RawURLEncoding.EncodeToString(make([]byte, 31))} {
		if ValidBrowserNonceHash(value) {
			t.Fatalf("hash %q must be rejected", value)
		}
	}
}
