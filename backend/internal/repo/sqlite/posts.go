package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

type PostRepo struct {
	db sqlExecutor
}

func NewPostRepo(db *sql.DB) *PostRepo {
	return &PostRepo{db: db}
}

const rawPostSelectColumns = `
	p.id, p.author_user_id, p.group_id, p.text, p.privacy, p.media_id,
	(SELECT COUNT(*) FROM post_comments comment_count WHERE comment_count.post_id = p.id),
	p.created_at,
	u.id, u.first_name, u.last_name, u.gender, u.nickname,
	u.avatar_media_id, u.is_private
`

const viewerPostSelectColumns = `
	p.id, p.author_user_id, p.group_id, p.text, p.privacy, p.media_id,
	(SELECT COUNT(*) FROM post_comments comment_count WHERE comment_count.post_id = p.id),
	p.created_at,
	u.id, u.first_name, u.last_name, u.gender, u.nickname,
	CASE
		WHEN u.avatar_media_id IS NULL THEN NULL
		WHEN u.id = ? OR u.is_private = 0 OR EXISTS (
			SELECT 1 FROM follows avatar_follow
			WHERE avatar_follow.follower_user_id = ?
				AND avatar_follow.followed_user_id = u.id
				AND avatar_follow.status = 'accepted'
		) THEN u.avatar_media_id
		ELSE NULL
	END,
	u.is_private
`

// Every personal post read uses this predicate. viewer_follow is the current
// accepted relation from the viewer to the post author, joined by each query.
const personalPostAccessPredicate = `
	(
		p.author_user_id = ?
		OR (
			(u.is_private = 0 OR viewer_follow.id IS NOT NULL)
			AND (
				p.privacy = 'public'
				OR (p.privacy = 'followers' AND viewer_follow.id IS NOT NULL)
				OR (
					p.privacy = 'selected'
					AND viewer_follow.id IS NOT NULL
					AND EXISTS (
						SELECT 1
						FROM post_selected_users audience
						WHERE audience.post_id = p.id AND audience.user_id = ?
					)
				)
			)
		)
	)
`

const postFromAndViewerJoin = `
	FROM posts p
	JOIN users u ON u.id = p.author_user_id
	LEFT JOIN follows viewer_follow
		ON viewer_follow.follower_user_id = ?
		AND viewer_follow.followed_user_id = p.author_user_id
		AND viewer_follow.status = 'accepted'
`

