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

test('group API uses cursor pages, strict JSON mutations, and bodyless transitions', async () => {
  const calls = [];
  const api = createAuthAPI(async (path, options) => {
    calls.push({ path, options });
    return jsonResponse(options.method === 'POST' && path === '/api/groups' ? 201 : 200, { id: 12 });
  });

  await api.groups('next page', 20);
  await api.createGroup('Readers', 'Long-form reading club');
  await api.groupMembers(12, 'members cursor', 50);
  await api.requestGroupJoin(12);
  await api.cancelGroupJoin(12);
  await api.groupJoinRequests(12, null, 20);
  await api.acceptGroupJoinRequest(12, 7);
  await api.rejectGroupJoinRequest(12, 7);
  await api.groupInvitations(12, null, 20);
  await api.inviteToGroup(12, 8);
  await api.groupInvitationInbox('inbox cursor', 10);
  await api.acceptGroupInvitation(12);
  await api.declineGroupInvitation(12);
  await api.leaveGroup(12);

  assert.equal(calls[0].path, '/api/groups?cursor=next%20page&limit=20');
  assert.equal(calls[1].path, '/api/groups');
  assert.equal(calls[1].options.headers['Content-Type'], 'application/json');
  assert.deepEqual(JSON.parse(calls[1].options.body), { title: 'Readers', description: 'Long-form reading club' });
  assert.equal(calls[2].path, '/api/groups/12/members?cursor=members%20cursor&limit=50');
  assert.equal(calls[3].options.headers['Content-Type'], undefined);
  assert.equal(calls[4].options.method, 'DELETE');
  assert.equal(calls[6].path, '/api/groups/12/join-requests/7/accept');
  assert.equal(calls[7].options.method, 'DELETE');
  assert.deepEqual(JSON.parse(calls[9].options.body), { user_id: 8 });
  assert.equal(calls[10].path, '/api/group-invitations?cursor=inbox%20cursor&limit=10');
  assert.equal(calls[13].path, '/api/groups/12/membership');
  assert.equal(calls[13].options.method, 'DELETE');
  calls.forEach(call => assert.equal(call.options.credentials, 'same-origin'));
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

test('group posts client keeps multipart intact and uses cursor pagination', async () => {
  const calls = [];
  const formData = { kind: 'group-post-form-data-test-double' };
  const api = createAuthAPI(async (path, options) => {
    calls.push({ path, options });
    return options.method === 'POST'
      ? jsonResponse(201, { id: 51, group_id: 9 })
      : jsonResponse(200, { posts: [], next_cursor: null });
  });

  await api.groupPosts(9, 'group cursor', 20);
  assert.deepEqual(await api.createGroupPost(9, formData), { id: 51, group_id: 9 });

  assert.equal(calls[0].path, '/api/groups/9/posts?cursor=group%20cursor&limit=20');
  assert.equal(calls[0].options.method, 'GET');
  assert.equal(calls[1].path, '/api/groups/9/posts');
  assert.equal(calls[1].options.method, 'POST');
  assert.equal(calls[1].options.body, formData);
  assert.equal(calls[1].options.headers['Content-Type'], undefined);
  assert.ok(calls.every(call => call.options.credentials === 'same-origin'));
});

test('group events client uses JSON create and idempotent RSVP endpoints', async () => {
  const calls = [];
  const api = createAuthAPI(async (path, options) => {
    calls.push({ path, options });
    return jsonResponse(options.method === 'POST' ? 201 : 200, { id: 14, group_id: 9 });
  });
  const event = {
    title: 'Planning', description: 'Plan the meetup', starts_at: '2026-07-24T12:00:00.000Z'
  };

  await api.groupEvents(9, 'event cursor', 20);
  await api.createGroupEvent(9, event);
  await api.respondToGroupEvent(9, 14, 'going');

  assert.equal(calls[0].path, '/api/groups/9/events?cursor=event%20cursor&limit=20');
  assert.equal(calls[0].options.method, 'GET');
  assert.equal(calls[1].path, '/api/groups/9/events');
  assert.equal(calls[1].options.method, 'POST');
  assert.deepEqual(JSON.parse(calls[1].options.body), event);
  assert.equal(calls[2].path, '/api/groups/9/events/14/response');
  assert.equal(calls[2].options.method, 'PUT');
  assert.deepEqual(JSON.parse(calls[2].options.body), { response: 'going' });
  assert.ok(calls.every(call => call.options.credentials === 'same-origin'));
});

test('comment create sends original FormData without setting Content-Type', async () => {
  let call;
  const api = createAuthAPI(async (path, options) => {
    call = { path, options };
    return jsonResponse(201, { id: 15, post_id: 7, text: 'Hello' });
  });
  const formData = { kind: 'comment-form-data-test-double' };

  await api.createComment(7, formData);
  assert.equal(call.path, '/api/posts/7/comments');
  assert.equal(call.options.method, 'POST');
  assert.equal(call.options.headers['Content-Type'], undefined);
  assert.equal(call.options.body, formData);
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

test('notification clients use persisted list, read and action contracts', async () => {
  const calls = [];
  const api = createAuthAPI(async (path, options) => {
    calls.push({ path, options });
    return jsonResponse(200, {});
  });

  await api.notifications('notification cursor', 20);
  await api.markNotificationRead(42);
  await api.markAllNotificationsRead();
  await api.actOnNotification(42, 'accept');

  assert.deepEqual(calls.map(call => [call.path, call.options.method]), [
    ['/api/notifications?cursor=notification%20cursor&limit=20', 'GET'],
    ['/api/notifications/42/read', 'PUT'],
    ['/api/notifications/read-all', 'PUT'],
    ['/api/notifications/42/action', 'PUT']
  ]);
  assert.deepEqual(JSON.parse(calls[3].options.body), { action: 'accept' });
  assert.equal(calls[3].options.headers['Content-Type'], 'application/json');
  assert.ok(calls.every(call => call.options.credentials === 'same-origin'));
});

test('chat history and list clients use cursor endpoints over same-origin HTTP', async () => {
  const calls = [];
  const api = createAuthAPI(async (path, options) => {
    calls.push({ path, options });
    return jsonResponse(200, path.startsWith('/api/chats?')
      ? { chats: [], next_cursor: null }
      : { messages: [], next_cursor: null });
  });

  await api.chats('chat cursor', 20);
  await api.directMessages(7, 'dm cursor', 50);
  await api.groupMessages(9, null, 20);
  await api.markDirectChatRead(7, 123);
  await api.markGroupChatRead(9, 456);

  assert.deepEqual(calls.map(call => [call.path, call.options.method]), [
    ['/api/chats?cursor=chat%20cursor&limit=20', 'GET'],
    ['/api/chats/direct/7/messages?cursor=dm%20cursor&limit=50', 'GET'],
    ['/api/groups/9/chat/messages?limit=20', 'GET'],
    ['/api/chats/direct/7/read', 'PUT'],
    ['/api/groups/9/chat/read', 'PUT']
  ]);
  assert.deepEqual(JSON.parse(calls[3].options.body), { through_message_id: 123 });
  assert.deepEqual(JSON.parse(calls[4].options.body), { through_message_id: 456 });
  assert.equal(calls[3].options.headers['Content-Type'], 'application/json');
  assert.equal(calls[4].options.headers['Content-Type'], 'application/json');
  assert.ok(calls.every(call => call.options.credentials === 'same-origin'));
});
