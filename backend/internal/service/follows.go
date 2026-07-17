package service

import (
	"context"
	"errors"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/repo"
)

type RelationshipStatus string

const (
	RelationshipNone      RelationshipStatus = "none"
	RelationshipPending   RelationshipStatus = "pending"
	RelationshipFollowing RelationshipStatus = "following"
)

type Relationship struct {
	Status    RelationshipStatus
	FollowsMe bool
}

type FollowService struct {
	users        repo.UserRepo
	follows      repo.FollowRepo
	transactions repo.TransactionManager
	clock        clock.Clock
}

func NewFollowService(users repo.UserRepo, follows repo.FollowRepo, transactions repo.TransactionManager, appClock clock.Clock) *FollowService {
	return &FollowService{users: users, follows: follows, transactions: transactions, clock: appClock}
}

func (s *FollowService) Follow(ctx context.Context, followerUserID, followedUserID int64) (*domain.Follow, error) {
	if s == nil || s.transactions == nil || s.clock == nil || followerUserID <= 0 || followedUserID <= 0 || followerUserID == followedUserID {
		return nil, ErrInvalidInput
	}

	var follow *domain.Follow
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		target, err := repositories.Users().GetByID(ctx, followedUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		desiredStatus := domain.FollowAccepted
		if target.IsPrivate {
			desiredStatus = domain.FollowPending
		}
		follow, err = repositories.Follows().Upsert(ctx, followerUserID, followedUserID, desiredStatus, s.clock.Now())
		return err
	})
	if err != nil {
		return nil, err
	}
	return follow, nil
}

func (s *FollowService) Unfollow(ctx context.Context, followerUserID, followedUserID int64) error {
	if s == nil || s.transactions == nil || followerUserID <= 0 || followedUserID <= 0 || followerUserID == followedUserID {
		return ErrInvalidInput
	}
	return s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := repositories.Users().GetByID(ctx, followedUserID); errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		return repositories.Follows().Delete(ctx, followerUserID, followedUserID)
	})
}

func (s *FollowService) Relationship(ctx context.Context, currentUserID, targetUserID int64) (*Relationship, error) {
	if s == nil || s.follows == nil || currentUserID <= 0 || targetUserID <= 0 || currentUserID == targetUserID {
		return nil, ErrInvalidInput
	}
	if _, err := s.userExists(ctx, targetUserID); err != nil {
		return nil, err
	}

	status := RelationshipNone
	follow, err := s.follows.Get(ctx, currentUserID, targetUserID)
	if err == nil {
		status = relationshipStatus(follow.Status)
	} else if !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}
	followsMe, err := s.follows.IsAccepted(ctx, targetUserID, currentUserID)
	if err != nil {
		return nil, err
	}
	return &Relationship{Status: status, FollowsMe: followsMe}, nil
}

func (s *FollowService) AcceptRequest(ctx context.Context, currentUserID, requestID int64) (*domain.Follow, error) {
	if s == nil || s.transactions == nil || s.clock == nil || currentUserID <= 0 || requestID <= 0 {
		return nil, ErrInvalidInput
	}
	var follow *domain.Follow
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		follow, err = repositories.Follows().Accept(ctx, requestID, currentUserID, s.clock.Now())
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return follow, nil
}

func (s *FollowService) RejectRequest(ctx context.Context, currentUserID, requestID int64) error {
	if s == nil || s.transactions == nil || currentUserID <= 0 || requestID <= 0 {
		return ErrInvalidInput
	}
	return s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		err := repositories.Follows().Reject(ctx, requestID, currentUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		return err
	})
}

func (s *FollowService) ListFollowers(ctx context.Context, userID int64) ([]*domain.User, error) {
	if s == nil || s.follows == nil || userID <= 0 {
		return nil, ErrInvalidInput
	}
	if _, err := s.userExists(ctx, userID); err != nil {
		return nil, err
	}
	return s.follows.ListFollowers(ctx, userID)
}

func (s *FollowService) ListFollowing(ctx context.Context, userID int64) ([]*domain.User, error) {
	if s == nil || s.follows == nil || userID <= 0 {
		return nil, ErrInvalidInput
	}
	if _, err := s.userExists(ctx, userID); err != nil {
		return nil, err
	}
	return s.follows.ListFollowing(ctx, userID)
}

func (s *FollowService) ListPendingRequests(ctx context.Context, currentUserID int64) ([]*domain.FollowRequest, error) {
	if s == nil || s.follows == nil || currentUserID <= 0 {
		return nil, ErrInvalidInput
	}
	return s.follows.ListPendingRequests(ctx, currentUserID)
}

func (s *FollowService) IsFollower(ctx context.Context, followerUserID, followedUserID int64) (bool, error) {
	if s == nil || s.follows == nil || followerUserID <= 0 || followedUserID <= 0 {
		return false, ErrInvalidInput
	}
	if followerUserID == followedUserID {
		return false, nil
	}
	return s.follows.IsAccepted(ctx, followerUserID, followedUserID)
}

func (s *FollowService) userExists(ctx context.Context, userID int64) (*domain.User, error) {
	if s == nil || s.users == nil {
		return nil, ErrInvalidInput
	}
	user, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, repo.ErrNotFound) {
		return nil, ErrNotFound
	}
	return user, err
}

func relationshipStatus(status domain.FollowStatus) RelationshipStatus {
	switch status {
	case domain.FollowPending:
		return RelationshipPending
	case domain.FollowAccepted:
		return RelationshipFollowing
	default:
		return RelationshipNone
	}
}
