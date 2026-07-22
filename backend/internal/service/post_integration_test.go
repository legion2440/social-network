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

func TestCreatePostRollbackRemovesFinalFileAndAllRows(t *testing.T) {
	root := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(root, "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	now := time.Date(2026, time.July, 18, 10, 0, 0, 0, time.UTC)
	users := sqlite.NewUserRepo(db)
	authorID := createPostTestUser(t, ctx, users, "post-rollback-author@example.com", now)
	followerID := createPostTestUser(t, ctx, users, "post-rollback-follower@example.com", now)
	if _, err := sqlite.NewFollowRepo(db).Upsert(ctx, followerID, authorID, domain.FollowAccepted, now); err != nil {
		t.Fatalf("create accepted follower: %v", err)
	}

	uploadDir := filepath.Join(root, "uploads")
	stager, err := service.NewMediaStager(&authTestIDGenerator{}, uploadDir, service.MaxMediaBytes)
	if err != nil {
		t.Fatalf("new post media stager: %v", err)
	}
	posts := service.NewPostService(
		failAfterCallbackTransactions{delegate: sqlite.NewTransactionManager(db)},
		fixedPostClock{now: now},
		stager,
	)
	png := []byte("\x89PNG\r\n\x1a\npost-image")
	post, err := posts.Create(ctx, authorID, service.CreatePostInput{
		Text:            "  rollback post  ",
		Privacy:         domain.PostSelected,
		SelectedUserIDs: []int64{followerID},
		Media: &service.MediaUpload{
			OriginalName: "post.png",
			Reader:       bytes.NewReader(png),
		},
	})
	if !errors.Is(err, errForcedCommitFailure) || post != nil {
		t.Fatalf("expected forced transaction failure, post=%+v err=%v", post, err)
	}

	for _, table := range []string{"posts", "post_selected_users", "media"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("rollback left %d rows in %s", count, table)
		}
	}
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("read uploads: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("rollback left files: %+v", files)
	}
}

func TestCreatePostCountsTrimmedUnicodeCodePoints(t *testing.T) {
	root := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(root, "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	now := time.Date(2026, time.July, 18, 11, 0, 0, 0, time.UTC)
	users := sqlite.NewUserRepo(db)
	authorID := createPostTestUser(t, ctx, users, "post-unicode@example.com", now)
	stager, err := service.NewMediaStager(&authTestIDGenerator{}, filepath.Join(root, "uploads"), service.MaxMediaBytes)
	if err != nil {
		t.Fatalf("new post media stager: %v", err)
	}
	posts := service.NewPostService(sqlite.NewTransactionManager(db), fixedPostClock{now: now}, stager)

	valid := strings.Repeat("🙂", service.MaxPostTextRunes)
	if utf8.RuneCountInString(valid) != service.MaxPostTextRunes {
		t.Fatal("test input does not contain the expected rune count")
	}
	created, err := posts.Create(ctx, authorID, service.CreatePostInput{Text: " \n" + valid + "\t ", Privacy: domain.PostPublic})
	if err != nil {
		t.Fatalf("create max-length Unicode post: %v", err)
	}
	if created.Text != valid {
		t.Fatal("post text was not trimmed before storage")
	}

	tooLong := valid + "a"
	if post, err := posts.Create(ctx, authorID, service.CreatePostInput{Text: tooLong, Privacy: domain.PostPublic}); !errors.Is(err, service.ErrInvalidInput) || post != nil {
		t.Fatalf("expected 5001-code-point post rejection, post=%+v err=%v", post, err)
	}
	if post, err := posts.Create(ctx, authorID, service.CreatePostInput{Text: string([]byte{0xff}), Privacy: domain.PostPublic}); !errors.Is(err, service.ErrInvalidInput) || post != nil {
		t.Fatalf("expected invalid UTF-8 rejection, post=%+v err=%v", post, err)
	}
}

func TestCreateGroupPostRollbackRemovesFinalFileAndRows(t *testing.T) {
	root := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(root, "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	now := time.Date(2026, time.July, 22, 10, 0, 0, 0, time.UTC)
	users := sqlite.NewUserRepo(db)
	ownerID := createPostTestUser(t, ctx, users, "group-post-rollback@example.com", now)
	transactions := sqlite.NewTransactionManager(db)
	group, err := service.NewGroupService(transactions, fixedPostClock{now: now}).Create(ctx, ownerID, "Rollback group", "Description")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	uploadDir := filepath.Join(root, "uploads")
	stager, err := service.NewMediaStager(&authTestIDGenerator{}, uploadDir, service.MaxMediaBytes)
	if err != nil {
		t.Fatalf("new post media stager: %v", err)
	}
	posts := service.NewPostService(
		failAfterCallbackTransactions{delegate: transactions}, fixedPostClock{now: now}, stager,
	)
	png := []byte("\x89PNG\r\n\x1a\ngroup-rollback-image")
	post, err := posts.CreateGroupPost(ctx, ownerID, group.ID, service.CreateGroupPostInput{
		Text:  "rollback group post",
		Media: &service.MediaUpload{OriginalName: "group.png", Reader: bytes.NewReader(png)},
	})
	if !errors.Is(err, errForcedCommitFailure) || post != nil {
		t.Fatalf("expected forced transaction failure, post=%+v err=%v", post, err)
	}
	for _, table := range []string{"posts", "post_selected_users", "post_comments", "media"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("rollback left %d rows in %s", count, table)
		}
	}
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("read uploads: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("rollback left files: %+v", files)
	}
}

type fixedPostClock struct {
	now time.Time
}

func (c fixedPostClock) Now() time.Time { return c.now }

func createPostTestUser(t *testing.T, ctx context.Context, users *sqlite.UserRepo, email string, now time.Time) int64 {
	t.Helper()
	id, err := users.Create(ctx, &domain.User{
		Email:        email,
		PasswordHash: "hash",
		FirstName:    "Post",
		LastName:     "User",
		DateOfBirth:  "18-07-1992",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return id
}
