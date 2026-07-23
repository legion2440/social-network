package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/repo"
)

const (
	MaxGroupTitleRunes       = 100
	MaxGroupDescriptionRunes = 2000
	DefaultGroupPageLimit    = 20
	MaxGroupPageLimit        = 50
	groupCursorVersion       = "v1"
	groupMemberCursorVersion = "v1"
	groupStateCursorVersion  = "v1"
	groupInboxCursorVersion  = "v1"
)

type GroupPage struct {
	Groups     []*domain.Group
	NextCursor *string
}

type GroupMemberPage struct {
	Members    []*domain.GroupMembership
	NextCursor *string
}

type GroupMembershipPage struct {
	Memberships []*domain.GroupMembership
	NextCursor  *string
}

type GroupInvitationPage struct {
	Invitations []*domain.GroupInvitation
	NextCursor  *string
}

type GroupService struct {
	transactions repo.TransactionManager
	clock        clock.Clock
}

type GroupMutationResult struct {
	Group               *domain.Group
	NotificationEffects *NotificationEffects
}

func NewGroupService(transactions repo.TransactionManager, appClock clock.Clock) *GroupService {
	return &GroupService{transactions: transactions, clock: appClock}
}

func (s *GroupService) Create(ctx context.Context, ownerUserID int64, titleValue, descriptionValue string) (*domain.Group, error) {
	if s == nil || s.transactions == nil || s.clock == nil || ownerUserID <= 0 {
		return nil, ErrInvalidInput
	}
	title := strings.TrimSpace(titleValue)
	description := strings.TrimSpace(descriptionValue)
	if !validRuneLength(title, 1, MaxGroupTitleRunes) || !validRuneLength(description, 1, MaxGroupDescriptionRunes) {
		return nil, ErrInvalidInput
	}
	now := s.clock.Now()
	group := &domain.Group{OwnerUserID: ownerUserID, Title: title, Description: description, CreatedAt: now}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := repositories.Users().GetByID(ctx, ownerUserID); err != nil {
			return mapGroupRepoError(err)
		}
		var err error
		group.ID, err = repositories.Groups().Create(ctx, group)
		if err != nil {
			return err
		}
		if _, err := repositories.Groups().CreateMembership(ctx, &domain.GroupMembership{
			GroupID: group.ID, UserID: ownerUserID, Status: domain.GroupOwner, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return mapGroupRepoError(err)
		}
		group, err = repositories.Groups().Get(ctx, group.ID, ownerUserID)
		return mapGroupRepoError(err)
	})
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (s *GroupService) Directory(ctx context.Context, viewerUserID int64, cursor *domain.GroupCursor, limit int) (*GroupPage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || !validGroupPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var groups []*domain.Group
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		groups, err = repositories.Groups().List(ctx, viewerUserID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildGroupPage(groups, limit), nil
}

func (s *GroupService) Detail(ctx context.Context, viewerUserID, groupID int64) (*domain.Group, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || groupID <= 0 {
		return nil, ErrInvalidInput
	}
	var group *domain.Group
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		group, err = repositories.Groups().Get(ctx, groupID, viewerUserID)
		return mapGroupRepoError(err)
	})
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (s *GroupService) Members(ctx context.Context, viewerUserID, groupID int64, cursor *domain.GroupMemberCursor, limit int) (*GroupMemberPage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || groupID <= 0 || !validGroupMemberPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var memberships []*domain.GroupMembership
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := repositories.Groups().Get(ctx, groupID, viewerUserID); err != nil {
			return mapGroupRepoError(err)
		}
		var err error
		memberships, err = repositories.Groups().ListMembers(ctx, groupID, viewerUserID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildGroupMemberPage(memberships, limit), nil
}

func (s *GroupService) JoinRequests(ctx context.Context, ownerUserID, groupID int64, cursor *domain.GroupMembershipCursor, limit int) (*GroupMembershipPage, error) {
	return s.membershipList(ctx, ownerUserID, groupID, domain.GroupRequested, cursor, limit)
}

func (s *GroupService) SentInvitations(ctx context.Context, ownerUserID, groupID int64, cursor *domain.GroupMembershipCursor, limit int) (*GroupMembershipPage, error) {
	return s.membershipList(ctx, ownerUserID, groupID, domain.GroupInvited, cursor, limit)
}

func (s *GroupService) membershipList(
	ctx context.Context,
	ownerUserID, groupID int64,
	status domain.GroupMembershipStatus,
	cursor *domain.GroupMembershipCursor,
	limit int,
) (*GroupMembershipPage, error) {
	if s == nil || s.transactions == nil || ownerUserID <= 0 || groupID <= 0 || !validGroupMembershipPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var memberships []*domain.GroupMembership
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		group, err := repositories.Groups().Get(ctx, groupID, ownerUserID)
		if err != nil {
			return mapGroupRepoError(err)
		}
		if group.OwnerUserID != ownerUserID {
			return ErrForbidden
		}
		memberships, err = repositories.Groups().ListMemberships(ctx, groupID, ownerUserID, status, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildGroupMembershipPage(memberships, limit), nil
}

func (s *GroupService) InvitationInbox(ctx context.Context, userID int64, cursor *domain.GroupInvitationCursor, limit int) (*GroupInvitationPage, error) {
	if s == nil || s.transactions == nil || userID <= 0 || !validGroupInvitationPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var invitations []*domain.GroupInvitation
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		invitations, err = repositories.Groups().ListInvitationInbox(ctx, userID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildGroupInvitationPage(invitations, limit), nil
}

func (s *GroupService) RequestJoin(ctx context.Context, userID, groupID int64) (*domain.Group, error) {
	result, err := s.RequestJoinWithEffects(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	return result.Group, nil
}

func (s *GroupService) RequestJoinWithEffects(ctx context.Context, userID, groupID int64) (*GroupMutationResult, error) {
	return s.createMembership(ctx, userID, groupID, userID, domain.GroupRequested, false)
}

func (s *GroupService) Invite(ctx context.Context, actorUserID, groupID, targetUserID int64) (*domain.Group, error) {
	result, err := s.InviteWithEffects(ctx, actorUserID, groupID, targetUserID)
	if err != nil {
		return nil, err
	}
	return result.Group, nil
}

func (s *GroupService) InviteWithEffects(ctx context.Context, actorUserID, groupID, targetUserID int64) (*GroupMutationResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || actorUserID <= 0 || groupID <= 0 || targetUserID <= 0 {
		return nil, ErrInvalidInput
	}
	if actorUserID == targetUserID {
		return nil, ErrConflict
	}
	result := &GroupMutationResult{NotificationEffects: emptyNotificationEffects()}
	now := s.clock.Now()
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		builder := newNotificationEffectBuilder()
		var err error
		result.Group, err = repositories.Groups().Get(ctx, groupID, actorUserID)
		if err != nil {
			return mapGroupRepoError(err)
		}
		status, err := repositories.Groups().GetMembershipStatus(ctx, groupID, actorUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		if status == nil || (*status != domain.GroupOwner && *status != domain.GroupMember) {
			return ErrForbidden
		}
		if _, err := repositories.Users().GetByID(ctx, targetUserID); err != nil {
			return mapGroupRepoError(err)
		}
		membership := &domain.GroupMembership{
			GroupID: groupID, UserID: targetUserID, Status: domain.GroupInvited, CreatedAt: now, UpdatedAt: now,
		}
		membershipID, err := repositories.Groups().CreateMembership(ctx, membership)
		if err != nil {
			return mapGroupRepoError(err)
		}
		if err := createNotificationTx(ctx, repositories, builder, &domain.Notification{
			RecipientUserID: targetUserID, ActorUserID: actorUserID, Type: domain.NotificationGroupInvitation,
			GroupID: &groupID, MembershipID: &membershipID, CreatedAt: now,
		}); err != nil {
			return err
		}
		result.Group, err = repositories.Groups().Get(ctx, groupID, actorUserID)
		if err != nil {
			return mapGroupRepoError(err)
		}
		result.NotificationEffects, err = builder.finish(ctx, repositories)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *GroupService) createMembership(
	ctx context.Context,
	viewerUserID, groupID, targetUserID int64,
	status domain.GroupMembershipStatus,
	requireOwner bool,
) (*GroupMutationResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || viewerUserID <= 0 || groupID <= 0 || targetUserID <= 0 {
		return nil, ErrInvalidInput
	}
	result := &GroupMutationResult{NotificationEffects: emptyNotificationEffects()}
	now := s.clock.Now()
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		builder := newNotificationEffectBuilder()
		var err error
		result.Group, err = repositories.Groups().Get(ctx, groupID, viewerUserID)
		if err != nil {
			return mapGroupRepoError(err)
		}
		if requireOwner && result.Group.OwnerUserID != viewerUserID {
			return ErrForbidden
		}
		if _, err := repositories.Users().GetByID(ctx, targetUserID); err != nil {
			return mapGroupRepoError(err)
		}
		membership := &domain.GroupMembership{
			GroupID: groupID, UserID: targetUserID, Status: status, CreatedAt: now, UpdatedAt: now,
		}
		membershipID, err := repositories.Groups().CreateMembership(ctx, membership)
		if err != nil {
			return mapGroupRepoError(err)
		}
		if status == domain.GroupRequested {
			ownerUserID := result.Group.OwnerUserID
			if err := createNotificationTx(ctx, repositories, builder, &domain.Notification{
				RecipientUserID: ownerUserID, ActorUserID: targetUserID, Type: domain.NotificationGroupJoinRequest,
				GroupID: &groupID, MembershipID: &membershipID, CreatedAt: now,
			}); err != nil {
				return err
			}
		}
		result.Group, err = repositories.Groups().Get(ctx, groupID, viewerUserID)
		if err != nil {
			return mapGroupRepoError(err)
		}
		result.NotificationEffects, err = builder.finish(ctx, repositories)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *GroupService) CancelJoinRequest(ctx context.Context, userID, groupID int64) (*domain.Group, error) {
	result, err := s.CancelJoinRequestWithEffects(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	return result.Group, nil
}

func (s *GroupService) CancelJoinRequestWithEffects(ctx context.Context, userID, groupID int64) (*GroupMutationResult, error) {
	return s.deleteMembership(ctx, userID, groupID, userID, domain.GroupRequested, false, domain.NotificationCancelled)
}

func (s *GroupService) RejectJoinRequest(ctx context.Context, ownerUserID, groupID, targetUserID int64) (*domain.Group, error) {
	result, err := s.RejectJoinRequestWithEffects(ctx, ownerUserID, groupID, targetUserID)
	if err != nil {
		return nil, err
	}
	return result.Group, nil
}

func (s *GroupService) RejectJoinRequestWithEffects(ctx context.Context, ownerUserID, groupID, targetUserID int64) (*GroupMutationResult, error) {
	return s.deleteMembership(ctx, ownerUserID, groupID, targetUserID, domain.GroupRequested, true, domain.NotificationDeclined)
}

func (s *GroupService) DeclineInvitation(ctx context.Context, userID, groupID int64) (*domain.Group, error) {
	result, err := s.DeclineInvitationWithEffects(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	return result.Group, nil
}

func (s *GroupService) DeclineInvitationWithEffects(ctx context.Context, userID, groupID int64) (*GroupMutationResult, error) {
	return s.deleteMembership(ctx, userID, groupID, userID, domain.GroupInvited, false, domain.NotificationDeclined)
}

func (s *GroupService) Leave(ctx context.Context, userID, groupID int64) (*domain.Group, error) {
	result, err := s.LeaveWithEffects(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	return result.Group, nil
}

func (s *GroupService) LeaveWithEffects(ctx context.Context, userID, groupID int64) (*GroupMutationResult, error) {
	return s.deleteMembership(ctx, userID, groupID, userID, domain.GroupMember, false, "")
}

func (s *GroupService) deleteMembership(
	ctx context.Context,
	viewerUserID, groupID, targetUserID int64,
	expected domain.GroupMembershipStatus,
	requireOwner bool,
	resolution domain.NotificationResolution,
) (*GroupMutationResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || viewerUserID <= 0 || groupID <= 0 || targetUserID <= 0 {
		return nil, ErrInvalidInput
	}
	result := &GroupMutationResult{NotificationEffects: emptyNotificationEffects()}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		builder := newNotificationEffectBuilder()
		var err error
		result.Group, err = repositories.Groups().Get(ctx, groupID, viewerUserID)
		if err != nil {
			return mapGroupRepoError(err)
		}
		if requireOwner && result.Group.OwnerUserID != viewerUserID {
			return ErrForbidden
		}
		if _, err := repositories.Users().GetByID(ctx, targetUserID); err != nil {
			return mapGroupRepoError(err)
		}
		if !requireOwner && expected == domain.GroupMember && result.Group.OwnerUserID == viewerUserID {
			return ErrConflict
		}
		membership, err := repositories.Groups().GetMembership(ctx, groupID, targetUserID)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return ErrConflict
			}
			return mapGroupRepoError(err)
		}
		var notification *domain.Notification
		if resolution.Valid() {
			typeValue := domain.NotificationGroupInvitation
			if expected == domain.GroupRequested {
				typeValue = domain.NotificationGroupJoinRequest
			}
			notification, err = repositories.Notifications().FindPendingByMembershipID(ctx, typeValue, membership.ID)
			if err != nil && !errors.Is(err, repo.ErrNotFound) {
				return err
			}
		}
		if err := deleteGroupMembershipTx(ctx, repositories, membership, expected); err != nil {
			return err
		}
		if notification != nil {
			if _, err := resolveNotificationTx(ctx, repositories, builder, notification, resolution, s.clock.Now().UTC()); err != nil {
				return err
			}
		}
		result.Group, err = repositories.Groups().Get(ctx, groupID, viewerUserID)
		if err != nil {
			return mapGroupRepoError(err)
		}
		result.NotificationEffects, err = builder.finish(ctx, repositories)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *GroupService) AcceptJoinRequest(ctx context.Context, ownerUserID, groupID, targetUserID int64) (*domain.Group, error) {
	result, err := s.AcceptJoinRequestWithEffects(ctx, ownerUserID, groupID, targetUserID)
	if err != nil {
		return nil, err
	}
	return result.Group, nil
}

func (s *GroupService) AcceptJoinRequestWithEffects(ctx context.Context, ownerUserID, groupID, targetUserID int64) (*GroupMutationResult, error) {
	return s.updateMembership(ctx, ownerUserID, groupID, targetUserID, domain.GroupRequested, domain.GroupMember, true)
}

func (s *GroupService) AcceptInvitation(ctx context.Context, userID, groupID int64) (*domain.Group, error) {
	result, err := s.AcceptInvitationWithEffects(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	return result.Group, nil
}

func (s *GroupService) AcceptInvitationWithEffects(ctx context.Context, userID, groupID int64) (*GroupMutationResult, error) {
	return s.updateMembership(ctx, userID, groupID, userID, domain.GroupInvited, domain.GroupMember, false)
}

func (s *GroupService) updateMembership(
	ctx context.Context,
	viewerUserID, groupID, targetUserID int64,
	expected, next domain.GroupMembershipStatus,
	requireOwner bool,
) (*GroupMutationResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil || viewerUserID <= 0 || groupID <= 0 || targetUserID <= 0 {
		return nil, ErrInvalidInput
	}
	result := &GroupMutationResult{NotificationEffects: emptyNotificationEffects()}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		builder := newNotificationEffectBuilder()
		var err error
		result.Group, err = repositories.Groups().Get(ctx, groupID, viewerUserID)
		if err != nil {
			return mapGroupRepoError(err)
		}
		if requireOwner && result.Group.OwnerUserID != viewerUserID {
			return ErrForbidden
		}
		if _, err := repositories.Users().GetByID(ctx, targetUserID); err != nil {
			return mapGroupRepoError(err)
		}
		membership, err := repositories.Groups().GetMembership(ctx, groupID, targetUserID)
		if err != nil {
			return mapGroupRepoError(err)
		}
		typeValue := domain.NotificationGroupInvitation
		if expected == domain.GroupRequested {
			typeValue = domain.NotificationGroupJoinRequest
		}
		notification, notificationErr := repositories.Notifications().FindPendingByMembershipID(ctx, typeValue, membership.ID)
		if notificationErr != nil && !errors.Is(notificationErr, repo.ErrNotFound) {
			return notificationErr
		}
		now := s.clock.Now().UTC()
		if err := updateGroupMembershipTx(ctx, repositories, membership, expected, next, now); err != nil {
			return err
		}
		if notification != nil {
			if _, err := resolveNotificationTx(ctx, repositories, builder, notification, domain.NotificationAccepted, now); err != nil {
				return err
			}
		}
		result.Group, err = repositories.Groups().Get(ctx, groupID, viewerUserID)
		if err != nil {
			return mapGroupRepoError(err)
		}
		result.NotificationEffects, err = builder.finish(ctx, repositories)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func validRuneLength(value string, min, max int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) >= min && utf8.RuneCountInString(value) <= max
}

func mapGroupRepoError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repo.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, repo.ErrConflict):
		return ErrConflict
	default:
		return err
	}
}

func validGroupPage(cursor *domain.GroupCursor, limit int) bool {
	return limit >= 1 && limit <= MaxGroupPageLimit && (cursor == nil || !cursor.CreatedAt.IsZero() && cursor.ID > 0)
}

func validGroupMemberPage(cursor *domain.GroupMemberCursor, limit int) bool {
	return limit >= 1 && limit <= MaxGroupPageLimit && (cursor == nil || (cursor.OwnerRank == 0 || cursor.OwnerRank == 1) && !cursor.UpdatedAt.IsZero() && cursor.UserID > 0)
}

func validGroupMembershipPage(cursor *domain.GroupMembershipCursor, limit int) bool {
	return limit >= 1 && limit <= MaxGroupPageLimit && (cursor == nil || !cursor.CreatedAt.IsZero() && cursor.UserID > 0)
}

func validGroupInvitationPage(cursor *domain.GroupInvitationCursor, limit int) bool {
	return limit >= 1 && limit <= MaxGroupPageLimit && (cursor == nil || !cursor.CreatedAt.IsZero() && cursor.GroupID > 0)
}

func buildGroupPage(groups []*domain.Group, limit int) *GroupPage {
	page := &GroupPage{Groups: groups}
	if len(groups) <= limit {
		return page
	}
	page.Groups = groups[:limit]
	last := page.Groups[len(page.Groups)-1]
	cursor := EncodeGroupCursor(domain.GroupCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	page.NextCursor = &cursor
	return page
}

func buildGroupMemberPage(memberships []*domain.GroupMembership, limit int) *GroupMemberPage {
	page := &GroupMemberPage{Members: memberships}
	if len(memberships) <= limit {
		return page
	}
	page.Members = memberships[:limit]
	last := page.Members[len(page.Members)-1]
	rank := 1
	if last.Status == domain.GroupOwner {
		rank = 0
	}
	cursor := EncodeGroupMemberCursor(domain.GroupMemberCursor{OwnerRank: rank, UpdatedAt: last.UpdatedAt, UserID: last.UserID})
	page.NextCursor = &cursor
	return page
}

func buildGroupMembershipPage(memberships []*domain.GroupMembership, limit int) *GroupMembershipPage {
	page := &GroupMembershipPage{Memberships: memberships}
	if len(memberships) <= limit {
		return page
	}
	page.Memberships = memberships[:limit]
	last := page.Memberships[len(page.Memberships)-1]
	cursor := EncodeGroupMembershipCursor(domain.GroupMembershipCursor{CreatedAt: last.CreatedAt, UserID: last.UserID})
	page.NextCursor = &cursor
	return page
}

func buildGroupInvitationPage(invitations []*domain.GroupInvitation, limit int) *GroupInvitationPage {
	page := &GroupInvitationPage{Invitations: invitations}
	if len(invitations) <= limit {
		return page
	}
	page.Invitations = invitations[:limit]
	last := page.Invitations[len(page.Invitations)-1]
	cursor := EncodeGroupInvitationCursor(domain.GroupInvitationCursor{CreatedAt: last.CreatedAt, GroupID: last.Group.ID})
	page.NextCursor = &cursor
	return page
}

func EncodeGroupCursor(cursor domain.GroupCursor) string {
	return encodeGroupCursorParts(groupCursorVersion, cursor.CreatedAt, cursor.ID)
}

func DecodeGroupCursor(value string) (*domain.GroupCursor, error) {
	timestamp, id, err := decodeGroupCursorParts(value, groupCursorVersion)
	if err != nil {
		return nil, err
	}
	return &domain.GroupCursor{CreatedAt: timestamp, ID: id}, nil
}

func EncodeGroupMemberCursor(cursor domain.GroupMemberCursor) string {
	payload := groupMemberCursorVersion + ":" + strconv.Itoa(cursor.OwnerRank) + ":" + strconv.FormatInt(cursor.UpdatedAt.UTC().Unix(), 10) + ":" + strconv.FormatInt(cursor.UserID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeGroupMemberCursor(value string) (*domain.GroupMemberCursor, error) {
	parts, err := decodeCursor(value, 4, groupMemberCursorVersion)
	if err != nil {
		return nil, err
	}
	rank, err := strconv.Atoi(parts[1])
	if err != nil || rank < 0 || rank > 1 {
		return nil, ErrInvalidInput
	}
	timestamp, id, err := parseCursorTail(parts[2], parts[3])
	if err != nil {
		return nil, err
	}
	return &domain.GroupMemberCursor{OwnerRank: rank, UpdatedAt: timestamp, UserID: id}, nil
}

func EncodeGroupMembershipCursor(cursor domain.GroupMembershipCursor) string {
	return encodeGroupCursorParts(groupStateCursorVersion, cursor.CreatedAt, cursor.UserID)
}

func DecodeGroupMembershipCursor(value string) (*domain.GroupMembershipCursor, error) {
	timestamp, id, err := decodeGroupCursorParts(value, groupStateCursorVersion)
	if err != nil {
		return nil, err
	}
	return &domain.GroupMembershipCursor{CreatedAt: timestamp, UserID: id}, nil
}

func EncodeGroupInvitationCursor(cursor domain.GroupInvitationCursor) string {
	return encodeGroupCursorParts(groupInboxCursorVersion, cursor.CreatedAt, cursor.GroupID)
}

func DecodeGroupInvitationCursor(value string) (*domain.GroupInvitationCursor, error) {
	timestamp, id, err := decodeGroupCursorParts(value, groupInboxCursorVersion)
	if err != nil {
		return nil, err
	}
	return &domain.GroupInvitationCursor{CreatedAt: timestamp, GroupID: id}, nil
}

func encodeGroupCursorParts(version string, timestamp time.Time, id int64) string {
	payload := version + ":" + strconv.FormatInt(timestamp.UTC().Unix(), 10) + ":" + strconv.FormatInt(id, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeGroupCursorParts(value, version string) (time.Time, int64, error) {
	parts, err := decodeCursor(value, 3, version)
	if err != nil {
		return time.Time{}, 0, err
	}
	return parseCursorTail(parts[1], parts[2])
}

func decodeCursor(value string, count int, version string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ErrInvalidInput
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidInput
	}
	parts := strings.Split(string(payload), ":")
	if len(parts) != count || parts[0] != version {
		return nil, ErrInvalidInput
	}
	return parts, nil
}

func parseCursorTail(rawTimestamp, rawID string) (time.Time, int64, error) {
	timestamp, err := strconv.ParseInt(rawTimestamp, 10, 64)
	if err != nil || timestamp <= 0 {
		return time.Time{}, 0, ErrInvalidInput
	}
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return time.Time{}, 0, ErrInvalidInput
	}
	return time.Unix(timestamp, 0).UTC(), id, nil
}
