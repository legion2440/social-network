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

type AvatarDelivery struct {
	MediaID int64
	MIME    string
	Path    string
}

type AvatarDeliveryService struct {
	transactions repo.TransactionManager
	uploadDir    string
}

func NewAvatarDeliveryService(transactions repo.TransactionManager, uploadDir string) *AvatarDeliveryService {
	uploadDir = strings.TrimSpace(uploadDir)
	if uploadDir == "" {
		uploadDir = filepath.Join(".", "var", "uploads")
	}
	return &AvatarDeliveryService{transactions: transactions, uploadDir: uploadDir}
}

func (s *AvatarDeliveryService) Open(ctx context.Context, currentUserID, targetUserID int64) (*AvatarDelivery, error) {
	if s == nil || s.transactions == nil || currentUserID <= 0 || targetUserID <= 0 {
		return nil, ErrInvalidInput
	}

	var media *domain.Media
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		target, err := authorizeProfileRead(
			ctx,
			repositories.Users(),
			repositories.Follows(),
			currentUserID,
			targetUserID,
		)
		if err != nil {
			return err
		}
		if target.AvatarMediaID == nil || *target.AvatarMediaID <= 0 {
			return ErrNotFound
		}

		media, err = repositories.Media().GetByID(ctx, *target.AvatarMediaID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if media.OwnerUserID != targetUserID {
			return ErrNotFound
		}
		if media.StorageKey == "" || filepath.Base(media.StorageKey) != media.StorageKey {
			return fmt.Errorf("invalid avatar storage key")
		}
		if _, ok := mediaExtension(media.MIME); !ok {
			return fmt.Errorf("invalid avatar MIME: %q", media.MIME)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &AvatarDelivery{
		MediaID: media.ID,
		MIME:    media.MIME,
		Path:    filepath.Join(s.uploadDir, media.StorageKey),
	}, nil
}
