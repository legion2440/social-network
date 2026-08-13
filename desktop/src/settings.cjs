'use strict';

const fsp = require('node:fs/promises');
const http = require('node:http');
const https = require('node:https');
const path = require('node:path');

const SETTINGS_FILE = 'settings.json';

function normalizeServerURL(value, fallback = '') {
  let raw = String(value || '').trim();
  if (!raw) raw = String(fallback || '').trim();
  if (!raw) return '';
  if (/^[a-z][a-z0-9+.-]*:\/\//i.test(raw) && !/^https?:\/\//i.test(raw)) {
    throw new Error('Server URL must use http:// or https://');
  }
  if (!/^https?:\/\//i.test(raw)) raw = 'http://' + raw;
  const parsed = new URL(raw);
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error('Server URL must use http:// or https://');
  }
  if (parsed.username || parsed.password) throw new Error('Server URL cannot contain credentials');
  return parsed.origin;
}

function settingsPath(userDataDir) {
  return path.join(userDataDir, SETTINGS_FILE);
}

async function loadSettings(userDataDir) {
  try {
    const raw = await fsp.readFile(settingsPath(userDataDir), 'utf8');
    const parsed = JSON.parse(raw);
    const serverURL = normalizeServerURL(parsed && parsed.serverURL || '');
    if (!serverURL) return null;
    return {
      serverURL,
      lastSuccessfulAt: String(parsed && parsed.lastSuccessfulAt || '')
    };
  } catch (error) {
    if (error && error.code === 'ENOENT') return null;
    if (error instanceof SyntaxError || /Server URL/.test(String(error && error.message || ''))) return null;
    throw error;
  }
}

async function saveSettings(userDataDir, settings) {
  const serverURL = normalizeServerURL(settings && settings.serverURL || '');
  if (!serverURL) throw new Error('Server URL is required');
  const payload = {
    version: 1,
    serverURL,
    lastSuccessfulAt: String(settings && settings.lastSuccessfulAt || '')
  };
  await fsp.mkdir(userDataDir, { recursive: true });
  const file = settingsPath(userDataDir);
  const tmp = file + '.tmp';
  await fsp.writeFile(tmp, JSON.stringify(payload, null, 2) + '\n', 'utf8');
  await fsp.rename(tmp, file);
  return payload;
}

function probeOrigin(origin, timeoutMs = 3000) {
  return new Promise(resolve => {
    let target;
    try {
      target = new URL(normalizeServerURL(origin));
    } catch (_error) {
      resolve(false);
      return;
    }
    const transport = target.protocol === 'https:' ? https : http;
    const req = transport.request({
      protocol: target.protocol,
      hostname: target.hostname,
      port: target.port || undefined,
      method: 'GET',
      path: '/api/health',
      headers: {
        Accept: 'application/json',
        Host: target.host,
        Origin: target.origin
      },
      timeout: timeoutMs
    }, res => {
      const ok = (res.statusCode || 0) >= 200 && (res.statusCode || 0) < 300;
      res.resume();
      res.on('end', () => resolve(ok));
    });
    req.on('timeout', () => req.destroy());
    req.on('error', () => resolve(false));
    req.end();
  });
}

module.exports = {
  SETTINGS_FILE,
  loadSettings,
  normalizeServerURL,
  probeOrigin,
  saveSettings,
  settingsPath
};
