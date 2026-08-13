'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs/promises');
const http = require('node:http');
const net = require('node:net');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');
const {
  CACHE_VERSION,
  PersistentCache,
  cacheKey,
  copyResponseHeaders,
  createDesktopServer,
  isCacheEntryFresh,
  safeStaticPath,
  shouldCache
} = require('../src/proxy.cjs');

function listen(server) {
  return new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const address = server.address();
      resolve(`http://127.0.0.1:${address.port}`);
    });
  });
}

function close(server) {
  return new Promise(resolve => server.close(resolve));
}

function request(origin, requestPath, options = {}) {
  return new Promise((resolve, reject) => {
    const req = http.request(new URL(requestPath, origin), {
      method: options.method || 'GET',
      headers: options.headers || {}
    }, res => {
      const chunks = [];
      res.on('data', chunk => chunks.push(chunk));
      res.on('end', () => resolve({
        status: res.statusCode,
        headers: res.headers,
        body: Buffer.concat(chunks).toString('utf8')
      }));
    });
    req.on('error', reject);
    if (options.body) req.write(options.body);
    req.end();
  });
}

async function desktopFixture(targetOrigin) {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'loop-desktop-test-'));
  const distDir = path.join(root, 'dist');
  const cacheDir = path.join(root, 'cache');
  await fs.mkdir(distDir, { recursive: true });
  await fs.writeFile(path.join(distDir, 'index.html'), '<!doctype html><title>Loop</title>', 'utf8');
  const desktop = createDesktopServer({ distDir, cacheDir, targetOrigin });
  const listener = await desktop.listen();
  return {
    desktop,
    cacheDir,
    origin: listener.origin,
    async cleanup() {
      await desktop.close();
      await fs.rm(root, { recursive: true, force: true });
    }
  };
}

test('cacheKey keeps method and full request URL', () => {
  assert.equal(cacheKey('get', '/api/chats?cursor=abc'), 'GET /api/chats?cursor=abc');
});

test('message and session endpoints are cacheable', () => {
  assert.equal(shouldCache('GET', '/api/auth/me'), true);
  assert.equal(shouldCache('GET', '/api/chats'), true);
  assert.equal(shouldCache('GET', '/api/chats/direct/42/messages?limit=20'), true);
  assert.equal(shouldCache('GET', '/api/groups/9/chat/messages'), true);
  assert.equal(shouldCache('POST', '/api/auth/me'), false);
  assert.equal(shouldCache('GET', '/api/posts/feed'), false);
});

test('safeStaticPath rejects traversal outside dist', () => {
  const root = path.resolve('/tmp/example-dist');
  assert.equal(safeStaticPath(root, '/css/base.css'), path.join(root, 'css', 'base.css'));
  assert.equal(safeStaticPath(root, '/%2e%2e/secret.txt'), null);
});

test('cached API response is available while offline', async () => {
  let hits = 0;
  const upstream = http.createServer((req, res) => {
    if (req.url === '/api/chats') {
      hits += 1;
      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify({ chats: [{ kind: 'direct', target_id: 2 }], next_cursor: null }));
      return;
    }
    res.writeHead(404).end();
  });
  const targetOrigin = await listen(upstream);
  const fixture = await desktopFixture(targetOrigin);
  try {
    const onlineResponse = await request(fixture.origin, '/api/chats');
    assert.equal(onlineResponse.status, 200);
    fixture.desktop.setClientOnline(false);
    const offlineResponse = await request(fixture.origin, '/api/chats');
    assert.equal(offlineResponse.status, 200);
    assert.equal(offlineResponse.headers['x-loop-offline'], '1');
    assert.deepEqual(JSON.parse(offlineResponse.body).chats, [{ kind: 'direct', target_id: 2 }]);
    assert.equal(hits, 1);
  } finally {
    await fixture.cleanup();
    await close(upstream);
  }
});

test('successful login seeds offline session cache without persisting session cookies', async () => {
  const user = { id: 7, display_name: 'Desktop User' };
  const upstream = http.createServer((req, res) => {
    if (req.url === '/api/auth/login' && req.method === 'POST') {
      req.resume();
      res.setHeader('Content-Type', 'application/json');
      res.setHeader('Set-Cookie', 'social_network_session=SECRET-TOKEN-123; Path=/; Secure; SameSite=None; Domain=example.com');
      res.end(JSON.stringify(user));
      return;
    }
    res.writeHead(404).end();
  });
  const targetOrigin = await listen(upstream);
  const fixture = await desktopFixture(targetOrigin);
  try {
    const login = await request(fixture.origin, '/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'a@example.com', password: 'secret' })
    });
    assert.equal(login.status, 200);
    assert.match(login.headers['set-cookie'][0], /SameSite=Lax/i);
    assert.doesNotMatch(login.headers['set-cookie'][0], /Secure/i);
    assert.doesNotMatch(login.headers['set-cookie'][0], /Domain=/i);

    fixture.desktop.setClientOnline(false);
    const me = await request(fixture.origin, '/api/auth/me');
    assert.equal(me.status, 200);
    assert.equal(me.headers['x-loop-offline'], '1');
    assert.equal(me.headers['set-cookie'], undefined);
    assert.deepEqual(JSON.parse(me.body), user);

    const cachePath = path.join(fixture.cacheDir, 'http-cache.json');
    let rawCache = '';
    for (let attempt = 0; attempt < 50; attempt += 1) {
      try { rawCache = await fs.readFile(cachePath, 'utf8'); break; } catch (error) {
        if (!error || error.code !== 'ENOENT') throw error;
        await new Promise(resolve => setTimeout(resolve, 10));
      }
    }
    assert.ok(rawCache);
    assert.doesNotMatch(rawCache, /SECRET-TOKEN-123/);
    assert.doesNotMatch(rawCache, /set-cookie/i);
  } finally {
    await fixture.cleanup();
    await close(upstream);
  }
});

