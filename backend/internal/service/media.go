package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/platform/id"
	"social-network/backend/internal/repo"
)

type MediaService struct {
	media  repo.MediaRepo
	clock  clock.Clock
	stager *MediaStager
}

func NewMediaService(media repo.MediaRepo, appClock clock.Clock, ids id.Generator, uploadDir string) (*MediaService, error) {
	stager, err := NewMediaStager(ids, uploadDir, MaxMediaBytes)
	if err != nil {
		return nil, err
	}
	return &MediaService{media: media, clock: appClock, stager: stager}, nil
}

func (s *MediaService) Upload(ctx context.Context, userID int64, upload MediaUpload) (*domain.Media, error) {
	if userID <= 0 || upload.Reader == nil || s == nil || s.media == nil || s.clock == nil || s.stager == nil {
		return nil, ErrInvalidInput
	}
	staged, err := s.stager.Stage(upload)
	if err != nil {
		return nil, err
	}
	defer staged.Discard()
	if err := staged.Finalize(); err != nil {
		return nil, err
	}

	media := &domain.Media{
		OwnerUserID:  userID,
		MIME:         staged.MIME,
		Size:         staged.Size,
		StorageKey:   staged.StorageKey,
		OriginalName: staged.OriginalName,
		CreatedAt:    s.clock.Now(),
	}
	media.ID, err = s.media.Create(ctx, media.OwnerUserID, media.MIME, media.Size, media.StorageKey, media.OriginalName, media.CreatedAt)
	if err != nil {
		return nil, err
	}
	staged.Keep()
	media.URL = domain.MediaURL(media.ID)
	return media.Public(), nil
}

func (s *MediaService) OpenOwned(ctx context.Context, mediaID, userID int64) (*domain.Media, string, error) {
	if mediaID <= 0 || userID <= 0 || s == nil || s.media == nil {
		return nil, "", ErrNotFound
	}

	media, err := s.media.GetByID(ctx, mediaID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	if media.OwnerUserID != userID {
		return nil, "", ErrNotFound
	}
	if media.StorageKey == "" || filepath.Base(media.StorageKey) != media.StorageKey {
		return nil, "", fmt.Errorf("invalid media storage key")
	}
	return media.Public(), filepath.Join(s.stager.uploadDir, media.StorageKey), nil
}
