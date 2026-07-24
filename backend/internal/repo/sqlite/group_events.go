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
)

type GroupEventRepo struct {
	db sqlExecutor
}

func (r *GroupEventRepo) Create(ctx context.Context, event *domain.GroupEvent) (int64, error) {
	if event == nil || event.GroupID <= 0 || event.CreatorUserID <= 0 || strings.TrimSpace(event.Title) == "" ||
		strings.TrimSpace(event.Description) == "" || event.StartsAt.IsZero() || event.CreatedAt.IsZero() {
		return 0, fmt.Errorf("invalid group event")
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO group_events (group_id, creator_user_id, title, description, starts_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.GroupID, event.CreatorUserID, strings.TrimSpace(event.Title), strings.TrimSpace(event.Description),
		timeToUnix(event.StartsAt), timeToUnix(event.CreatedAt))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *GroupEventRepo) Get(ctx context.Context, viewerUserID, eventID int64) (*domain.GroupEvent, error) {
	if viewerUserID <= 0 || eventID <= 0 {
		return nil, repo.ErrNotFound
	}
	return scanGroupEvent(r.db.QueryRowContext(ctx, groupEventSelect+`
		WHERE ge.id = ?
	`, viewerUserID, eventID))
}

func (r *GroupEventRepo) List(
	ctx context.Context,
	viewerUserID, groupID int64,
	cursor *domain.GroupEventCursor,
	limit int,
) ([]*domain.GroupEvent, error) {
	if viewerUserID <= 0 || groupID <= 0 || limit <= 0 {
		return []*domain.GroupEvent{}, nil
	}
	query := groupEventSelect + ` WHERE ge.group_id = ?`
	args := []any{viewerUserID, groupID}
	if cursor != nil {
		startsAt := timeToUnix(cursor.StartsAt)
		query += ` AND (ge.starts_at > ? OR (ge.starts_at = ? AND ge.id > ?))`
		args = append(args, startsAt, startsAt, cursor.ID)
	}
	query += ` ORDER BY ge.starts_at ASC, ge.id ASC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]*domain.GroupEvent, 0)
	for rows.Next() {
		event, err := scanGroupEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *GroupEventRepo) UpsertResponse(
	ctx context.Context,
	eventID, userID int64,
	response domain.GroupEventResponse,
	now time.Time,
) error {
	if eventID <= 0 || userID <= 0 || !response.Valid() || now.IsZero() {
		return fmt.Errorf("invalid group event response")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO group_event_responses (event_id, user_id, response, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(event_id, user_id) DO UPDATE SET
			response = excluded.response,
			updated_at = excluded.updated_at
	`, eventID, userID, response, timeToUnix(now), timeToUnix(now))
	return err
}

const groupEventSelect = `
	SELECT
		ge.id, ge.group_id, ge.creator_user_id, ge.title, ge.description, ge.starts_at, ge.created_at,
		(
			SELECT COUNT(*)
			FROM group_event_responses counted
			JOIN group_memberships active
				ON active.group_id = ge.group_id
				AND active.user_id = counted.user_id
				AND active.status IN ('owner', 'member')
			WHERE counted.event_id = ge.id AND counted.response = 'going'
		),
		(
			SELECT COUNT(*)
			FROM group_event_responses counted
			JOIN group_memberships active
				ON active.group_id = ge.group_id
				AND active.user_id = counted.user_id
				AND active.status IN ('owner', 'member')
			WHERE counted.event_id = ge.id AND counted.response = 'not_going'
		),
		viewer_response.response,
		creator.id, creator.email, creator.password_hash, creator.first_name, creator.last_name,
		creator.date_of_birth, creator.gender, creator.nickname, creator.about_me,
		creator.avatar_media_id,
		creator.is_private, creator.created_at, creator.updated_at
	FROM group_events ge
	JOIN users creator ON creator.id = ge.creator_user_id
	LEFT JOIN group_event_responses viewer_response
		ON viewer_response.event_id = ge.id AND viewer_response.user_id = ?
`

type scannedGroupEvent struct {
	event               domain.GroupEvent
	startsAt, createdAt int64
	viewerResponse      sql.NullString
	creator             scannedGroupUser
}

func (s *scannedGroupEvent) destinations() []any {
	return append([]any{
		&s.event.ID, &s.event.GroupID, &s.event.CreatorUserID,
		&s.event.Title, &s.event.Description, &s.startsAt, &s.createdAt,
		&s.event.GoingCount, &s.event.NotGoingCount, &s.viewerResponse,
	}, s.creator.destinations()...)
}

func (s *scannedGroupEvent) value() (*domain.GroupEvent, error) {
	creator, err := s.creator.value()
	if err != nil {
		return nil, err
	}
	s.event.Title = strings.TrimSpace(s.event.Title)
	s.event.Description = strings.TrimSpace(s.event.Description)
	s.event.StartsAt = unixToTime(s.startsAt)
	s.event.CreatedAt = unixToTime(s.createdAt)
	s.event.Creator = creator
	if s.viewerResponse.Valid {
		response := domain.GroupEventResponse(s.viewerResponse.String)
		if !response.Valid() {
			return nil, fmt.Errorf("invalid group event response: %q", s.viewerResponse.String)
		}
		s.event.ViewerResponse = &response
	}
	return &s.event, nil
}

func scanGroupEvent(row rowScanner) (*domain.GroupEvent, error) {
	var value scannedGroupEvent
	if err := row.Scan(value.destinations()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	return value.value()
}
