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

type NotificationRepo struct {
	db sqlExecutor
}

func NewNotificationRepo(db *sql.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) EnsureUserState(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid notification user state")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notification_user_states (user_id, revision)
		VALUES (?, 0)
		ON CONFLICT(user_id) DO NOTHING
	`, userID)
	return err
}

func (r *NotificationRepo) Create(ctx context.Context, notification *domain.Notification) (int64, error) {
	if err := validateNotification(notification); err != nil {
		return 0, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO notifications (
			recipient_user_id, actor_user_id, type, follow_id, group_id, event_id,
			membership_id, resolution, resolved_at, read_at, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, notification.RecipientUserID, notification.ActorUserID, notification.Type,
		nullableNotificationInt64(notification.FollowID), nullableNotificationInt64(notification.GroupID), nullableNotificationInt64(notification.EventID),
		nullableNotificationInt64(notification.MembershipID), nullableResolution(notification.Resolution), nullableTime(notification.ResolvedAt),
		nullableTime(notification.ReadAt), timeToUnix(notification.CreatedAt))
	if err != nil {
		var sqliteErr githubsqlite.Error
		if errors.As(err, &sqliteErr) && (sqliteErr.ExtendedCode == githubsqlite.ErrConstraintUnique || sqliteErr.ExtendedCode == githubsqlite.ErrConstraintPrimaryKey) {
			return 0, repo.ErrConflict
		}
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	notification.ID = id
	return id, nil
}

func (r *NotificationRepo) GetForRecipient(ctx context.Context, recipientUserID, notificationID int64) (*domain.Notification, error) {
	if recipientUserID <= 0 || notificationID <= 0 {
		return nil, repo.ErrNotFound
	}
	return scanNotification(r.db.QueryRowContext(ctx, notificationSelect+`
		WHERE n.recipient_user_id = ? AND n.id = ?
	`, recipientUserID, recipientUserID, recipientUserID, notificationID))
}

func (r *NotificationRepo) ListForRecipient(
	ctx context.Context,
	recipientUserID int64,
	cursor *domain.NotificationCursor,
	limit int,
) ([]*domain.Notification, error) {
	if recipientUserID <= 0 || limit <= 0 {
		return []*domain.Notification{}, nil
	}
	query := notificationSelect + ` WHERE n.recipient_user_id = ?`
	args := []any{recipientUserID, recipientUserID, recipientUserID}
	if cursor != nil {
		timestamp := timeToUnix(cursor.CreatedAt)
		query += ` AND (n.created_at < ? OR (n.created_at = ? AND n.id < ?))`
		args = append(args, timestamp, timestamp, cursor.ID)
	}
	query += ` ORDER BY n.created_at DESC, n.id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.Notification, 0)
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepo) FindPendingByFollowID(ctx context.Context, notificationType domain.NotificationType, followID int64) (*domain.Notification, error) {
	if !notificationType.Actionable() || followID <= 0 {
		return nil, repo.ErrNotFound
	}
	var recipientUserID, notificationID int64
	err := r.db.QueryRowContext(ctx, `
		SELECT recipient_user_id, id
		FROM notifications
		WHERE type = ? AND follow_id = ? AND resolution IS NULL
	`, notificationType, followID).Scan(&recipientUserID, &notificationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.GetForRecipient(ctx, recipientUserID, notificationID)
}

func (r *NotificationRepo) FindPendingByMembershipID(ctx context.Context, notificationType domain.NotificationType, membershipID int64) (*domain.Notification, error) {
	if !notificationType.Actionable() || membershipID <= 0 {
		return nil, repo.ErrNotFound
	}
	var recipientUserID, notificationID int64
	err := r.db.QueryRowContext(ctx, `
		SELECT recipient_user_id, id
		FROM notifications
		WHERE type = ? AND membership_id = ? AND resolution IS NULL
	`, notificationType, membershipID).Scan(&recipientUserID, &notificationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.GetForRecipient(ctx, recipientUserID, notificationID)
}

func (r *NotificationRepo) ReferencesByFollowID(ctx context.Context, followID int64) ([]domain.NotificationReference, error) {
	if followID <= 0 {
		return []domain.NotificationReference{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, recipient_user_id
		FROM notifications
		WHERE follow_id = ?
		ORDER BY id ASC
	`, followID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.NotificationReference, 0)
	for rows.Next() {
		var item domain.NotificationReference
		if err := rows.Scan(&item.ID, &item.RecipientUserID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepo) Resolve(ctx context.Context, notificationID int64, resolution domain.NotificationResolution, now time.Time) (bool, error) {
	if notificationID <= 0 || !resolution.Valid() || now.IsZero() {
		return false, fmt.Errorf("invalid notification resolution")
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE notifications
		SET resolution = ?, resolved_at = ?, read_at = COALESCE(read_at, ?)
		WHERE id = ? AND resolution IS NULL
	`, resolution, timeToUnix(now), timeToUnix(now), notificationID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (r *NotificationRepo) MarkRead(ctx context.Context, notificationID, recipientUserID int64, now time.Time) (bool, error) {
	if notificationID <= 0 || recipientUserID <= 0 || now.IsZero() {
		return false, fmt.Errorf("invalid notification read")
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = ?
		WHERE id = ? AND recipient_user_id = ? AND read_at IS NULL
	`, timeToUnix(now), notificationID, recipientUserID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, recipientUserID int64, now time.Time) (int64, error) {
	if recipientUserID <= 0 || now.IsZero() {
		return 0, fmt.Errorf("invalid notification read-all")
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = ?
		WHERE recipient_user_id = ? AND read_at IS NULL
	`, timeToUnix(now), recipientUserID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, recipientUserID int64) (int64, error) {
	if recipientUserID <= 0 {
		return 0, nil
	}
	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE recipient_user_id = ? AND read_at IS NULL
	`, recipientUserID).Scan(&count)
	return count, err
}

func (r *NotificationRepo) CurrentRevision(ctx context.Context, userID int64) (int64, error) {
	if userID <= 0 {
		return 0, nil
	}
	var revision int64
	err := r.db.QueryRowContext(ctx, `
		SELECT revision FROM notification_user_states WHERE user_id = ?
	`, userID).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return revision, err
}

func (r *NotificationRepo) BumpRevision(ctx context.Context, userID int64) (int64, error) {
	if userID <= 0 {
		return 0, fmt.Errorf("invalid notification revision")
	}
	var revision int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO notification_user_states (user_id, revision)
		VALUES (?, 1)
		ON CONFLICT(user_id) DO UPDATE SET revision = notification_user_states.revision + 1
		RETURNING revision
	`, userID).Scan(&revision)
	return revision, err
}

const notificationSelect = `
	SELECT
		n.id, n.recipient_user_id, n.actor_user_id, n.type,
		n.follow_id, n.group_id, g.title, n.event_id, ge.title, ge.starts_at,
		n.membership_id, n.resolution, n.resolved_at, n.read_at, n.created_at,
		actor.id, actor.email, actor.password_hash, actor.first_name, actor.last_name,
		actor.date_of_birth, actor.gender, actor.nickname, actor.about_me,
		CASE
			WHEN actor.avatar_media_id IS NULL THEN NULL
			WHEN actor.id = ? OR actor.is_private = 0 OR EXISTS (
				SELECT 1 FROM follows avatar_follow
				WHERE avatar_follow.follower_user_id = ?
					AND avatar_follow.followed_user_id = actor.id
					AND avatar_follow.status = 'accepted'
			) THEN actor.avatar_media_id
			ELSE NULL
		END,
		actor.is_private, actor.created_at, actor.updated_at
	FROM notifications n
	JOIN users actor ON actor.id = n.actor_user_id
	LEFT JOIN groups g ON g.id = n.group_id
	LEFT JOIN group_events ge ON ge.id = n.event_id
`

type scannedNotification struct {
	notification domain.Notification
	typeValue    string
	followID     sql.NullInt64
	groupID      sql.NullInt64
	groupTitle   sql.NullString
	eventID      sql.NullInt64
	eventTitle   sql.NullString
	eventStarts  sql.NullInt64
	membershipID sql.NullInt64
	resolution   sql.NullString
	resolvedAt   sql.NullInt64
	readAt       sql.NullInt64
	createdAt    int64
	actor        scannedGroupUser
}

func (s *scannedNotification) destinations() []any {
	return append([]any{
		&s.notification.ID, &s.notification.RecipientUserID, &s.notification.ActorUserID, &s.typeValue,
		&s.followID, &s.groupID, &s.groupTitle, &s.eventID, &s.eventTitle, &s.eventStarts,
		&s.membershipID, &s.resolution, &s.resolvedAt, &s.readAt, &s.createdAt,
	}, s.actor.destinations()...)
}

func (s *scannedNotification) value() (*domain.Notification, error) {
	s.notification.Type = domain.NotificationType(s.typeValue)
	if !s.notification.Type.Valid() {
		return nil, fmt.Errorf("invalid notification type: %q", s.typeValue)
	}
	if s.followID.Valid {
		value := s.followID.Int64
		s.notification.FollowID = &value
	}
	if s.groupID.Valid {
		value := s.groupID.Int64
		s.notification.GroupID = &value
	}
	if s.groupTitle.Valid {
		value := strings.TrimSpace(s.groupTitle.String)
		s.notification.GroupTitle = &value
	}
	if s.eventID.Valid {
		value := s.eventID.Int64
		s.notification.EventID = &value
	}
	if s.eventTitle.Valid {
		value := strings.TrimSpace(s.eventTitle.String)
		s.notification.EventTitle = &value
	}
	if s.eventStarts.Valid {
		value := unixToTime(s.eventStarts.Int64)
		s.notification.EventStartsAt = &value
	}
	if s.membershipID.Valid {
		value := s.membershipID.Int64
		s.notification.MembershipID = &value
	}
	if s.resolution.Valid {
		value := domain.NotificationResolution(s.resolution.String)
		if !value.Valid() {
			return nil, fmt.Errorf("invalid notification resolution: %q", s.resolution.String)
		}
		s.notification.Resolution = &value
	}
	if s.resolvedAt.Valid {
		value := unixToTime(s.resolvedAt.Int64)
		s.notification.ResolvedAt = &value
	}
	if s.readAt.Valid {
		value := unixToTime(s.readAt.Int64)
		s.notification.ReadAt = &value
	}
	s.notification.CreatedAt = unixToTime(s.createdAt)
	actor, err := s.actor.value()
	if err != nil {
		return nil, err
	}
	s.notification.Actor = actor
	return &s.notification, nil
}

func scanNotification(row rowScanner) (*domain.Notification, error) {
	var value scannedNotification
	if err := row.Scan(value.destinations()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	return value.value()
}

func validateNotification(notification *domain.Notification) error {
	if notification == nil || notification.RecipientUserID <= 0 || notification.ActorUserID <= 0 ||
		notification.RecipientUserID == notification.ActorUserID || !notification.Type.Valid() || notification.CreatedAt.IsZero() {
		return fmt.Errorf("invalid notification")
	}
	if notification.Resolution != nil && (!notification.Type.Actionable() || !notification.Resolution.Valid() || notification.ResolvedAt == nil) {
		return fmt.Errorf("invalid notification resolution")
	}
	return nil
}

func nullableNotificationInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableResolution(value *domain.NotificationResolution) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return timeToUnix(*value)
}