test('legacy cache is sanitized and expired session entries are discarded', async () => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'loop-cache-migration-'));
  try {
    await fs.writeFile(path.join(root, 'http-cache.json'), JSON.stringify({
      version: 1,
      entries: {
        'GET /api/auth/me': {
          status: 200,
          headers: { 'set-cookie': ['social_network_session=OLD-SECRET'], 'content-type': 'application/json' },
          body: Buffer.from('{"id":1}').toString('base64'),
          savedAt: '2020-01-01T00:00:00.000Z'
        }
      }
    }), 'utf8');
    const cache = new PersistentCache(root);
    await cache.load();
    assert.equal(cache.get('GET /api/auth/me'), null);
    const migrated = JSON.parse(await fs.readFile(path.join(root, 'http-cache.json'), 'utf8'));
    assert.equal(migrated.version, CACHE_VERSION);
    assert.deepEqual(migrated.entries, {});
  } finally {
    await fs.rm(root, { recursive: true, force: true });
  }
});

test('response header cache sanitization strips authentication material', () => {
  assert.deepEqual(copyResponseHeaders({
    'content-type': 'application/json',
    'set-cookie': ['session=secret'],
    authorization: 'Bearer secret',
    'www-authenticate': 'Basic realm=test',
    'proxy-authenticate': 'Basic realm=proxy'
  }), { 'content-type': 'application/json' });
});

test('session cache expires after backend session TTL while chat cache lasts longer', () => {
  const now = Date.parse('2026-08-13T00:00:00.000Z');
  const twentyFiveHoursAgo = new Date(now - 25 * 60 * 60 * 1000).toISOString();
  const entry = { savedAt: twentyFiveHoursAgo };
  assert.equal(isCacheEntryFresh('GET /api/auth/me', entry, now), false);
  assert.equal(isCacheEntryFresh('GET /api/chats', entry, now), true);
});

test('backend health check reports recovery after the upstream becomes available', async () => {
  const upstream = http.createServer((req, res) => {
    if (req.url === '/api/health') return res.end('{"ok":true}');
    res.writeHead(404).end();
  });
  const targetOrigin = await listen(upstream);
  const fixture = await desktopFixture(targetOrigin);
  try {
    assert.equal(await fixture.desktop.checkBackend(), true);
  } finally {
    await fixture.cleanup();
    await close(upstream);
  }
});

test('websocket proxy rewrites Host and Origin to the backend origin', async () => {
  let seenHeaders = null;
  const upstream = http.createServer();
  upstream.on('upgrade', (req, socket) => {
    seenHeaders = req.headers;
    socket.write([
      'HTTP/1.1 101 Switching Protocols',
      'Connection: Upgrade',
      'Upgrade: websocket',
      '',
      ''
    ].join('\r\n'));
    socket.end();
  });
  const targetOrigin = await listen(upstream);
  const fixture = await desktopFixture(targetOrigin);
  try {
    const local = new URL(fixture.origin);
    await new Promise((resolve, reject) => {
      const socket = net.connect(Number(local.port), local.hostname, () => {
        socket.write([
          'GET /ws HTTP/1.1',
          `Host: ${local.host}`,
          `Origin: ${fixture.origin}`,
          'Connection: Upgrade',
          'Upgrade: websocket',
          'Sec-WebSocket-Version: 13',
          'Sec-WebSocket-Key: dGVzdGtleTEyMzQ1Ng==',
          '',
          ''
        ].join('\r\n'));
      });
      let response = '';
      socket.on('data', chunk => { response += chunk.toString('utf8'); });
      socket.on('end', () => {
        try {
          assert.match(response, /101 Switching Protocols/);
          resolve();
        } catch (error) { reject(error); }
      });
      socket.on('error', reject);
    });
    const target = new URL(targetOrigin);
    assert.equal(seenHeaders.host, target.host);
    assert.equal(seenHeaders.origin, target.origin);
  } finally {
    await fixture.cleanup();
    await close(upstream);
  }
});
