package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/repo"
)

const (
	MaxGroupEventTitleRunes       = 100
	MaxGroupEventDescriptionRunes = 2000
	DefaultGroupEventPageLimit    = 20
	MaxGroupEventPageLimit        = 50
	groupEventCursorVersion       = "v1"
)

type CreateGroupEventInput struct {
	Title       string
	Description string
	StartsAt    time.Time
}

type GroupEventPage struct {
	Events     []*domain.GroupEvent
	NextCursor *string
}

type GroupEventService struct {
	transactions repo.TransactionManager
	clock        clock.Clock
}

type GroupEventMutationResult struct {
	Event               *domain.GroupEvent
	NotificationEffects *NotificationEffects
}

func NewGroupEventService(transactions repo.TransactionManager, appClock clock.Clock) *GroupEventService {
	return &GroupEventService{transactions: transactions, clock: appClock}
}

func (s *GroupEventService) Create(
	ctx context.Context,
	creatorUserID, groupID int64,
	input CreateGroupEventInput,
) (*domain.GroupEvent, error) {
	result, err := s.CreateWithEffects(ctx, creatorUserID, groupID, input)
	if err != nil {
		return nil, err
	}
	return result.Event, nil
}

func (s *GroupEventService) CreateWithEffects(
	ctx context.Context,
	creatorUserID, groupID int64,
	input CreateGroupEventInput,
) (*GroupEventMutationResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || creatorUserID <= 0 || groupID <= 0 {
		return nil, ErrInvalidInput
	}
	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)
	startsAt := time.Unix(input.StartsAt.UTC().Unix(), 0).UTC()
	if !validRuneLength(title, 1, MaxGroupEventTitleRunes) ||
		!validRuneLength(description, 1, MaxGroupEventDescriptionRunes) || startsAt.IsZero() {
		return nil, ErrInvalidInput
	}

	result := &GroupEventMutationResult{NotificationEffects: emptyNotificationEffects()}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		builder := newNotificationEffectBuilder()
		if err := requireActiveGroupMember(ctx, repositories, creatorUserID, groupID); err != nil {
			return err
		}
		now := s.clock.Now().UTC()
		if !startsAt.After(now) {
			return ErrInvalidInput
		}
		result.Event = &domain.GroupEvent{
			GroupID: groupID, CreatorUserID: creatorUserID, Title: title,
			Description: description, StartsAt: startsAt, CreatedAt: now,
		}
		id, err := repositories.GroupEvents().Create(ctx, result.Event)
		if err != nil {
			return err
		}
		memberIDs, err := repositories.Groups().ListActiveMemberIDs(ctx, groupID)
		if err != nil {
			return err
		}
		for _, recipientUserID := range memberIDs {
			if recipientUserID == creatorUserID {
				continue
			}
			groupIDValue, eventIDValue := groupID, id
			if err := createNotificationTx(ctx, repositories, builder, &domain.Notification{
				RecipientUserID: recipientUserID, ActorUserID: creatorUserID, Type: domain.NotificationGroupEvent,
				GroupID: &groupIDValue, EventID: &eventIDValue, CreatedAt: now,
			}); err != nil {
				return err
			}
		}
		result.Event, err = repositories.GroupEvents().Get(ctx, creatorUserID, id)
		if err != nil {
			return mapGroupEventRepoError(err)
		}
		result.NotificationEffects, err = builder.finish(ctx, repositories)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *GroupEventService) List(
	ctx context.Context,
	viewerUserID, groupID int64,
	cursor *domain.GroupEventCursor,
	limit int,
) (*GroupEventPage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || groupID <= 0 || !validGroupEventPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var events []*domain.GroupEvent
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if err := requireActiveGroupMember(ctx, repositories, viewerUserID, groupID); err != nil {
			return err
		}
		var err error
		events, err = repositories.GroupEvents().List(ctx, viewerUserID, groupID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildGroupEventPage(events, limit), nil
}

func (s *GroupEventService) Respond(
	ctx context.Context,
	viewerUserID, groupID, eventID int64,
	response domain.GroupEventResponse,
) (*domain.GroupEvent, error) {
	if s == nil || s.transactions == nil || s.clock == nil || viewerUserID <= 0 || groupID <= 0 || eventID <= 0 || !response.Valid() {
		return nil, ErrInvalidInput
	}
	var event *domain.GroupEvent
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if err := requireActiveGroupMember(ctx, repositories, viewerUserID, groupID); err != nil {
			return err
		}
		var err error
		event, err = repositories.GroupEvents().Get(ctx, viewerUserID, eventID)
		if err != nil {
			return mapGroupEventRepoError(err)
		}
		if event.GroupID != groupID {
			return ErrNotFound
		}
		if err := repositories.GroupEvents().UpsertResponse(ctx, eventID, viewerUserID, response, s.clock.Now().UTC()); err != nil {
			return err
		}
		event, err = repositories.GroupEvents().Get(ctx, viewerUserID, eventID)
		return mapGroupEventRepoError(err)
	})
	if err != nil {
		return nil, err
	}
	return event, nil
}

func requireActiveGroupMember(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	viewerUserID, groupID int64,
) error {
	if _, err := repositories.Groups().Get(ctx, groupID, viewerUserID); err != nil {
		return mapGroupEventRepoError(err)
	}
	status, err := repositories.Groups().GetMembershipStatus(ctx, groupID, viewerUserID)
	if errors.Is(err, repo.ErrNotFound) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if status == nil || (*status != domain.GroupOwner && *status != domain.GroupMember) {
		return ErrForbidden
	}
	return nil
}

func mapGroupEventRepoError(err error) error {
	if errors.Is(err, repo.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func validGroupEventPage(cursor *domain.GroupEventCursor, limit int) bool {
	return limit >= 1 && limit <= MaxGroupEventPageLimit &&
		(cursor == nil || !cursor.StartsAt.IsZero() && cursor.ID > 0)
}

func buildGroupEventPage(events []*domain.GroupEvent, limit int) *GroupEventPage {
	page := &GroupEventPage{Events: events}
	if len(events) <= limit {
		return page
	}
	page.Events = events[:limit]
	last := page.Events[len(page.Events)-1]
	cursor := EncodeGroupEventCursor(domain.GroupEventCursor{StartsAt: last.StartsAt, ID: last.ID})
	page.NextCursor = &cursor
	return page
}

func EncodeGroupEventCursor(cursor domain.GroupEventCursor) string {
	payload := groupEventCursorVersion + ":" + strconv.FormatInt(cursor.StartsAt.UTC().Unix(), 10) + ":" + strconv.FormatInt(cursor.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeGroupEventCursor(value string) (*domain.GroupEventCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ErrInvalidInput
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidInput
	}
	parts := strings.Split(string(payload), ":")
	if len(parts) != 3 || parts[0] != groupEventCursorVersion {
		return nil, ErrInvalidInput
	}
	startsAtUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || startsAtUnix <= 0 {
		return nil, ErrInvalidInput
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || id <= 0 {
		return nil, ErrInvalidInput
	}
	return &domain.GroupEventCursor{StartsAt: time.Unix(startsAtUnix, 0).UTC(), ID: id}, nil
}
