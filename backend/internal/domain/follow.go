package domain

import "time"

type FollowStatus string

const (
	FollowPending  FollowStatus = "pending"
	FollowAccepted FollowStatus = "accepted"
)

func (s FollowStatus) Valid() bool {
	return s == FollowPending || s == FollowAccepted
}

type Follow struct {
	ID             int64
	FollowerUserID int64
	FollowedUserID int64
	Status         FollowStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type FollowRequest struct {
	Follow *Follow
	User   *User
}

// RelatedUser carries the relationship between the current viewer and User.
// A nil Status is the external "none" state; pending and accepted reuse the
// persisted follow vocabulary.
type RelatedUser struct {
	User      *User
	Status    *FollowStatus
	FollowsMe bool
}
