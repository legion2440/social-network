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
	DefaultNotificationPageLimit = 20
	MaxNotificationPageLimit     = 50
	notificationCursorVersion    = "v1"
)

type NotificationUserState struct {
	Revision    int64
	UnreadCount int64
}

type NotificationEffects struct {
	Upserts      []*domain.Notification
	StatesByUser map[int64]NotificationUserState
}

func emptyNotificationEffects() *NotificationEffects {
	return &NotificationEffects{Upserts: []*domain.Notification{}, StatesByUser: map[int64]NotificationUserState{}}
}

type notificationEffectBuilder struct {
	changedByUser map[int64]map[int64]struct{}
}

func newNotificationEffectBuilder() *notificationEffectBuilder {
	return &notificationEffectBuilder{changedByUser: make(map[int64]map[int64]struct{})}
}

func (b *notificationEffectBuilder) changed(recipientUserID, notificationID int64) {
	if b == nil || recipientUserID <= 0 || notificationID <= 0 {
		return
	}
	if b.changedByUser[recipientUserID] == nil {
		b.changedByUser[recipientUserID] = make(map[int64]struct{})
	}
	b.changedByUser[recipientUserID][notificationID] = struct{}{}
}

func (b *notificationEffectBuilder) finish(
	ctx context.Context,
	repositories repo.TransactionRepositories,
) (*NotificationEffects, error) {
	effects := emptyNotificationEffects()
	if b == nil {
		return effects, nil
	}
	for recipientUserID, notificationIDs := range b.changedByUser {
		revision, err := repositories.Notifications().BumpRevision(ctx, recipientUserID)
		if err != nil {
			return nil, err
		}
		unreadCount, err := repositories.Notifications().UnreadCount(ctx, recipientUserID)
		if err != nil {
			return nil, err
		}
		effects.StatesByUser[recipientUserID] = NotificationUserState{Revision: revision, UnreadCount: unreadCount}
		for notificationID := range notificationIDs {
			notification, err := repositories.Notifications().GetForRecipient(ctx, recipientUserID, notificationID)
			if err != nil {
				return nil, err
			}
			effects.Upserts = append(effects.Upserts, notification)
		}
	}
	return effects, nil
}

func createNotificationTx(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	builder *notificationEffectBuilder,
	notification *domain.Notification,
) error {
	id, err := repositories.Notifications().Create(ctx, notification)
	if err != nil {
		return err
	}
	builder.changed(notification.RecipientUserID, id)
	return nil
}

func resolveNotificationTx(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	builder *notificationEffectBuilder,
	notification *domain.Notification,
	resolution domain.NotificationResolution,
	now time.Time,
) (bool, error) {
	if notification == nil {
		return false, repo.ErrNotFound
	}
	changed, err := repositories.Notifications().Resolve(ctx, notification.ID, resolution, now)
	if err != nil {
		return false, err
	}
	if changed {
		builder.changed(notification.RecipientUserID, notification.ID)
	}
	return changed, nil
}

type NotificationPage struct {
	Notifications []*domain.Notification
	NextCursor    *string
	UnreadCount   int64
	Revision      int64
}

type NotificationReadResult struct {
	Notification *domain.Notification
	UnreadCount  int64
	Revision     int64
}

type NotificationReadAllResult struct {
	ReadAt      time.Time
	UnreadCount int64
	Revision    int64
	Changed     bool
}

type NotificationAction string

const (
	NotificationActionAccept  NotificationAction = "accept"
	NotificationActionDecline NotificationAction = "decline"
)

func (a NotificationAction) Valid() bool {
	return a == NotificationActionAccept || a == NotificationActionDecline
}

func (a NotificationAction) resolution() domain.NotificationResolution {
	if a == NotificationActionAccept {
		return domain.NotificationAccepted
	}
	return domain.NotificationDeclined
}

type NotificationActionSourceKind string

const (
	NotificationSourceRelationship NotificationActionSourceKind = "relationship"
	NotificationSourceGroup        NotificationActionSourceKind = "group"
)

type NotificationActionSource struct {
	Kind         NotificationActionSourceKind
	UserID       int64
	Relationship *Relationship
	Group        *domain.Group
}

type NotificationActionResult struct {
	Notification            *domain.Notification
	Source                  *NotificationActionSource
	UnreadCount             int64
	Revision                int64
	SourceTransitionApplied bool
	NotificationEffects     *NotificationEffects
}

type NotificationService struct {
	transactions repo.TransactionManager
	clock        clock.Clock
}

func NewNotificationService(transactions repo.TransactionManager, appClock clock.Clock) *NotificationService {
	return &NotificationService{transactions: transactions, clock: appClock}
}

