const test = require('node:test');
const assert = require('node:assert/strict');

const { isCustomAvatarURL } = require('./avatar-url.js');

test('controlled user avatar URLs are recognized as custom avatars', () => {
  assert.equal(isCustomAvatarURL('/api/users/7/avatar?v=42'), true);
  assert.equal(isCustomAvatarURL('/api/users/7/avatar'), true);
});

test('placeholders, legacy uploads, and malformed routes are not custom avatars', () => {
  assert.equal(isCustomAvatarURL('/static/avatars/neutral.svg'), false);
  assert.equal(isCustomAvatarURL('/static/avatars/female.svg'), false);
  assert.equal(isCustomAvatarURL('/uploads/42'), false);
  assert.equal(isCustomAvatarURL('/api/users/0/avatar?v=42'), false);
  assert.equal(isCustomAvatarURL('/api/users/7/avatar/extra?v=42'), false);
});
