(function (root, factory) {
  var create = factory(root && root.createFeatureController, root && root.createControllerContext);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createRealtimeController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController, createContext) {
  return function createRealtimeController(dependencies) {
    var context = createContext(dependencies);
    var AuthAPI = dependencies.api;
    var ChatModel = dependencies.models.chats;
    var NotificationModel = dependencies.models.notifications;

  context.startAuthenticatedRealtime = function (authGeneration) {
    if (!context.authGate.isCurrent(authGeneration) || context.state.authStatus !== 'authenticated') return;
    context.wsHasOpened = false;
    context.chatSendLock = false;
    context.loadChats(true);
    context.connectRealtime(authGeneration);
  };

  context.realtimeURL = function () {
    if (typeof window === 'undefined' || !window.location) return '';
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return protocol + '//' + window.location.host + '/ws';
  };

  context.connectRealtime = function (authGeneration = context.authGate.current()) {
    if (!context.authGate.isCurrent(authGeneration) || context.state.authStatus !== 'authenticated') return;
    if (context.ws && (context.ws.readyState === 0 || context.ws.readyState === 1)) return;
    if (typeof WebSocket !== 'function') {
      context.setState({ wsStatus: 'disconnected', chatError: 'Realtime is unavailable in this browser.' });
      return;
    }
    if (context.wsReconnectTimer) {
      clearTimeout(context.wsReconnectTimer);
      context.wsReconnectTimer = null;
    }
    const url = context.realtimeURL();
    if (!url) return;
    const generation = ++context.wsGeneration;
    const socket = new WebSocket(url);
    context.ws = socket;
    context.setState({ wsStatus: 'connecting' });
    socket.onopen = () => {
      if (!context.authGate.isCurrent(authGeneration) || generation !== context.wsGeneration || context.ws !== socket) {
        socket.close();
        return;
      }
      const reconnect = context.wsHasOpened;
      context.wsHasOpened = true;
      context.setState({ wsStatus: 'connected', wsReconnectAttempt: 0, chatError: '' }, () => {
        if (!reconnect) return;
        context.loadChats(true);
        context.loadNotifications(true);
        if (context.state.activeChatKey) context.loadChatHistory(context.state.activeChatKey, true);
      });
    };
    socket.onmessage = event => {
      if (!context.authGate.isCurrent(authGeneration) || generation !== context.wsGeneration || context.ws !== socket) return;
      context.handleRealtimeEvent(event && event.data);
    };
    socket.onclose = () => {
      if (!context.authGate.isCurrent(authGeneration) || generation !== context.wsGeneration || context.ws !== socket) return;
      context.ws = null;
      context.stopTyping(false);
      context.setState({ wsStatus: 'reconnecting' }, () => context.scheduleRealtimeReconnect(authGeneration));
    };
    socket.onerror = () => {};
  };

  context.scheduleRealtimeReconnect = function (authGeneration) {
    if (context.wsReconnectTimer || !context.authGate.isCurrent(authGeneration) || context.state.authStatus !== 'authenticated') return;
    const attempt = Math.max(0, Number(context.state.wsReconnectAttempt) || 0);
    const base = Math.min(15000, 500 * Math.pow(2, attempt));
    const delay = Math.round(base * (0.75 + Math.random() * 0.5));
    context.setState({ wsReconnectAttempt: attempt + 1 });
    context.wsReconnectTimer = setTimeout(() => {
      context.wsReconnectTimer = null;
      context.connectRealtime(authGeneration);
    }, delay);
  };

  context.stopRealtime = function (updateState = true) {
    context.stopTyping(false);
    context.wsGeneration += 1;
    if (context.wsReconnectTimer) {
      clearTimeout(context.wsReconnectTimer);
      context.wsReconnectTimer = null;
    }
    const socket = context.ws;
    context.ws = null;
    if (socket && typeof socket.close === 'function') {
      try { socket.close(); } catch (ignore) {}
    }
    Object.keys(context.pendingMessageTimers).forEach(id => clearTimeout(context.pendingMessageTimers[id]));
    context.pendingMessageTimers = {};
    Object.keys(context.typingExpiryTimers).forEach(id => clearTimeout(context.typingExpiryTimers[id]));
    context.typingExpiryTimers = {};
    context.wsHasOpened = false;
    if (updateState && context.state) {
      context.setState({ wsStatus: 'disconnected', wsReconnectAttempt: 0, onlineUserIDs: {}, typingByChatKey: {} });
    }
  };

  context.handleRealtimeEvent = function (raw) {
    let event;
    try { event = JSON.parse(String(raw || '')); } catch (ignore) { return; }
    if (!event || typeof event.type !== 'string') return;
    if (event.type === 'notification:upsert') {
      context.applyNotificationPayload(event, true);
      return;
    }
    if (event.type === 'notifications:read-all') {
      const revision = Number(event.revision);
      const unreadCount = Number(event.unread_count);
      if (!Number.isInteger(revision) || revision < Number(context.state.notificationRevision || 0) ||
          !Number.isInteger(unreadCount) || unreadCount < 0) return;
      context.setState(current => {
        if (revision < Number(current.notificationRevision || 0)) return {};
        return {
          notifications: NotificationModel.markAllRead(current.notifications, event.read_at),
          notificationUnreadCount: unreadCount,
          notificationRevision: revision
        };
      });
      return;
    }
    if (event.type === 'presence:init') {
      const onlineUserIDs = {};
      (event.online_user_ids || []).forEach(id => {
        id = Number(id);
        if (Number.isInteger(id) && id > 0) onlineUserIDs[String(id)] = true;
      });
      context.setState({ onlineUserIDs });
      return;
    }
    if (event.type === 'presence:update') {
      const userID = Number(event.user_id);
      if (!Number.isInteger(userID) || userID <= 0) return;
      context.setState(current => {
        const onlineUserIDs = Object.assign({}, current.onlineUserIDs);
        if (event.online === true) onlineUserIDs[String(userID)] = true;
        else delete onlineUserIDs[String(userID)];
        return { onlineUserIDs };
      });
      return;
    }
    if (event.type === 'presence:remove') {
      const userID = Number(event.user_id);
      context.setState(current => {
        const onlineUserIDs = Object.assign({}, current.onlineUserIDs);
        delete onlineUserIDs[String(userID)];
        return { onlineUserIDs };
      });
      return;
    }
    if (event.type === 'chat:remove') {
      if (!event.chat || event.chat.kind !== 'group') return;
	  const groupID = Number(event.chat.target_id);
	  if (!Number.isInteger(groupID) || groupID <= 0) return;
      let key;
      try { key = ChatModel.chatKey(event.chat.kind, event.chat.target_id); } catch (ignore) { return; }
	  context.revokeGroupAccess(groupID);
      context.purgeChat(key);
      return;
    }
    if (event.type === 'typing:update') {
      context.handleTypingUpdate(event);
      return;
    }
    if (event.type === 'chat:unread') {
      context.applyChatUnreadPayload(event);
      return;
    }
    if (event.type === 'chat:message' && event.message) {
      context.handleRealtimeMessage(event.message);
      return;
    }
    if (event.type === 'chat:error') context.handleRealtimeError(event);
  };

    return createController('realtime', dependencies, {
      startAuthenticatedRealtime: context.startAuthenticatedRealtime,
      realtimeURL: context.realtimeURL,
      connectRealtime: context.connectRealtime,
      scheduleRealtimeReconnect: context.scheduleRealtimeReconnect,
      stopRealtime: context.stopRealtime,
      handleRealtimeEvent: context.handleRealtimeEvent
    }, function (state) {
      return { status: state ? state.wsStatus : 'disconnected' };
    }, { stop: context.stopRealtime });
  };
});
