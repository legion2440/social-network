package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	githubsqlite "github.com/mattn/go-sqlite3"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

type AuthIdentityRepo struct {
	db sqlExecutor
}

func NewAuthIdentityRepo(db *sql.DB) *AuthIdentityRepo {
	return &AuthIdentityRepo{db: db}
}

func (r *AuthIdentityRepo) Create(ctx context.Context, identity *domain.AuthIdentity) (int64, error) {
	if r == nil || r.db == nil || identity == nil {
		return 0, errors.New("auth identity repository is not configured")
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO auth_identities (
			user_id, provider, provider_user_id, provider_email,
			provider_email_verified, provider_username, provider_display_name,
			linked_at, last_login_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		identity.UserID,
		strings.TrimSpace(identity.Provider),
		strings.TrimSpace(identity.ProviderUserID),
		strings.TrimSpace(identity.ProviderEmail),
		boolToInt(identity.ProviderEmailVerified),
		strings.TrimSpace(identity.ProviderUsername),
		strings.TrimSpace(identity.ProviderDisplayName),
		timeToUnix(identity.LinkedAt),
		timeToUnix(identity.LastLoginAt),
	)
	if err != nil {
		return 0, mapAuthIdentityError(err)
	}
	return result.LastInsertId()
}

func (r *AuthIdentityRepo) GetByProviderUserID(
	ctx context.Context,
	provider string,
	providerUserID string,
) (*domain.AuthIdentity, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("auth identity repository is not configured")
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, user_id, provider, provider_user_id, provider_email,
			provider_email_verified, provider_username, provider_display_name,
			linked_at, last_login_at
		FROM auth_identities
		WHERE provider = ? AND provider_user_id = ?
		LIMIT 1
	`, strings.TrimSpace(provider), strings.TrimSpace(providerUserID))
	return scanAuthIdentity(row)
}

func (r *AuthIdentityRepo) UpdateMetadata(ctx context.Context, identity *domain.AuthIdentity) error {
	if r == nil || r.db == nil || identity == nil || identity.ID <= 0 || identity.UserID <= 0 {
		return errors.New("invalid auth identity update")
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE auth_identities
		SET
			provider_email = ?,
			provider_email_verified = ?,
			provider_username = ?,
			provider_display_name = ?,
			last_login_at = ?
		WHERE id = ? AND user_id = ?
	`,
		strings.TrimSpace(identity.ProviderEmail),
		boolToInt(identity.ProviderEmailVerified),
		strings.TrimSpace(identity.ProviderUsername),
		strings.TrimSpace(identity.ProviderDisplayName),
		timeToUnix(identity.LastLoginAt),
		identity.ID,
		identity.UserID,
	)
	if err != nil {
		return mapAuthIdentityError(err)
	}
	return requireOneRow(result)
}

func scanAuthIdentity(row *sql.Row) (*domain.AuthIdentity, error) {
	identity := &domain.AuthIdentity{}
	var verified int
	var linkedAt int64
	var lastLoginAt int64
	err := row.Scan(
		&identity.ID,
		&identity.UserID,
		&identity.Provider,
		&identity.ProviderUserID,
		&identity.ProviderEmail,
		&verified,
		&identity.ProviderUsername,
		&identity.ProviderDisplayName,
		&linkedAt,
		&lastLoginAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	identity.ProviderEmailVerified = verified != 0
	identity.LinkedAt = unixToTime(linkedAt)
	identity.LastLoginAt = unixToTime(lastLoginAt)
	return identity, nil
}

func mapAuthIdentityError(err error) error {
	var sqliteErr githubsqlite.Error
	if errors.As(err, &sqliteErr) &&
		(sqliteErr.ExtendedCode == githubsqlite.ErrConstraintUnique ||
			sqliteErr.ExtendedCode == githubsqlite.ErrConstraintForeignKey) {
		return fmt.Errorf("%w: auth identity", repo.ErrConflict)
	}
	return err
}
