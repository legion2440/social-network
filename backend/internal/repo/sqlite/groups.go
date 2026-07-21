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

type GroupRepo struct {
	db sqlExecutor
}

func NewGroupRepo(db *sql.DB) *GroupRepo {
	return &GroupRepo{db: db}
}

func (r *GroupRepo) Create(ctx context.Context, group *domain.Group) (int64, error) {
	if group == nil || group.OwnerUserID <= 0 || strings.TrimSpace(group.Title) == "" || strings.TrimSpace(group.Description) == "" || group.CreatedAt.IsZero() {
		return 0, fmt.Errorf("invalid group")
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO groups (owner_user_id, title, description, created_at)
		VALUES (?, ?, ?, ?)
	`, group.OwnerUserID, strings.TrimSpace(group.Title), strings.TrimSpace(group.Description), timeToUnix(group.CreatedAt))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *GroupRepo) Get(ctx context.Context, groupID, viewerUserID int64) (*domain.Group, error) {
	if groupID <= 0 || viewerUserID <= 0 {
		return nil, repo.ErrNotFound
	}
	return scanGroup(r.db.QueryRowContext(ctx, groupSelect+`
		WHERE g.id = ?
	`, viewerUserID, viewerUserID, viewerUserID, groupID))
}

func (r *GroupRepo) List(ctx context.Context, viewerUserID int64, cursor *domain.GroupCursor, limit int) ([]*domain.Group, error) {
	if viewerUserID <= 0 || limit <= 0 {
		return []*domain.Group{}, nil
	}
	query := groupSelect + ` WHERE 1 = 1`
	args := []any{viewerUserID, viewerUserID, viewerUserID}
	if cursor != nil {
		timestamp := timeToUnix(cursor.CreatedAt)
		query += ` AND (g.created_at < ? OR (g.created_at = ? AND g.id < ?))`
		args = append(args, timestamp, timestamp, cursor.ID)
	}
	query += ` ORDER BY g.created_at DESC, g.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]*domain.Group, 0)
	for rows.Next() {
		group, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *GroupRepo) CreateMembership(ctx context.Context, membership *domain.GroupMembership) error {
	if membership == nil || membership.GroupID <= 0 || membership.UserID <= 0 || !membership.Status.Valid() || membership.CreatedAt.IsZero() || membership.UpdatedAt.IsZero() {
		return fmt.Errorf("invalid group membership")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, membership.GroupID, membership.UserID, membership.Status, timeToUnix(membership.CreatedAt), timeToUnix(membership.UpdatedAt))
	if err != nil {
		var sqliteErr githubsqlite.Error
		if errors.As(err, &sqliteErr) && (sqliteErr.ExtendedCode == githubsqlite.ErrConstraintPrimaryKey || sqliteErr.ExtendedCode == githubsqlite.ErrConstraintUnique) {
			return repo.ErrConflict
		}
	}
	return err
}

func (r *GroupRepo) GetMembershipStatus(ctx context.Context, groupID, userID int64) (*domain.GroupMembershipStatus, error) {
	if groupID <= 0 || userID <= 0 {
		return nil, repo.ErrNotFound
	}
	var raw string
	if err := r.db.QueryRowContext(ctx, `
		SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?
	`, groupID, userID).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	status := domain.GroupMembershipStatus(raw)
	if !status.Valid() {
		return nil, fmt.Errorf("invalid group membership status: %q", raw)
	}
	return &status, nil
}

func (r *GroupRepo) UpdateMembershipStatus(
	ctx context.Context,
	groupID, userID int64,
	expected, next domain.GroupMembershipStatus,
	now time.Time,
) error {
	if groupID <= 0 || userID <= 0 || !expected.Valid() || !next.Valid() || now.IsZero() {
		return fmt.Errorf("invalid group membership transition")
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE group_memberships
		SET status = ?, updated_at = ?
		WHERE group_id = ? AND user_id = ? AND status = ?
	`, next, timeToUnix(now), groupID, userID, expected)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return repo.ErrConflict
	}
	return nil
}

func (r *GroupRepo) DeleteMembership(ctx context.Context, groupID, userID int64, expected domain.GroupMembershipStatus) error {
	if groupID <= 0 || userID <= 0 || !expected.Valid() {
		return fmt.Errorf("invalid group membership deletion")
	}
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM group_memberships
		WHERE group_id = ? AND user_id = ? AND status = ?
	`, groupID, userID, expected)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return repo.ErrConflict
	}
	return nil
}

