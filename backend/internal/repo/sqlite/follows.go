package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

type FollowRepo struct {
	db sqlExecutor
}

func NewFollowRepo(db *sql.DB) *FollowRepo {
	return &FollowRepo{db: db}
}

func (r *FollowRepo) Upsert(
	ctx context.Context,
	followerUserID, followedUserID int64,
	desiredStatus domain.FollowStatus,
	now time.Time,
) (*domain.Follow, error) {
	if followerUserID <= 0 || followedUserID <= 0 || followerUserID == followedUserID || !desiredStatus.Valid() {
		return nil, fmt.Errorf("invalid follow relation")
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO follows (
			follower_user_id,
			followed_user_id,
			status,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(follower_user_id, followed_user_id) DO UPDATE SET
			status = CASE
				WHEN follows.status = 'pending' AND excluded.status = 'accepted' THEN 'accepted'
				ELSE follows.status
			END,
			updated_at = CASE
				WHEN follows.status = 'pending' AND excluded.status = 'accepted' THEN excluded.updated_at
				ELSE follows.updated_at
			END
		RETURNING id, follower_user_id, followed_user_id, status, created_at, updated_at
	`, followerUserID, followedUserID, desiredStatus, timeToUnix(now), timeToUnix(now))
	return scanFollow(row)
}

func (r *FollowRepo) Get(ctx context.Context, followerUserID, followedUserID int64) (*domain.Follow, error) {
	if followerUserID <= 0 || followedUserID <= 0 {
		return nil, repo.ErrNotFound
	}
	return scanFollow(r.db.QueryRowContext(ctx, `
		SELECT id, follower_user_id, followed_user_id, status, created_at, updated_at
		FROM follows
		WHERE follower_user_id = ? AND followed_user_id = ?
	`, followerUserID, followedUserID))
}

func (r *FollowRepo) GetByID(ctx context.Context, id int64) (*domain.Follow, error) {
	if id <= 0 {
		return nil, repo.ErrNotFound
	}
	return scanFollow(r.db.QueryRowContext(ctx, `
		SELECT id, follower_user_id, followed_user_id, status, created_at, updated_at
		FROM follows
		WHERE id = ?
	`, id))
}

func (r *FollowRepo) Accept(ctx context.Context, id, followedUserID int64, now time.Time) (*domain.Follow, error) {
	if id <= 0 || followedUserID <= 0 {
		return nil, repo.ErrNotFound
	}
	return scanFollow(r.db.QueryRowContext(ctx, `
		UPDATE follows
		SET
			status = 'accepted',
			updated_at = CASE WHEN status = 'pending' THEN ? ELSE updated_at END
		WHERE id = ? AND followed_user_id = ?
		RETURNING id, follower_user_id, followed_user_id, status, created_at, updated_at
	`, timeToUnix(now), id, followedUserID))
}

func (r *FollowRepo) Delete(ctx context.Context, followerUserID, followedUserID int64) error {
	if followerUserID <= 0 || followedUserID <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM follows
		WHERE follower_user_id = ? AND followed_user_id = ?
	`, followerUserID, followedUserID)
	return err
}

func (r *FollowRepo) Reject(ctx context.Context, id, followedUserID int64) error {
	if id <= 0 || followedUserID <= 0 {
		return repo.ErrNotFound
	}
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM follows
		WHERE id = ? AND followed_user_id = ? AND status = 'pending'
	`, id, followedUserID)
	if err != nil {
		return err
	}
	return requireOneRow(result)
}

func (r *FollowRepo) ListFollowers(ctx context.Context, userID int64) ([]*domain.User, error) {
	return r.listUsers(ctx, `
		SELECT
			u.id, u.email, u.password_hash, u.first_name, u.last_name,
			u.date_of_birth, u.gender, u.nickname, u.about_me,
			u.avatar_media_id, u.is_private, u.created_at, u.updated_at
		FROM follows f
		JOIN users u ON u.id = f.follower_user_id
		WHERE f.followed_user_id = ? AND f.status = 'accepted'
		ORDER BY f.updated_at DESC, f.id DESC
	`, userID)
}

func (r *FollowRepo) ListFollowing(ctx context.Context, userID int64) ([]*domain.User, error) {
	return r.listUsers(ctx, `
		SELECT
			u.id, u.email, u.password_hash, u.first_name, u.last_name,
			u.date_of_birth, u.gender, u.nickname, u.about_me,
			u.avatar_media_id, u.is_private, u.created_at, u.updated_at
		FROM follows f
		JOIN users u ON u.id = f.followed_user_id
		WHERE f.follower_user_id = ? AND f.status = 'accepted'
		ORDER BY f.updated_at DESC, f.id DESC
	`, userID)
}

func (r *FollowRepo) ListPendingRequests(ctx context.Context, followedUserID int64) ([]*domain.FollowRequest, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			f.id, f.follower_user_id, f.followed_user_id, f.status, f.created_at, f.updated_at,
			u.id, u.email, u.password_hash, u.first_name, u.last_name,
			u.date_of_birth, u.gender, u.nickname, u.about_me,
			u.avatar_media_id, u.is_private, u.created_at, u.updated_at
		FROM follows f
		JOIN users u ON u.id = f.follower_user_id
		WHERE f.followed_user_id = ? AND f.status = 'pending'
		ORDER BY f.created_at ASC, f.id ASC
	`, followedUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]*domain.FollowRequest, 0)
	for rows.Next() {
		follow, user, err := scanFollowRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, &domain.FollowRequest{Follow: follow, User: user})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

