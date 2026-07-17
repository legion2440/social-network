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
	ListFollowers(ctx context.Context, userID int64) ([]*domain.User, error)
	ListFollowing(ctx context.Context, userID int64) ([]*domain.User, error)
	ListPendingRequests(ctx context.Context, followedUserID int64) ([]*domain.FollowRequest, error)
	IsAccepted(ctx context.Context, followerUserID, followedUserID int64) (bool, error)
}

type TransactionRepositories interface {
	Users() UserRepo
	Sessions() SessionRepo
	Media() MediaRepo
	Follows() FollowRepo
}

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(TransactionRepositories) error) error
}
