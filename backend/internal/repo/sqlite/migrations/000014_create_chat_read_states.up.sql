CREATE TABLE chat_user_states (
  user_id INTEGER PRIMARY KEY,
  revision INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO chat_user_states (user_id, revision)
SELECT id, 0
FROM users;

CREATE TABLE direct_chat_read_states (
  user_id INTEGER NOT NULL,
  direct_conversation_id INTEGER NOT NULL,
  last_read_message_id INTEGER NULL,
  unread_count INTEGER NOT NULL DEFAULT 0 CHECK (unread_count >= 0),
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (user_id, direct_conversation_id),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (direct_conversation_id) REFERENCES direct_conversations(id) ON DELETE CASCADE,
  FOREIGN KEY (last_read_message_id) REFERENCES chat_messages(id) ON DELETE SET NULL
);

INSERT INTO direct_chat_read_states (
  user_id,
  direct_conversation_id,
  last_read_message_id,
  unread_count,
  updated_at
)
SELECT
  participant.user_id,
  participant.conversation_id,
  (
    SELECT latest.id
    FROM chat_messages latest
    WHERE latest.direct_conversation_id = participant.conversation_id
    ORDER BY latest.created_at DESC, latest.id DESC
    LIMIT 1
  ),
  0,
  COALESCE((
    SELECT latest.created_at
    FROM chat_messages latest
    WHERE latest.direct_conversation_id = participant.conversation_id
    ORDER BY latest.created_at DESC, latest.id DESC
    LIMIT 1
  ), participant.created_at)
FROM (
  SELECT id AS conversation_id, user_low_id AS user_id, created_at
  FROM direct_conversations
  UNION ALL
  SELECT id AS conversation_id, user_high_id AS user_id, created_at
  FROM direct_conversations
) participant;

CREATE INDEX idx_direct_chat_read_states_conversation_user
ON direct_chat_read_states(direct_conversation_id, user_id);

CREATE INDEX idx_direct_chat_read_states_last_read
ON direct_chat_read_states(last_read_message_id);

CREATE TABLE group_chat_read_states (
  membership_id INTEGER PRIMARY KEY,
  last_read_message_id INTEGER NULL,
  unread_count INTEGER NOT NULL DEFAULT 0 CHECK (unread_count >= 0),
  updated_at INTEGER NOT NULL,
  FOREIGN KEY (membership_id) REFERENCES group_memberships(id) ON DELETE CASCADE,
  FOREIGN KEY (last_read_message_id) REFERENCES chat_messages(id) ON DELETE SET NULL
);

INSERT INTO group_chat_read_states (
  membership_id,
  last_read_message_id,
  unread_count,
  updated_at
)
SELECT
  membership.id,
  (
    SELECT latest.id
    FROM chat_messages latest
    WHERE latest.group_id = membership.group_id
    ORDER BY latest.created_at DESC, latest.id DESC
    LIMIT 1
  ),
  0,
  COALESCE((
    SELECT latest.created_at
    FROM chat_messages latest
    WHERE latest.group_id = membership.group_id
    ORDER BY latest.created_at DESC, latest.id DESC
    LIMIT 1
  ), membership.updated_at)
FROM group_memberships membership
WHERE membership.status IN ('owner', 'member');

CREATE INDEX idx_group_chat_read_states_last_read
ON group_chat_read_states(last_read_message_id);