func (r *FollowRepo) IsAccepted(ctx context.Context, followerUserID, followedUserID int64) (bool, error) {
	if followerUserID <= 0 || followedUserID <= 0 {
		return false, nil
	}
	var accepted bool
	if err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM follows
			WHERE follower_user_id = ? AND followed_user_id = ? AND status = 'accepted'
		)
	`, followerUserID, followedUserID).Scan(&accepted); err != nil {
		return false, err
	}
	return accepted, nil
}

func (r *FollowRepo) listUsers(ctx context.Context, query string, userID int64) ([]*domain.User, error) {
	if userID <= 0 {
		return []*domain.User{}, nil
	}
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func scanFollow(row rowScanner) (*domain.Follow, error) {
	var (
		follow               domain.Follow
		status               string
		createdAt, updatedAt int64
	)
	if err := row.Scan(
		&follow.ID,
		&follow.FollowerUserID,
		&follow.FollowedUserID,
		&status,
		&createdAt,
		&updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	follow.Status = domain.FollowStatus(status)
	if !follow.Status.Valid() {
		return nil, fmt.Errorf("invalid follow status: %q", status)
	}
	follow.CreatedAt = unixToTime(createdAt)
	follow.UpdatedAt = unixToTime(updatedAt)
	return &follow, nil
}

func scanFollowRequest(row rowScanner) (*domain.Follow, *domain.User, error) {
	var (
		follow                           domain.Follow
		user                             domain.User
		status                           string
		followCreatedAt, followUpdatedAt int64
		userCreatedAt, userUpdatedAt     int64
		gender, nickname, aboutMe        sql.NullString
		avatarMediaID                    sql.NullInt64
		isPrivate                        int
	)
	if err := row.Scan(
		&follow.ID,
		&follow.FollowerUserID,
		&follow.FollowedUserID,
		&status,
		&followCreatedAt,
		&followUpdatedAt,
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
		&userCreatedAt,
		&userUpdatedAt,
	); err != nil {
		return nil, nil, err
	}

	follow.Status = domain.FollowStatus(status)
	if !follow.Status.Valid() {
		return nil, nil, fmt.Errorf("invalid follow status: %q", status)
	}
	follow.CreatedAt = unixToTime(followCreatedAt)
	follow.UpdatedAt = unixToTime(followUpdatedAt)

	user.FirstName = strings.TrimSpace(user.FirstName)
	user.LastName = strings.TrimSpace(user.LastName)
	if !domain.ValidDateOfBirth(user.DateOfBirth) {
		return nil, nil, fmt.Errorf("%w: %q", domain.ErrInvalidDateOfBirth, user.DateOfBirth)
	}
	parsedGender, err := genderFromNullString(gender)
	if err != nil {
		return nil, nil, err
	}
	user.Gender = parsedGender
	user.Nickname = stringFromNullString(nickname)
	user.AboutMe = stringFromNullString(aboutMe)
	if avatarMediaID.Valid {
		value := avatarMediaID.Int64
		user.AvatarMediaID = &value
	}
	if isPrivate != 0 && isPrivate != 1 {
		return nil, nil, fmt.Errorf("invalid is_private value: %d", isPrivate)
	}
	user.IsPrivate = isPrivate == 1
	user.CreatedAt = unixToTime(userCreatedAt)
	user.UpdatedAt = unixToTime(userUpdatedAt)
	return &follow, &user, nil
}
