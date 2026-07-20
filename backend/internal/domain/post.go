package domain

import (
	"strconv"
	"time"
)

type PostPrivacy string

const (
	PostPublic    PostPrivacy = "public"
	PostFollowers PostPrivacy = "followers"
	PostSelected  PostPrivacy = "selected"
)

func (p PostPrivacy) Valid() bool {
	return p == PostPublic || p == PostFollowers || p == PostSelected
}

type Post struct {
	ID           int64
	AuthorUserID int64
	Author       *User
	Text         string
	Privacy      PostPrivacy
	MediaID      *int64
	CreatedAt    time.Time
}

func PostMediaURL(post *Post) *string {
	if post == nil || post.ID <= 0 || post.MediaID == nil || *post.MediaID <= 0 {
		return nil
	}
	value := "/api/posts/" + strconv.FormatInt(post.ID, 10) + "/media"
	return &value
}

type PostCursor struct {
	CreatedAt time.Time
	ID        int64
}
