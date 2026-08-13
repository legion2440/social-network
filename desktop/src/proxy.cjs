'use strict';

const fs = require('node:fs');
const fsp = require('node:fs/promises');
const http = require('node:http');
const https = require('node:https');
const path = require('node:path');
const { URL } = require('node:url');

const CACHE_FILE = 'http-cache.json';
const CACHE_VERSION = 2;
const MAX_CACHE_BODY = 5 * 1024 * 1024;
const SESSION_CACHE_TTL_MS = 24 * 60 * 60 * 1000;
const MESSAGE_CACHE_TTL_MS = 30 * 24 * 60 * 60 * 1000;
const SENSITIVE_RESPONSE_HEADERS = new Set([
  'set-cookie',
  'authorization',
  'www-authenticate',
  'proxy-authenticate'
]);

function mimeType(filePath) {
  switch (path.extname(filePath).toLowerCase()) {
    case '.html': return 'text/html; charset=utf-8';
    case '.js': return 'text/javascript; charset=utf-8';
    case '.css': return 'text/css; charset=utf-8';
    case '.json': return 'application/json; charset=utf-8';
    case '.svg': return 'image/svg+xml';
    case '.png': return 'image/png';
    case '.jpg':
    case '.jpeg': return 'image/jpeg';
    case '.gif': return 'image/gif';
    case '.webp': return 'image/webp';
    case '.woff2': return 'font/woff2';
    default: return 'application/octet-stream';
  }
}

function requestPathname(requestURL) {
  return new URL(String(requestURL || '/'), 'http://desktop.local').pathname;
}

function cacheKey(method, requestURL) {
  return String(method || 'GET').toUpperCase() + ' ' + String(requestURL || '/');
}

function shouldCache(method, requestURL) {
  if (String(method || '').toUpperCase() !== 'GET') return false;
  const pathname = requestPathname(requestURL);
  if (pathname === '/api/auth/me') return true;
  if (pathname === '/api/chats') return true;
  if (/^\/api\/chats\/direct\/\d+\/messages$/.test(pathname)) return true;
  if (/^\/api\/groups\/\d+\/chat\/messages$/.test(pathname)) return true;
  return false;
}

function safeStaticPath(distDir, requestURL) {
  const rawPath = String(requestURL || '/').split('?', 1)[0].split('#', 1)[0];
  const decoded = decodeURIComponent(rawPath);
  if (decoded.split('/').includes('..')) return null;
  const relative = decoded.replace(/^\/+/, '');
  const root = path.resolve(distDir);
  const candidate = path.resolve(root, relative || 'index.html');
  if (candidate !== root && !candidate.startsWith(root + path.sep)) return null;
  return candidate;
}

function copyResponseHeaders(headers) {
  const result = {};
  Object.entries(headers || {}).forEach(([name, value]) => {
    if (value === undefined) return;
    const lower = name.toLowerCase();
    if (lower === 'transfer-encoding' || lower === 'connection' || lower === 'content-length') return;
    if (SENSITIVE_RESPONSE_HEADERS.has(lower)) return;
    result[name] = value;
  });
  return result;
}

function cacheTTLForKey(key) {
  return String(key || '') === 'GET /api/auth/me' ? SESSION_CACHE_TTL_MS : MESSAGE_CACHE_TTL_MS;
}

function isCacheEntryFresh(key, entry, now = Date.now()) {
  if (!entry || !entry.savedAt) return false;
  const savedAt = Date.parse(String(entry.savedAt));
  if (!Number.isFinite(savedAt)) return false;
  return now - savedAt <= cacheTTLForKey(key);
}

function sanitizeCacheEntry(entry) {
  if (!entry || typeof entry !== 'object') return null;
  return {
    status: Number(entry.status) || 200,
    headers: copyResponseHeaders(entry.headers || {}),
    body: String(entry.body || ''),
    savedAt: String(entry.savedAt || '')
  };
}

class PersistentCache {
  constructor(directory) {
    this.directory = directory;
    this.filePath = path.join(directory, CACHE_FILE);
    this.entries = {};
    this.loaded = false;
    this.writeChain = Promise.resolve();
  }

