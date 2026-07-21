const test = require('node:test');
const assert = require('node:assert/strict');

const { APIError, createAuthAPI } = require('./auth-api.js');

function jsonResponse(status, body) {
  return {
    status,
    ok: status >= 200 && status < 300,
    headers: { get: () => 'application/json; charset=utf-8' },
    json: async () => body,
    text: async () => JSON.stringify(body)
  };
}

function noContentResponse(status = 204) {
  return {
    status,
    ok: status >= 200 && status < 300,
    headers: { get: () => '' },
    text: async () => ''
  };
}

test('me uses a same-origin GET and returns the current user', async () => {
  const user = { id: 7, email: 'user@example.com' };
  const calls = [];
  const api = createAuthAPI(async (path, options) => {
    calls.push({ path, options });
    return jsonResponse(200, user);
  });

  assert.deepEqual(await api.me(), user);
  assert.equal(calls[0].path, '/api/auth/me');
  assert.equal(calls[0].options.method, 'GET');
  assert.equal(calls[0].options.credentials, 'same-origin');
});

test('register sends the original FormData without setting Content-Type', async () => {
  const formData = { kind: 'form-data-test-double' };
  let call;
  const api = createAuthAPI(async (path, options) => {
    call = { path, options };
    return jsonResponse(201, { id: 8 });
  });

  assert.deepEqual(await api.register(formData), { id: 8 });
  assert.equal(call.path, '/api/auth/register');
  assert.equal(call.options.method, 'POST');
  assert.equal(call.options.body, formData);
  assert.equal(call.options.headers['Content-Type'], undefined);
});

test('login sends JSON credentials and returns the authenticated user', async () => {
  let call;
  const api = createAuthAPI(async (path, options) => {
    call = { path, options };
    return jsonResponse(200, { id: 9 });
  });

  assert.deepEqual(await api.login('login@example.com', 'secret'), { id: 9 });
  assert.equal(call.path, '/api/auth/login');
  assert.equal(call.options.headers['Content-Type'], 'application/json');
  assert.deepEqual(JSON.parse(call.options.body), {
    email: 'login@example.com',
    password: 'secret'
  });
});

test('logout accepts only the backend 204 contract', async () => {
  const calls = [];
  const api = createAuthAPI(async (path, options) => {
    calls.push({ path, options });
    return noContentResponse();
  });

  assert.equal(await api.logout(), null);
  assert.equal(calls[0].path, '/api/auth/logout');
  assert.equal(calls[0].options.method, 'POST');

  const wrongStatusAPI = createAuthAPI(async () => jsonResponse(200, { ok: true }));
  await assert.rejects(wrongStatusAPI.logout(), (error) => {
    assert.ok(error instanceof APIError);
    assert.equal(error.status, 200);
    assert.equal(error.message, 'Unexpected server response.');
    return true;
  });
});

test('JSON API errors keep their status and backend message', async () => {
  const api = createAuthAPI(async () => jsonResponse(401, { error: 'invalid email or password' }));

  await assert.rejects(api.login('bad@example.com', 'bad'), (error) => {
    assert.ok(error instanceof APIError);
    assert.equal(error.status, 401);
    assert.equal(error.message, 'invalid email or password');
    return true;
  });
});

test('network failures use the common API error shape', async () => {
  const api = createAuthAPI(async () => {
    throw new Error('offline');
  });

  await assert.rejects(api.me(), (error) => {
    assert.ok(error instanceof APIError);
    assert.equal(error.status, 0);
    assert.equal(error.message, 'Network error. Please try again.');
    return true;
  });
});

test('profile update sends JSON to the protected profile endpoint', async () => {
  let call;
  const api = createAuthAPI(async (path, options) => {
    call = { path, options };
    return jsonResponse(200, { id: 7, first_name: 'Updated' });
  });
  const patch = { first_name: 'Updated', gender: null, is_private: true };

  assert.equal((await api.updateProfile(patch)).first_name, 'Updated');
  assert.equal(call.path, '/api/profile');
  assert.equal(call.options.method, 'PATCH');
  assert.equal(call.options.headers['Content-Type'], 'application/json');
  assert.deepEqual(JSON.parse(call.options.body), patch);
  assert.equal(call.options.credentials, 'same-origin');
});

