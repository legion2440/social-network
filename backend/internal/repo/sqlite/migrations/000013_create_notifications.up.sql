CREATE TABLE group_memberships_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  group_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('owner', 'member', 'invited', 'requested')),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  UNIQUE (group_id, user_id),
  FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO group_memberships_new (group_id, user_id, status, created_at, updated_at)
SELECT group_id, user_id, status, created_at, updated_at
FROM group_memberships
ORDER BY group_id ASC, user_id ASC;

DROP TABLE group_memberships;
ALTER TABLE group_memberships_new RENAME TO group_memberships;

CREATE INDEX idx_group_memberships_group_status_updated
ON group_memberships(group_id, status, updated_at ASC, user_id ASC);

CREATE INDEX idx_group_memberships_user_status_updated
ON group_memberships(user_id, status, updated_at ASC, group_id ASC);

CREATE INDEX idx_group_memberships_user_status_created
ON group_memberships(user_id, status, created_at ASC, group_id ASC);

CREATE TABLE notifications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  recipient_user_id INTEGER NOT NULL,
  actor_user_id INTEGER NOT NULL,
  type TEXT NOT NULL CHECK (type IN (
    'follow_started',
    'follow_request',
    'group_invitation',
    'group_join_request',
    'group_event'
  )),
  follow_id INTEGER NULL,
  group_id INTEGER NULL,
  event_id INTEGER NULL,
  membership_id INTEGER NULL,
  resolution TEXT NULL CHECK (resolution IS NULL OR resolution IN ('accepted', 'declined', 'cancelled')),
  resolved_at INTEGER NULL,
  read_at INTEGER NULL,
  created_at INTEGER NOT NULL,
  CHECK (actor_user_id != recipient_user_id),
  CHECK ((resolution IS NULL AND resolved_at IS NULL) OR (resolution IS NOT NULL AND resolved_at IS NOT NULL)),
  CHECK (type IN ('follow_request', 'group_invitation', 'group_join_request') OR resolution IS NULL),
  CHECK (
    (type IN ('follow_started', 'follow_request') AND group_id IS NULL AND event_id IS NULL AND membership_id IS NULL) OR
    (type IN ('group_invitation', 'group_join_request') AND follow_id IS NULL AND event_id IS NULL) OR
    (type = 'group_event' AND follow_id IS NULL AND membership_id IS NULL)
  ),
  FOREIGN KEY (recipient_user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (follow_id) REFERENCES follows(id) ON DELETE SET NULL,
  FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL,
  FOREIGN KEY (event_id) REFERENCES group_events(id) ON DELETE SET NULL,
  FOREIGN KEY (membership_id) REFERENCES group_memberships(id) ON DELETE SET NULL
);

CREATE INDEX idx_notifications_recipient_created
ON notifications(recipient_user_id, created_at DESC, id DESC);

CREATE INDEX idx_notifications_recipient_unread
ON notifications(recipient_user_id, id)
WHERE read_at IS NULL;

CREATE INDEX idx_notifications_type_follow
ON notifications(type, follow_id);

CREATE INDEX idx_notifications_type_group_actor
ON notifications(type, group_id, actor_user_id);

CREATE INDEX idx_notifications_type_event
ON notifications(type, event_id);

CREATE UNIQUE INDEX idx_notifications_unique_follow_lifecycle
ON notifications(follow_id)
WHERE follow_id IS NOT NULL AND type IN ('follow_started', 'follow_request');

CREATE UNIQUE INDEX idx_notifications_unique_event_recipient
ON notifications(recipient_user_id, type, event_id)
WHERE event_id IS NOT NULL AND type = 'group_event';

CREATE UNIQUE INDEX idx_notifications_unique_membership_lifecycle
ON notifications(membership_id)
WHERE membership_id IS NOT NULL AND type IN ('group_invitation', 'group_join_request');

CREATE UNIQUE INDEX idx_notifications_one_pending_invitation
ON notifications(recipient_user_id, type, group_id)
WHERE type = 'group_invitation' AND resolution IS NULL AND group_id IS NOT NULL;

CREATE UNIQUE INDEX idx_notifications_one_pending_join_request
ON notifications(recipient_user_id, type, group_id, actor_user_id)
WHERE type = 'group_join_request' AND resolution IS NULL AND group_id IS NOT NULL;

CREATE TABLE notification_user_states (
  user_id INTEGER PRIMARY KEY,
  revision INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO notification_user_states (user_id, revision)
SELECT id, 0 FROM users;
