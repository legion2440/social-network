package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

const (
	DefaultUserPageLimit = 20
	MaxUserPageLimit     = 50
	userCursorVersion    = "v1"
)

type UserProfileStats struct {
	Posts     int64
	Followers int64
	Following int64
}

type UserProfile struct {
	User       *domain.User
	CanView    bool
	Statistics *UserProfileStats
}

type UserPage struct {
	Users      []*domain.RelatedUser
	NextCursor *string
}

type UserService struct {
	transactions repo.TransactionManager
}

func NewUserService(transactions repo.TransactionManager) *UserService {
	return &UserService{transactions: transactions}
}

func (s *UserService) Profile(ctx context.Context, viewerUserID, targetUserID int64) (*UserProfile, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || targetUserID <= 0 {
		return nil, ErrInvalidInput
	}

	profile := &UserProfile{}
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		target, err := repositories.Users().GetByID(ctx, targetUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		profile.User = target
		profile.CanView = viewerUserID == targetUserID || !target.IsPrivate
		if !profile.CanView {
			profile.CanView, err = repositories.Follows().IsAccepted(ctx, viewerUserID, targetUserID)
			if err != nil {
				return err
			}
		}
		if !profile.CanView {
			return nil
		}

		stats := &UserProfileStats{}
		stats.Posts, err = repositories.Posts().CountAccessibleByAuthor(ctx, viewerUserID, targetUserID)
		if err != nil {
			return err
		}
		stats.Followers, err = repositories.Follows().CountFollowers(ctx, targetUserID)
		if err != nil {
			return err
		}
		stats.Following, err = repositories.Follows().CountFollowing(ctx, targetUserID)
		if err != nil {
			return err
		}
		profile.Statistics = stats
		return nil
	})
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *UserService) Directory(
	ctx context.Context,
	viewerUserID int64,
	cursor *domain.UserCursor,
	limit int,
) (*UserPage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || !validUserPage(cursor, limit) {
		return nil, ErrInvalidInput
	}

	var users []*domain.RelatedUser
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		users, err = repositories.Users().ListDirectory(ctx, viewerUserID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildUserPage(users, limit), nil
}

func validUserPage(cursor *domain.UserCursor, limit int) bool {
	if limit < 1 || limit > MaxUserPageLimit {
		return false
	}
	return cursor == nil || (!cursor.CreatedAt.IsZero() && cursor.ID > 0)
}

func buildUserPage(users []*domain.RelatedUser, limit int) *UserPage {
	page := &UserPage{Users: users}
	if len(users) <= limit {
		return page
	}
	page.Users = users[:limit]
	last := page.Users[len(page.Users)-1].User
	if last != nil {
		cursor := EncodeUserCursor(domain.UserCursor{CreatedAt: last.CreatedAt, ID: last.ID})
		page.NextCursor = &cursor
	}
	return page
}

func EncodeUserCursor(cursor domain.UserCursor) string {
	payload := userCursorVersion + ":" + strconv.FormatInt(cursor.CreatedAt.UTC().Unix(), 10) + ":" + strconv.FormatInt(cursor.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeUserCursor(value string) (*domain.UserCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ErrInvalidInput
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidInput
	}
	parts := strings.Split(string(payload), ":")
	if len(parts) != 3 || parts[0] != userCursorVersion {
		return nil, ErrInvalidInput
	}
	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || timestamp <= 0 {
		return nil, ErrInvalidInput
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || id <= 0 {
		return nil, ErrInvalidInput
	}
	return &domain.UserCursor{CreatedAt: time.Unix(timestamp, 0).UTC(), ID: id}, nil
}
