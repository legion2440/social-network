const test = require('node:test');
const assert = require('node:assert/strict');

const GroupEventModel = require('./group-event-model.js');

function rawEvent(id, startsAt, response) {
  return {
    id,
    group_id: 7,
    creator: { id: 2 },
    title: 'Event ' + id,
    description: 'Description',
    starts_at: startsAt,
    created_at: '2026-07-22T10:00:00Z',
    going_count: 3,
    not_going_count: 1,
    viewer_response: response
  };
}

test('normalizes authoritative event counts and viewer response', () => {
  const event = GroupEventModel.normalizeEventResponse(rawEvent(8, '2026-07-23T12:00:00Z', 'going'));
  assert.deepEqual(event, {
    id: 8,
    groupID: 7,
    creatorID: 2,
    title: 'Event 8',
    description: 'Description',
    startsAt: '2026-07-23T12:00:00Z',
    createdAt: '2026-07-22T10:00:00Z',
    goingCount: 3,
    notGoingCount: 1,
    viewerResponse: 'going'
  });
});

test('authoritative event merge replaces by id and sorts by starts_at then id', () => {
  const first = GroupEventModel.normalizeEventResponse(rawEvent(5, '2026-07-24T12:00:00Z', null));
  const second = GroupEventModel.normalizeEventResponse(rawEvent(7, '2026-07-23T12:00:00Z', null));
  const replacement = GroupEventModel.normalizeEventResponse(rawEvent(5, '2026-07-22T12:00:00Z', 'not_going'));
  const merged = GroupEventModel.mergeAuthoritative([first, second], replacement);
  assert.deepEqual(merged.map(event => event.id), [5, 7]);
  assert.equal(merged[0].viewerResponse, 'not_going');
});
