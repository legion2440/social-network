package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/repo"
)

const (
	MaxPostTextRunes     = 5000
	MaxSelectedPostUsers = 100
	DefaultPostPageLimit = 20
	MaxPostPageLimit     = 50
	postCursorVersion    = "v1"
)

type CreatePostInput struct {
	Text            string
	Privacy         domain.PostPrivacy
	SelectedUserIDs []int64
	Media           *MediaUpload
}

type CreateGroupPostInput struct {
	Text  string
	Media *MediaUpload
}

type PostPage struct {
	Posts      []*domain.Post
	NextCursor *string
}

type PostService struct {
	transactions repo.TransactionManager
	clock        clock.Clock
	media        *MediaStager
}

func NewPostService(transactions repo.TransactionManager, appClock clock.Clock, media *MediaStager) *PostService {
	return &PostService{transactions: transactions, clock: appClock, media: media}
}

func (s *PostService) Create(ctx context.Context, authorUserID int64, input CreatePostInput) (*domain.Post, error) {
	if s == nil || s.transactions == nil || s.clock == nil || s.media == nil || authorUserID <= 0 {
		return nil, ErrInvalidInput
	}

	text := strings.TrimSpace(input.Text)
	if !utf8.ValidString(text) {
		return nil, ErrInvalidInput
	}
	runeCount := utf8.RuneCountInString(text)
	if runeCount < 1 || runeCount > MaxPostTextRunes || !input.Privacy.Valid() {
		return nil, ErrInvalidInput
	}
	selectedUserIDs, err := normalizeSelectedPostUsers(authorUserID, input.Privacy, input.SelectedUserIDs)
	if err != nil {
		return nil, err
	}

	var staged *StagedMedia
	if input.Media != nil {
		staged, err = s.media.Stage(*input.Media)
		if err != nil {
			return nil, err
		}
		defer staged.Discard()
	}

	now := s.clock.Now()
	var post *domain.Post
	err = s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		author, err := repositories.Users().GetByID(ctx, authorUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrUnauthorized
		}
		if err != nil {
			return err
		}

		for _, selectedUserID := range selectedUserIDs {
			accepted, err := repositories.Follows().IsAccepted(ctx, selectedUserID, authorUserID)
			if err != nil {
				return err
			}
			if !accepted {
				return ErrInvalidInput
			}
		}

		var mediaID *int64
		if staged != nil {
			createdMediaID, err := repositories.Media().Create(
				ctx,
				authorUserID,
				staged.MIME,
				staged.Size,
				staged.StorageKey,
				staged.OriginalName,
				now,
			)
			if err != nil {
				return err
			}
			createdMedia, err := repositories.Media().GetByID(ctx, createdMediaID)
			if err != nil {
				return err
			}
			if createdMedia.OwnerUserID != authorUserID {
				return fmt.Errorf("post media %d is not owned by user %d", createdMediaID, authorUserID)
			}
			mediaID = &createdMediaID
		}

		privacy := input.Privacy
		post = &domain.Post{
			AuthorUserID: authorUserID,
			Author:       author,
			Text:         text,
			Privacy:      &privacy,
			MediaID:      mediaID,
			CreatedAt:    now,
		}
		post.ID, err = repositories.Posts().Create(ctx, post)
		if err != nil {
			return err
		}
		if input.Privacy == domain.PostSelected {
			if err := repositories.Posts().AddSelectedUsers(ctx, post.ID, selectedUserIDs); err != nil {
				return err
			}
		}
		if staged != nil {
			if err := staged.Finalize(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if staged != nil {
		staged.Keep()
	}
	return post, nil
}

func (s *PostService) CreateGroupPost(
	ctx context.Context,
	authorUserID, groupID int64,
	input CreateGroupPostInput,
) (*domain.Post, error) {
	if s == nil || s.transactions == nil || s.clock == nil || s.media == nil || authorUserID <= 0 || groupID <= 0 {
		return nil, ErrInvalidInput
	}
	text := strings.TrimSpace(input.Text)
	if !utf8.ValidString(text) {
		return nil, ErrInvalidInput
	}
	runeCount := utf8.RuneCountInString(text)
	if runeCount < 1 || runeCount > MaxPostTextRunes {
		return nil, ErrInvalidInput
	}

	var staged *StagedMedia
	var err error
	if input.Media != nil {
		staged, err = s.media.Stage(*input.Media)
		if err != nil {
			return nil, err
		}
		defer staged.Discard()
	}

	now := s.clock.Now()
	var post *domain.Post
	err = s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := authorizeGroupContentAccess(ctx, repositories, authorUserID, groupID); err != nil {
			return err
		}
		author, err := repositories.Users().GetByID(ctx, authorUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrUnauthorized
		}
		if err != nil {
			return err
		}

		var mediaID *int64
		if staged != nil {
			createdMediaID, err := repositories.Media().Create(
				ctx, authorUserID, staged.MIME, staged.Size, staged.StorageKey, staged.OriginalName, now,
			)
			if err != nil {
				return err
			}
			createdMedia, err := repositories.Media().GetByID(ctx, createdMediaID)
			if err != nil {
				return err
			}
			if createdMedia.OwnerUserID != authorUserID {
				return fmt.Errorf("post media %d is not owned by user %d", createdMediaID, authorUserID)
			}
			mediaID = &createdMediaID
		}

		storedGroupID := groupID
		post = &domain.Post{
			AuthorUserID: authorUserID,
			Author:       author,
			GroupID:      &storedGroupID,
			Text:         text,
			MediaID:      mediaID,
			CreatedAt:    now,
		}
		post.ID, err = repositories.Posts().Create(ctx, post)
		if err != nil {
			return err
		}
		if staged != nil {
			if err := staged.Finalize(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if staged != nil {
		staged.Keep()
	}
	return post, nil
}

func (s *PostService) Feed(
	ctx context.Context,
	viewerUserID int64,
	cursor *domain.PostCursor,
	limit int,
) (*PostPage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || !validPostPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var posts []*domain.Post
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var err error
		posts, err = repositories.Posts().ListFeed(ctx, viewerUserID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildPostPage(posts, limit), nil
}

func (s *PostService) UserPosts(
	ctx context.Context,
	viewerUserID, authorUserID int64,
	cursor *domain.PostCursor,
	limit int,
) (*PostPage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || authorUserID <= 0 || !validPostPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var posts []*domain.Post
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := authorizeProfileRead(ctx, repositories.Users(), repositories.Follows(), viewerUserID, authorUserID); err != nil {
			return err
		}
		var err error
		posts, err = repositories.Posts().ListByAuthor(ctx, viewerUserID, authorUserID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildPostPage(posts, limit), nil
}

func (s *PostService) GroupPosts(
	ctx context.Context,
	viewerUserID, groupID int64,
	cursor *domain.PostCursor,
	limit int,
) (*PostPage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || groupID <= 0 || !validPostPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var posts []*domain.Post
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := authorizeGroupContentAccess(ctx, repositories, viewerUserID, groupID); err != nil {
			return err
		}
		var err error
		posts, err = repositories.Posts().ListByGroup(ctx, viewerUserID, groupID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildPostPage(posts, limit), nil
}

func normalizeSelectedPostUsers(authorUserID int64, privacy domain.PostPrivacy, values []int64) ([]int64, error) {
	if privacy != domain.PostSelected {
		if len(values) != 0 {
			return nil, ErrInvalidInput
		}
		return []int64{}, nil
	}

	unique := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value <= 0 || value == authorUserID {
			return nil, ErrInvalidInput
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
		if len(unique) > MaxSelectedPostUsers {
			return nil, ErrInvalidInput
		}
	}
	if len(unique) == 0 {
		return nil, ErrInvalidInput
	}
	return unique, nil
}

func validPostPage(cursor *domain.PostCursor, limit int) bool {
	if limit < 1 || limit > MaxPostPageLimit {
		return false
	}
	return cursor == nil || (!cursor.CreatedAt.IsZero() && cursor.ID > 0)
}

func buildPostPage(posts []*domain.Post, limit int) *PostPage {
	page := &PostPage{Posts: posts, NextCursor: nil}
	if len(posts) <= limit {
		return page
	}
	page.Posts = posts[:limit]
	last := page.Posts[len(page.Posts)-1]
	cursor := EncodePostCursor(domain.PostCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	page.NextCursor = &cursor
	return page
}

func EncodePostCursor(cursor domain.PostCursor) string {
	payload := postCursorVersion + ":" + strconv.FormatInt(cursor.CreatedAt.UTC().Unix(), 10) + ":" + strconv.FormatInt(cursor.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodePostCursor(value string) (*domain.PostCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ErrInvalidInput
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidInput
	}
	parts := strings.Split(string(payload), ":")
	if len(parts) != 3 || parts[0] != postCursorVersion {
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
	return &domain.PostCursor{CreatedAt: time.Unix(timestamp, 0).UTC(), ID: id}, nil
}
