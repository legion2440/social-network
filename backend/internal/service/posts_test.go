package service

import (
	"errors"
	"testing"
	"time"

	"social-network/backend/internal/domain"
)

func TestNormalizeSelectedPostUsersDeduplicatesAndEnforcesBounds(t *testing.T) {
	got, err := normalizeSelectedPostUsers(1, domain.PostSelected, []int64{2, 3, 2})
	if err != nil || len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Fatalf("deduplicate audience: got=%v err=%v", got, err)
	}

	hundred := make([]int64, MaxSelectedPostUsers)
	for index := range hundred {
		hundred[index] = int64(index + 2)
	}
	if got, err := normalizeSelectedPostUsers(1, domain.PostSelected, hundred); err != nil || len(got) != MaxSelectedPostUsers {
		t.Fatalf("accept maximum audience: len=%d err=%v", len(got), err)
	}
	if _, err := normalizeSelectedPostUsers(1, domain.PostSelected, append(hundred, 1002)); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected oversized audience rejection, got %v", err)
	}

	for name, test := range map[string]struct {
		privacy domain.PostPrivacy
		values  []int64
	}{
		"selected empty":     {privacy: domain.PostSelected},
		"selected author":    {privacy: domain.PostSelected, values: []int64{1}},
		"selected invalid":   {privacy: domain.PostSelected, values: []int64{0}},
		"public audience":    {privacy: domain.PostPublic, values: []int64{2}},
		"followers audience": {privacy: domain.PostFollowers, values: []int64{2}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeSelectedPostUsers(1, test.privacy, test.values); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
}

func TestPostCursorRoundTripAndStrictRejection(t *testing.T) {
	want := domain.PostCursor{CreatedAt: time.Unix(1_700_000_000, 0).UTC(), ID: 42}
	encoded := EncodePostCursor(want)
	got, err := DecodePostCursor(encoded)
	if err != nil || !got.CreatedAt.Equal(want.CreatedAt) || got.ID != want.ID {
		t.Fatalf("cursor round trip: got=%+v err=%v", got, err)
	}
	for _, invalid := range []string{"", "not-base64", "djI6MTcwMDAwMDAwMDo0Mg", "djE6MDox", "djE6MTcwMDAwMDAwMDow"} {
		if _, err := DecodePostCursor(invalid); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("cursor %q: expected invalid input, got %v", invalid, err)
		}
	}
}
