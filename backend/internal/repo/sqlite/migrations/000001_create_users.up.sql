CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL COLLATE NOCASE CHECK (LENGTH(TRIM(email)) > 0),
  password_hash TEXT NOT NULL CHECK (LENGTH(password_hash) > 0),
  first_name TEXT NOT NULL CHECK (LENGTH(TRIM(first_name)) > 0),
  last_name TEXT NOT NULL CHECK (LENGTH(TRIM(last_name)) > 0),
  date_of_birth TEXT NOT NULL CHECK (
    TYPEOF(date_of_birth) = 'text'
    AND LENGTH(date_of_birth) = 10
    AND date_of_birth GLOB '[0-9][0-9]-[0-9][0-9]-[0-9][0-9][0-9][0-9]'
    AND CAST(SUBSTR(date_of_birth, 7, 4) AS INTEGER) BETWEEN 1 AND 9999
    AND CAST(SUBSTR(date_of_birth, 4, 2) AS INTEGER) BETWEEN 1 AND 12
    AND CAST(SUBSTR(date_of_birth, 1, 2) AS INTEGER) BETWEEN 1 AND CASE
      WHEN CAST(SUBSTR(date_of_birth, 4, 2) AS INTEGER) IN (1, 3, 5, 7, 8, 10, 12) THEN 31
      WHEN CAST(SUBSTR(date_of_birth, 4, 2) AS INTEGER) IN (4, 6, 9, 11) THEN 30
      WHEN (
        CAST(SUBSTR(date_of_birth, 7, 4) AS INTEGER) % 400 = 0
        OR (
          CAST(SUBSTR(date_of_birth, 7, 4) AS INTEGER) % 4 = 0
          AND CAST(SUBSTR(date_of_birth, 7, 4) AS INTEGER) % 100 != 0
        )
      ) THEN 29
      ELSE 28
    END
  ),
  gender TEXT CHECK (gender IS NULL OR gender IN ('male', 'female')),
  nickname TEXT,
  about_me TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_users_email_nocase ON users(email COLLATE NOCASE);
