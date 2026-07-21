package domain

import "time"

type Comment struct {
	ID           int64
	PostID       int64
	AuthorUserID int64
	Author       *User
	Text         string
	CreatedAt    time.Time
}

type CommentCursor struct {
	CreatedAt time.Time
	ID        int64
}
