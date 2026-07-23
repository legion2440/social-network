DROP TABLE IF EXISTS notification_user_states;
DROP TABLE IF EXISTS notifications;

CREATE TABLE group_memberships_old (
  group_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('owner', 'member', 'invited', 'requested')),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (group_id, user_id),
  FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO group_memberships_old (group_id, user_id, status, created_at, updated_at)
SELECT group_id, user_id, status, created_at, updated_at
FROM group_memberships
ORDER BY group_id ASC, user_id ASC;

DROP TABLE group_memberships;
ALTER TABLE group_memberships_old RENAME TO group_memberships;

CREATE INDEX idx_group_memberships_group_status_updated
ON group_memberships(group_id, status, updated_at ASC, user_id ASC);

CREATE INDEX idx_group_memberships_user_status_updated
ON group_memberships(user_id, status, updated_at ASC, group_id ASC);

CREATE INDEX idx_group_memberships_user_status_created
ON group_memberships(user_id, status, created_at ASC, group_id ASC);
