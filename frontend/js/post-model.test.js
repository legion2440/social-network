const test = require('node:test');
const assert = require('node:assert/strict');

const PostModel = require('./post-model.js');

class FakeFormData {
  constructor() { this.entries = []; }
  append(...args) { this.entries.push(args); }
}

test('create post FormData uses selected and repeated selected_user_id values', () => {
  const media = { name: 'post.webp' };
  const form = PostModel.buildCreatePostForm({
    text: '  hello 🙂  ',
    privacy: 'selected',
    selectedUserIDs: [12, 15, 12],
    media
  }, FakeFormData);

  assert.deepEqual(form.entries, [
    ['text', 'hello 🙂'],
    ['privacy', 'selected'],
    ['selected_user_id', '12'],
    ['selected_user_id', '15'],
    ['media', media, 'post.webp']
  ]);
});

test('public and followers FormData never send selected audience values', () => {
  for (const privacy of ['public', 'followers']) {
    const form = PostModel.buildCreatePostForm({
      text: 'post', privacy, selectedUserIDs: [12], media: null
    }, FakeFormData);
    assert.deepEqual(form.entries, [['text', 'post'], ['privacy', privacy]]);
  }
});

test('post response mapping keeps controlled media and author fields', () => {
  const mapped = PostModel.normalizePostResponse({
    id: 42,
    author: {
      id: 7,
      first_name: 'Ada',
      last_name: 'Lovelace',
      nickname: 'ada',
      avatar_url: '/api/users/7/avatar?v=9',
      is_private: true
    },
    text: 'Mapped post',
    privacy: 'selected',
    media_url: '/api/posts/42/media',
    comments_count: 8,
    created_at: '2026-07-18T10:00:00Z'
  }, 7);

  assert.equal(mapped.id, '42');
  assert.equal(mapped.isOwn, true);
  assert.equal(mapped.privacy, 'selected');
  assert.equal(mapped.mediaUrl, '/api/posts/42/media');
  assert.equal(mapped.commentsCount, 8);
  assert.deepEqual(mapped.author, {
    apiId: 7,
    firstName: 'Ada',
    lastName: 'Lovelace',
    nickname: 'ada',
    avatarUrl: '/api/users/7/avatar?v=9',
    isPrivate: true
  });
});
