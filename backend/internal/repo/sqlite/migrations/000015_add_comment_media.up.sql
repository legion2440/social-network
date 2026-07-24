ALTER TABLE post_comments
ADD COLUMN media_id INTEGER REFERENCES media(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX idx_post_comments_media
ON post_comments(media_id)
WHERE media_id IS NOT NULL;
