(function (root) {
  'use strict';

  const desktop = root && root.loopDesktop;
  if (!desktop || desktop.isDesktop !== true) return;

  const OFFLINE_MESSAGE = 'No internet connection. You are offline.';
  const NativeWebSocket = root.WebSocket;
  const sockets = new Set();
  let browserOnline = typeof navigator === 'undefined' || navigator.onLine !== false;
  let backendOnline = true;
  let currentUserID = 0;
  let cachedUserID = 0;
  let searchQuery = '';
  let applyingUI = false;

  function online() {
    return browserOnline && backendOnline;
  }

  function closeRealtimeSockets() {
    sockets.forEach(socket => {
      try { socket.close(); } catch (_error) {}
    });
  }

  function notifyMessage(message) {
    if (!message || !online() || currentUserID <= 0) return;
    const sender = message.sender || {};
    const senderID = Number(sender.id);
    if (!Number.isInteger(senderID) || senderID <= 0 || senderID === currentUserID) return;
    const title = String(sender.display_name || 'New message');
    const body = String(message.body || 'New message');
    desktop.notify({ title, body }).catch(() => {});
  }

  if (typeof NativeWebSocket === 'function') {
    function DesktopWebSocket(url, protocols) {
      const socket = protocols === undefined
        ? new NativeWebSocket(url)
        : new NativeWebSocket(url, protocols);
      sockets.add(socket);
      socket.addEventListener('open', () => refreshCurrentUser());
      socket.addEventListener('close', () => sockets.delete(socket));
      socket.addEventListener('message', event => {
        if (!online()) {
          if (typeof event.stopImmediatePropagation === 'function') event.stopImmediatePropagation();
          return;
        }
        try {
          const envelope = JSON.parse(String(event.data || ''));
          if (envelope && envelope.type === 'chat:message') notifyMessage(envelope.message);
        } catch (_error) {}
      });
      return socket;
    }
    DesktopWebSocket.prototype = NativeWebSocket.prototype;
    Object.setPrototypeOf(DesktopWebSocket, NativeWebSocket);
    root.WebSocket = DesktopWebSocket;
  }

  async function fetchJSON(url) {
    const response = await fetch(url, {
      method: 'GET',
      credentials: 'same-origin',
      headers: { Accept: 'application/json' }
    });
    if (!response.ok) throw new Error('request failed');
    return response.json();
  }

  async function warmChatHistory(chat) {
    if (!chat || !chat.kind || !chat.target_id) return;
    const targetID = encodeURIComponent(String(chat.target_id));
    const base = chat.kind === 'direct'
      ? '/api/chats/direct/' + targetID + '/messages'
      : chat.kind === 'group'
        ? '/api/groups/' + targetID + '/chat/messages'
        : '';
    if (!base) return;

    let cursor = '';
    for (let page = 0; page < 100 && online(); page += 1) {
      const suffix = cursor ? '?cursor=' + encodeURIComponent(cursor) + '&limit=20' : '?limit=20';
      const data = await fetchJSON(base + suffix);
      cursor = String(data && data.next_cursor || '');
      if (!cursor) return;
    }
  }

  async function warmOfflineCache(userID) {
    if (!online() || !Number.isInteger(userID) || userID <= 0 || cachedUserID === userID) return;
    cachedUserID = userID;
    try {
      let cursor = '';
      const chats = [];
      for (let page = 0; page < 100 && online(); page += 1) {
        const suffix = cursor ? '?cursor=' + encodeURIComponent(cursor) + '&limit=20' : '?limit=20';
        const data = await fetchJSON('/api/chats' + suffix);
        (data && data.chats || []).forEach(chat => chats.push(chat));
        cursor = String(data && data.next_cursor || '');
        if (!cursor) break;
      }
      for (const chat of chats) {
        if (!online()) break;
        try { await warmChatHistory(chat); } catch (_error) {}
      }
    } catch (_error) {
      cachedUserID = 0;
    }
  }

  async function refreshCurrentUser() {
    if (!online()) return;
    try {
      const user = await fetchJSON('/api/auth/me');
      const id = Number(user && user.id);
      currentUserID = Number.isInteger(id) && id > 0 ? id : 0;
      if (currentUserID > 0) warmOfflineCache(currentUserID);
    } catch (_error) {
      currentUserID = 0;
    }
  }

  function ensureStyles() {
    if (document.getElementById('loop-desktop-style')) return;
    const style = document.createElement('style');
    style.id = 'loop-desktop-style';
    style.textContent = [
      '#loop-desktop-offline{position:fixed;left:50%;top:12px;transform:translateX(-50%);z-index:2147483000;',
      'background:#8d2430;color:#fff;padding:8px 14px;border-radius:999px;font:600 13px/1.2 system-ui,sans-serif;',
      'box-shadow:0 8px 26px rgba(0,0,0,.18)}',
      '#loop-desktop-search{display:flex;gap:8px;align-items:center;padding:8px 12px;border-bottom:1px solid var(--border,#ddd);',
      'background:var(--surface,#fff)}',
      '#loop-desktop-search input{flex:1;min-width:0;border:1px solid var(--border,#d6d6de);background:var(--surface2,#f6f6f8);',
      'color:inherit;border-radius:10px;padding:8px 10px;font:inherit;outline:none}',
      '#loop-desktop-search span{font-size:12px;color:var(--muted,#777);white-space:nowrap}',
      '#loop-desktop-toast{position:fixed;right:18px;bottom:18px;z-index:2147483000;background:#8d2430;color:#fff;',
      'max-width:340px;padding:10px 14px;border-radius:12px;font:600 13px/1.35 system-ui,sans-serif;',
      'box-shadow:0 10px 28px rgba(0,0,0,.22)}'
    ].join('');
    document.head.appendChild(style);
  }

  function showOfflineToast() {
    let toast = document.getElementById('loop-desktop-toast');
    if (!toast) {
      toast = document.createElement('div');
      toast.id = 'loop-desktop-toast';
      toast.setAttribute('role', 'alert');
      document.body.appendChild(toast);
    }
    toast.textContent = OFFLINE_MESSAGE;
    clearTimeout(showOfflineToast.timer);
    showOfflineToast.timer = setTimeout(() => {
      if (toast && toast.parentNode) toast.remove();
    }, 3500);
  }

  function ensureOfflineBanner() {
    const existing = document.getElementById('loop-desktop-offline');
    if (online()) {
      if (existing) existing.remove();
      return;
    }
    if (existing) return;
    const banner = document.createElement('div');
    banner.id = 'loop-desktop-offline';
    banner.setAttribute('role', 'status');
    banner.textContent = OFFLINE_MESSAGE;
    document.body.appendChild(banner);
  }

  function ensureSearch() {
    const chatHeader = document.querySelector('.ui-050');
    const messageList = document.querySelector('.ui-054');
    if (!chatHeader || !messageList) return;

    let search = document.getElementById('loop-desktop-search');
    if (!search) {
      search = document.createElement('div');
      search.id = 'loop-desktop-search';
      const input = document.createElement('input');
      input.type = 'search';
      input.placeholder = 'Search messages…';
      input.setAttribute('aria-label', 'Search messages');
      input.value = searchQuery;
      const count = document.createElement('span');
      count.id = 'loop-desktop-search-count';
      count.textContent = 'Search messages';
      input.addEventListener('input', event => {
        searchQuery = String(event.target.value || '');
        applySearch();
      });
      search.append(input, count);
      chatHeader.insertAdjacentElement('afterend', search);
    }
    applySearch();
  }

  function applySearch() {
    const needle = searchQuery.trim().toLocaleLowerCase();
    const rows = Array.from(document.querySelectorAll('.ui-054 .ui-058, .ui-054 .ui-063'));
    let matches = 0;
    rows.forEach(row => {
      const bubble = row.querySelector('.ui-059, .ui-066');
      const text = String(bubble && bubble.textContent || '').toLocaleLowerCase();
      const visible = !needle || text.includes(needle);
      row.style.display = visible ? '' : 'none';
      if (needle && visible) matches += 1;
    });
    const count = document.getElementById('loop-desktop-search-count');
    if (count) count.textContent = needle ? (matches + (matches === 1 ? ' match' : ' matches')) : 'Search messages';
  }

  function applyOfflineControls() {
    const disabled = !online();
    document.querySelectorAll('.ui-079, .ui-080').forEach(element => {
      if (disabled) {
        if (!element.disabled) element.dataset.loopDesktopDisabled = '1';
        element.setAttribute('disabled', 'disabled');
        element.setAttribute('aria-disabled', 'true');
        element.title = OFFLINE_MESSAGE;
        return;
      }
      element.removeAttribute('aria-disabled');
      if (element.title === OFFLINE_MESSAGE) element.removeAttribute('title');
      if (element.dataset.loopDesktopDisabled === '1') {
        delete element.dataset.loopDesktopDisabled;
        element.removeAttribute('disabled');
      }
    });
  }

  function applyDesktopUI() {
    if (applyingUI || !document.body) return;
    applyingUI = true;
    try {
      ensureStyles();
      ensureOfflineBanner();
      ensureSearch();
      applyOfflineControls();
    } finally {
      applyingUI = false;
    }
  }

  function setBrowserOnline(value) {
    browserOnline = value === true;
    if (!browserOnline) closeRealtimeSockets();
    desktop.setConnectivity(browserOnline).then(valueFromMain => {
      backendOnline = valueFromMain !== false;
      applyDesktopUI();
      if (online()) refreshCurrentUser();
    }).catch(() => {
      backendOnline = false;
      applyDesktopUI();
    });
    applyDesktopUI();
  }

  root.addEventListener('online', () => setBrowserOnline(true));
  root.addEventListener('offline', () => setBrowserOnline(false));

  desktop.onNetworkStatus(value => {
    backendOnline = value === true;
    if (!backendOnline) closeRealtimeSockets();
    applyDesktopUI();
    if (online()) refreshCurrentUser();
  });

  document.addEventListener('click', event => {
    const button = event.target && event.target.closest ? event.target.closest('button') : null;
    if (!button) return;

    if (button.classList.contains('ui-013') && /register|sign\s*up|create\s+account/i.test(button.textContent || '')) {
      event.preventDefault();
      event.stopImmediatePropagation();
      desktop.openRegistration().catch(() => {});
      return;
    }

    if (!online() && button.classList.contains('ui-080')) {
      event.preventDefault();
      event.stopImmediatePropagation();
      showOfflineToast();
    }
  }, true);

  document.addEventListener('keydown', event => {
    if (online() || event.key !== 'Enter') return;
    const target = event.target;
    if (!target || !target.classList || !target.classList.contains('ui-079')) return;
    event.preventDefault();
    event.stopImmediatePropagation();
    showOfflineToast();
  }, true);

  const observer = new MutationObserver(() => applyDesktopUI());
  function start() {
    observer.observe(document.documentElement, { childList: true, subtree: true });
    applyDesktopUI();
    setBrowserOnline(browserOnline);
    refreshCurrentUser();
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start, { once: true });
  else start();
})(typeof window !== 'undefined' ? window : null);
