package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

const BrowserNonceBytes = 32

func NewBrowserNonce() (string, error) {
	value := make([]byte, BrowserNonceBytes)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func HashBrowserNonce(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func ValidBrowserNonce(value, expectedHash string) bool {
	nonce, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(nonce) != BrowserNonceBytes ||
		base64.RawURLEncoding.EncodeToString(nonce) != value {
		return false
	}
	expected, err := base64.RawURLEncoding.DecodeString(expectedHash)
	if err != nil || len(expected) != sha256.Size ||
		base64.RawURLEncoding.EncodeToString(expected) != expectedHash {
		return false
	}
	actual := sha256.Sum256([]byte(value))
	return subtle.ConstantTimeCompare(actual[:], expected) == 1
}

func ValidBrowserNonceHash(value string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil &&
		len(decoded) == sha256.Size &&
		base64.RawURLEncoding.EncodeToString(decoded) == value
}
