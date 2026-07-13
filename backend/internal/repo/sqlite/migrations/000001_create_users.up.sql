CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL COLLATE NOCASE CHECK (LENGTH(TRIM(email)) > 0),
  password_hash TEXT NOT NULL CHECK (LENGTH(password_hash) > 0),
  first_name TEXT NOT NULL CHECK (LENGTH(TRIM(first_name)) > 0),
  last_name TEXT NOT NULL CHECK (LENGTH(TRIM(last_name)) > 0),
  date_of_birth INTEGER NOT NULL,
  gender TEXT CHECK (gender IS NULL OR gender IN ('male', 'female')),
  nickname TEXT,
  about_me TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_users_email_nocase ON users(email COLLATE NOCASE);
