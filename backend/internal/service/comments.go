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
	MaxCommentTextRunes     = 5000
	DefaultCommentPageLimit = 20
	MaxCommentPageLimit     = 50
	commentCursorVersion    = "v1"
)

type CommentPage struct {
	Comments   []*domain.Comment
	NextCursor *string
}

type CreateCommentInput struct {
	Text  string
	Media *MediaUpload
}

type CommentService struct {
	transactions repo.TransactionManager
	clock        clock.Clock
	media        *MediaStager
}

func NewCommentService(transactions repo.TransactionManager, appClock clock.Clock, media *MediaStager) *CommentService {
	return &CommentService{transactions: transactions, clock: appClock, media: media}
}

func (s *CommentService) Create(ctx context.Context, authorUserID, postID int64, input CreateCommentInput) (*domain.Comment, error) {
	if s == nil || s.transactions == nil || s.clock == nil || s.media == nil || authorUserID <= 0 || postID <= 0 {
		return nil, ErrInvalidInput
	}
	text := strings.TrimSpace(input.Text)
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) < 1 || utf8.RuneCountInString(text) > MaxCommentTextRunes {
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
	comment := &domain.Comment{
		PostID:       postID,
		AuthorUserID: authorUserID,
		Text:         text,
		CreatedAt:    now,
	}
	err = s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := authorizePostAccess(ctx, repositories, authorUserID, postID); err != nil {
			return err
		}
		author, err := repositories.Users().GetByID(ctx, authorUserID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrUnauthorized
		}
		if err != nil {
			return err
		}
		comment.Author = author

		if staged != nil {
			mediaID, err := repositories.Media().Create(
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
			createdMedia, err := repositories.Media().GetByID(ctx, mediaID)
			if err != nil {
				return err
			}
			if createdMedia.OwnerUserID != authorUserID {
				return fmt.Errorf("comment media %d is not owned by user %d", mediaID, authorUserID)
			}
			comment.MediaID = &mediaID
		}

		comment.ID, err = repositories.Comments().Create(ctx, comment)
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
	return comment, nil
}

func (s *CommentService) List(
	ctx context.Context,
	viewerUserID, postID int64,
	cursor *domain.CommentCursor,
	limit int,
) (*CommentPage, error) {
	if s == nil || s.transactions == nil || viewerUserID <= 0 || postID <= 0 || !validCommentPage(cursor, limit) {
		return nil, ErrInvalidInput
	}
	var comments []*domain.Comment
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		if _, err := authorizePostAccess(ctx, repositories, viewerUserID, postID); err != nil {
			return err
		}
		var err error
		comments, err = repositories.Comments().ListByPost(ctx, viewerUserID, postID, cursor, limit+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buildCommentPage(comments, limit), nil
}

func validCommentPage(cursor *domain.CommentCursor, limit int) bool {
	if limit < 1 || limit > MaxCommentPageLimit {
		return false
	}
	return cursor == nil || (!cursor.CreatedAt.IsZero() && cursor.ID > 0)
}

func buildCommentPage(comments []*domain.Comment, limit int) *CommentPage {
	page := &CommentPage{Comments: comments, NextCursor: nil}
	if len(comments) <= limit {
		return page
	}
	page.Comments = comments[:limit]
	last := page.Comments[len(page.Comments)-1]
	cursor := EncodeCommentCursor(domain.CommentCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	page.NextCursor = &cursor
	return page
}

func EncodeCommentCursor(cursor domain.CommentCursor) string {
	payload := commentCursorVersion + ":" + strconv.FormatInt(cursor.CreatedAt.UTC().Unix(), 10) + ":" + strconv.FormatInt(cursor.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeCommentCursor(value string) (*domain.CommentCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ErrInvalidInput
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidInput
	}
	parts := strings.Split(string(payload), ":")
	if len(parts) != 3 || parts[0] != commentCursorVersion {
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
	return &domain.CommentCursor{CreatedAt: time.Unix(timestamp, 0).UTC(), ID: id}, nil
}
