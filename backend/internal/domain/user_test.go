package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidDateOfBirth(t *testing.T) {
	for _, testCase := range []struct {
		value string
		valid bool
	}{
		{value: "14-03-1992", valid: true},
		{value: "29-02-2000", valid: true},
		{value: "31-02-1992", valid: false},
		{value: "29-02-1900", valid: false},
		{value: "1992-03-14", valid: false},
		{value: "14/03/1992", valid: false},
		{value: "1-03-1992", valid: false},
		{value: "14-3-1992", valid: false},
		{value: "14-03-0000", valid: false},
	} {
		t.Run(testCase.value, func(t *testing.T) {
			if got := ValidDateOfBirth(testCase.value); got != testCase.valid {
				t.Fatalf("ValidDateOfBirth(%q) = %t, want %t", testCase.value, got, testCase.valid)
			}
		})
	}
}

func TestUserJSONUsesDDMMYYYYDateOfBirth(t *testing.T) {
	payload, err := json.Marshal(User{DateOfBirth: "14-03-1992"})
	if err != nil {
		t.Fatalf("marshal user: %v", err)
	}
	if !strings.Contains(string(payload), `"date_of_birth":"14-03-1992"`) {
		t.Fatalf("unexpected user JSON: %s", payload)
	}
}
