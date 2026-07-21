CREATE TABLE groups (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_user_id INTEGER NOT NULL,
  title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 100 AND title = trim(title)),
  description TEXT NOT NULL CHECK (length(description) BETWEEN 1 AND 2000 AND description = trim(description)),
  created_at INTEGER NOT NULL,
  FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_groups_created
ON groups(created_at DESC, id DESC);

CREATE TABLE group_memberships (
  group_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('owner', 'member', 'invited', 'requested')),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (group_id, user_id),
  FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_group_memberships_group_status_updated
ON group_memberships(group_id, status, updated_at ASC, user_id ASC);

CREATE INDEX idx_group_memberships_user_status_updated
ON group_memberships(user_id, status, updated_at ASC, group_id ASC);

CREATE INDEX idx_group_memberships_user_status_created
ON group_memberships(user_id, status, created_at ASC, group_id ASC);
