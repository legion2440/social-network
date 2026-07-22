CREATE TABLE direct_conversations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_low_id INTEGER NOT NULL,
  user_high_id INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  CHECK (user_low_id < user_high_id),
  UNIQUE (user_low_id, user_high_id),
  FOREIGN KEY (user_low_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (user_high_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE chat_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  direct_conversation_id INTEGER,
  group_id INTEGER,
  sender_user_id INTEGER NOT NULL,
  client_message_id TEXT NOT NULL,
  body TEXT NOT NULL CHECK (length(body) BETWEEN 1 AND 2000 AND body = trim(body)),
  created_at INTEGER NOT NULL,
  CHECK ((direct_conversation_id IS NOT NULL) != (group_id IS NOT NULL)),
  UNIQUE (sender_user_id, client_message_id),
  FOREIGN KEY (direct_conversation_id) REFERENCES direct_conversations(id) ON DELETE CASCADE,
  FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
  FOREIGN KEY (sender_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_chat_messages_direct_created
ON chat_messages(direct_conversation_id, created_at DESC, id DESC)
WHERE direct_conversation_id IS NOT NULL;

CREATE INDEX idx_chat_messages_group_created
ON chat_messages(group_id, created_at DESC, id DESC)
WHERE group_id IS NOT NULL;

CREATE INDEX idx_chat_messages_sender_client
ON chat_messages(sender_user_id, client_message_id);
