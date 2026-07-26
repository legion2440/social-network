package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	githubsqlite "github.com/mattn/go-sqlite3"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

type AuthFlowRepo struct {
	db sqlExecutor
}

func NewAuthFlowRepo(db *sql.DB) *AuthFlowRepo {
	return &AuthFlowRepo{db: db}
}

func (r *AuthFlowRepo) Create(ctx context.Context, flow *domain.AuthFlow) error {
	if r == nil || r.db == nil || flow == nil {
		return errors.New("auth flow repository is not configured")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO auth_flows (token, kind, provider, payload, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		strings.TrimSpace(flow.Token),
		strings.TrimSpace(flow.Kind),
		strings.TrimSpace(flow.Provider),
		flow.Payload,
		timeToUnix(flow.CreatedAt),
		timeToUnix(flow.ExpiresAt),
	)
	if err != nil {
		var sqliteErr githubsqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == githubsqlite.ErrConstraintPrimaryKey {
			return fmt.Errorf("%w: auth flow", repo.ErrConflict)
		}
	}
	return err
}

func (r *AuthFlowRepo) GetByToken(ctx context.Context, token string) (*domain.AuthFlow, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("auth flow repository is not configured")
	}
	return scanAuthFlow(r.db.QueryRowContext(ctx, `
		SELECT token, kind, provider, payload, created_at, expires_at
		FROM auth_flows
		WHERE token = ?
		LIMIT 1
	`, strings.TrimSpace(token)))
}

func (r *AuthFlowRepo) TakeByToken(ctx context.Context, token string) (*domain.AuthFlow, error) {
	flow, err := r.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM auth_flows WHERE token = ?`, flow.Token)
	if err != nil {
		return nil, err
	}
	if err := requireOneRow(result); err != nil {
		return nil, err
	}
	return flow, nil
}

func (r *AuthFlowRepo) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("auth flow repository is not configured")
	}
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM auth_flows
		WHERE expires_at <= ?
	`, timeToUnix(before))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanAuthFlow(row *sql.Row) (*domain.AuthFlow, error) {
	flow := &domain.AuthFlow{}
	var createdAt int64
	var expiresAt int64
	err := row.Scan(
		&flow.Token,
		&flow.Kind,
		&flow.Provider,
		&flow.Payload,
		&createdAt,
		&expiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	flow.CreatedAt = unixToTime(createdAt)
	flow.ExpiresAt = unixToTime(expiresAt)
	return flow, nil
}
