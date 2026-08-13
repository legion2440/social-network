'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs/promises');
const http = require('node:http');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');
const {
  loadSettings,
  normalizeServerURL,
  probeOrigin,
  saveSettings
} = require('../src/settings.cjs');

test('normalizeServerURL accepts host shorthand and keeps only the origin', () => {
  assert.equal(normalizeServerURL('192.168.1.20:8080/path?q=1'), 'http://192.168.1.20:8080');
  assert.equal(normalizeServerURL('https://social.example/app'), 'https://social.example');
  assert.throws(() => normalizeServerURL('ftp://example.com'), /http/);
});

test('settings persist a validated server address', async () => {
  const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'loop-settings-test-'));
  try {
    await saveSettings(dir, {
      serverURL: 'http://127.0.0.1:8123/path',
      lastSuccessfulAt: '2026-08-13T00:00:00.000Z'
    });
    assert.deepEqual(await loadSettings(dir), {
      serverURL: 'http://127.0.0.1:8123',
      lastSuccessfulAt: '2026-08-13T00:00:00.000Z'
    });
  } finally {
    await fs.rm(dir, { recursive: true, force: true });
  }
});

test('probeOrigin checks the public health endpoint', async () => {
  const server = http.createServer((req, res) => {
    if (req.url === '/api/health') return res.end('{"ok":true}');
    res.writeHead(404).end();
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  const address = server.address();
  try {
    assert.equal(await probeOrigin(`http://127.0.0.1:${address.port}`, 1000), true);
  } finally {
    await new Promise(resolve => server.close(resolve));
  }
});