func (r *GroupRepo) ListMembers(
	ctx context.Context,
	groupID, viewerUserID int64,
	cursor *domain.GroupMemberCursor,
	limit int,
) ([]*domain.GroupMembership, error) {
	if groupID <= 0 || viewerUserID <= 0 || limit <= 0 {
		return []*domain.GroupMembership{}, nil
	}
	query := membershipSelect + `
		WHERE gm.group_id = ? AND gm.status IN ('owner', 'member')
	`
	args := []any{viewerUserID, viewerUserID, groupID}
	if cursor != nil {
		timestamp := timeToUnix(cursor.UpdatedAt)
		query += ` AND (
			(CASE WHEN gm.status = 'owner' THEN 0 ELSE 1 END) > ? OR
			((CASE WHEN gm.status = 'owner' THEN 0 ELSE 1 END) = ? AND gm.updated_at > ?) OR
			((CASE WHEN gm.status = 'owner' THEN 0 ELSE 1 END) = ? AND gm.updated_at = ? AND gm.user_id > ?)
		)`
		args = append(args, cursor.OwnerRank, cursor.OwnerRank, timestamp, cursor.OwnerRank, timestamp, cursor.UserID)
	}
	query += ` ORDER BY CASE WHEN gm.status = 'owner' THEN 0 ELSE 1 END ASC, gm.updated_at ASC, gm.user_id ASC LIMIT ?`
	args = append(args, limit)
	return r.listMembershipRows(ctx, query, args...)
}

func (r *GroupRepo) ListMemberships(
	ctx context.Context,
	groupID, viewerUserID int64,
	status domain.GroupMembershipStatus,
	cursor *domain.GroupMembershipCursor,
	limit int,
) ([]*domain.GroupMembership, error) {
	if groupID <= 0 || viewerUserID <= 0 || !status.Valid() || limit <= 0 {
		return []*domain.GroupMembership{}, nil
	}
	query := membershipSelect + ` WHERE gm.group_id = ? AND gm.status = ?`
	args := []any{viewerUserID, viewerUserID, groupID, status}
	if cursor != nil {
		timestamp := timeToUnix(cursor.CreatedAt)
		query += ` AND (gm.created_at > ? OR (gm.created_at = ? AND gm.user_id > ?))`
		args = append(args, timestamp, timestamp, cursor.UserID)
	}
	query += ` ORDER BY gm.created_at ASC, gm.user_id ASC LIMIT ?`
	args = append(args, limit)
	return r.listMembershipRows(ctx, query, args...)
}

