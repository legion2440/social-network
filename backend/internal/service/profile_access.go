package service

import (
	"context"
	"errors"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

func authorizeProfileRead(
	ctx context.Context,
	users repo.UserRepo,
	follows repo.FollowRepo,
	currentUserID, targetUserID int64,
) (*domain.User, error) {
	target, err := users.GetByID(ctx, targetUserID)
	if errors.Is(err, repo.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if currentUserID == targetUserID || !target.IsPrivate {
		return target, nil
	}
	accepted, err := follows.IsAccepted(ctx, currentUserID, targetUserID)
	if err != nil {
		return nil, err
	}
	if !accepted {
		return nil, ErrForbidden
	}
	return target, nil
}
