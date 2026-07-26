package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

func TestAuthIdentityRepositoryLifecycleAndUniqueness(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "oauth-identities.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	seedOAuthUser(t, db, 1, "oauth-user@example.com")

	repository := NewAuthIdentityRepo(db)
	now := time.Unix(1_700_000_000, 0).UTC()
	identity := &domain.AuthIdentity{
		UserID:                1,
		Provider:              "github",
		ProviderUserID:        "123",
		ProviderEmail:         "verified@example.com",
		ProviderEmailVerified: true,
		ProviderUsername:      "octocat",
		ProviderDisplayName:   "The Octocat",
		LinkedAt:              now,
		LastLoginAt:           now,
	}
	identityID, err := repository.Create(ctx, identity)
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	identity.ID = identityID

	stored, err := repository.GetByProviderUserID(ctx, "github", "123")
	if err != nil {
		t.Fatalf("get identity: %v", err)
	}
	if stored.UserID != 1 || stored.ProviderEmail != "verified@example.com" || !stored.ProviderEmailVerified {
		t.Fatalf("unexpected stored identity: %+v", stored)
	}

	stored.ProviderEmail = "new@example.com"
	stored.ProviderUsername = "new-login"
	stored.ProviderDisplayName = "New Name"
	stored.LastLoginAt = now.Add(time.Hour)
	if err := repository.UpdateMetadata(ctx, stored); err != nil {
		t.Fatalf("update identity: %v", err)
	}
	updated, err := repository.GetByProviderUserID(ctx, "github", "123")
	if err != nil {
		t.Fatalf("get updated identity: %v", err)
	}
	if updated.ProviderEmail != "new@example.com" ||
		updated.ProviderUsername != "new-login" ||
		!updated.LastLoginAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("unexpected updated identity: %+v", updated)
	}

	if _, err := repository.Create(ctx, identity); !errors.Is(err, repo.ErrConflict) {
		t.Fatalf("expected duplicate identity conflict, got %v", err)
	}
}

func TestAuthFlowTakeIsOneTimeAndRollbackSafe(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "oauth-flows.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	repository := NewAuthFlowRepo(db)
	now := time.Unix(1_700_000_000, 0).UTC()
	flow := &domain.AuthFlow{
		Token:     "flow-token",
		Kind:      "oauth_registration",
		Provider:  "github",
		Payload:   `{"email":"verified@example.com"}`,
		CreatedAt: now,
		ExpiresAt: now.Add(30 * time.Minute),
	}
	if err := repository.Create(ctx, flow); err != nil {
		t.Fatalf("create flow: %v", err)
	}

	rollbackErr := errors.New("force rollback")
	transactions := NewTransactionManager(db)
	err = transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		taken, err := repositories.AuthFlows().TakeByToken(ctx, flow.Token)
		if err != nil {
			return err
		}
		if taken.Provider != "github" {
			t.Fatalf("unexpected flow: %+v", taken)
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error, got %v", err)
	}
	if _, err := repository.GetByToken(ctx, flow.Token); err != nil {
		t.Fatalf("flow must survive rollback: %v", err)
	}

	if err := transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		_, err := repositories.AuthFlows().TakeByToken(ctx, flow.Token)
		return err
	}); err != nil {
		t.Fatalf("take flow: %v", err)
	}
	if _, err := repository.GetByToken(ctx, flow.Token); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected consumed flow to be absent, got %v", err)
	}
}

func TestAuthFlowDeleteExpiredKeepsActiveFlows(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "oauth-flow-cleanup.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	repository := NewAuthFlowRepo(db)
	now := time.Unix(1_700_000_000, 0).UTC()
	for _, flow := range []*domain.AuthFlow{
		{
			Token: "expired", Kind: "oauth_state", Provider: "github", Payload: "{}",
			CreatedAt: now.Add(-time.Hour), ExpiresAt: now,
		},
		{
			Token: "active", Kind: "oauth_registration", Provider: "github", Payload: "{}",
			CreatedAt: now, ExpiresAt: now.Add(time.Second),
		},
	} {
		if err := repository.Create(ctx, flow); err != nil {
			t.Fatalf("create flow %q: %v", flow.Token, err)
		}
	}

	deleted, err := repository.DeleteExpired(ctx, now)
	if err != nil || deleted != 1 {
		t.Fatalf("delete expired flows: deleted=%d err=%v", deleted, err)
	}
	if _, err := repository.GetByToken(ctx, "expired"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expired flow still exists: %v", err)
	}
	if _, err := repository.GetByToken(ctx, "active"); err != nil {
		t.Fatalf("active flow was removed: %v", err)
	}
}

func seedOAuthUser(t *testing.T, db *sql.DB, id int64, email string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO users (
			id, email, password_hash, first_name, last_name, date_of_birth,
			is_private, created_at, updated_at
		)
		VALUES (?, ?, 'hash', 'OAuth', 'User', '01-01-1990', 0, 1, 1)
	`, id, email)
	if err != nil {
		t.Fatalf("seed OAuth user: %v", err)
	}
}
