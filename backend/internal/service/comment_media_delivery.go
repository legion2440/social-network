package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

type CommentMediaDelivery struct {
	CommentID int64
	MediaID   int64
	MIME      string
	Path      string
}

type CommentMediaDeliveryService struct {
	transactions repo.TransactionManager
	uploadDir    string
}

func NewCommentMediaDeliveryService(transactions repo.TransactionManager, uploadDir string) *CommentMediaDeliveryService {
	uploadDir = strings.TrimSpace(uploadDir)
	if uploadDir == "" {
		uploadDir = filepath.Join(".", "var", "uploads")
	}
	return &CommentMediaDeliveryService{transactions: transactions, uploadDir: uploadDir}
}

func (s *CommentMediaDeliveryService) Open(
	ctx context.Context,
	viewerUserID, commentID int64,
) (*CommentMediaDelivery, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || commentID <= 0 {
		return nil, ErrInvalidInput
	}

	var comment *domain.Comment
	var media *domain.Media
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		comment, err = repositories.Comments().GetByID(ctx, commentID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		if _, err := authorizePostAccess(ctx, repositories, viewerUserID, comment.PostID); err != nil {
			return err
		}
		if comment.MediaID == nil || *comment.MediaID <= 0 {
			return ErrNotFound
		}

		media, err = repositories.Media().GetByID(ctx, *comment.MediaID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if media.OwnerUserID != comment.AuthorUserID {
			return ErrNotFound
		}
		if media.StorageKey == "" || filepath.Base(media.StorageKey) != media.StorageKey {
			return ErrNotFound
		}
		if _, ok := mediaExtension(media.MIME); !ok {
			return ErrNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if media == nil {
		return nil, fmt.Errorf("comment media metadata is missing")
	}

	return &CommentMediaDelivery{
		CommentID: comment.ID,
		MediaID:   media.ID,
		MIME:      media.MIME,
		Path:      filepath.Join(s.uploadDir, media.StorageKey),
	}, nil
}
