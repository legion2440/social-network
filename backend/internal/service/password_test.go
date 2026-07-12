package service

import "testing"

func TestBcryptPasswordHelpers(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("password was not hashed")
	}
	if err := VerifyPassword(hash, "correct horse battery staple"); err != nil {
		t.Fatalf("verify correct password: %v", err)
	}
	if err := VerifyPassword(hash, "wrong password"); err == nil {
		t.Fatal("wrong password unexpectedly verified")
	}
}
