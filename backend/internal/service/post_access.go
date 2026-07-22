package service

import (
	"context"
	"errors"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

func authorizeGroupContentAccess(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	viewerUserID, groupID int64,
) (*domain.Group, error) {
	group, err := repositories.Groups().Get(ctx, groupID, viewerUserID)
	if errors.Is(err, repo.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if group.ViewerStatus == nil || (*group.ViewerStatus != domain.GroupOwner && *group.ViewerStatus != domain.GroupMember) {
		return nil, ErrForbidden
	}
	return group, nil
}

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
