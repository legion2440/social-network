const test = require('node:test');
const assert = require('node:assert/strict');

const CommentModel = require('./comment-model.js');

function response(id, createdAt) {
  return {
    id,
    post_id: 7,
    text: 'comment ' + id,
    created_at: createdAt,
    author: { id: 3 }
  };
}

test('comment response keeps backend ids and author reference', () => {
  const comment = CommentModel.normalizeCommentResponse(response(15, '2026-07-21T10:00:00Z'));
  assert.deepEqual(comment, {
    id: '15', apiId: 15, postID: 7, apiAuthorID: 3,
    text: 'comment 15', mediaUrl: '', createdAt: '2026-07-21T10:00:00Z'
  });
});

test('comment response maps a controlled media URL', () => {
  const raw = response(16, '2026-07-21T10:00:00Z');
  raw.media_url = '/api/comments/16/media';
  assert.equal(CommentModel.normalizeCommentResponse(raw).mediaUrl, '/api/comments/16/media');
});

test('comment merge deduplicates and orders by created_at then id', () => {
  const comments = [
    response(4, '2026-07-21T10:00:01Z'),
    response(2, '2026-07-21T10:00:00Z'),
    response(3, '2026-07-21T10:00:00Z'),
    response(4, '2026-07-21T10:00:01Z')
  ].map(CommentModel.normalizeCommentResponse);
  assert.deepEqual(CommentModel.mergeComments([comments[0]], comments.slice(1)).map(item => item.apiId), [2, 3, 4]);
});
