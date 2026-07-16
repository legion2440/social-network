package sqlite

import (
	"context"
	"database/sql"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

type SessionRepo struct {
	db sqlExecutor
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Create(ctx context.Context, session *domain.Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, session.Token, session.UserID, timeToUnix(session.ExpiresAt), timeToUnix(session.CreatedAt))
	return err
}

func (r *SessionRepo) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT token, user_id, expires_at, created_at
		FROM sessions
		WHERE token = ?
	`, token)

	var session domain.Session
	var expiresAt, createdAt int64
	if err := row.Scan(&session.Token, &session.UserID, &expiresAt, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	session.ExpiresAt = unixToTime(expiresAt)
	session.CreatedAt = unixToTime(createdAt)
	return &session, nil
}

func (r *SessionRepo) DeleteByToken(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	return err
}
