package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

type MediaRepo struct {
	db sqlExecutor
}

func NewMediaRepo(db *sql.DB) *MediaRepo {
	return &MediaRepo{db: db}
}

func (r *MediaRepo) Create(ctx context.Context, ownerUserID int64, mime string, size int64, storageKey, originalName string, createdAt time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO media (owner_user_id, mime, size, storage_key, original_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, ownerUserID, strings.TrimSpace(mime), size, strings.TrimSpace(storageKey), strings.TrimSpace(originalName), timeToUnix(createdAt))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *MediaRepo) GetByID(ctx context.Context, id int64) (*domain.Media, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, owner_user_id, mime, size, storage_key, original_name, created_at
		FROM media
		WHERE id = ?
	`, id)

	var media domain.Media
	var createdAt int64
	if err := row.Scan(&media.ID, &media.OwnerUserID, &media.MIME, &media.Size, &media.StorageKey, &media.OriginalName, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	media.MIME = strings.TrimSpace(media.MIME)
	media.StorageKey = strings.TrimSpace(media.StorageKey)
	media.OriginalName = strings.TrimSpace(media.OriginalName)
	media.CreatedAt = unixToTime(createdAt)
	media.URL = domain.MediaURL(media.ID)
	return &media, nil
}
