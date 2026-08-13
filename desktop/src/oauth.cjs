'use strict';

function normalizedOrigin(value) {
  const parsed = new URL(String(value || ''));
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error('unsupported origin');
  }
  return parsed.origin;
}

function sameSiteForTarget(value, secure) {
  const normalized = String(value || 'unspecified').toLowerCase();
  if (normalized === 'none' || normalized === 'no_restriction') {
    return secure ? 'no_restriction' : 'lax';
  }
  if (normalized === 'strict') return 'strict';
  if (normalized === 'lax') return 'lax';
  return 'unspecified';
}

function cookieForOrigin(cookie, targetOrigin) {
  const origin = new URL(normalizedOrigin(targetOrigin));
  const path = String(cookie && cookie.path || '/').startsWith('/')
    ? String(cookie.path || '/')
    : '/';
  const secure = origin.protocol === 'https:' && cookie && cookie.secure === true;
  const result = {
    url: origin.origin + path,
    name: String(cookie && cookie.name || ''),
    value: String(cookie && cookie.value || ''),
    path,
    httpOnly: cookie && cookie.httpOnly === true,
    secure,
    sameSite: sameSiteForTarget(cookie && cookie.sameSite, secure)
  };
  if (cookie && Number.isFinite(cookie.expirationDate) && cookie.expirationDate > 0) {
    result.expirationDate = cookie.expirationDate;
  }
  return result;
}

async function mirrorSessionCookies(session, sourceOrigin, targetOrigin) {
  if (!session || !session.cookies || typeof session.cookies.get !== 'function' ||
      typeof session.cookies.set !== 'function') {
    throw new TypeError('Electron session with cookies API is required');
  }
  const source = normalizedOrigin(sourceOrigin);
  const target = normalizedOrigin(targetOrigin);
  if (source === target) return 0;

  const cookies = await session.cookies.get({ url: source });
  let copied = 0;
  for (const cookie of cookies) {
    if (!cookie || !cookie.name) continue;
    await session.cookies.set(cookieForOrigin(cookie, target));
    copied += 1;
  }
  return copied;
}

function isOriginURL(value, origin) {
  try {
    return new URL(String(value || '')).origin === normalizedOrigin(origin);
  } catch (_error) {
    return false;
  }
}

function isAllowedOAuthURL(value, targetOrigin) {
  try {
    const parsed = new URL(String(value || ''));
    if (parsed.origin === normalizedOrigin(targetOrigin)) return true;
    return parsed.protocol === 'https:' && parsed.hostname.toLowerCase() === 'github.com';
  } catch (_error) {
    return false;
  }
}

module.exports = {
  cookieForOrigin,
  isAllowedOAuthURL,
  isOriginURL,
  mirrorSessionCookies,
  normalizedOrigin
};
