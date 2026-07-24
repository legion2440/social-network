package domain

import "time"

type Media struct {
	ID           int64     `json:"id"`
	OwnerUserID  int64     `json:"-"`
	MIME         string    `json:"mime"`
	Size         int64     `json:"size"`
	StorageKey   string    `json:"-"`
	OriginalName string    `json:"-"`
	CreatedAt    time.Time `json:"-"`
}