func (s *NotificationService) List(
	ctx context.Context,
	recipientUserID int64,
	cursor *domain.NotificationCursor,
	limit int,
) (*NotificationPage, error) {
	if s == nil || s.transactions == nil || recipientUserID <= 0 || !validNotificationPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	page := &NotificationPage{}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		items, err := repositories.Notifications().ListForRecipient(ctx, recipientUserID, cursor, limit+1)
		if err != nil {
			return err
		}
		page.Notifications = items
		if len(items) > limit {
			page.Notifications = items[:limit]
			last := page.Notifications[len(page.Notifications)-1]
			encoded := EncodeNotificationCursor(domain.NotificationCursor{CreatedAt: last.CreatedAt, ID: last.ID})
			page.NextCursor = &encoded
		}
		page.UnreadCount, err = repositories.Notifications().UnreadCount(ctx, recipientUserID)
		if err != nil {
			return err
		}
		page.Revision, err = repositories.Notifications().CurrentRevision(ctx, recipientUserID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return page, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, recipientUserID, notificationID int64) (*NotificationReadResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || recipientUserID <= 0 || notificationID <= 0 {
		return nil, ErrInvalidInput
	}
	result := &NotificationReadResult{}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := repositories.Notifications().GetForRecipient(ctx, recipientUserID, notificationID); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		changed, err := repositories.Notifications().MarkRead(ctx, notificationID, recipientUserID, s.clock.Now().UTC())
		if err != nil {
			return err
		}
		if changed {
			result.Revision, err = repositories.Notifications().BumpRevision(ctx, recipientUserID)
		} else {
			result.Revision, err = repositories.Notifications().CurrentRevision(ctx, recipientUserID)
		}
		if err != nil {
			return err
		}
		result.UnreadCount, err = repositories.Notifications().UnreadCount(ctx, recipientUserID)
		if err != nil {
			return err
		}
		result.Notification, err = repositories.Notifications().GetForRecipient(ctx, recipientUserID, notificationID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *NotificationService) MarkAllRead(ctx context.Context, recipientUserID int64) (*NotificationReadAllResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || recipientUserID <= 0 {
		return nil, ErrInvalidInput
	}
	result := &NotificationReadAllResult{}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		result.ReadAt = s.clock.Now().UTC()
		rows, err := repositories.Notifications().MarkAllRead(ctx, recipientUserID, result.ReadAt)
		if err != nil {
			return err
		}
		result.Changed = rows > 0
		if result.Changed {
			result.Revision, err = repositories.Notifications().BumpRevision(ctx, recipientUserID)
		} else {
			result.Revision, err = repositories.Notifications().CurrentRevision(ctx, recipientUserID)
		}
		if err != nil {
			return err
		}
		result.UnreadCount, err = repositories.Notifications().UnreadCount(ctx, recipientUserID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *NotificationService) Action(
	ctx context.Context,
	recipientUserID, notificationID int64,
	action NotificationAction,
) (*NotificationActionResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || recipientUserID <= 0 || notificationID <= 0 || !action.Valid() {
		return nil, ErrInvalidInput
	}
	result := &NotificationActionResult{NotificationEffects: emptyNotificationEffects()}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		notification, err := repositories.Notifications().GetForRecipient(ctx, recipientUserID, notificationID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !notification.Type.Actionable() {
			return ErrConflict
		}
		desiredResolution := action.resolution()
		if notification.Resolution != nil {
			if *notification.Resolution != desiredResolution {
				return ErrConflict
			}
			result.Notification = notification
			result.Source, err = notificationActionSourceTx(ctx, repositories, notification)
			if err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
			if errors.Is(err, ErrNotFound) {
				result.Source = nil
			}
			result.UnreadCount, err = repositories.Notifications().UnreadCount(ctx, recipientUserID)
			if err != nil {
				return err
			}
			result.Revision, err = repositories.Notifications().CurrentRevision(ctx, recipientUserID)
			return err
		}

		if err := validateNotificationLifecycleTx(ctx, repositories, notification); err != nil {
			return err
		}
		now := s.clock.Now().UTC()
		switch notification.Type {
		case domain.NotificationFollowRequest:
			if action == NotificationActionAccept {
				if _, err := acceptFollowRequestTx(ctx, repositories, *notification.FollowID, recipientUserID, now); err != nil {
					return mapNotificationActionSourceError(err)
				}
			} else if err := declineFollowRequestTx(ctx, repositories, *notification.FollowID, recipientUserID); err != nil {
				return mapNotificationActionSourceError(err)
			}
		case domain.NotificationGroupInvitation, domain.NotificationGroupJoinRequest:
			membership, err := repositories.Groups().GetMembershipByID(ctx, *notification.MembershipID)
			if err != nil {
				return ErrConflict
			}
			if action == NotificationActionAccept {
				if err := updateGroupMembershipTx(ctx, repositories, membership, membership.Status, domain.GroupMember, now); err != nil {
					return mapNotificationActionSourceError(err)
				}
			} else if err := deleteGroupMembershipTx(ctx, repositories, membership, membership.Status); err != nil {
				return mapNotificationActionSourceError(err)
			}
		default:
			return ErrConflict
		}

		result.Source, err = notificationActionSourceTx(ctx, repositories, notification)
		if err != nil {
			return err
		}
		builder := newNotificationEffectBuilder()
		changed, err := resolveNotificationTx(ctx, repositories, builder, notification, desiredResolution, now)
		if err != nil {
			return err
		}
		if !changed {
			return ErrConflict
		}
		result.NotificationEffects, err = builder.finish(ctx, repositories)
		if err != nil {
			return err
		}
		state, ok := result.NotificationEffects.StatesByUser[recipientUserID]
		if !ok || len(result.NotificationEffects.Upserts) != 1 {
			return errors.New("notification action effect invariant failed")
		}
		result.Notification = result.NotificationEffects.Upserts[0]
		result.UnreadCount = state.UnreadCount
		result.Revision = state.Revision
		result.SourceTransitionApplied = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func validateNotificationLifecycleTx(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	notification *domain.Notification,
) error {
	if notification == nil || notification.Resolution != nil {
		return ErrConflict
	}
	switch notification.Type {
	case domain.NotificationFollowRequest:
		if notification.FollowID == nil {
			return ErrConflict
		}
		follow, err := repositories.Follows().GetByID(ctx, *notification.FollowID)
		if err != nil || follow.FollowerUserID != notification.ActorUserID ||
			follow.FollowedUserID != notification.RecipientUserID || follow.Status != domain.FollowPending {
			return ErrConflict
		}
	case domain.NotificationGroupInvitation:
		if notification.GroupID == nil || notification.MembershipID == nil {
			return ErrConflict
		}
		membership, err := repositories.Groups().GetMembershipByID(ctx, *notification.MembershipID)
		if err != nil || membership.GroupID != *notification.GroupID ||
			membership.UserID != notification.RecipientUserID || membership.Status != domain.GroupInvited {
			return ErrConflict
		}
	case domain.NotificationGroupJoinRequest:
		if notification.GroupID == nil || notification.MembershipID == nil {
			return ErrConflict
		}
		membership, err := repositories.Groups().GetMembershipByID(ctx, *notification.MembershipID)
		if err != nil || membership.GroupID != *notification.GroupID ||
			membership.UserID != notification.ActorUserID || membership.Status != domain.GroupRequested {
			return ErrConflict
		}
		group, err := repositories.Groups().Get(ctx, *notification.GroupID, notification.RecipientUserID)
		if err != nil || group.OwnerUserID != notification.RecipientUserID {
			return ErrConflict
		}
	default:
		return ErrConflict
	}
	return nil
}

func notificationActionSourceTx(
	ctx context.Context,
	repositories repo.TransactionRepositories,
	notification *domain.Notification,
) (*NotificationActionSource, error) {
	if notification == nil {
		return nil, ErrNotFound
	}
	switch notification.Type {
	case domain.NotificationFollowRequest:
		relationship, err := relationshipTx(ctx, repositories, notification.RecipientUserID, notification.ActorUserID)
		if err != nil {
			return nil, err
		}
		return &NotificationActionSource{
			Kind: NotificationSourceRelationship, UserID: notification.ActorUserID, Relationship: relationship,
		}, nil
	case domain.NotificationGroupInvitation, domain.NotificationGroupJoinRequest:
		if notification.GroupID == nil {
			return nil, ErrNotFound
		}
		group, err := repositories.Groups().Get(ctx, *notification.GroupID, notification.RecipientUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		return &NotificationActionSource{Kind: NotificationSourceGroup, Group: group}, nil
	default:
		return nil, ErrConflict
	}
}

func mapNotificationActionSourceError(err error) error {
	if errors.Is(err, ErrNotFound) || errors.Is(err, repo.ErrNotFound) || errors.Is(err, ErrConflict) || errors.Is(err, repo.ErrConflict) {
		return ErrConflict
	}
	return err
}

func validNotificationPage(cursor *domain.NotificationCursor, limit int) bool {
	return limit >= 1 && limit <= MaxNotificationPageLimit &&
		(cursor == nil || !cursor.CreatedAt.IsZero() && cursor.ID > 0)
}

func EncodeNotificationCursor(cursor domain.NotificationCursor) string {
	payload := notificationCursorVersion + ":" + strconv.FormatInt(cursor.CreatedAt.UTC().Unix(), 10) + ":" + strconv.FormatInt(cursor.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeNotificationCursor(value string) (*domain.NotificationCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ErrInvalidInput
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidInput
	}
	parts := strings.Split(string(payload), ":")
	if len(parts) != 3 || parts[0] != notificationCursorVersion {
		return nil, ErrInvalidInput
	}
	createdAtUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || createdAtUnix <= 0 {
		return nil, ErrInvalidInput
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || id <= 0 {
		return nil, ErrInvalidInput
	}
	return &domain.NotificationCursor{CreatedAt: time.Unix(createdAtUnix, 0).UTC(), ID: id}, nil
}
