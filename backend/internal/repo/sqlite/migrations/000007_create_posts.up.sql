CREATE TABLE posts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  author_user_id INTEGER NOT NULL,
  text TEXT NOT NULL CHECK (length(text) BETWEEN 1 AND 5000 AND text = trim(text)),
  privacy TEXT NOT NULL CHECK (privacy IN ('public', 'followers', 'selected')),
  media_id INTEGER,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE SET NULL
);

CREATE INDEX idx_posts_created
ON posts(created_at DESC, id DESC);

CREATE INDEX idx_posts_author_created
ON posts(author_user_id, created_at DESC, id DESC);

CREATE UNIQUE INDEX idx_posts_media_unique
ON posts(media_id)
WHERE media_id IS NOT NULL;

CREATE TABLE post_selected_users (
  post_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  UNIQUE (post_id, user_id),
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_post_selected_users_user_post
ON post_selected_users(user_id, post_id);