func (r *PostRepo) Create(ctx context.Context, post *domain.Post) (int64, error) {
	if r == nil || r.db == nil || !validPostForCreate(post) {
		return 0, fmt.Errorf("invalid post")
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO posts (author_user_id, group_id, text, privacy, media_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, post.AuthorUserID, post.GroupID, post.Text, post.Privacy, post.MediaID, timeToUnix(post.CreatedAt))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func validPostForCreate(post *domain.Post) bool {
	if post == nil || post.AuthorUserID <= 0 || post.Text == "" || post.CreatedAt.IsZero() {
		return false
	}
	if post.GroupID == nil {
		return post.Privacy != nil && post.Privacy.Valid()
	}
	return *post.GroupID > 0 && post.Privacy == nil
}

func (r *PostRepo) AddSelectedUsers(ctx context.Context, postID int64, userIDs []int64) error {
	if r == nil || r.db == nil || postID <= 0 {
		return fmt.Errorf("invalid post audience")
	}
	var selected int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM posts
		WHERE id = ? AND group_id IS NULL AND privacy = 'selected'
	`, postID).Scan(&selected); err != nil {
		return err
	}
	if selected != 1 {
		return fmt.Errorf("post %d cannot have a selected audience", postID)
	}
	for _, userID := range userIDs {
		if userID <= 0 {
			return fmt.Errorf("invalid post audience user")
		}
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO post_selected_users (post_id, user_id)
			VALUES (?, ?)
		`, postID, userID); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostRepo) GetByID(ctx context.Context, postID int64) (*domain.Post, error) {
	if r == nil || r.db == nil || postID <= 0 {
		return nil, repo.ErrNotFound
	}
	query := `SELECT ` + rawPostSelectColumns + `
		FROM posts p
		JOIN users u ON u.id = p.author_user_id
		WHERE p.id = ?`
	return scanPost(r.db.QueryRowContext(ctx, query, postID))
}

func (r *PostRepo) GetAccessibleByID(ctx context.Context, viewerUserID, postID int64) (*domain.Post, error) {
	if r == nil || r.db == nil || viewerUserID <= 0 || postID <= 0 {
		return nil, repo.ErrNotFound
	}
	query := `SELECT ` + viewerPostSelectColumns + postFromAndViewerJoin + `
		WHERE p.id = ? AND (
			(p.group_id IS NULL AND ` + personalPostAccessPredicate + `)
			OR (
				p.group_id IS NOT NULL
				AND EXISTS (
					SELECT 1 FROM group_memberships content_member
					WHERE content_member.group_id = p.group_id
						AND content_member.user_id = ?
						AND content_member.status IN ('owner', 'member')
				)
			)
		)`
	return scanPost(r.db.QueryRowContext(
		ctx, query,
		viewerUserID, viewerUserID,
		viewerUserID,
		postID,
		viewerUserID, viewerUserID,
		viewerUserID,
	))
}

func (r *PostRepo) ListFeed(
	ctx context.Context,
	viewerUserID int64,
	cursor *domain.PostCursor,
	limit int,
) ([]*domain.Post, error) {
	if r == nil || r.db == nil || viewerUserID <= 0 || limit <= 0 {
		return []*domain.Post{}, nil
	}
	query := `SELECT ` + viewerPostSelectColumns + postFromAndViewerJoin + `
		WHERE p.group_id IS NULL
		AND (p.author_user_id = ? OR viewer_follow.id IS NOT NULL)
		AND ` + personalPostAccessPredicate
	args := []any{
		viewerUserID, viewerUserID,
		viewerUserID,
		viewerUserID,
		viewerUserID, viewerUserID,
	}
	query, args = appendPostCursor(query, args, cursor)
	query += ` ORDER BY p.created_at DESC, p.id DESC LIMIT ?`
	args = append(args, limit)
	return r.list(ctx, query, args...)
}

func (r *PostRepo) ListByAuthor(
	ctx context.Context,
	viewerUserID, authorUserID int64,
	cursor *domain.PostCursor,
	limit int,
) ([]*domain.Post, error) {
	if r == nil || r.db == nil || viewerUserID <= 0 || authorUserID <= 0 || limit <= 0 {
		return []*domain.Post{}, nil
	}
	query := `SELECT ` + viewerPostSelectColumns + postFromAndViewerJoin + `
		WHERE p.group_id IS NULL AND p.author_user_id = ? AND ` + personalPostAccessPredicate
	args := []any{
		viewerUserID, viewerUserID,
		viewerUserID,
		authorUserID,
		viewerUserID, viewerUserID,
	}
	query, args = appendPostCursor(query, args, cursor)
	query += ` ORDER BY p.created_at DESC, p.id DESC LIMIT ?`
	args = append(args, limit)
	return r.list(ctx, query, args...)
}

func (r *PostRepo) ListByGroup(
	ctx context.Context,
	viewerUserID, groupID int64,
	cursor *domain.PostCursor,
	limit int,
) ([]*domain.Post, error) {
	if r == nil || r.db == nil || viewerUserID <= 0 || groupID <= 0 || limit <= 0 {
		return []*domain.Post{}, nil
	}
	query := `SELECT ` + viewerPostSelectColumns + `
		FROM posts p
		JOIN users u ON u.id = p.author_user_id
		WHERE p.group_id = ?
		AND EXISTS (
			SELECT 1 FROM group_memberships content_member
			WHERE content_member.group_id = p.group_id
				AND content_member.user_id = ?
				AND content_member.status IN ('owner', 'member')
		)`
	args := []any{viewerUserID, viewerUserID, groupID, viewerUserID}
	query, args = appendPostCursor(query, args, cursor)
	query += ` ORDER BY p.created_at DESC, p.id DESC LIMIT ?`
	args = append(args, limit)
	return r.list(ctx, query, args...)
}

func (r *PostRepo) CountAccessibleByAuthor(ctx context.Context, viewerUserID, authorUserID int64) (int64, error) {
	if r == nil || r.db == nil || viewerUserID <= 0 || authorUserID <= 0 {
		return 0, nil
	}
	query := `SELECT COUNT(*)` + postFromAndViewerJoin + `
		WHERE p.group_id IS NULL AND p.author_user_id = ? AND ` + personalPostAccessPredicate
	var count int64
	if err := r.db.QueryRowContext(
		ctx,
		query,
		viewerUserID,
		authorUserID,
		viewerUserID,
		viewerUserID,
	).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func appendPostCursor(query string, args []any, cursor *domain.PostCursor) (string, []any) {
	if cursor == nil {
		return query, args
	}
	timestamp := timeToUnix(cursor.CreatedAt)
	query += ` AND (p.created_at < ? OR (p.created_at = ? AND p.id < ?))`
	return query, append(args, timestamp, timestamp, cursor.ID)
}

func (r *PostRepo) list(ctx context.Context, query string, args ...any) ([]*domain.Post, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]*domain.Post, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

func scanPost(row rowScanner) (*domain.Post, error) {
	var (
		post          domain.Post
		author        domain.User
		groupID       sql.NullInt64
		privacy       sql.NullString
		mediaID       sql.NullInt64
		createdAt     int64
		gender        sql.NullString
		nickname      sql.NullString
		avatarMediaID sql.NullInt64
		isPrivate     int
	)
	if err := row.Scan(
		&post.ID,
		&post.AuthorUserID,
		&groupID,
		&post.Text,
		&privacy,
		&mediaID,
		&post.CommentsCount,
		&createdAt,
		&author.ID,
		&author.FirstName,
		&author.LastName,
		&gender,
		&nickname,
		&avatarMediaID,
		&isPrivate,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}

	if groupID.Valid {
		value := groupID.Int64
		post.GroupID = &value
	}
	if privacy.Valid {
		value := domain.PostPrivacy(privacy.String)
		if !value.Valid() {
			return nil, fmt.Errorf("invalid post privacy: %q", privacy.String)
		}
		post.Privacy = &value
	}
	if (post.GroupID == nil) == (post.Privacy == nil) {
		return nil, fmt.Errorf("invalid personal/group post state")
	}
	if mediaID.Valid {
		value := mediaID.Int64
		post.MediaID = &value
	}
	post.CreatedAt = unixToTime(createdAt)

	parsedGender, err := genderFromNullString(gender)
	if err != nil {
		return nil, err
	}
	author.Gender = parsedGender
	author.Nickname = stringFromNullString(nickname)
	if avatarMediaID.Valid {
		value := avatarMediaID.Int64
		author.AvatarMediaID = &value
	}
	if isPrivate != 0 && isPrivate != 1 {
		return nil, fmt.Errorf("invalid is_private value: %d", isPrivate)
	}
	author.IsPrivate = isPrivate == 1
	post.Author = &author
	return &post, nil
}
