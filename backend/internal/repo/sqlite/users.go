package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, created_at)
		VALUES (?, ?, ?)
	`, strings.TrimSpace(user.Email), user.PasswordHash, timeToUnix(user.CreatedAt))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE id = ?
	`, id)

	var user domain.User
	var createdAt int64
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	user.CreatedAt = unixToTime(createdAt)
	return &user, nil
}
