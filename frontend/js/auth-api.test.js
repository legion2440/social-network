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
