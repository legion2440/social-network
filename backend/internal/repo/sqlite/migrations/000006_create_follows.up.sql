CREATE TABLE follows (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  follower_user_id INTEGER NOT NULL,
  followed_user_id INTEGER NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'accepted')),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  CHECK (follower_user_id != followed_user_id),
  UNIQUE (follower_user_id, followed_user_id),
  FOREIGN KEY (follower_user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (followed_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_follows_followed_status_updated
ON follows(followed_user_id, status, updated_at DESC, id DESC);

CREATE INDEX idx_follows_follower_status_updated
ON follows(follower_user_id, status, updated_at DESC, id DESC);
