DROP INDEX IF EXISTS idx_post_comments_media;

ALTER TABLE post_comments DROP COLUMN media_id;
