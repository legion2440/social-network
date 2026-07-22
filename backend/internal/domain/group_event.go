package domain

import "time"

type GroupEventResponse string

const (
	GroupEventGoing    GroupEventResponse = "going"
	GroupEventNotGoing GroupEventResponse = "not_going"
)

func (r GroupEventResponse) Valid() bool {
	return r == GroupEventGoing || r == GroupEventNotGoing
}

type GroupEvent struct {
	ID             int64
	GroupID        int64
	CreatorUserID  int64
	Creator        *User
	Title          string
	Description    string
	StartsAt       time.Time
	CreatedAt      time.Time
	GoingCount     int64
	NotGoingCount  int64
	ViewerResponse *GroupEventResponse
}

type GroupEventCursor struct {
	StartsAt time.Time
	ID       int64
}