test('profile avatar replace keeps FormData intact and delete expects 200', async () => {
  const calls = [];
  const formData = { kind: 'avatar-form-data-test-double' };
  const responses = [
    { id: 7, avatar_url: '/api/users/7/avatar?v=42' },
    { id: 7, avatar_url: '/static/avatars/neutral.svg' }
  ];
  const api = createAuthAPI(async (path, options) => {
    calls.push({ path, options });
    return jsonResponse(200, responses[calls.length - 1]);
  });

  const replaced = await api.replaceAvatar(formData);
  const deleted = await api.deleteAvatar();

  assert.equal(replaced.avatar_url, '/api/users/7/avatar?v=42');
  assert.equal(deleted.avatar_url, '/static/avatars/neutral.svg');
  assert.equal(calls[0].path, '/api/profile/avatar');
  assert.equal(calls[0].options.method, 'PUT');
  assert.equal(calls[0].options.body, formData);
  assert.equal(calls[0].options.headers['Content-Type'], undefined);
  assert.equal(calls[1].path, '/api/profile/avatar');
  assert.equal(calls[1].options.method, 'DELETE');
});

test('posts client sends multipart unchanged and builds cursor requests', async () => {
  const calls = [];
  const formData = { kind: 'post-form-data-test-double' };
  const api = createAuthAPI(async (path, options) => {
    calls.push({ path, options });
    if (options.method === 'POST') return jsonResponse(201, { id: 42 });
    return jsonResponse(200, { posts: [], next_cursor: null });
  });

  assert.deepEqual(await api.createPost(formData), { id: 42 });
  await api.feed('opaque+/=', 20);
  await api.userPosts(7, 'next cursor', 50);
  await api.postComments(42, 'comment cursor', 20);
  await api.followers(7);

  assert.equal(calls[0].path, '/api/posts');
  assert.equal(calls[0].options.method, 'POST');
  assert.equal(calls[0].options.body, formData);
  assert.equal(calls[0].options.headers['Content-Type'], undefined);
  assert.equal(calls[1].path, '/api/posts/feed?cursor=opaque%2B%2F%3D&limit=20');
  assert.equal(calls[2].path, '/api/users/7/posts?cursor=next%20cursor&limit=50');
  assert.equal(calls[3].path, '/api/posts/42/comments?cursor=comment%20cursor&limit=20');
  assert.equal(calls[4].path, '/api/users/7/followers');
  assert.ok(calls.every(call => call.options.credentials === 'same-origin'));
});

test('comment create sends strict JSON to the post comments endpoint', async () => {
  let call;
  const api = createAuthAPI(async (path, options) => {
    call = { path, options };
    return jsonResponse(201, { id: 15, post_id: 7, text: 'Hello' });
  });

  await api.createComment(7, 'Hello');
  assert.equal(call.path, '/api/posts/7/comments');
  assert.equal(call.options.method, 'POST');
  assert.equal(call.options.headers['Content-Type'], 'application/json');
  assert.deepEqual(JSON.parse(call.options.body), { text: 'Hello' });
  assert.equal(call.options.credentials, 'same-origin');
});

test('users and follow clients use backend IDs and exact endpoint contracts', async () => {
  const calls = [];
  const api = createAuthAPI(async (path, options) => {
    calls.push({ path, options });
    if (options.method === 'DELETE') return noContentResponse();
    if (path === '/api/follow-requests/41/accept') return jsonResponse(200, { status: 'accepted' });
    return jsonResponse(200, {});
  });

  await api.users('user cursor', 20);
  await api.userProfile(7);
  await api.relationship(7);
  await api.follow(7);
  await api.unfollow(7);
  await api.followers(7);
  await api.following(7);
  await api.followRequests();
  await api.acceptFollowRequest(41);
  await api.rejectFollowRequest(41);

  assert.deepEqual(calls.map(call => [call.path, call.options.method]), [
    ['/api/users?cursor=user%20cursor&limit=20', 'GET'],
    ['/api/users/7', 'GET'],
    ['/api/users/7/follow', 'GET'],
    ['/api/users/7/follow', 'PUT'],
    ['/api/users/7/follow', 'DELETE'],
    ['/api/users/7/followers', 'GET'],
    ['/api/users/7/following', 'GET'],
    ['/api/follow-requests', 'GET'],
    ['/api/follow-requests/41/accept', 'POST'],
    ['/api/follow-requests/41', 'DELETE']
  ]);
  assert.ok(calls.every(call => call.options.credentials === 'same-origin'));
});

test('follow mutations preserve exact backend errors', async () => {
  const api = createAuthAPI(async () => jsonResponse(500, { error: 'internal server error' }));
  await assert.rejects(api.follow(9), (error) => {
    assert.ok(error instanceof APIError);
    assert.equal(error.status, 500);
    return true;
  });
});
