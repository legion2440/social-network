package service

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"social-network/backend/internal/platform/id"
)

const (
	MaxMediaBytes      int64 = 20 << 20
	MaxMediaBodyBytes  int64 = MaxMediaBytes + (1 << 20)
	MaxAvatarBytes     int64 = 20 << 20
	MaxAvatarBodyBytes int64 = MaxAvatarBytes + (1 << 20)
)

type MediaUpload struct {
	OriginalName string
	Reader       io.Reader
}

type MediaStager struct {
	ids       id.Generator
	uploadDir string
	maxBytes  int64
}

func NewMediaStager(ids id.Generator, uploadDir string, maxBytes int64) (*MediaStager, error) {
	uploadDir = strings.TrimSpace(uploadDir)
	if uploadDir == "" {
		uploadDir = filepath.Join(".", "var", "uploads")
	}
	if ids == nil || maxBytes <= 0 {
		return nil, ErrInvalidInput
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, err
	}
	return &MediaStager{ids: ids, uploadDir: uploadDir, maxBytes: maxBytes}, nil
}

func (s *MediaStager) Stage(upload MediaUpload) (*StagedMedia, error) {
	if s == nil || s.ids == nil || upload.Reader == nil || s.maxBytes <= 0 {
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
	removeTemp := true
	defer func() {
		_ = tempFile.Close()
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	size, head, err := copyMedia(tempFile, upload.Reader, s.maxBytes)
	if err != nil {
		return nil, err
	}
	if err := tempFile.Close(); err != nil {
		return nil, err
	}

	mime := detectMediaMIME(head)
	extension, ok := mediaExtension(mime)
	if !ok || size <= 0 {
		return nil, ErrInvalidMediaType
	}
	storageID, err := s.ids.New()
	if err != nil {
		return nil, err
	}
	storageKey := storageID + extension
	removeTemp = false
	return &StagedMedia{
		MIME:         mime,
		Size:         size,
		StorageKey:   storageKey,
		OriginalName: strings.TrimSpace(upload.OriginalName),
		tempPath:     tempPath,
		finalPath:    filepath.Join(s.uploadDir, storageKey),
	}, nil
}

type StagedMedia struct {
	MIME         string
	Size         int64
	StorageKey   string
	OriginalName string
	tempPath     string
	finalPath    string
	finalized    bool
	kept         bool
}

func (s *MediaStager) Remove(storageKey string) error {
	storageKey = strings.TrimSpace(storageKey)
	if s == nil || storageKey == "" || filepath.Base(storageKey) != storageKey {
		return ErrInvalidInput
	}
	if err := os.Remove(filepath.Join(s.uploadDir, storageKey)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *StagedMedia) Finalize() error {
	if m == nil || m.tempPath == "" || m.finalPath == "" || m.finalized {
		return ErrInvalidInput
	}
	if err := os.Rename(m.tempPath, m.finalPath); err != nil {
		return err
	}
	m.finalized = true
	return nil
}

func (m *StagedMedia) Keep() {
	if m != nil && m.finalized {
		m.kept = true
	}
}

func (m *StagedMedia) Discard() {
	if m == nil || m.kept {
		return
	}
	_ = os.Remove(m.tempPath)
	if m.finalized {
		_ = os.Remove(m.finalPath)
	}
}

func copyMedia(dst io.Writer, src io.Reader, maxBytes int64) (int64, []byte, error) {
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
			if size > maxBytes {
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

func detectMediaMIME(head []byte) string {
	if len(head) >= 12 && bytes.Equal(head[:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")) {
		return "image/webp"
	}
	return http.DetectContentType(head)
}

func mediaExtension(mime string) (string, bool) {
	switch strings.TrimSpace(mime) {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}
