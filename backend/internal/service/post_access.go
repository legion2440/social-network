package service

import (
	"context"
	"errors"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

func authorizePostAccess(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	viewerUserID, postID int64,
) (*domain.Post, error) {
	post, err := repositories.Posts().GetAccessibleByID(ctx, viewerUserID, postID)
	if err == nil {
		return post, nil
	}
	if !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}
	if _, err := repositories.Posts().GetByID(ctx, postID); errors.Is(err, repo.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return nil, ErrForbidden
}
