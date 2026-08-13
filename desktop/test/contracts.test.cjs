'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const repoRoot = path.resolve(__dirname, '..', '..');

function read(relative) {
  return fs.readFileSync(path.join(repoRoot, relative), 'utf8');
}

test('desktop chat integration uses stable data hooks', () => {
  const template = read('frontend/src/templates/chat.html');
  const adapter = read('frontend/src/js/desktop-adapter.js');
  [
    'data-loop-chat-header',
    'data-loop-message-list',
    'data-loop-message-row',
    'data-loop-message-body',
    'data-loop-load-older',
    'data-loop-chat-input',
    'data-loop-chat-send'
  ].forEach(hook => assert.match(template, new RegExp(hook)));
  assert.doesNotMatch(adapter, /\.ui-0(?:50|54|58|59|63|66|79|80)\b/);
});

test('registration route is included in the production frontend source', () => {
  const index = read('frontend/src/index.html');
  const route = read('frontend/src/js/register-route.js');
  assert.match(index, /\/js\/register-route\.js/);
  assert.match(route, /searchParams|URLSearchParams/);
  assert.match(route, /Create\\s\+account|create\\s\+account/i);
});