  async load() {
    if (this.loaded) return;
    this.loaded = true;
    try {
      const raw = await fsp.readFile(this.filePath, 'utf8');
      const parsed = JSON.parse(raw);
      const next = {};
      let changed = !parsed || parsed.version !== CACHE_VERSION;
      const source = parsed && parsed.entries && typeof parsed.entries === 'object' ? parsed.entries : {};
      Object.entries(source).forEach(([key, value]) => {
        const sanitized = sanitizeCacheEntry(value);
        if (!sanitized || !isCacheEntryFresh(key, sanitized)) {
          changed = true;
          return;
        }
        if (JSON.stringify(value && value.headers || {}) !== JSON.stringify(sanitized.headers)) changed = true;
        next[key] = sanitized;
      });
      this.entries = next;
      if (changed) await this.persist();
    } catch (error) {
      if (error && error.code !== 'ENOENT') console.warn('desktop cache read failed:', error.message);
    }
  }

  get(key) {
    const entry = this.entries[key] || null;
    if (!entry) return null;
    if (isCacheEntryFresh(key, entry)) return entry;
    delete this.entries[key];
    this.persist().catch(() => {});
    return null;
  }

  async set(key, value) {
    this.entries[key] = sanitizeCacheEntry(value);
    await this.persist();
  }

  async clear() {
    this.entries = {};
    await this.persist();
  }

  async persist() {
    const snapshot = JSON.stringify({ version: CACHE_VERSION, entries: this.entries });
    this.writeChain = this.writeChain.then(async () => {
      await fsp.mkdir(this.directory, { recursive: true });
      const tmp = this.filePath + '.tmp';
      await fsp.writeFile(tmp, snapshot, 'utf8');
      await fsp.rename(tmp, this.filePath);
    }).catch(error => {
      console.warn('desktop cache write failed:', error.message);
    });
    return this.writeChain;
  }
}

function rewriteSetCookie(value) {
  const cookies = Array.isArray(value) ? value : [value];
  return cookies.filter(Boolean).map(cookie => String(cookie)
    .replace(/;\s*Domain=[^;]+/gi, '')
    .replace(/;\s*Secure\b/gi, '')
    .replace(/;\s*SameSite=None\b/gi, '; SameSite=Lax'));
}

function rewriteResponseHeaders(headers) {
  const result = Object.assign({}, headers || {});
  if (result['set-cookie']) result['set-cookie'] = rewriteSetCookie(result['set-cookie']);
  return result;
}

function cacheValue(status, headers, body) {
  return {
    status: status || 200,
    headers: copyResponseHeaders(headers),
    body: body.toString('base64'),
    savedAt: new Date().toISOString()
  };
}

