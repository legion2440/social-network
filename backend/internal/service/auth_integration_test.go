package service_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/repo"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

var errForcedCommitFailure = errors.New("forced commit failure")

type authTestIDGenerator struct {
	mu sync.Mutex
	n  int
}

func (g *authTestIDGenerator) New() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return fmt.Sprintf("auth-test-%d", g.n), nil
}

type authTestHasher struct{}

func (authTestHasher) Hash(password string) (string, error) {
	return "hash:" + password, nil
}

func (authTestHasher) Compare(hash, password string) error {
	if hash != "hash:"+password {
		return errors.New("password mismatch")
	}
	return nil
}

type failAfterCallbackTransactions struct {
	delegate repo.TransactionManager
}

func (m failAfterCallbackTransactions) WithinTransaction(ctx context.Context, fn func(repo.TransactionRepositories) error) error {
	return m.delegate.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if err := fn(repositories); err != nil {
			return err
		}
		return errForcedCommitFailure
	})
}

func TestRegisterRemovesFinalAvatarWhenTransactionFailsAfterMove(t *testing.T) {
	root := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(root, "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ids := &authTestIDGenerator{}
	appClock := clock.RealClock{}
	sessions := service.NewSessionService(sqlite.NewSessionRepo(db), appClock, ids, 24*time.Hour)
	uploadDir := filepath.Join(root, "uploads")
	stager, err := service.NewMediaStager(ids, uploadDir, service.MaxAvatarBytes)
	if err != nil {
		t.Fatalf("new media stager: %v", err)
	}
	auth := service.NewAuthService(
		sqlite.NewUserRepo(db),
		failAfterCallbackTransactions{delegate: sqlite.NewTransactionManager(db)},
		sessions,
		authTestHasher{},
		appClock,
		stager,
	)
	png := []byte("\x89PNG\r\n\x1a\nrollback-avatar")
	result, err := auth.Register(context.Background(), service.RegisterInput{
		Email:       "rollback@example.com",
		Password:    "password",
		FirstName:   "Rollback",
		LastName:    "Test",
		DateOfBirth: "14-03-1992",
		Avatar: &service.MediaUpload{
			OriginalName: "avatar.png",
			Reader:       bytes.NewReader(png),
		},
	})
	if !errors.Is(err, errForcedCommitFailure) || result != nil {
		t.Fatalf("expected forced transaction failure, result=%+v err=%v", result, err)
	}
	for _, table := range []string{"users", "media", "sessions"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("transaction left %d rows in %s", count, table)
		}
	}
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("read upload directory: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("failed transaction left avatar files: %+v", files)
	}
}
