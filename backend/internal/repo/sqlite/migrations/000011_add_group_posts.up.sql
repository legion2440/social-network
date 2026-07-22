CREATE TABLE posts_000011 (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  author_user_id INTEGER NOT NULL,
  group_id INTEGER,
  text TEXT NOT NULL CHECK (length(text) BETWEEN 1 AND 5000 AND text = trim(text)),
  privacy TEXT,
  media_id INTEGER,
  created_at INTEGER NOT NULL,
  CHECK (
    (group_id IS NULL AND privacy IS NOT NULL AND privacy IN ('public', 'followers', 'selected'))
    OR (group_id IS NOT NULL AND privacy IS NULL)
  ),
  FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
  FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE SET NULL
);

INSERT INTO posts_000011 (id, author_user_id, group_id, text, privacy, media_id, created_at)
SELECT id, author_user_id, NULL, text, privacy, media_id, created_at
FROM posts;

CREATE TABLE post_selected_users_000011_backup AS
SELECT post_id, user_id FROM post_selected_users;

CREATE TABLE post_comments_000011_backup AS
SELECT id, post_id, author_user_id, text, created_at FROM post_comments;

DROP TABLE post_selected_users;
DROP TABLE post_comments;
DROP TABLE posts;
ALTER TABLE posts_000011 RENAME TO posts;

CREATE INDEX idx_posts_created
ON posts(created_at DESC, id DESC);

CREATE INDEX idx_posts_author_created
ON posts(author_user_id, created_at DESC, id DESC);

CREATE INDEX idx_posts_group_created
ON posts(group_id, created_at DESC, id DESC)
WHERE group_id IS NOT NULL;

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

INSERT INTO post_selected_users (post_id, user_id)
SELECT post_id, user_id FROM post_selected_users_000011_backup;

DROP TABLE post_selected_users_000011_backup;

CREATE INDEX idx_post_selected_users_user_post
ON post_selected_users(user_id, post_id);

CREATE TABLE post_comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  post_id INTEGER NOT NULL,
  author_user_id INTEGER NOT NULL,
  text TEXT NOT NULL CHECK (length(text) BETWEEN 1 AND 5000 AND text = trim(text)),
  created_at INTEGER NOT NULL,
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO post_comments (id, post_id, author_user_id, text, created_at)
SELECT id, post_id, author_user_id, text, created_at
FROM post_comments_000011_backup;

DROP TABLE post_comments_000011_backup;

CREATE INDEX idx_post_comments_post_created
ON post_comments(post_id, created_at ASC, id ASC);
