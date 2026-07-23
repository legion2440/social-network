package repo

import (
	"context"
	"time"

	"social-network/backend/internal/domain"
)

type UserRepo interface {
	Create(ctx context.Context, user *domain.User) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	ListDirectory(ctx context.Context, viewerUserID int64, cursor *domain.UserCursor, limit int) ([]*domain.RelatedUser, error)
	UpdateProfile(ctx context.Context, user *domain.User) error
	SetAvatarMediaID(ctx context.Context, userID int64, mediaID *int64, updatedAt time.Time) error
}

type SessionRepo interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByToken(ctx context.Context, token string) (*domain.Session, error)
	DeleteByToken(ctx context.Context, token string) error
}

type MediaRepo interface {
	Create(ctx context.Context, ownerUserID int64, mime string, size int64, storageKey, originalName string, createdAt time.Time) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Media, error)
	DeleteByID(ctx context.Context, id int64) error
}

type FollowRepo interface {
	Upsert(ctx context.Context, followerUserID, followedUserID int64, desiredStatus domain.FollowStatus, now time.Time) (*domain.Follow, error)
	Get(ctx context.Context, followerUserID, followedUserID int64) (*domain.Follow, error)
	GetByID(ctx context.Context, id int64) (*domain.Follow, error)
	Accept(ctx context.Context, id, followedUserID int64, now time.Time) (*domain.Follow, error)
	Delete(ctx context.Context, followerUserID, followedUserID int64) error
	Reject(ctx context.Context, id, followedUserID int64) error
	ListFollowers(ctx context.Context, userID, viewerUserID int64) ([]*domain.RelatedUser, error)
	ListFollowing(ctx context.Context, userID, viewerUserID int64) ([]*domain.RelatedUser, error)
	ListPendingRequests(ctx context.Context, followedUserID int64) ([]*domain.FollowRequest, error)
	IsAccepted(ctx context.Context, followerUserID, followedUserID int64) (bool, error)
	CountFollowers(ctx context.Context, userID int64) (int64, error)
	CountFollowing(ctx context.Context, userID int64) (int64, error)
}

type PostRepo interface {
	Create(ctx context.Context, post *domain.Post) (int64, error)
	AddSelectedUsers(ctx context.Context, postID int64, userIDs []int64) error
	GetByID(ctx context.Context, postID int64) (*domain.Post, error)
	GetAccessibleByID(ctx context.Context, viewerUserID, postID int64) (*domain.Post, error)
	ListFeed(ctx context.Context, viewerUserID int64, cursor *domain.PostCursor, limit int) ([]*domain.Post, error)
	ListByAuthor(ctx context.Context, viewerUserID, authorUserID int64, cursor *domain.PostCursor, limit int) ([]*domain.Post, error)
	ListByGroup(ctx context.Context, viewerUserID, groupID int64, cursor *domain.PostCursor, limit int) ([]*domain.Post, error)
	CountAccessibleByAuthor(ctx context.Context, viewerUserID, authorUserID int64) (int64, error)
}

type CommentRepo interface {
	Create(ctx context.Context, comment *domain.Comment) (int64, error)
	ListByPost(ctx context.Context, viewerUserID, postID int64, cursor *domain.CommentCursor, limit int) ([]*domain.Comment, error)
}

type GroupRepo interface {
	Create(ctx context.Context, group *domain.Group) (int64, error)
	Get(ctx context.Context, groupID, viewerUserID int64) (*domain.Group, error)
	List(ctx context.Context, viewerUserID int64, cursor *domain.GroupCursor, limit int) ([]*domain.Group, error)
	CreateMembership(ctx context.Context, membership *domain.GroupMembership) (int64, error)
	GetMembership(ctx context.Context, groupID, userID int64) (*domain.GroupMembership, error)
	GetMembershipByID(ctx context.Context, membershipID int64) (*domain.GroupMembership, error)
	GetMembershipStatus(ctx context.Context, groupID, userID int64) (*domain.GroupMembershipStatus, error)
	UpdateMembershipStatusByID(ctx context.Context, membershipID int64, expected, next domain.GroupMembershipStatus, now time.Time) error
	DeleteMembershipByID(ctx context.Context, membershipID int64, expected domain.GroupMembershipStatus) error
	ListMembers(ctx context.Context, groupID, viewerUserID int64, cursor *domain.GroupMemberCursor, limit int) ([]*domain.GroupMembership, error)
	ListMemberships(ctx context.Context, groupID, viewerUserID int64, status domain.GroupMembershipStatus, cursor *domain.GroupMembershipCursor, limit int) ([]*domain.GroupMembership, error)
	ListInvitationInbox(ctx context.Context, userID int64, cursor *domain.GroupInvitationCursor, limit int) ([]*domain.GroupInvitation, error)
	ListActiveMemberIDs(ctx context.Context, groupID int64) ([]int64, error)
}

