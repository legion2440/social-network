package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

type CommentRepo struct {
	db sqlExecutor
}

func NewCommentRepo(db *sql.DB) *CommentRepo {
	return &CommentRepo{db: db}
}

func (r *CommentRepo) Create(ctx context.Context, comment *domain.Comment) (int64, error) {
	if r == nil || r.db == nil || comment == nil || comment.PostID <= 0 || comment.AuthorUserID <= 0 || comment.Text == "" || comment.CreatedAt.IsZero() {
		return 0, fmt.Errorf("invalid comment")
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO post_comments (post_id, author_user_id, text, created_at)
		VALUES (?, ?, ?, ?)
	`, comment.PostID, comment.AuthorUserID, comment.Text, timeToUnix(comment.CreatedAt))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *CommentRepo) ListByPost(
	ctx context.Context,
	postID int64,
	cursor *domain.CommentCursor,
	limit int,
) ([]*domain.Comment, error) {
	if r == nil || r.db == nil || postID <= 0 || limit <= 0 {
		return []*domain.Comment{}, nil
	}
	query := `
		SELECT
			c.id, c.post_id, c.author_user_id, c.text, c.created_at,
			u.id, u.first_name, u.last_name, u.nickname, u.avatar_media_id, u.is_private
		FROM post_comments c
		JOIN users u ON u.id = c.author_user_id
		WHERE c.post_id = ?`
	args := []any{postID}
	if cursor != nil {
		timestamp := timeToUnix(cursor.CreatedAt)
		query += ` AND (c.created_at > ? OR (c.created_at = ? AND c.id > ?))`
		args = append(args, timestamp, timestamp, cursor.ID)
	}
	query += ` ORDER BY c.created_at ASC, c.id ASC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]*domain.Comment, 0)
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

func scanComment(row rowScanner) (*domain.Comment, error) {
	var (
		comment       domain.Comment
		author        domain.User
		createdAt     int64
		nickname      sql.NullString
		avatarMediaID sql.NullInt64
		isPrivate     int
	)
	if err := row.Scan(
		&comment.ID,
		&comment.PostID,
		&comment.AuthorUserID,
		&comment.Text,
		&createdAt,
		&author.ID,
		&author.FirstName,
		&author.LastName,
		&nickname,
		&avatarMediaID,
		&isPrivate,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	author.Nickname = stringFromNullString(nickname)
	if avatarMediaID.Valid {
		value := avatarMediaID.Int64
		author.AvatarMediaID = &value
	}
	if isPrivate != 0 && isPrivate != 1 {
		return nil, fmt.Errorf("invalid is_private value: %d", isPrivate)
	}
	author.IsPrivate = isPrivate == 1
	comment.Author = &author
	comment.CreatedAt = unixToTime(createdAt)
	return &comment, nil
}
