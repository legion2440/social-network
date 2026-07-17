package service_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

func TestReplaceAvatarRollbackPreservesOldAvatarAndRemovesNewFile(t *testing.T) {
	root := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(root, "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	now := time.Date(2026, time.July, 17, 10, 0, 0, 0, time.UTC)
	users := sqlite.NewUserRepo(db)
	userID, err := users.Create(ctx, &domain.User{
		Email:        "avatar-rollback@example.com",
		PasswordHash: "hash",
		FirstName:    "Avatar",
		LastName:     "Rollback",
		DateOfBirth:  "14-03-1992",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	uploadDir := filepath.Join(root, "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		t.Fatalf("create uploads: %v", err)
	}
	oldStorageKey := "old-avatar.png"
	oldContents := []byte("\x89PNG\r\n\x1a\nold-avatar")
	if err := os.WriteFile(filepath.Join(uploadDir, oldStorageKey), oldContents, 0o600); err != nil {
		t.Fatalf("write old avatar: %v", err)
	}
	media := sqlite.NewMediaRepo(db)
	oldMediaID, err := media.Create(ctx, userID, "image/png", int64(len(oldContents)), oldStorageKey, "old.png", now)
	if err != nil {
		t.Fatalf("create old media: %v", err)
	}
	if err := users.SetAvatarMediaID(ctx, userID, &oldMediaID, now); err != nil {
		t.Fatalf("link old avatar: %v", err)
	}

	stager, err := service.NewMediaStager(&authTestIDGenerator{}, uploadDir, service.MaxAvatarBytes)
	if err != nil {
		t.Fatalf("new media stager: %v", err)
	}
	profile := service.NewProfileService(
		failAfterCallbackTransactions{delegate: sqlite.NewTransactionManager(db)},
		clock.RealClock{},
		stager,
		nil,
	)
	newContents := []byte("\x89PNG\r\n\x1a\nnew-avatar")
	updated, err := profile.ReplaceAvatar(ctx, userID, service.MediaUpload{
		OriginalName: "new.png",
		Reader:       bytes.NewReader(newContents),
	})
	if !errors.Is(err, errForcedCommitFailure) || updated != nil {
		t.Fatalf("expected forced transaction failure, user=%+v err=%v", updated, err)
	}

	storedUser, err := users.GetByID(ctx, userID)
	if err != nil {
		t.Fatalf("get user after rollback: %v", err)
	}
	if storedUser.AvatarMediaID == nil || *storedUser.AvatarMediaID != oldMediaID {
		t.Fatalf("old avatar relation was not preserved: %v", storedUser.AvatarMediaID)
	}
	if _, err := media.GetByID(ctx, oldMediaID); err != nil {
		t.Fatalf("old media row was not preserved: %v", err)
	}
	var mediaCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM media`).Scan(&mediaCount); err != nil {
		t.Fatalf("count media: %v", err)
	}
	if mediaCount != 1 {
		t.Fatalf("expected only old media row after rollback, got %d", mediaCount)
	}
	storedContents, err := os.ReadFile(filepath.Join(uploadDir, oldStorageKey))
	if err != nil {
		t.Fatalf("read old avatar after rollback: %v", err)
	}
	if !bytes.Equal(storedContents, oldContents) {
		t.Fatalf("old avatar contents changed: %q", storedContents)
	}
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("read uploads: %v", err)
	}
	if len(files) != 1 || files[0].Name() != oldStorageKey {
		t.Fatalf("rollback left unexpected files: %+v", files)
	}
}
