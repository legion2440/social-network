DROP INDEX IF EXISTS idx_users_avatar_media_id;

ALTER TABLE users DROP COLUMN avatar_media_id;
