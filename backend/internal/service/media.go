package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/platform/id"
	"social-network/backend/internal/repo"
)

const (
	MaxMediaBytes     int64 = 20 << 20
	MaxMediaBodyBytes int64 = MaxMediaBytes + (1 << 20)
)

type MediaUpload struct {
	OriginalName string
	Reader       io.Reader
}

type MediaService struct {
	media     repo.MediaRepo
	clock     clock.Clock
	ids       id.Generator
	uploadDir string
}

func NewMediaService(media repo.MediaRepo, appClock clock.Clock, ids id.Generator, uploadDir string) (*MediaService, error) {
	uploadDir = strings.TrimSpace(uploadDir)
	if uploadDir == "" {
		uploadDir = filepath.Join(".", "var", "uploads")
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, err
	}
	return &MediaService{media: media, clock: appClock, ids: ids, uploadDir: uploadDir}, nil
}

func (s *MediaService) Upload(ctx context.Context, userID int64, upload MediaUpload) (*domain.Media, error) {
	if userID <= 0 || upload.Reader == nil || s == nil || s.media == nil || s.clock == nil || s.ids == nil {
		return nil, ErrInvalidInput
	}

	tempID, err := s.ids.New()
	if err != nil {
		return nil, err
	}
	tempPath := filepath.Join(s.uploadDir, tempID+".tmp")
	tempFile, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	keepTemp := false
	defer func() {
		_ = tempFile.Close()
		if !keepTemp {
			_ = os.Remove(tempPath)
		}
	}()

	size, head, err := copyMedia(tempFile, upload.Reader)
	if err != nil {
		return nil, err
	}
	if err := tempFile.Close(); err != nil {
		return nil, err
	}

	mime := http.DetectContentType(head)
	ext, ok := mediaExtension(mime)
	if !ok || size <= 0 {
		return nil, ErrInvalidMediaType
	}

	finalID, err := s.ids.New()
	if err != nil {
		return nil, err
	}
	storageKey := finalID + ext
	finalPath := filepath.Join(s.uploadDir, storageKey)
	if err := os.Rename(tempPath, finalPath); err != nil {
		return nil, err
	}
	keepTemp = true

	media := &domain.Media{
		OwnerUserID:  userID,
		MIME:         mime,
		Size:         size,
		StorageKey:   storageKey,
		OriginalName: strings.TrimSpace(upload.OriginalName),
		CreatedAt:    s.clock.Now(),
	}
	media.ID, err = s.media.Create(ctx, media.OwnerUserID, media.MIME, media.Size, media.StorageKey, media.OriginalName, media.CreatedAt)
	if err != nil {
		_ = os.Remove(finalPath)
		return nil, err
	}
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
	return media.Public(), filepath.Join(s.uploadDir, media.StorageKey), nil
}

func copyMedia(dst io.Writer, src io.Reader) (int64, []byte, error) {
	buffer := make([]byte, 32*1024)
	head := make([]byte, 0, 512)
	var size int64
	for {
		n, readErr := src.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if len(head) < 512 {
				needed := 512 - len(head)
				if needed > len(chunk) {
					needed = len(chunk)
				}
				head = append(head, chunk[:needed]...)
			}
			size += int64(n)
			if size > MaxMediaBytes {
				return 0, nil, ErrMediaTooBig
			}
			if _, err := dst.Write(chunk); err != nil {
				return 0, nil, err
			}
		}
		if errors.Is(readErr, io.EOF) {
			return size, head, nil
		}
		if readErr != nil {
			return 0, nil, readErr
		}
	}
}

func mediaExtension(mime string) (string, bool) {
	switch strings.TrimSpace(mime) {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/gif":
		return ".gif", true
	default:
		return "", false
	}
}
