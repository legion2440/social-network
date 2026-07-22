CREATE TABLE group_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  group_id INTEGER NOT NULL,
  creator_user_id INTEGER NOT NULL,
  title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 100 AND title = trim(title)),
  description TEXT NOT NULL CHECK (length(description) BETWEEN 1 AND 2000 AND description = trim(description)),
  starts_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
  FOREIGN KEY (creator_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_group_events_group_starts
ON group_events(group_id, starts_at ASC, id ASC);

CREATE TABLE group_event_responses (
  event_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  response TEXT NOT NULL CHECK (response IN ('going', 'not_going')),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (event_id, user_id),
  FOREIGN KEY (event_id) REFERENCES group_events(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_group_event_responses_user_event
ON group_event_responses(user_id, event_id);
