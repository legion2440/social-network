const test = require('node:test');
const assert = require('node:assert/strict');

const ChatModel = require('./chat-model.js');

function rawMessage(id, clientMessageID, kind, targetID, senderID, createdAt) {
  return {
    id,
    client_message_id: clientMessageID,
    chat: { kind, target_id: targetID },
    sender: {
      id: senderID,
      first_name: 'User',
      last_name: String(senderID),
      avatar_url: '/static/avatars/neutral.svg'
    },
    body: 'message ' + id,
    created_at: createdAt || '2026-07-22T10:00:00Z'
  };
}

test('chat keys are strict and reversible', () => {
  assert.equal(ChatModel.chatKey('direct', 15), 'direct:15');
  assert.deepEqual(ChatModel.parseChatKey('group:7'), { kind: 'group', target_id: 7 });
  assert.equal(ChatModel.parseChatKey('group:nope'), null);
  assert.throws(() => ChatModel.chatKey('unknown', 1), /valid chat/);
});

test('authoritative message replaces optimistic pending by client_message_id', () => {
  const clientID = '47cd9266-b43f-4a89-9338-4f9c197ff12a';
  const pending = ChatModel.pendingMessage(
    clientID, { kind: 'direct', target_id: 2 }, 1, 'message 8', '2026-07-22T09:59:59Z'
  );
  const authoritative = ChatModel.normalizeMessage(rawMessage(8, clientID, 'direct', 2, 1));

  const merged = ChatModel.mergeMessages([pending], [authoritative, authoritative]);
  assert.equal(merged.length, 1);
  assert.equal(merged[0].apiId, 8);
  assert.equal(merged[0].pending, false);
});

test('HTTP and WebSocket copies deduplicate and keep chronological order', () => {
  const first = ChatModel.normalizeMessage(rawMessage(
    1, 'e641ac02-ed21-44a8-a1c4-46094b03ecfa', 'group', 7, 2, '2026-07-22T10:00:00Z'
  ));
  const second = ChatModel.normalizeMessage(rawMessage(
    2, 'cb493e86-ecb4-4f98-b666-4beedb4c4909', 'group', 7, 3, '2026-07-22T10:00:01Z'
  ));

  assert.deepEqual(
    ChatModel.mergeMessages([second], [first, second]).map(message => message.apiId),
    [1, 2]
  );
});

test('chat summaries sort by activity, direct rank, and descending target id', () => {
  const store = ChatModel.mergeChatSummaries({}, [
    { key: 'group:9', kind: 'group', targetID: 9, activityAt: '2026-07-22T10:00:00Z' },
    { key: 'direct:2', kind: 'direct', targetID: 2, activityAt: '2026-07-22T10:00:00Z' },
    { key: 'direct:5', kind: 'direct', targetID: 5, activityAt: '2026-07-22T10:00:00Z' },
    { key: 'group:3', kind: 'group', targetID: 3, activityAt: '2026-07-22T10:00:01Z' }
  ]);
  assert.deepEqual(ChatModel.sortedChatKeys(store), ['group:3', 'direct:5', 'direct:2', 'group:9']);
});

test('older HTTP chat list cannot overwrite a newer local last message', () => {
  const newer = {
    key: 'direct:2', kind: 'direct', targetID: 2,
    activityAt: '2026-07-22T10:00:02Z', lastMessage: { apiId: 2 }
  };
  const older = {
    key: 'direct:2', kind: 'direct', targetID: 2,
    activityAt: '2026-07-22T10:00:01Z', lastMessage: { apiId: 1 }
  };
  const merged = ChatModel.mergeChatSummaries({ 'direct:2': newer }, [older]);
  assert.equal(merged['direct:2'].lastMessage.apiId, 2);
  assert.equal(merged['direct:2'].activityAt, newer.activityAt);
});
