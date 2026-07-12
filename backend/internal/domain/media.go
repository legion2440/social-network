package domain

import (
	"strconv"
	"time"
)

type Media struct {
	ID           int64     `json:"id"`
	OwnerUserID  int64     `json:"-"`
	MIME         string    `json:"mime"`
	Size         int64     `json:"size"`
	StorageKey   string    `json:"-"`
	OriginalName string    `json:"-"`
	CreatedAt    time.Time `json:"-"`
	URL          string    `json:"url"`
}

func MediaURL(id int64) string {
	if id <= 0 {
		return ""
	}
	return "/uploads/" + strconv.FormatInt(id, 10)
}

func (m *Media) Public() *Media {
	if m == nil {
		return nil
	}
	return &Media{
		ID:   m.ID,
		MIME: m.MIME,
		Size: m.Size,
		URL:  MediaURL(m.ID),
	}
}
