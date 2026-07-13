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
		INSERT INTO users (
			email,
			password_hash,
			first_name,
			last_name,
			date_of_birth,
			gender,
			nickname,
			about_me,
			avatar_media_id,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		strings.TrimSpace(user.Email),
		user.PasswordHash,
		strings.TrimSpace(user.FirstName),
		strings.TrimSpace(user.LastName),
		timeToUnix(user.DateOfBirth),
		nullableGender(user.Gender),
		nullableTrimmedText(user.Nickname),
		nullableTrimmedText(user.AboutMe),
		nullableInt64(user.AvatarMediaID),
		timeToUnix(user.CreatedAt),
		timeToUnix(user.UpdatedAt),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			email,
			password_hash,
			first_name,
			last_name,
			date_of_birth,
			gender,
			nickname,
			about_me,
			avatar_media_id,
			created_at,
			updated_at
		FROM users
		WHERE id = ?
	`, id)
	return scanUser(row)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			email,
			password_hash,
			first_name,
			last_name,
			date_of_birth,
			gender,
			nickname,
			about_me,
			avatar_media_id,
			created_at,
			updated_at
		FROM users
		WHERE email = ? COLLATE NOCASE
		LIMIT 1
	`, strings.TrimSpace(email))
	return scanUser(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*domain.User, error) {
	var (
		user                              domain.User
		dateOfBirth, createdAt, updatedAt int64
		gender, nickname, aboutMe         sql.NullString
		avatarMediaID                     sql.NullInt64
	)
	if err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&dateOfBirth,
		&gender,
		&nickname,
		&aboutMe,
		&avatarMediaID,
		&createdAt,
		&updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}

	user.FirstName = strings.TrimSpace(user.FirstName)
	user.LastName = strings.TrimSpace(user.LastName)
	user.DateOfBirth = unixToTime(dateOfBirth)
	user.Gender = genderFromNullString(gender)
	user.Nickname = stringFromNullString(nickname)
	user.AboutMe = stringFromNullString(aboutMe)
	if avatarMediaID.Valid {
		value := avatarMediaID.Int64
		user.AvatarMediaID = &value
	}
	user.CreatedAt = unixToTime(createdAt)
	user.UpdatedAt = unixToTime(updatedAt)
	return &user, nil
}

func nullableGender(gender *domain.Gender) any {
	if gender == nil || !gender.Valid() {
		return nil
	}
	return string(*gender)
}

func nullableTrimmedText(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func genderFromNullString(value sql.NullString) *domain.Gender {
	if !value.Valid {
		return nil
	}
	gender := domain.Gender(strings.TrimSpace(value.String))
	if !gender.Valid() {
		return nil
	}
	return &gender
}

func stringFromNullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