func (r *GroupRepo) listMembershipRows(ctx context.Context, query string, args ...any) ([]*domain.GroupMembership, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	memberships := make([]*domain.GroupMembership, 0)
	for rows.Next() {
		membership, err := scanGroupMembership(rows)
		if err != nil {
			return nil, err
		}
		memberships = append(memberships, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return memberships, nil
}

func (r *GroupRepo) ListInvitationInbox(
	ctx context.Context,
	userID int64,
	cursor *domain.GroupInvitationCursor,
	limit int,
) ([]*domain.GroupInvitation, error) {
	if userID <= 0 || limit <= 0 {
		return []*domain.GroupInvitation{}, nil
	}
	query := groupSelect + `
		WHERE viewer.status = 'invited'
	`
	args := []any{userID, userID, userID}
	if cursor != nil {
		timestamp := timeToUnix(cursor.CreatedAt)
		query += ` AND (viewer.created_at > ? OR (viewer.created_at = ? AND g.id > ?))`
		args = append(args, timestamp, timestamp, cursor.GroupID)
	}
	query += ` ORDER BY viewer.created_at ASC, g.id ASC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	invitations := make([]*domain.GroupInvitation, 0)
	for rows.Next() {
		group, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		if group.ViewerStatus == nil || *group.ViewerStatus != domain.GroupInvited || group.ViewerMembershipCreatedAt == nil {
			return nil, fmt.Errorf("invalid invitation state")
		}
		invitations = append(invitations, &domain.GroupInvitation{Group: group, CreatedAt: *group.ViewerMembershipCreatedAt})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invitations, nil
}

const groupSelect = `
	SELECT
		g.id, g.owner_user_id, g.title, g.description, g.created_at,
		(SELECT COUNT(*) FROM group_memberships counted WHERE counted.group_id = g.id AND counted.status IN ('owner', 'member')),
		viewer.status, viewer.created_at,
		owner.id, owner.email, owner.password_hash, owner.first_name, owner.last_name,
		owner.date_of_birth, owner.gender, owner.nickname, owner.about_me,
		CASE
			WHEN owner.avatar_media_id IS NULL THEN NULL
			WHEN owner.id = ? OR owner.is_private = 0 OR EXISTS (
				SELECT 1 FROM follows avatar_follow
				WHERE avatar_follow.follower_user_id = ?
					AND avatar_follow.followed_user_id = owner.id
					AND avatar_follow.status = 'accepted'
			) THEN owner.avatar_media_id
			ELSE NULL
		END,
		owner.is_private, owner.created_at, owner.updated_at
	FROM groups g
	JOIN users owner ON owner.id = g.owner_user_id
	LEFT JOIN group_memberships viewer ON viewer.group_id = g.id AND viewer.user_id = ?
`

const membershipSelect = `
	SELECT
		gm.group_id, gm.user_id, gm.status, gm.created_at, gm.updated_at,
		u.id, u.email, u.password_hash, u.first_name, u.last_name,
		u.date_of_birth, u.gender, u.nickname, u.about_me,
		CASE
			WHEN u.avatar_media_id IS NULL THEN NULL
			WHEN u.id = ? OR u.is_private = 0 OR EXISTS (
				SELECT 1 FROM follows avatar_follow
				WHERE avatar_follow.follower_user_id = ?
					AND avatar_follow.followed_user_id = u.id
					AND avatar_follow.status = 'accepted'
			) THEN u.avatar_media_id
			ELSE NULL
		END,
		u.is_private, u.created_at, u.updated_at
	FROM group_memberships gm
	JOIN users u ON u.id = gm.user_id
`

type scannedGroup struct {
	group           domain.Group
	groupCreatedAt  int64
	viewerStatus    sql.NullString
	viewerCreatedAt sql.NullInt64
	owner           scannedGroupUser
}

func (s *scannedGroup) destinations() []any {
	return append([]any{
		&s.group.ID,
		&s.group.OwnerUserID,
		&s.group.Title,
		&s.group.Description,
		&s.groupCreatedAt,
		&s.group.MembersCount,
		&s.viewerStatus,
		&s.viewerCreatedAt,
	}, s.owner.destinations()...)
}

func (s *scannedGroup) value() (*domain.Group, error) {
	owner, err := s.owner.value()
	if err != nil {
		return nil, err
	}
	s.group.Title = strings.TrimSpace(s.group.Title)
	s.group.Description = strings.TrimSpace(s.group.Description)
	s.group.CreatedAt = unixToTime(s.groupCreatedAt)
	s.group.Owner = owner
	if s.viewerStatus.Valid {
		status := domain.GroupMembershipStatus(s.viewerStatus.String)
		if !status.Valid() {
			return nil, fmt.Errorf("invalid group membership status: %q", s.viewerStatus.String)
		}
		s.group.ViewerStatus = &status
		if s.viewerCreatedAt.Valid {
			createdAt := unixToTime(s.viewerCreatedAt.Int64)
			s.group.ViewerMembershipCreatedAt = &createdAt
		}
	}
	return &s.group, nil
}

func scanGroup(row rowScanner) (*domain.Group, error) {
	var value scannedGroup
	if err := row.Scan(value.destinations()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	return value.value()
}

type scannedGroupUser struct {
	user                 domain.User
	gender               sql.NullString
	nickname             sql.NullString
	about                sql.NullString
	avatarMediaID        sql.NullInt64
	isPrivate            int
	createdAt, updatedAt int64
}

func (s *scannedGroupUser) destinations() []any {
	return []any{
		&s.user.ID, &s.user.Email, &s.user.PasswordHash, &s.user.FirstName, &s.user.LastName,
		&s.user.DateOfBirth, &s.gender, &s.nickname, &s.about, &s.avatarMediaID,
		&s.isPrivate, &s.createdAt, &s.updatedAt,
	}
}

func (s *scannedGroupUser) value() (*domain.User, error) {
	s.user.FirstName = strings.TrimSpace(s.user.FirstName)
	s.user.LastName = strings.TrimSpace(s.user.LastName)
	if !domain.ValidDateOfBirth(s.user.DateOfBirth) {
		return nil, fmt.Errorf("%w: %q", domain.ErrInvalidDateOfBirth, s.user.DateOfBirth)
	}
	gender, err := genderFromNullString(s.gender)
	if err != nil {
		return nil, err
	}
	s.user.Gender = gender
	s.user.Nickname = stringFromNullString(s.nickname)
	s.user.AboutMe = stringFromNullString(s.about)
	if s.avatarMediaID.Valid {
		value := s.avatarMediaID.Int64
		s.user.AvatarMediaID = &value
	}
	if s.isPrivate != 0 && s.isPrivate != 1 {
		return nil, fmt.Errorf("invalid is_private value: %d", s.isPrivate)
	}
	s.user.IsPrivate = s.isPrivate == 1
	s.user.CreatedAt = unixToTime(s.createdAt)
	s.user.UpdatedAt = unixToTime(s.updatedAt)
	return &s.user, nil
}

func scanGroupMembership(row rowScanner) (*domain.GroupMembership, error) {
	var (
		membership domain.GroupMembership
		status     string
		createdAt  int64
		updatedAt  int64
		user       scannedGroupUser
	)
	destinations := []any{&membership.GroupID, &membership.UserID, &status, &createdAt, &updatedAt}
	destinations = append(destinations, user.destinations()...)
	if err := row.Scan(destinations...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	membership.Status = domain.GroupMembershipStatus(status)
	if !membership.Status.Valid() {
		return nil, fmt.Errorf("invalid group membership status: %q", status)
	}
	membership.CreatedAt = unixToTime(createdAt)
	membership.UpdatedAt = unixToTime(updatedAt)
	var err error
	membership.User, err = user.value()
	return &membership, err
}
