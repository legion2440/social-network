package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/repo"
)

const MaxPasswordBytes = 72

type RegisterInput struct {
	Email       string
	Password    string
	FirstName   string
	LastName    string
	DateOfBirth string
	Gender      *domain.Gender
	Nickname    *string
	AboutMe     *string
	Avatar      *MediaUpload
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResult struct {
	User    *domain.User
	Session *domain.Session
}

type AuthService struct {
	users        repo.UserRepo
	transactions repo.TransactionManager
	sessions     *SessionService
	passwords    PasswordHasher
	clock        clock.Clock
	avatars      *MediaStager
}

func NewAuthService(
	users repo.UserRepo,
	transactions repo.TransactionManager,
	sessions *SessionService,
	passwords PasswordHasher,
	appClock clock.Clock,
	avatars *MediaStager,
) *AuthService {
	return &AuthService{
		users:        users,
		transactions: transactions,
		sessions:     sessions,
		passwords:    passwords,
		clock:        appClock,
		avatars:      avatars,
	}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	if s == nil || s.users == nil || s.transactions == nil || s.sessions == nil || s.passwords == nil || s.clock == nil || s.avatars == nil {
		return nil, ErrInvalidInput
	}

	input.Email = strings.TrimSpace(input.Email)
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Nickname = optionalTrimmed(input.Nickname)
	input.AboutMe = optionalTrimmed(input.AboutMe)
	if !validEmail(input.Email) || input.FirstName == "" || input.LastName == "" || !validPassword(input.Password) || !domain.ValidDateOfBirth(input.DateOfBirth) {
		return nil, ErrInvalidInput
	}
	if input.Gender != nil && !input.Gender.Valid() {
		return nil, ErrInvalidInput
	}

	passwordHash, err := s.passwords.Hash(input.Password)
	if err != nil {
		return nil, err
	}

	var stagedAvatar *StagedMedia
	if input.Avatar != nil {
		stagedAvatar, err = s.avatars.Stage(*input.Avatar)
		if err != nil {
			return nil, err
		}
		defer stagedAvatar.Discard()
	}

	now := s.clock.Now()
	user := &domain.User{
		Email:        input.Email,
		PasswordHash: passwordHash,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		DateOfBirth:  input.DateOfBirth,
		Gender:       input.Gender,
		Nickname:     input.Nickname,
		AboutMe:      input.AboutMe,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	var session *domain.Session
	err = s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		userID, err := repositories.Users().Create(ctx, user)
		if errors.Is(err, repo.ErrConflict) {
			return ErrEmailTaken
		}
		if err != nil {
			return err
		}
		user.ID = userID

		if stagedAvatar != nil {
			mediaID, err := repositories.Media().Create(
				ctx,
				user.ID,
				stagedAvatar.MIME,
				stagedAvatar.Size,
				stagedAvatar.StorageKey,
				stagedAvatar.OriginalName,
				now,
			)
			if err != nil {
				return err
			}
			if err := repositories.Users().SetAvatarMediaID(ctx, user.ID, &mediaID, now); err != nil {
				return err
			}
			user.AvatarMediaID = &mediaID
		}

		session, err = s.sessions.New(user.ID)
		if err != nil {
			return err
		}
		if err := repositories.Sessions().Create(ctx, session); err != nil {
			return err
		}
		if stagedAvatar != nil {
			if err := stagedAvatar.Finalize(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if stagedAvatar != nil {
		stagedAvatar.Keep()
	}
	return &AuthResult{User: user, Session: session}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	if s == nil || s.users == nil || s.sessions == nil || s.passwords == nil {
		return nil, ErrInvalidInput
	}
	input.Email = strings.TrimSpace(input.Email)
	if input.Email == "" || input.Password == "" {
		return nil, ErrInvalidCredentials
	}
	user, err := s.users.GetByEmail(ctx, input.Email)
	if errors.Is(err, repo.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if err := s.passwords.Compare(user.PasswordHash, input.Password); err != nil {
		return nil, ErrInvalidCredentials
	}
	session, err := s.sessions.Create(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{User: user, Session: session}, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if s == nil || s.sessions == nil {
		return ErrInvalidInput
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return s.sessions.Delete(ctx, token)
}

func (s *AuthService) Me(ctx context.Context, userID int64) (*domain.User, error) {
	if s == nil || s.users == nil || userID <= 0 {
		return nil, ErrUnauthorized
	}
	user, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, repo.ErrNotFound) {
		return nil, ErrUnauthorized
	}
	return user, err
}

func validEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func validPassword(value string) bool {
	return len(value) > 0 && len([]byte(value)) <= MaxPasswordBytes
}

func optionalTrimmed(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
