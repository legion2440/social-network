const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

test('confirmation dialog and custom date inputs expose accessible instructions', () => {
  const templates = path.resolve(__dirname, '..', 'src', 'templates');
  const modal = fs.readFileSync(path.join(templates, 'confirmation-modal.html'), 'utf8');
  const auth = fs.readFileSync(path.join(templates, 'auth.html'), 'utf8');
  const group = fs.readFileSync(path.join(templates, 'group.html'), 'utf8');
  const shell = fs.readFileSync(path.join(templates, 'shell.html'), 'utf8');
  const chat = fs.readFileSync(path.join(templates, 'chat.html'), 'utf8');

  assert.match(modal, /role="dialog"/);
  assert.match(modal, /aria-modal="true"/);
  assert.match(modal, /aria-labelledby="confirmation-title"/);
  assert.match(modal, /aria-describedby="confirmation-description"/);
  assert.match(auth, /aria-describedby="date-of-birth-help"/);
  assert.match(group, /aria-describedby="event-start-help"/);
  assert.match(shell, /aria-label="{{n\.label}}"/);
  assert.match(chat, /aria-label="Back to conversations"/);
});
