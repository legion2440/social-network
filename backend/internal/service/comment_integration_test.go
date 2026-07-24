package service_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

func TestCreateCommentRollsBackWhenTransactionFails(t *testing.T) {
	root := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(root, "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)
	users := sqlite.NewUserRepo(db)
	authorID := createPostTestUser(t, ctx, users, "comment-rollback-author@example.com", now)
	commenterID := createPostTestUser(t, ctx, users, "comment-rollback-user@example.com", now)
	privacy := domain.PostPublic
	postID, err := sqlite.NewPostRepo(db).Create(ctx, &domain.Post{
		AuthorUserID: authorID,
		Text:         "comment target",
		Privacy:      &privacy,
		CreatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	uploadDir := filepath.Join(root, "uploads")
	stager, err := service.NewMediaStager(&authTestIDGenerator{}, uploadDir, service.MaxMediaBytes)
	if err != nil {
		t.Fatalf("new comment media stager: %v", err)
	}

	comments := service.NewCommentService(
		failAfterCallbackTransactions{delegate: sqlite.NewTransactionManager(db)},
		fixedPostClock{now: now},
		stager,
	)
	comment, err := comments.Create(ctx, commenterID, postID, service.CreateCommentInput{
		Text:  "rollback comment",
		Media: &service.MediaUpload{OriginalName: "rollback.png", Reader: bytes.NewReader([]byte("\x89PNG\r\n\x1a\ncomment"))},
	})
	if !errors.Is(err, errForcedCommitFailure) || comment != nil {
		t.Fatalf("expected forced transaction failure, comment=%+v err=%v", comment, err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM post_comments WHERE post_id = ?`, postID).Scan(&count); err != nil {
		t.Fatalf("count comments: %v", err)
	}
	if count != 0 {
		t.Fatalf("rollback left %d comment rows", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM media WHERE owner_user_id = ?`, commenterID).Scan(&count); err != nil {
		t.Fatalf("count comment media: %v", err)
	}
	if count != 0 {
		t.Fatalf("rollback left %d media rows", count)
	}
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("read upload directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("rollback left upload files: %+v", entries)
	}
}

func TestCreateCommentCountsTrimmedUnicodeCodePoints(t *testing.T) {
	root := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(root, "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	now := time.Date(2026, time.July, 21, 12, 1, 0, 0, time.UTC)
	users := sqlite.NewUserRepo(db)
	authorID := createPostTestUser(t, ctx, users, "comment-unicode-author@example.com", now)
	commenterID := createPostTestUser(t, ctx, users, "comment-unicode-user@example.com", now)
	privacy := domain.PostPublic
	postID, err := sqlite.NewPostRepo(db).Create(ctx, &domain.Post{
		AuthorUserID: authorID,
		Text:         "Unicode comments",
		Privacy:      &privacy,
		CreatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	stager, err := service.NewMediaStager(&authTestIDGenerator{}, filepath.Join(root, "uploads"), service.MaxMediaBytes)
	if err != nil {
		t.Fatalf("new comment media stager: %v", err)
	}
	comments := service.NewCommentService(sqlite.NewTransactionManager(db), fixedPostClock{now: now}, stager)

	valid := strings.Repeat("🙂", service.MaxCommentTextRunes)
	if utf8.RuneCountInString(valid) != service.MaxCommentTextRunes {
		t.Fatal("test input does not contain the expected rune count")
	}
	created, err := comments.Create(ctx, commenterID, postID, service.CreateCommentInput{Text: " \n" + valid + "\t "})
	if err != nil {
		t.Fatalf("create max-length Unicode comment: %v", err)
	}
	if created.Text != valid {
		t.Fatal("comment text was not trimmed before storage")
	}

	for name, value := range map[string]string{
		"too long":      valid + "a",
		"invalid UTF-8": string([]byte{0xff}),
	} {
		t.Run(name, func(t *testing.T) {
			comment, err := comments.Create(ctx, commenterID, postID, service.CreateCommentInput{Text: value})
			if !errors.Is(err, service.ErrInvalidInput) || comment != nil {
				t.Fatalf("expected invalid comment rejection, comment=%+v err=%v", comment, err)
			}
		})
	}
}
