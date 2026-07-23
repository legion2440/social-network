DROP INDEX IF EXISTS idx_group_chat_read_states_last_read;
DROP TABLE IF EXISTS group_chat_read_states;

DROP INDEX IF EXISTS idx_direct_chat_read_states_last_read;
DROP INDEX IF EXISTS idx_direct_chat_read_states_conversation_user;
DROP TABLE IF EXISTS direct_chat_read_states;

DROP TABLE IF EXISTS chat_user_states;
