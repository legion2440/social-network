package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/repo"
)

type ProfileField struct {
	Present bool
	Value   *string
}

type UpdateProfileInput struct {
	FirstName   ProfileField
	LastName    ProfileField
	DateOfBirth ProfileField
	Gender      ProfileField
	Nickname    ProfileField
	AboutMe     ProfileField
}

func (i UpdateProfileInput) Empty() bool {
	return !i.FirstName.Present &&
		!i.LastName.Present &&
		!i.DateOfBirth.Present &&
		!i.Gender.Present &&
		!i.Nickname.Present &&
		!i.AboutMe.Present
}

type ProfileService struct {
	transactions repo.TransactionManager
	clock        clock.Clock
	avatars      *MediaStager
	logger       *log.Logger
}

func NewProfileService(
	transactions repo.TransactionManager,
	appClock clock.Clock,
	avatars *MediaStager,
	logger *log.Logger,
) *ProfileService {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &ProfileService{
		transactions: transactions,
		clock:        appClock,
		avatars:      avatars,
		logger:       logger,
	}
}

func (s *ProfileService) Update(ctx context.Context, userID int64, input UpdateProfileInput) (*domain.User, error) {
	if s == nil || s.transactions == nil || s.clock == nil || userID <= 0 || input.Empty() {
		return nil, ErrInvalidInput
	}

	var user *domain.User
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		user, err = repositories.Users().GetByID(ctx, userID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrUnauthorized
		}
		if err != nil {
			return err
		}

		if err := applyProfileUpdate(user, input); err != nil {
			return err
		}
		user.UpdatedAt = s.clock.Now()
		return repositories.Users().UpdateProfile(ctx, user)
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *ProfileService) ReplaceAvatar(ctx context.Context, userID int64, upload MediaUpload) (*domain.User, error) {
	if s == nil || s.transactions == nil || s.clock == nil || s.avatars == nil || userID <= 0 {
		return nil, ErrInvalidInput
	}

	staged, err := s.avatars.Stage(upload)
	if err != nil {
		return nil, err
	}
	defer staged.Discard()

	var user *domain.User
	var oldMedia *domain.Media
	now := s.clock.Now()
	err = s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		user, err = repositories.Users().GetByID(ctx, userID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrUnauthorized
		}
		if err != nil {
			return err
		}

		if user.AvatarMediaID != nil {
			oldMedia, err = repositories.Media().GetByID(ctx, *user.AvatarMediaID)
			if err != nil {
				return err
			}
			if oldMedia.OwnerUserID != user.ID {
				return fmt.Errorf("avatar media %d is not owned by user %d", oldMedia.ID, user.ID)
			}
		}

		mediaID, err := repositories.Media().Create(
			ctx,
			user.ID,
			staged.MIME,
			staged.Size,
			staged.StorageKey,
			staged.OriginalName,
			now,
		)
		if err != nil {
			return err
		}
		if err := repositories.Users().SetAvatarMediaID(ctx, user.ID, &mediaID, now); err != nil {
			return err
		}
		if oldMedia != nil {
			if err := repositories.Media().DeleteByID(ctx, oldMedia.ID); err != nil {
				return err
			}
		}
		if err := staged.Finalize(); err != nil {
			return err
		}

		user.AvatarMediaID = &mediaID
		user.UpdatedAt = now
		return nil
	})
	if err != nil {
		return nil, err
	}

	staged.Keep()
	if oldMedia != nil {
		s.removeCommittedAvatar(oldMedia)
	}
	return user, nil
}

func (s *ProfileService) DeleteAvatar(ctx context.Context, userID int64) (*domain.User, error) {
	if s == nil || s.transactions == nil || s.clock == nil || s.avatars == nil || userID <= 0 {
		return nil, ErrInvalidInput
	}

	var user *domain.User
	var oldMedia *domain.Media
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		user, err = repositories.Users().GetByID(ctx, userID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrUnauthorized
		}
		if err != nil {
			return err
		}
		if user.AvatarMediaID == nil {
			return nil
		}

		oldMedia, err = repositories.Media().GetByID(ctx, *user.AvatarMediaID)
		if err != nil {
			return err
		}
		if oldMedia.OwnerUserID != user.ID {
			return fmt.Errorf("avatar media %d is not owned by user %d", oldMedia.ID, user.ID)
		}

		now := s.clock.Now()
		if err := repositories.Users().SetAvatarMediaID(ctx, user.ID, nil, now); err != nil {
			return err
		}
		if err := repositories.Media().DeleteByID(ctx, oldMedia.ID); err != nil {
			return err
		}
		user.AvatarMediaID = nil
		user.UpdatedAt = now
		return nil
	})
	if err != nil {
		return nil, err
	}

	if oldMedia != nil {
		s.removeCommittedAvatar(oldMedia)
	}
	return user, nil
}

func applyProfileUpdate(user *domain.User, input UpdateProfileInput) error {
	if user == nil {
		return ErrInvalidInput
	}
	if input.FirstName.Present {
		if input.FirstName.Value == nil || strings.TrimSpace(*input.FirstName.Value) == "" {
			return ErrInvalidInput
		}
		user.FirstName = strings.TrimSpace(*input.FirstName.Value)
	}
	if input.LastName.Present {
		if input.LastName.Value == nil || strings.TrimSpace(*input.LastName.Value) == "" {
			return ErrInvalidInput
		}
		user.LastName = strings.TrimSpace(*input.LastName.Value)
	}
	if input.DateOfBirth.Present {
		if input.DateOfBirth.Value == nil || !domain.ValidDateOfBirth(*input.DateOfBirth.Value) {
			return ErrInvalidInput
		}
		user.DateOfBirth = *input.DateOfBirth.Value
	}
	if input.Gender.Present {
		if input.Gender.Value == nil {
			user.Gender = nil
		} else {
			gender := domain.Gender(*input.Gender.Value)
			if !gender.Valid() {
				return ErrInvalidInput
			}
			user.Gender = &gender
		}
	}
	if input.Nickname.Present {
		user.Nickname = optionalTrimmed(input.Nickname.Value)
	}
	if input.AboutMe.Present {
		user.AboutMe = optionalTrimmed(input.AboutMe.Value)
	}
	return nil
}

func (s *ProfileService) removeCommittedAvatar(media *domain.Media) {
	if media == nil {
		return
	}
	if err := s.avatars.Remove(media.StorageKey); err != nil {
		s.logger.Printf("remove replaced avatar %d: %v", media.ID, err)
	}
}
