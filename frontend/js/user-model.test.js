const test = require('node:test');
const assert = require('node:assert/strict');

const UserModel = require('./user-model.js');

test('normalization keeps one backend user object and merges full profile data', () => {
  let store = UserModel.mergeUsers({}, [{
    id: 7, first_name: 'Ada', last_name: 'Lovelace', nickname: 'ada',
    avatar_url: '/api/users/7/avatar?v=3', is_private: false,
    relationship: { status: 'none', follows_me: true }
  }], 1);
  const first = store['7'];

  store = UserModel.mergeUsers(store, [{
    id: 7, first_name: 'Ada', last_name: 'Lovelace', nickname: 'ada',
    avatar_url: '/api/users/7/avatar?v=3', is_private: false,
    can_view_profile: true, email: 'ada@example.com',
    date_of_birth: '10-12-1815', gender: 'female',
    about_me: 'Analytical engine', posts_count: 4, followers_count: 8, following_count: 2
  }], 1);

  assert.equal(store['7'], first);
  assert.equal(first.name, 'Ada Lovelace');
  assert.equal(first.email, 'ada@example.com');
  assert.equal(first.dob, '10-12-1815');
  assert.equal(first.gender, 'female');
  assert.equal(first.postsCount, 4);
  assert.deepEqual(first.relationship, { status: 'none', follows_me: true });
});

test('locked private user clears sensitive data but keeps a custom avatar', () => {
  const user = UserModel.normalizeUser({
    id: 9, first_name: 'Private', last_name: 'User', nickname: null,
    is_private: true, avatar_url: '/api/users/9/avatar?v=4',
    can_view_profile: false,
    relationship: { status: 'pending', follows_me: false }
  }, {
    email: 'private@example.com', dob: '01-01-1990', gender: 'male',
    bio: 'old private value', postsCount: 10
  }, 1);

  assert.equal(user.relationship.status, 'pending');
  assert.equal(user.email, '');
  assert.equal(user.dob, '');
  assert.equal(user.gender, '');
  assert.equal(user.bio, '');
  assert.equal(user.postsCount, 0);
  assert.equal(user.avatarUrl, '/api/users/9/avatar?v=4');
  assert.equal(user.hasCustomAvatar, true);
  assert.equal(user.rawAvatarUrl, '/api/users/9/avatar?v=4');
});

test('server-locked profile keeps custom avatar independent of profile access', () => {
  const user = UserModel.normalizeUser({
    id: 9, first_name: 'Private', last_name: 'User', nickname: null,
    is_private: true, avatar_url: '/api/users/9/avatar?v=5',
    can_view_profile: false,
    relationship: { status: 'accepted', follows_me: false }
  }, null, 1);

  assert.equal(user.relationship.status, 'accepted');
  assert.equal(user.canViewProfile, false);
  assert.equal(user.avatarUrl, '/api/users/9/avatar?v=5');
  assert.equal(user.hasCustomAvatar, true);
  assert.equal(user.rawAvatarUrl, '/api/users/9/avatar?v=5');
});

test('relationship changes do not gate a private custom avatar', () => {
  const previous = UserModel.normalizeUser({
    id: 9, first_name: 'Private', last_name: 'User', is_private: true,
    avatar_url: '/api/users/9/avatar?v=4', can_view_profile: false,
    relationship: { status: 'none', follows_me: false }
  }, null, 1);
  const accepted = UserModel.normalizeUser({
    id: 9,
    relationship: { status: 'accepted', follows_me: false }
  }, previous, 1);

  assert.equal(accepted.avatarUrl, '/api/users/9/avatar?v=4');
  assert.equal(UserModel.followButton(accepted, false).label, 'Following');
});

test('selected post audience is pruned to current accepted followers', () => {
  assert.deepEqual(UserModel.pruneSelected(
    { 2: true, 3: true, 4: false },
    [{ apiId: 3 }, { apiId: 5 }]
  ), { 3: true });
});

test('request gate rejects stale profile responses', () => {
  const gate = UserModel.createRequestGate();
  const first = gate.begin();
  const second = gate.begin();
  assert.equal(gate.isCurrent(first), false);
  assert.equal(gate.isCurrent(second), true);
});
