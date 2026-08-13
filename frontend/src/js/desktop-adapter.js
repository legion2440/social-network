(function (root) {
  'use strict';

  const desktop = root && root.loopDesktop;
  if (!desktop || desktop.isDesktop !== true) return;

  const searchEngine = root.LoopDesktopSearch || null;
  const OFFLINE_MESSAGE = 'No internet connection. You are offline.';
  const NativeWebSocket = root.WebSocket;
  const sockets = new Set();
  const selectors = {
    githubAuth: '.oauth-github-button',
    chatHeader: '[data-loop-chat-header]',
    messageList: '[data-loop-message-list]',
    messageRow: '[data-loop-message-row]',
    messageBody: '[data-loop-message-body]',
    loadOlder: '[data-loop-load-older]',
    chatInput: '[data-loop-chat-input]',
    chatSend: '[data-loop-chat-send]'
  };
  let browserOnline = typeof navigator === 'undefined' || navigator.onLine !== false;
  let backendOnline = true;
  let currentUserID = 0;
  let cachedUserID = 0;
  let searchQuery = '';
  let searchGeneration = 0;
  let searchLoading = false;
  let applyingUI = false;
  let uiScheduled = false;
  let reconnectReloadScheduled = false;
  let sawDisconnected = false;

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
      '#loop-desktop-search{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:6px 8px;align-items:center;',
      'padding:8px 12px;border-bottom:1px solid var(--border,#ddd);background:var(--surface,#fff)}',
      '#loop-desktop-search input{min-width:0;border:1px solid var(--border,#d6d6de);background:var(--surface2,#f6f6f8);',
      'color:inherit;border-radius:10px;padding:8px 10px;font:inherit;outline:none}',
      '#loop-desktop-search span{font-size:12px;color:var(--muted,#777);white-space:nowrap}',
      '#loop-desktop-search-help{grid-column:1/-1;font-size:11px;color:var(--muted,#777);line-height:1.3}',
      '#loop-desktop-toast{position:fixed;right:18px;bottom:18px;z-index:2147483000;background:#8d2430;color:#fff;',
      'max-width:340px;padding:10px 14px;border-radius:12px;font:600 13px/1.35 system-ui,sans-serif;',
      'box-shadow:0 10px 28px rgba(0,0,0,.22)}',
      '[data-loop-offline-guard="1"]{opacity:.68;cursor:not-allowed}'
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
    if (toast.textContent !== OFFLINE_MESSAGE) toast.textContent = OFFLINE_MESSAGE;
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

  function messageMatches(text, query) {
    if (searchEngine && typeof searchEngine.matchesMessage === 'function') {
      return searchEngine.matchesMessage(text, query);
    }
    const needle = String(query || '').trim().toLocaleLowerCase();
    return !needle || String(text || '').toLocaleLowerCase().includes(needle);
  }

  function updateSearchCount(matches) {
    const count = document.getElementById('loop-desktop-search-count');
    if (!count) return;
    const query = searchQuery.trim();
    let nextText = 'Search messages';
    if (query) {
      nextText = matches + (matches === 1 ? ' match' : ' matches');
      if (searchLoading) nextText += ' · loading history…';
    }
    if (count.textContent !== nextText) count.textContent = nextText;
  }

  function applySearch() {
    const query = searchQuery.trim();
    const rows = Array.from(document.querySelectorAll(selectors.messageRow));
    let matches = 0;
    rows.forEach(row => {
      const bubble = row.querySelector(selectors.messageBody);
      const text = String(bubble && bubble.textContent || '');
      const visible = messageMatches(text, query);
      row.style.display = visible ? '' : 'none';
      if (query && visible) matches += 1;
    });
    updateSearchCount(matches);
  }

  function waitForHistoryPage(button, previousCount, generation) {
    return new Promise(resolve => {
      const deadline = Date.now() + 6000;
      const tick = () => {
        if (generation !== searchGeneration || !searchQuery.trim()) {
          resolve(false);
          return;
        }
        const current = document.querySelector(selectors.loadOlder);
        const count = document.querySelectorAll(selectors.messageRow).length;
        if (!current || count > previousCount || current !== button) {
          resolve(true);
          return;
        }
        if (Date.now() >= deadline) {
          resolve(false);
          return;
        }
        setTimeout(tick, 60);
      };
      setTimeout(tick, 60);
    });
  }

  async function loadCompleteHistoryForSearch(generation) {
    if (searchLoading || !searchQuery.trim()) return;
    searchLoading = true;
    applySearch();
    try {
      for (let page = 0; page < 100; page += 1) {
        if (generation !== searchGeneration || !searchQuery.trim()) break;
        const button = document.querySelector(selectors.loadOlder);
        if (!button) break;
        if (button.disabled) {
          const progressed = await waitForHistoryPage(button, document.querySelectorAll(selectors.messageRow).length, generation);
          if (!progressed) break;
          continue;
        }
        const previousCount = document.querySelectorAll(selectors.messageRow).length;
        button.click();
        const progressed = await waitForHistoryPage(button, previousCount, generation);
        if (!progressed) break;
        applySearch();
      }
    } finally {
      if (generation === searchGeneration) {
        searchLoading = false;
        applySearch();
      }
    }
  }

  function beginSearch(value) {
    searchQuery = String(value || '');
    searchGeneration += 1;
    searchLoading = false;
    const generation = searchGeneration;
    applySearch();
    if (searchQuery.trim()) loadCompleteHistoryForSearch(generation).catch(() => {
      if (generation === searchGeneration) {
        searchLoading = false;
        applySearch();
      }
    });
  }

  function ensureSearch() {
    const chatHeader = document.querySelector(selectors.chatHeader);
    const messageList = document.querySelector(selectors.messageList);
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
      const help = document.createElement('div');
      help.id = 'loop-desktop-search-help';
      help.textContent = 'Operators: +include  -exclude  ~fuzzy  =10  !=10  >10  <10';
      input.addEventListener('input', event => beginSearch(event.target.value));
      search.append(input, count, help);
      chatHeader.insertAdjacentElement('afterend', search);
    }
    applySearch();
  }

  function applyOfflineControls() {
    const guarded = !online();
    document.querySelectorAll(selectors.chatInput + ', ' + selectors.chatSend).forEach(element => {
      if (guarded) {
        if (element.disabled && element.getAttribute('data-loop-disabled-overridden') !== '1') {
          element.setAttribute('data-loop-disabled-overridden', '1');
          element.removeAttribute('disabled');
        }
        if (element.getAttribute('aria-disabled') !== 'true') element.setAttribute('aria-disabled', 'true');
        if (element.getAttribute('data-loop-offline-guard') !== '1') element.setAttribute('data-loop-offline-guard', '1');
        if (element.title !== OFFLINE_MESSAGE) element.title = OFFLINE_MESSAGE;
        return;
      }
      if (element.getAttribute('data-loop-disabled-overridden') === '1') {
        element.setAttribute('disabled', 'disabled');
        element.removeAttribute('data-loop-disabled-overridden');
      }
      if (element.getAttribute('data-loop-offline-guard') === '1') element.removeAttribute('data-loop-offline-guard');
      if (element.getAttribute('aria-disabled') === 'true') element.removeAttribute('aria-disabled');
      if (element.title === OFFLINE_MESSAGE) element.removeAttribute('title');
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

  function scheduleDesktopUI() {
    if (uiScheduled) return;
    uiScheduled = true;
    const run = () => {
      uiScheduled = false;
      applyDesktopUI();
    };
    if (typeof root.requestAnimationFrame === 'function') root.requestAnimationFrame(run);
    else setTimeout(run, 0);
  }

  function scheduleReconnectReload() {
    if (reconnectReloadScheduled || !sawDisconnected || !online()) return;
    reconnectReloadScheduled = true;
    sawDisconnected = false;
    setTimeout(() => root.location.reload(), 150);
  }

  function setBackendOnline(value) {
    backendOnline = value === true;
    if (!backendOnline) {
      sawDisconnected = true;
      closeRealtimeSockets();
    }
    scheduleDesktopUI();
    if (online()) {
      refreshCurrentUser();
      scheduleReconnectReload();
    }
  }

  function setBrowserOnline(value) {
    browserOnline = value === true;
    if (!browserOnline) {
      sawDisconnected = true;
      closeRealtimeSockets();
    }
    desktop.setConnectivity(browserOnline).then(valueFromMain => {
      setBackendOnline(valueFromMain !== false);
    }).catch(() => setBackendOnline(false));
    scheduleDesktopUI();
  }

  root.addEventListener('online', () => setBrowserOnline(true));
  root.addEventListener('offline', () => setBrowserOnline(false));

  desktop.onNetworkStatus(value => setBackendOnline(value));

  if (typeof desktop.onOAuthComplete === 'function') {
    desktop.onOAuthComplete(() => {
      cachedUserID = 0;
      currentUserID = 0;
      root.location.reload();
    });
  }

  document.addEventListener('click', event => {
    const button = event.target && event.target.closest ? event.target.closest('button') : null;
    if (!button) return;

    if (button.matches(selectors.githubAuth) && typeof desktop.startGitHubOAuth === 'function') {
      event.preventDefault();
      event.stopImmediatePropagation();
      desktop.startGitHubOAuth().catch(() => {});
      return;
    }

    if (document.querySelector('[data-screen-label="Auth"]') && /register|sign\s*up|create\s+account/i.test(button.textContent || '')) {
      event.preventDefault();
      event.stopImmediatePropagation();
      desktop.openRegistration().catch(() => {});
      return;
    }

    if (!online() && button.matches(selectors.chatSend)) {
      event.preventDefault();
      event.stopImmediatePropagation();
      showOfflineToast();
    }
  }, true);

  document.addEventListener('keydown', event => {
    if (online() || event.key !== 'Enter') return;
    const target = event.target;
    if (!target || !target.matches || !target.matches(selectors.chatInput)) return;
    event.preventDefault();
    event.stopImmediatePropagation();
    showOfflineToast();
  }, true);

  const observer = new MutationObserver(() => scheduleDesktopUI());
  function start() {
    observer.observe(document.documentElement, { childList: true, subtree: true });
    applyDesktopUI();
    setBrowserOnline(browserOnline);
    refreshCurrentUser();
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start, { once: true });
  else start();
})(typeof window !== 'undefined' ? window : null);