type GroupEventRepo interface {
	Create(ctx context.Context, event *domain.GroupEvent) (int64, error)
	Get(ctx context.Context, viewerUserID, eventID int64) (*domain.GroupEvent, error)
	List(ctx context.Context, viewerUserID, groupID int64, cursor *domain.GroupEventCursor, limit int) ([]*domain.GroupEvent, error)
	UpsertResponse(ctx context.Context, eventID, userID int64, response domain.GroupEventResponse, now time.Time) error
}

type NotificationRepo interface {
	EnsureUserState(ctx context.Context, userID int64) error
	Create(ctx context.Context, notification *domain.Notification) (int64, error)
	GetForRecipient(ctx context.Context, recipientUserID, notificationID int64) (*domain.Notification, error)
	ListForRecipient(ctx context.Context, recipientUserID int64, cursor *domain.NotificationCursor, limit int) ([]*domain.Notification, error)
	FindPendingByFollowID(ctx context.Context, notificationType domain.NotificationType, followID int64) (*domain.Notification, error)
	FindPendingByMembershipID(ctx context.Context, notificationType domain.NotificationType, membershipID int64) (*domain.Notification, error)
	ReferencesByFollowID(ctx context.Context, followID int64) ([]domain.NotificationReference, error)
	Resolve(ctx context.Context, notificationID int64, resolution domain.NotificationResolution, now time.Time) (bool, error)
	MarkRead(ctx context.Context, notificationID, recipientUserID int64, now time.Time) (bool, error)
	MarkAllRead(ctx context.Context, recipientUserID int64, now time.Time) (int64, error)
	UnreadCount(ctx context.Context, recipientUserID int64) (int64, error)
	CurrentRevision(ctx context.Context, userID int64) (int64, error)
	BumpRevision(ctx context.Context, userID int64) (int64, error)
}

type ChatRepo interface {
	GetDirectConversation(ctx context.Context, userLowID, userHighID int64) (*domain.DirectConversation, error)
	EnsureDirectConversation(ctx context.Context, userLowID, userHighID int64, createdAt time.Time) (*domain.DirectConversation, error)
	CreateMessage(ctx context.Context, message *domain.ChatMessage) (int64, error)
	GetMessageByClientID(ctx context.Context, senderUserID int64, clientMessageID string) (*domain.ChatMessage, error)
	ListDirectMessages(ctx context.Context, viewerUserID, targetUserID int64, cursor *domain.ChatMessageCursor, limit int) ([]*domain.ChatMessage, error)
	ListGroupMessages(ctx context.Context, viewerUserID, groupID int64, cursor *domain.ChatMessageCursor, limit int) ([]*domain.ChatMessage, error)
	ListChats(ctx context.Context, viewerUserID int64, cursor *domain.ChatListCursor, limit int) ([]*domain.ChatSummary, error)
	ListDirectPeerIDs(ctx context.Context, userID int64) ([]int64, error)
}

type TransactionRepositories interface {
	Users() UserRepo
	Sessions() SessionRepo
	Media() MediaRepo
	Follows() FollowRepo
	Posts() PostRepo
	Comments() CommentRepo
	Groups() GroupRepo
	GroupEvents() GroupEventRepo
	Notifications() NotificationRepo
	Chats() ChatRepo
}

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(TransactionRepositories) error) error
}
