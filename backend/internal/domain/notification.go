package domain

import "time"

type NotificationType string

const (
	NotificationFollowStarted    NotificationType = "follow_started"
	NotificationFollowRequest    NotificationType = "follow_request"
	NotificationGroupInvitation  NotificationType = "group_invitation"
	NotificationGroupJoinRequest NotificationType = "group_join_request"
	NotificationGroupEvent       NotificationType = "group_event"
)

func (t NotificationType) Valid() bool {
	return t == NotificationFollowStarted || t == NotificationFollowRequest ||
		t == NotificationGroupInvitation || t == NotificationGroupJoinRequest || t == NotificationGroupEvent
}

func (t NotificationType) Actionable() bool {
	return t == NotificationFollowRequest || t == NotificationGroupInvitation || t == NotificationGroupJoinRequest
}

type NotificationResolution string

const (
	NotificationAccepted  NotificationResolution = "accepted"
	NotificationDeclined  NotificationResolution = "declined"
	NotificationCancelled NotificationResolution = "cancelled"
)

func (r NotificationResolution) Valid() bool {
	return r == NotificationAccepted || r == NotificationDeclined || r == NotificationCancelled
}

type Notification struct {
	ID              int64
	RecipientUserID int64
	ActorUserID     int64
	Actor           *User
	Type            NotificationType
	FollowID        *int64
	GroupID         *int64
	GroupTitle      *string
	EventID         *int64
	EventTitle      *string
	EventStartsAt   *time.Time
	MembershipID    *int64
	Resolution      *NotificationResolution
	ResolvedAt      *time.Time
	ReadAt          *time.Time
	CreatedAt       time.Time
}

type NotificationCursor struct {
	CreatedAt time.Time
	ID        int64
}

type NotificationReference struct {
	ID              int64
	RecipientUserID int64
}
