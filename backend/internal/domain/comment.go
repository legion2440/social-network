package domain

import (
	"strconv"
	"time"
)

type Comment struct {
	ID           int64
	PostID       int64
	AuthorUserID int64
	Author       *User
	Text         string
	MediaID      *int64
	CreatedAt    time.Time
}

func CommentMediaURL(comment *Comment) *string {
	if comment == nil || comment.ID <= 0 || comment.MediaID == nil || *comment.MediaID <= 0 {
		return nil
	}
	value := "/api/comments/" + strconv.FormatInt(comment.ID, 10) + "/media"
	return &value
}

type CommentCursor struct {
	CreatedAt time.Time
	ID        int64
}
