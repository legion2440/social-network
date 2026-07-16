package service

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type stagingIDGenerator struct {
	mu sync.Mutex
	n  int
}

func (g *stagingIDGenerator) New() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return fmt.Sprintf("stage-%d", g.n), nil
}

func TestMediaStagerAcceptsAvatarFormats(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		contents []byte
		wantMIME string
		wantExt  string
	}{
		{name: "jpeg", contents: []byte("\xff\xd8\xff\xe0avatar"), wantMIME: "image/jpeg", wantExt: ".jpg"},
		{name: "png", contents: []byte("\x89PNG\r\n\x1a\navatar"), wantMIME: "image/png", wantExt: ".png"},
		{name: "gif", contents: []byte("GIF89aavatar"), wantMIME: "image/gif", wantExt: ".gif"},
		{name: "webp", contents: []byte("RIFF\x10\x00\x00\x00WEBPVP8 avatar"), wantMIME: "image/webp", wantExt: ".webp"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			uploadDir := t.TempDir()
			stager, err := NewMediaStager(&stagingIDGenerator{}, uploadDir, MaxAvatarBytes)
			if err != nil {
				t.Fatalf("new stager: %v", err)
			}
			staged, err := stager.Stage(MediaUpload{OriginalName: "avatar", Reader: bytes.NewReader(testCase.contents)})
			if err != nil {
				t.Fatalf("stage media: %v", err)
			}
			defer staged.Discard()
			if staged.MIME != testCase.wantMIME || filepath.Ext(staged.StorageKey) != testCase.wantExt || staged.Size != int64(len(testCase.contents)) {
				t.Fatalf("unexpected staged media: %+v", staged)
			}
			if err := staged.Finalize(); err != nil {
				t.Fatalf("finalize media: %v", err)
			}
			stored, err := os.ReadFile(filepath.Join(uploadDir, staged.StorageKey))
			if err != nil {
				t.Fatalf("read finalized media: %v", err)
			}
			if !bytes.Equal(stored, testCase.contents) {
				t.Fatalf("stored contents changed: %q", stored)
			}
			staged.Discard()
			if _, err := os.Stat(filepath.Join(uploadDir, staged.StorageKey)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("discard did not remove final file: %v", err)
			}
		})
	}
}

func TestMediaStagerRejectsFilesOverConfiguredLimitAndCleansTemp(t *testing.T) {
	uploadDir := t.TempDir()
	stager, err := NewMediaStager(&stagingIDGenerator{}, uploadDir, 4)
	if err != nil {
		t.Fatalf("new stager: %v", err)
	}
	if _, err := stager.Stage(MediaUpload{OriginalName: "too-big.png", Reader: bytes.NewReader([]byte("12345"))}); !errors.Is(err, ErrMediaTooBig) {
		t.Fatalf("expected media too big error, got %v", err)
	}
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("read upload directory: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("oversized upload left temp files: %+v", files)
	}
}

func TestAvatarLimitIsTwentyMegabytes(t *testing.T) {
	if MaxAvatarBytes != 20<<20 {
		t.Fatalf("expected 20MB avatar limit, got %d", MaxAvatarBytes)
	}
}
