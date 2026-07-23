package service

import (
	"context"
	"errors"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

func acceptFollowRequestTx(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	requestID, followedUserID int64,
	now time.Time,
) (*domain.Follow, error) {
	follow, err := repositories.Follows().Accept(ctx, requestID, followedUserID, now)
	if errors.Is(err, repo.ErrNotFound) {
		return nil, ErrNotFound
	}
	return follow, err
}

func declineFollowRequestTx(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	requestID, followedUserID int64,
) error {
	err := repositories.Follows().Reject(ctx, requestID, followedUserID)
	if errors.Is(err, repo.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func updateGroupMembershipTx(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	membership *domain.GroupMembership,
	expected, next domain.GroupMembershipStatus,
	now time.Time,
) error {
	if membership == nil || membership.Status != expected {
		return ErrConflict
	}
	if err := repositories.Groups().UpdateMembershipStatusByID(ctx, membership.ID, expected, next, now); err != nil {
		return mapGroupRepoError(err)
	}
	membership.Status = next
	membership.UpdatedAt = now
	return nil
}

func deleteGroupMembershipTx(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	membership *domain.GroupMembership,
	expected domain.GroupMembershipStatus,
) error {
	if membership == nil || membership.Status != expected {
		return ErrConflict
	}
	return mapGroupRepoError(repositories.Groups().DeleteMembershipByID(ctx, membership.ID, expected))
}

func relationshipTx(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	currentUserID, targetUserID int64,
) (*Relationship, error) {
	if _, err := repositories.Users().GetByID(ctx, targetUserID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	status := RelationshipNone
	follow, err := repositories.Follows().Get(ctx, currentUserID, targetUserID)
	if err == nil {
		status = relationshipStatus(follow.Status)
	} else if !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}
	followsMe, err := repositories.Follows().IsAccepted(ctx, targetUserID, currentUserID)
	if err != nil {
		return nil, err
	}
	return &Relationship{Status: status, FollowsMe: followsMe}, nil
}
