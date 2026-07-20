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

type PostMediaDelivery struct {
	PostID  int64
	MediaID int64
	MIME    string
	Path    string
}

type PostMediaDeliveryService struct {
	transactions repo.TransactionManager
	uploadDir    string
}

func NewPostMediaDeliveryService(transactions repo.TransactionManager, uploadDir string) *PostMediaDeliveryService {
	uploadDir = strings.TrimSpace(uploadDir)
	if uploadDir == "" {
		uploadDir = filepath.Join(".", "var", "uploads")
	}
	return &PostMediaDeliveryService{transactions: transactions, uploadDir: uploadDir}
}

func (s *PostMediaDeliveryService) Open(ctx context.Context, viewerUserID, postID int64) (*PostMediaDelivery, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || postID <= 0 {
		return nil, ErrInvalidInput
	}

	var post *domain.Post
	var media *domain.Media
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		post, err = repositories.Posts().GetByID(ctx, postID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := repositories.Posts().GetAccessibleByID(ctx, viewerUserID, postID); errors.Is(err, repo.ErrNotFound) {
			return ErrForbidden
		} else if err != nil {
			return err
		}
		if post.MediaID == nil || *post.MediaID <= 0 {
			return ErrNotFound
		}
		media, err = repositories.Media().GetByID(ctx, *post.MediaID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if media.OwnerUserID != post.AuthorUserID {
			return ErrNotFound
		}
		if media.StorageKey == "" || filepath.Base(media.StorageKey) != media.StorageKey {
			return fmt.Errorf("invalid post media storage key")
		}
		if _, ok := mediaExtension(media.MIME); !ok {
			return fmt.Errorf("invalid post media MIME: %q", media.MIME)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &PostMediaDelivery{
		PostID:  post.ID,
		MediaID: media.ID,
		MIME:    media.MIME,
		Path:    filepath.Join(s.uploadDir, media.StorageKey),
	}, nil
}
