package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"

	githubsqlite "github.com/mattn/go-sqlite3"
)

type UserRepo struct {
	db sqlExecutor
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) (int64, error) {
	if !domain.ValidDateOfBirth(user.DateOfBirth) {
		return 0, fmt.Errorf("%w: %q", domain.ErrInvalidDateOfBirth, user.DateOfBirth)
	}
	gender, err := nullableGender(user.Gender)
	if err != nil {
		return 0, err
	}

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
			is_private,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		strings.TrimSpace(user.Email),
		user.PasswordHash,
		strings.TrimSpace(user.FirstName),
		strings.TrimSpace(user.LastName),
		user.DateOfBirth,
		gender,
		nullableTrimmedText(user.Nickname),
		nullableTrimmedText(user.AboutMe),
		nullableInt64(user.AvatarMediaID),
		boolToInt(user.IsPrivate),
		timeToUnix(user.CreatedAt),
		timeToUnix(user.UpdatedAt),
	)
	if err != nil {
		var sqliteErr githubsqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == githubsqlite.ErrConstraintUnique {
			return 0, fmt.Errorf("%w: email", repo.ErrConflict)
		}
		return 0, err
	}
	return result.LastInsertId()
}

func (r *UserRepo) UpdateProfile(ctx context.Context, user *domain.User) error {
	if user == nil || user.ID <= 0 || strings.TrimSpace(user.FirstName) == "" || strings.TrimSpace(user.LastName) == "" {
		return fmt.Errorf("invalid profile update")
	}
	if !domain.ValidDateOfBirth(user.DateOfBirth) {
		return fmt.Errorf("%w: %q", domain.ErrInvalidDateOfBirth, user.DateOfBirth)
	}
	gender, err := nullableGender(user.Gender)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET
			first_name = ?,
			last_name = ?,
			date_of_birth = ?,
			gender = ?,
			nickname = ?,
			about_me = ?,
			is_private = ?,
			updated_at = ?
		WHERE id = ?
	`,
		strings.TrimSpace(user.FirstName),
		strings.TrimSpace(user.LastName),
		user.DateOfBirth,
		gender,
		nullableTrimmedText(user.Nickname),
		nullableTrimmedText(user.AboutMe),
		boolToInt(user.IsPrivate),
		timeToUnix(user.UpdatedAt),
		user.ID,
	)
	if err != nil {
		return err
	}
	return requireOneRow(result)
}

func (r *UserRepo) SetAvatarMediaID(ctx context.Context, userID int64, mediaID *int64, updatedAt time.Time) error {
	if userID <= 0 || mediaID != nil && *mediaID <= 0 {
		return fmt.Errorf("invalid avatar relation")
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET avatar_media_id = ?, updated_at = ?
		WHERE id = ?
	`, nullableInt64(mediaID), timeToUnix(updatedAt), userID)
	if err != nil {
		return err
	}
	return requireOneRow(result)
}

func requireOneRow(result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return repo.ErrNotFound
	}
	return nil
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
			is_private,
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
			is_private,
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
		user                      domain.User
		createdAt, updatedAt      int64
		isPrivate                 int
		gender, nickname, aboutMe sql.NullString
		avatarMediaID             sql.NullInt64
	)
	if err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.DateOfBirth,
		&gender,
		&nickname,
		&aboutMe,
		&avatarMediaID,
		&isPrivate,
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
	if !domain.ValidDateOfBirth(user.DateOfBirth) {
		return nil, fmt.Errorf("%w: %q", domain.ErrInvalidDateOfBirth, user.DateOfBirth)
	}
	parsedGender, err := genderFromNullString(gender)
	if err != nil {
		return nil, err
	}
	user.Gender = parsedGender
	user.Nickname = stringFromNullString(nickname)
	user.AboutMe = stringFromNullString(aboutMe)
	if isPrivate != 0 && isPrivate != 1 {
		return nil, fmt.Errorf("invalid is_private value: %d", isPrivate)
	}
	user.IsPrivate = isPrivate == 1
	if avatarMediaID.Valid {
		value := avatarMediaID.Int64
		user.AvatarMediaID = &value
	}
	user.CreatedAt = unixToTime(createdAt)
	user.UpdatedAt = unixToTime(updatedAt)
	return &user, nil
}

func nullableGender(gender *domain.Gender) (any, error) {
	if gender == nil {
		return nil, nil
	}
	if !gender.Valid() {
		return nil, fmt.Errorf("%w: %q", domain.ErrInvalidGender, *gender)
	}
	return string(*gender), nil
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

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func genderFromNullString(value sql.NullString) (*domain.Gender, error) {
	if !value.Valid {
		return nil, nil
	}
	gender := domain.Gender(value.String)
	if !gender.Valid() {
		return nil, fmt.Errorf("%w: %q", domain.ErrInvalidGender, value.String)
	}
	return &gender, nil
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
