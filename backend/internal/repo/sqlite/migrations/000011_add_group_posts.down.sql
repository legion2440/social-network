CREATE TABLE autoincrement_000011_down_backup (
  name TEXT PRIMARY KEY,
  seq INTEGER NOT NULL
);

INSERT INTO autoincrement_000011_down_backup (name, seq)
SELECT name, seq
FROM sqlite_sequence
WHERE name IN ('posts', 'post_comments');

CREATE TABLE posts_000011_down (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  author_user_id INTEGER NOT NULL,
  text TEXT NOT NULL CHECK (length(text) BETWEEN 1 AND 5000 AND text = trim(text)),
  privacy TEXT NOT NULL CHECK (privacy IN ('public', 'followers', 'selected')),
  media_id INTEGER,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE SET NULL
);

INSERT INTO posts_000011_down (id, author_user_id, text, privacy, media_id, created_at)
SELECT id, author_user_id, text, privacy, media_id, created_at
FROM posts;

CREATE TABLE post_selected_users_000011_down_backup AS
SELECT post_id, user_id FROM post_selected_users;

CREATE TABLE post_comments_000011_down_backup AS
SELECT id, post_id, author_user_id, text, created_at FROM post_comments;

DROP TABLE post_selected_users;
DROP TABLE post_comments;
DROP TABLE posts;
ALTER TABLE posts_000011_down RENAME TO posts;

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

INSERT INTO post_selected_users (post_id, user_id)
SELECT post_id, user_id FROM post_selected_users_000011_down_backup;

DROP TABLE post_selected_users_000011_down_backup;

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
FROM post_comments_000011_down_backup;

DROP TABLE post_comments_000011_down_backup;

CREATE INDEX idx_post_comments_post_created
ON post_comments(post_id, created_at ASC, id ASC);

UPDATE sqlite_sequence
SET seq = max(
  seq,
  COALESCE((SELECT saved.seq FROM autoincrement_000011_down_backup saved WHERE saved.name = 'posts'), 0),
  COALESCE((SELECT MAX(id) FROM posts), 0)
)
WHERE name = 'posts';

INSERT INTO sqlite_sequence (name, seq)
SELECT
  'posts',
  max(
    COALESCE((SELECT saved.seq FROM autoincrement_000011_down_backup saved WHERE saved.name = 'posts'), 0),
    COALESCE((SELECT MAX(id) FROM posts), 0)
  )
WHERE NOT EXISTS (SELECT 1 FROM sqlite_sequence WHERE name = 'posts')
  AND (
    EXISTS (SELECT 1 FROM autoincrement_000011_down_backup WHERE name = 'posts')
    OR EXISTS (SELECT 1 FROM posts)
  );

UPDATE sqlite_sequence
SET seq = max(
  seq,
  COALESCE((SELECT saved.seq FROM autoincrement_000011_down_backup saved WHERE saved.name = 'post_comments'), 0),
  COALESCE((SELECT MAX(id) FROM post_comments), 0)
)
WHERE name = 'post_comments';

INSERT INTO sqlite_sequence (name, seq)
SELECT
  'post_comments',
  max(
    COALESCE((SELECT saved.seq FROM autoincrement_000011_down_backup saved WHERE saved.name = 'post_comments'), 0),
    COALESCE((SELECT MAX(id) FROM post_comments), 0)
  )
WHERE NOT EXISTS (SELECT 1 FROM sqlite_sequence WHERE name = 'post_comments')
  AND (
    EXISTS (SELECT 1 FROM autoincrement_000011_down_backup WHERE name = 'post_comments')
    OR EXISTS (SELECT 1 FROM post_comments)
  );

DROP TABLE autoincrement_000011_down_backup;
