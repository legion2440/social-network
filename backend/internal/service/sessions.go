package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/platform/id"
	"social-network/backend/internal/repo"
)

type SessionService struct {
	sessions repo.SessionRepo
	clock    clock.Clock
	ids      id.Generator
	ttl      time.Duration
}

func NewSessionService(sessions repo.SessionRepo, appClock clock.Clock, ids id.Generator, ttl time.Duration) *SessionService {
	return &SessionService{
		sessions: sessions,
		clock:    appClock,
		ids:      ids,
		ttl:      ttl,
	}
}

func (s *SessionService) Create(ctx context.Context, userID int64) (*domain.Session, error) {
	if userID <= 0 || s == nil || s.sessions == nil || s.clock == nil || s.ids == nil || s.ttl <= 0 {
		return nil, ErrInvalidInput
	}

	token, err := s.ids.New()
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	session := &domain.Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: now.Add(s.ttl),
		CreatedAt: now,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *SessionService) Get(ctx context.Context, token string) (*domain.Session, error) {
	token = strings.TrimSpace(token)
	if token == "" || s == nil || s.sessions == nil || s.clock == nil {
		return nil, ErrUnauthorized
	}

	session, err := s.sessions.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	if !session.ExpiresAt.After(s.clock.Now()) {
		if err := s.sessions.DeleteByToken(ctx, token); err != nil {
			return nil, err
		}
		return nil, ErrUnauthorized
	}
	return session, nil
}

func (s *SessionService) Delete(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" || s == nil || s.sessions == nil {
		return ErrInvalidInput
	}
	return s.sessions.DeleteByToken(ctx, token)
}
