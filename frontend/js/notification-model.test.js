const test = require('node:test');
const assert = require('node:assert/strict');

const NotificationModel = require('./notification-model.js');

function rawNotification(id, type, options) {
  options = options || {};
  return {
    id,
    type,
    actor: { id: options.actorID || 2 },
    follow_id: options.followID == null ? null : options.followID,
    group: options.groupID ? { id: options.groupID, title: 'Group' } : null,
    event: null,
    resolution: options.resolution || null,
    resolved_at: null,
    read_at: options.readAt || null,
    created_at: options.createdAt || '2026-07-23T10:00:00Z'
  };
}

test('normalizes and merges authoritative notification DTOs by descending cursor order', () => {
  const oldItem = NotificationModel.normalize(rawNotification(4, 'follow_started'));
  const newItem = NotificationModel.normalize(rawNotification(5, 'follow_request', {
    createdAt: '2026-07-23T11:00:00Z'
  }));
  const replacement = Object.assign({}, oldItem, { readAt: '2026-07-23T12:00:00Z' });
  const merged = NotificationModel.merge([oldItem], [newItem, replacement]);
  assert.deepEqual(merged.map(item => item.id), [5, 4]);
  assert.equal(merged[1].readAt, '2026-07-23T12:00:00Z');
});

test('logical lifecycle keys use notification source identity while the id identifies a lifecycle', () => {
  assert.equal(NotificationModel.sourceKey(NotificationModel.normalize(rawNotification(1, 'follow_request', { actorID: 7 }))), 'follow_request:7');
  assert.equal(NotificationModel.sourceKey(NotificationModel.normalize(rawNotification(2, 'group_invitation', { groupID: 9 }))), 'group_invitation:9');
  assert.equal(NotificationModel.sourceKey(NotificationModel.normalize(rawNotification(3, 'group_join_request', { groupID: 9, actorID: 7 }))), 'group_join_request:9:7');
  assert.equal(NotificationModel.sourceKey(NotificationModel.normalize(rawNotification(4, 'group_invitation', { groupID: 9, resolution: 'declined' }))), null);
});

test('mark all read preserves already-read timestamps', () => {
  const items = [
    NotificationModel.normalize(rawNotification(1, 'follow_started', { readAt: '2026-07-23T09:00:00Z' })),
    NotificationModel.normalize(rawNotification(2, 'follow_started'))
  ];
  const marked = NotificationModel.markAllRead(items, '2026-07-23T12:00:00Z');
  assert.equal(marked[0].readAt, '2026-07-23T09:00:00Z');
  assert.equal(marked[1].readAt, '2026-07-23T12:00:00Z');
});
