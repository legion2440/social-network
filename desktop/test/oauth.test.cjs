'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const {
  cookieForOrigin,
  isAllowedOAuthURL,
  isOriginURL,
  mirrorSessionCookies
} = require('../src/oauth.cjs');

test('cookieForOrigin converts remote secure cookies for a local HTTP origin', () => {
  const cookie = cookieForOrigin({
    name: 'session',
    value: 'abc',
    path: '/',
    httpOnly: true,
    secure: true,
    sameSite: 'no_restriction',
    expirationDate: 2000000000
  }, 'http://127.0.0.1:43123');
  assert.equal(cookie.url, 'http://127.0.0.1:43123/');
  assert.equal(cookie.secure, false);
  assert.equal(cookie.sameSite, 'lax');
  assert.equal(cookie.httpOnly, true);
  assert.equal('domain' in cookie, false);
});

test('mirrorSessionCookies copies cookies between the configured server and loopback app', async () => {
  const writes = [];
  const session = {
    cookies: {
      get: async query => {
        assert.deepEqual(query, { url: 'https://social.example' });
        return [
          { name: 'session', value: 'token', path: '/', httpOnly: true, secure: true, sameSite: 'lax' },
          { name: 'oauth_nonce', value: 'nonce', path: '/api/auth/oauth', httpOnly: true, secure: true }
        ];
      },
      set: async cookie => writes.push(cookie)
    }
  };
  const copied = await mirrorSessionCookies(session, 'https://social.example', 'http://127.0.0.1:5010');
  assert.equal(copied, 2);
  assert.equal(writes.length, 2);
  assert.equal(writes[0].name, 'session');
  assert.equal(writes[0].url, 'http://127.0.0.1:5010/');
  assert.equal(writes[1].url, 'http://127.0.0.1:5010/api/auth/oauth');
});

test('isOriginURL compares normalized URL origins', () => {
  assert.equal(isOriginURL('https://example.com/oauth/complete?flow=1', 'https://example.com'), true);
  assert.equal(isOriginURL('https://github.com/login/oauth', 'https://example.com'), false);
});

test('OAuth navigation is restricted to GitHub and the configured server', () => {
  assert.equal(isAllowedOAuthURL('https://github.com/login/oauth/authorize', 'https://social.example'), true);
  assert.equal(isAllowedOAuthURL('https://social.example/api/auth/oauth/github/callback', 'https://social.example'), true);
  assert.equal(isAllowedOAuthURL('https://evil.example/phish', 'https://social.example'), false);
  assert.equal(isAllowedOAuthURL('http://github.com/login', 'https://social.example'), false);
});
