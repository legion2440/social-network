package domain

import "time"

type AuthIdentity struct {
	ID                    int64
	UserID                int64
	Provider              string
	ProviderUserID        string
	ProviderEmail         string
	ProviderEmailVerified bool
	ProviderUsername      string
	ProviderDisplayName   string
	LinkedAt              time.Time
	LastLoginAt           time.Time
}

type AuthFlow struct {
	Token     string
	Kind      string
	Provider  string
	Payload   string
	CreatedAt time.Time
	ExpiresAt time.Time
}
