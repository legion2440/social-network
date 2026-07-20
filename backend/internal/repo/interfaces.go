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
	CountAccessibleByAuthor(ctx context.Context, viewerUserID, authorUserID int64) (int64, error)
}

type TransactionRepositories interface {
	Users() UserRepo
	Sessions() SessionRepo
	Media() MediaRepo
	Follows() FollowRepo
	Posts() PostRepo
}

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(TransactionRepositories) error) error
}