function createDesktopServer(options) {
  const distDir = path.resolve(options.distDir);
  const target = new URL(options.targetOrigin);
  const transport = target.protocol === 'https:' ? https : http;
  const cache = new PersistentCache(options.cacheDir);
  let server = null;
  let clientOnline = true;
  let backendOnline = null;

  const reportNetwork = online => {
    const next = online === true;
    if (backendOnline === next) return;
    backendOnline = next;
    if (typeof options.onNetworkStatus === 'function') options.onNetworkStatus(next && clientOnline);
  };

  const effectiveOnline = () => clientOnline && backendOnline !== false;

  const sendOffline = async (req, res) => {
    await cache.load();
    const entry = cache.get(cacheKey(req.method, req.url));
    if (entry && String(req.method || 'GET').toUpperCase() === 'GET') {
      const body = Buffer.from(entry.body || '', 'base64');
      res.writeHead(entry.status || 200, Object.assign({}, rewriteResponseHeaders(entry.headers || {}), {
        'Content-Length': String(body.length),
        'X-Loop-Offline': '1'
      }));
      res.end(body);
      return;
    }
    const body = Buffer.from(JSON.stringify({ error: 'No internet connection. You are offline.' }));
    res.writeHead(503, {
      'Content-Type': 'application/json; charset=utf-8',
      'Content-Length': String(body.length),
      'Cache-Control': 'no-store',
      'X-Loop-Offline': '1'
    });
    res.end(body);
  };

  const upstreamHeaders = headers => {
    const next = Object.assign({}, headers || {}, {
      host: target.host,
      origin: target.origin
    });
    if (next.referer) next.referer = target.origin + '/';
    return next;
  };

  const proxyHTTP = async (req, res) => {
    await cache.load();
    if (!clientOnline) {
      await sendOffline(req, res);
      return;
    }

    const pathname = requestPathname(req.url);
    const upstream = transport.request({
      protocol: target.protocol,
      hostname: target.hostname,
      port: target.port || undefined,
      method: req.method,
      path: req.url,
      headers: upstreamHeaders(req.headers)
    }, upstreamRes => {
      reportNetwork(true);
      const status = upstreamRes.statusCode || 502;
      const cacheable = shouldCache(req.method, req.url) && status >= 200 && status < 300;
      const captureLogin = String(req.method || '').toUpperCase() === 'POST' &&
        pathname === '/api/auth/login' && status >= 200 && status < 300;

      if (!cacheable && !captureLogin) {
        res.writeHead(status, rewriteResponseHeaders(upstreamRes.headers));
        upstreamRes.pipe(res);
        if (String(req.method || '').toUpperCase() === 'POST' && pathname === '/api/auth/logout' &&
            status >= 200 && status < 300) {
          cache.clear();
        }
        return;
      }

      const chunks = [];
      let size = 0;
      let withinLimit = true;
      res.writeHead(status, rewriteResponseHeaders(upstreamRes.headers));
      upstreamRes.on('data', chunk => {
        res.write(chunk);
        size += chunk.length;
        if (withinLimit && size <= MAX_CACHE_BODY) chunks.push(chunk);
        else withinLimit = false;
      });
      upstreamRes.on('end', async () => {
        res.end();
        if (!withinLimit) return;
        const body = Buffer.concat(chunks);
        const value = cacheValue(status, upstreamRes.headers, body);
        if (cacheable) await cache.set(cacheKey(req.method, req.url), value);
        if (captureLogin) await cache.set(cacheKey('GET', '/api/auth/me'), value);
      });
    });

    upstream.on('error', async () => {
      reportNetwork(false);
      if (!res.headersSent) await sendOffline(req, res);
      else res.destroy();
    });
    req.pipe(upstream);
  };

  const serveStatic = async (req, res) => {
    let filePath;
    try {
      filePath = safeStaticPath(distDir, req.url);
    } catch (_error) {
      res.writeHead(400);
      res.end('Bad request');
      return;
    }
    if (!filePath) {
      res.writeHead(403);
      res.end('Forbidden');
      return;
    }

    try {
      const stat = await fsp.stat(filePath);
      if (stat.isDirectory()) filePath = path.join(filePath, 'index.html');
      const finalStat = await fsp.stat(filePath);
      if (!finalStat.isFile()) throw Object.assign(new Error('not a file'), { code: 'ENOENT' });
    } catch (error) {
      if (error && error.code !== 'ENOENT') {
        res.writeHead(500);
        res.end('Internal error');
        return;
      }
      filePath = path.join(distDir, 'index.html');
    }

    res.writeHead(200, {
      'Content-Type': mimeType(filePath),
      'Cache-Control': filePath.endsWith('index.html') ? 'no-store' : 'public, max-age=3600'
    });
    fs.createReadStream(filePath).pipe(res);
  };

  const handleUpgrade = (req, socket, head) => {
    if (!clientOnline) {
      socket.destroy();
      return;
    }
    const upstream = transport.request({
      protocol: target.protocol,
      hostname: target.hostname,
      port: target.port || undefined,
      method: 'GET',
      path: req.url,
      headers: upstreamHeaders(req.headers)
    });

    upstream.on('upgrade', (upstreamRes, upstreamSocket, upstreamHead) => {
      reportNetwork(true);
      const status = upstreamRes.statusCode || 101;
      const statusMessage = upstreamRes.statusMessage || 'Switching Protocols';
      const responseLines = [`HTTP/1.1 ${status} ${statusMessage}`];
      for (let index = 0; index < upstreamRes.rawHeaders.length; index += 2) {
        responseLines.push(`${upstreamRes.rawHeaders[index]}: ${upstreamRes.rawHeaders[index + 1]}`);
      }
      socket.write(responseLines.join('\r\n') + '\r\n\r\n');
      if (head && head.length) upstreamSocket.write(head);
      if (upstreamHead && upstreamHead.length) socket.write(upstreamHead);
      socket.pipe(upstreamSocket).pipe(socket);
    });

    upstream.on('response', upstreamRes => {
      reportNetwork(true);
      const responseLines = [`HTTP/1.1 ${upstreamRes.statusCode || 502} ${upstreamRes.statusMessage || 'Bad Gateway'}`];
      for (let index = 0; index < upstreamRes.rawHeaders.length; index += 2) {
        responseLines.push(`${upstreamRes.rawHeaders[index]}: ${upstreamRes.rawHeaders[index + 1]}`);
      }
      socket.write(responseLines.join('\r\n') + '\r\n\r\n');
      upstreamRes.pipe(socket);
    });

    upstream.on('error', () => {
      reportNetwork(false);
      socket.destroy();
    });
    upstream.end();
  };

  const checkBackend = () => new Promise(resolve => {
    if (!clientOnline) {
      if (typeof options.onNetworkStatus === 'function') options.onNetworkStatus(false);
      resolve(false);
      return;
    }
    const req = transport.request({
      protocol: target.protocol,
      hostname: target.hostname,
      port: target.port || undefined,
      method: 'GET',
      path: '/api/health',
      headers: { host: target.host, origin: target.origin },
      timeout: 2500
    }, res => {
      const ok = (res.statusCode || 0) >= 200 && (res.statusCode || 0) < 300;
      reportNetwork(ok);
      res.resume();
      res.on('end', () => resolve(ok));
    });
    req.on('timeout', () => req.destroy());
    req.on('error', () => {
      reportNetwork(false);
      resolve(false);
    });
    req.end();
  });

  return {
    async listen() {
      await cache.load();
      if (server) throw new Error('desktop server is already running');
      server = http.createServer((req, res) => {
        const pathname = requestPathname(req.url);
        if (pathname === '/api' || pathname.startsWith('/api/') || pathname === '/static/avatars' || pathname.startsWith('/static/avatars/')) {
          proxyHTTP(req, res).catch(error => {
            console.error('desktop proxy error:', error);
            if (!res.headersSent) res.writeHead(500);
            res.end();
          });
          return;
        }
        serveStatic(req, res).catch(error => {
          console.error('desktop static server error:', error);
          if (!res.headersSent) res.writeHead(500);
          res.end();
        });
      });
      server.on('upgrade', (req, socket, head) => {
        const pathname = requestPathname(req.url);
        if (pathname !== '/ws' && !pathname.startsWith('/ws/')) {
          socket.destroy();
          return;
        }
        handleUpgrade(req, socket, head);
      });
      await new Promise((resolve, reject) => {
        server.once('error', reject);
        server.listen(0, '127.0.0.1', resolve);
      });
      const address = server.address();
      return { origin: `http://127.0.0.1:${address.port}` };
    },
    setClientOnline(online) {
      clientOnline = online === true;
      if (typeof options.onNetworkStatus === 'function') options.onNetworkStatus(effectiveOnline());
    },
    isOnline() {
      return effectiveOnline();
    },
    checkBackend,
    async close() {
      if (!server) return;
      const active = server;
      server = null;
      await new Promise(resolve => active.close(resolve));
    }
  };
}

module.exports = {
  CACHE_VERSION,
  PersistentCache,
  cacheKey,
  copyResponseHeaders,
  createDesktopServer,
  isCacheEntryFresh,
  safeStaticPath,
  shouldCache
};
