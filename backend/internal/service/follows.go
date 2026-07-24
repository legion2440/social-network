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
	RelationshipNone     RelationshipStatus = "none"
	RelationshipPending  RelationshipStatus = "pending"
	RelationshipAccepted RelationshipStatus = "accepted"
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

type FollowMutationResult struct {
	Follow              *domain.Follow
	NotificationEffects *NotificationEffects
}

func NewFollowService(users repo.UserRepo, follows repo.FollowRepo, transactions repo.TransactionManager, appClock clock.Clock) *FollowService {
	return &FollowService{users: users, follows: follows, transactions: transactions, clock: appClock}
}

func (s *FollowService) Follow(ctx context.Context, followerUserID, followedUserID int64) (*domain.Follow, error) {
	result, err := s.FollowWithEffects(ctx, followerUserID, followedUserID)
	if err != nil {
		return nil, err
	}
	return result.Follow, nil
}

func (s *FollowService) FollowWithEffects(ctx context.Context, followerUserID, followedUserID int64) (*FollowMutationResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || followerUserID <= 0 || followedUserID <= 0 || followerUserID == followedUserID {
		return nil, ErrInvalidInput
	}

	result := &FollowMutationResult{NotificationEffects: emptyNotificationEffects()}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		builder := newNotificationEffectBuilder()
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
		existing, existingErr := repositories.Follows().Get(ctx, followerUserID, followedUserID)
		if existingErr != nil && !errors.Is(existingErr, repo.ErrNotFound) {
			return existingErr
		}
		now := s.clock.Now().UTC()
		result.Follow, err = repositories.Follows().Upsert(ctx, followerUserID, followedUserID, desiredStatus, now)
		if err != nil {
			return err
		}
		if errors.Is(existingErr, repo.ErrNotFound) {
			typeValue := domain.NotificationFollowStarted
			if result.Follow.Status == domain.FollowPending {
				typeValue = domain.NotificationFollowRequest
			}
			followID := result.Follow.ID
			if err := createNotificationTx(ctx, repositories, builder, &domain.Notification{
				RecipientUserID: followedUserID, ActorUserID: followerUserID, Type: typeValue,
				FollowID: &followID, CreatedAt: now,
			}); err != nil {
				return err
			}
		} else if existing.Status == domain.FollowPending && result.Follow.Status == domain.FollowAccepted {
			notification, findErr := repositories.Notifications().FindPendingByFollowID(ctx, domain.NotificationFollowRequest, result.Follow.ID)
			if findErr != nil && !errors.Is(findErr, repo.ErrNotFound) {
				return findErr
			}
			if findErr == nil {
				if _, err := resolveNotificationTx(ctx, repositories, builder, notification, domain.NotificationAccepted, now); err != nil {
					return err
				}
			}
		}
		result.NotificationEffects, err = builder.finish(ctx, repositories)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *FollowService) Unfollow(ctx context.Context, followerUserID, followedUserID int64) error {
	_, err := s.UnfollowWithEffects(ctx, followerUserID, followedUserID)
	return err
}

func (s *FollowService) UnfollowWithEffects(ctx context.Context, followerUserID, followedUserID int64) (*NotificationEffects, error) {
	if s == nil || s.transactions == nil || s.clock == nil || followerUserID <= 0 || followedUserID <= 0 || followerUserID == followedUserID {
		return nil, ErrInvalidInput
	}
	effects := emptyNotificationEffects()
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := repositories.Users().GetByID(ctx, followedUserID); errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		follow, err := repositories.Follows().Get(ctx, followerUserID, followedUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		builder := newNotificationEffectBuilder()
		references, err := repositories.Notifications().ReferencesByFollowID(ctx, follow.ID)
		if err != nil {
			return err
		}
		var pending *domain.Notification
		if follow.Status == domain.FollowPending {
			pending, err = repositories.Notifications().FindPendingByFollowID(ctx, domain.NotificationFollowRequest, follow.ID)
			if err != nil && !errors.Is(err, repo.ErrNotFound) {
				return err
			}
		}
		if err := repositories.Follows().Delete(ctx, followerUserID, followedUserID); err != nil {
			return err
		}
		if pending != nil {
			if _, err := resolveNotificationTx(ctx, repositories, builder, pending, domain.NotificationCancelled, s.clock.Now().UTC()); err != nil {
				return err
			}
		}
		for _, reference := range references {
			builder.changed(reference.RecipientUserID, reference.ID)
		}
		effects, err = builder.finish(ctx, repositories)
		return err
	})
	return effects, err
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

