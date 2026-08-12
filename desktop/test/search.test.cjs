'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const { matchesMessage, parseQuery } = require('../../frontend/src/js/desktop-search.js');

test('text search supports include, exclude and fuzzy operators', () => {
  assert.equal(matchesMessage('Meet me at Astana Arena tonight', '+astana -tomorrow'), true);
  assert.equal(matchesMessage('Meet me tomorrow in Astana', '+astana -tomorrow'), false);
  assert.equal(matchesMessage('deployment completed', '~deplyoment'), true);
  assert.equal(matchesMessage('deployment completed', 'include:completed exclude:failed'), true);
  assert.equal(matchesMessage('deployment failed', 'include:deployment exclude:failed'), false);
});

test('quoted search terms stay together', () => {
  assert.equal(matchesMessage('The release candidate is ready', '"release candidate"'), true);
  assert.equal(matchesMessage('release is a candidate', '"release candidate"'), false);
  assert.deepEqual(parseQuery('exclude:"not ready"'), [{ type: 'exclude', value: 'not ready' }]);
});

test('numeric operators compare numbers contained in a message', () => {
  const text = 'Build 42 took 18.5 seconds and used 512 MB';
  assert.equal(matchesMessage(text, '=42'), true);
  assert.equal(matchesMessage(text, '!=42'), false);
  assert.equal(matchesMessage(text, '>500'), true);
  assert.equal(matchesMessage(text, '<20'), true);
  assert.equal(matchesMessage(text, '>=512'), true);
  assert.equal(matchesMessage(text, '<=18.5'), true);
  assert.equal(matchesMessage('no metrics here', '>1'), false);
});

test('operators can be combined and remain interactive query predicates', () => {
  const text = 'invoice 120 approved for Astana office';
  assert.equal(matchesMessage(text, '+invoice -rejected >100 <200'), true);
  assert.equal(matchesMessage(text, '+invoice -approved >100'), false);
});
