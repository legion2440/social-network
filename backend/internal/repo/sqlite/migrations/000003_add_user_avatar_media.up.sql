ALTER TABLE users
ADD COLUMN avatar_media_id INTEGER REFERENCES media(id) ON DELETE SET NULL;

CREATE INDEX idx_users_avatar_media_id ON users(avatar_media_id);
