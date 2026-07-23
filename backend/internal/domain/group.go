package domain

import "time"

type GroupMembershipStatus string

const (
	GroupOwner     GroupMembershipStatus = "owner"
	GroupMember    GroupMembershipStatus = "member"
	GroupInvited   GroupMembershipStatus = "invited"
	GroupRequested GroupMembershipStatus = "requested"
)

func (s GroupMembershipStatus) Valid() bool {
	return s == GroupOwner || s == GroupMember || s == GroupInvited || s == GroupRequested
}

type Group struct {
	ID                        int64
	OwnerUserID               int64
	Owner                     *User
	Title                     string
	Description               string
	CreatedAt                 time.Time
	MembersCount              int64
	ViewerStatus              *GroupMembershipStatus
	ViewerMembershipCreatedAt *time.Time
}

type GroupMembership struct {
	ID        int64
	GroupID   int64
	UserID    int64
	Status    GroupMembershipStatus
	User      *User
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GroupInvitation struct {
	Group     *Group
	CreatedAt time.Time
}

type GroupCursor struct {
	CreatedAt time.Time
	ID        int64
}

type GroupMemberCursor struct {
	OwnerRank int
	UpdatedAt time.Time
	UserID    int64
}

type GroupMembershipCursor struct {
	CreatedAt time.Time
	UserID    int64
}

type GroupInvitationCursor struct {
	CreatedAt time.Time
	GroupID   int64
}