func (s *FollowService) AcceptRequestWithEffects(ctx context.Context, currentUserID, requestID int64) (*FollowMutationResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || currentUserID <= 0 || requestID <= 0 {
		return nil, ErrInvalidInput
	}
	result := &FollowMutationResult{NotificationEffects: emptyNotificationEffects()}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		builder := newNotificationEffectBuilder()
		notification, notificationErr := repositories.Notifications().FindPendingByFollowID(ctx, domain.NotificationFollowRequest, requestID)
		if notificationErr != nil && !errors.Is(notificationErr, repo.ErrNotFound) {
			return notificationErr
		}
		var err error
		result.Follow, err = acceptFollowRequestTx(ctx, repositories, requestID, currentUserID, s.clock.Now().UTC())
		if err != nil {
			return err
		}
		if notification != nil {
			if _, err := resolveNotificationTx(ctx, repositories, builder, notification, domain.NotificationAccepted, s.clock.Now().UTC()); err != nil {
				return err
			}
		}
		result.NotificationEffects, err = builder.finish(ctx, repositories)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *FollowService) RejectRequestWithEffects(ctx context.Context, currentUserID, requestID int64) (*FollowMutationResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || currentUserID <= 0 || requestID <= 0 {
		return nil, ErrInvalidInput
	}
	result := &FollowMutationResult{NotificationEffects: emptyNotificationEffects()}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		builder := newNotificationEffectBuilder()
		var err error
		result.Follow, err = repositories.Follows().GetByID(ctx, requestID)
		if errors.Is(err, repo.ErrNotFound) || err == nil && result.Follow.FollowedUserID != currentUserID {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		notification, notificationErr := repositories.Notifications().FindPendingByFollowID(ctx, domain.NotificationFollowRequest, requestID)
		if notificationErr != nil && !errors.Is(notificationErr, repo.ErrNotFound) {
			return notificationErr
		}
		if err := declineFollowRequestTx(ctx, repositories, requestID, currentUserID); err != nil {
			return err
		}
		if notification != nil {
			if _, err := resolveNotificationTx(ctx, repositories, builder, notification, domain.NotificationDeclined, s.clock.Now().UTC()); err != nil {
				return err
			}
			builder.changed(notification.RecipientUserID, notification.ID)
		}
		result.NotificationEffects, err = builder.finish(ctx, repositories)
		return err
	})
	return result, err
}

func (s *FollowService) ListFollowers(ctx context.Context, currentUserID, targetUserID int64) ([]*domain.RelatedUser, error) {
	return s.listFollowUsers(ctx, currentUserID, targetUserID, true)
}

func (s *FollowService) ListFollowing(ctx context.Context, currentUserID, targetUserID int64) ([]*domain.RelatedUser, error) {
	return s.listFollowUsers(ctx, currentUserID, targetUserID, false)
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

func (s *FollowService) listFollowUsers(
	ctx context.Context,
	currentUserID, targetUserID int64,
	followers bool,
) ([]*domain.RelatedUser, error) {
	if s == nil || s.transactions == nil || currentUserID <= 0 || targetUserID <= 0 {
		return nil, ErrInvalidInput
	}

	var users []*domain.RelatedUser
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := authorizeProfileRead(ctx, repositories.Users(), repositories.Follows(), currentUserID, targetUserID); err != nil {
			return err
		}
		var err error
		if followers {
			users, err = repositories.Follows().ListFollowers(ctx, targetUserID, currentUserID)
		} else {
			users, err = repositories.Follows().ListFollowing(ctx, targetUserID, currentUserID)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}

func relationshipStatus(status domain.FollowStatus) RelationshipStatus {
	switch status {
	case domain.FollowPending:
		return RelationshipPending
	case domain.FollowAccepted:
		return RelationshipAccepted
	default:
		return RelationshipNone
	}
}
