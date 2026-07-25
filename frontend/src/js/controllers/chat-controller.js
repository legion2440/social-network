(function (root, factory) {
  var create = factory(root && root.createFeatureController, root && root.createControllerContext);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createChatController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController, createContext) {
  return function createChatController(dependencies) {
    var context = createContext(dependencies);
    var AuthAPI = dependencies.api;
    var USERS = dependencies.session.users;
    var UserModel = dependencies.models.users;
    var ChatModel = dependencies.models.chats;
    var emptyChatMessages = dependencies.helpers.emptyChatMessages;
    var createClientMessageID = dependencies.helpers.createClientMessageID;
    var requestErrorMessage = dependencies.helpers.requestErrorMessage;
    var decorateUser = dependencies.helpers.decorateUser;
    var formatPostTime = dependencies.helpers.formatPostTime;
    var num = dependencies.helpers.num;
    var GROUP_COLORS = dependencies.values.GROUP_COLORS;
    var EMOJIS = dependencies.values.EMOJIS;

  context.chatHistoryGate = function (key) {
    if (!context.chatHistoryGatesByKey[key]) context.chatHistoryGatesByKey[key] = UserModel.createRequestGate();
    return context.chatHistoryGatesByKey[key];
  };

  context.chatAccessGate = function (key) {
    if (!context.chatAccessGatesByKey[key]) context.chatAccessGatesByKey[key] = UserModel.createRequestGate();
    return context.chatAccessGatesByKey[key];
  };

  context.chatReadGate = function (key) {
    if (!context.chatReadGatesByKey[key]) context.chatReadGatesByKey[key] = UserModel.createRequestGate();
    return context.chatReadGatesByKey[key];
  };

  context.chatMessages = function (key) {
    return context.state.messagesByChatKey[key] || emptyChatMessages();
  };

  context.documentIsVisible = function () {
    return typeof document === 'undefined' || !document || document.visibilityState === undefined ||
      document.visibilityState === 'visible';
  };

  context.compareChatReadCandidates = function (left, right) {
    if (!left) return right ? -1 : 0;
    if (!right) return 1;
    const leftTime = Date.parse(left.createdAt) || 0;
    const rightTime = Date.parse(right.createdAt) || 0;
    if (leftTime !== rightTime) return leftTime - rightTime;
    return Number(left.id) - Number(right.id);
  };

  context.latestAuthoritativeChatCandidate = function (key) {
    const messages = context.chatMessages(key).messages || [];
    let latest = null;
    messages.forEach(message => {
      const id = Number(message && message.apiId);
      if (!Number.isInteger(id) || id <= 0 || !message.createdAt) return;
      const candidate = { id, createdAt: String(message.createdAt) };
      if (!latest || context.compareChatReadCandidates(candidate, latest) > 0) latest = candidate;
    });
    return latest;
  };

  context.chatReadEligible = function (key) {
    const chat = ChatModel.parseChatKey(key);
    if (!chat || context.state.screen !== 'chat' || context.state.activeChatKey !== key ||
        !context.documentIsVisible() || !context.chatMessages(key).loaded || context.revokedChatKeys.has(key)) return false;
    return chat.kind !== 'group' || !context.groupAccessIsRevoked(chat.target_id);
  };

  context.enqueueChatRead = function (key) {
    if (!context.chatReadEligible(key)) return;
    const candidate = context.latestAuthoritativeChatCandidate(key);
    if (!candidate) return;
    const queued = context.state.chatReadQueuedThroughByKey[key];
    const sent = context.chatReadSentCandidateByKey[key];
    if (queued && context.compareChatReadCandidates(candidate, queued) <= 0) {
      if (!context.chatReadInFlightByKey[key]) {
        context.drainChatReadQueue(key);
      }
      return;
    }
    if (!queued && sent && context.compareChatReadCandidates(candidate, sent) <= 0) return;
    context.setState(current => ({
      chatReadQueuedThroughByKey: Object.assign({}, current.chatReadQueuedThroughByKey, { [key]: candidate }),
      chatReadErrorByKey: Object.assign({}, current.chatReadErrorByKey, { [key]: '' })
    }), () => context.drainChatReadQueue(key));
  };

  context.drainChatReadQueue = async key => {
    if (context.chatReadInFlightByKey[key] || !context.chatReadEligible(key)) return;
    const chat = ChatModel.parseChatKey(key);
    const sentCandidate = context.state.chatReadQueuedThroughByKey[key];
    if (!chat || !sentCandidate) return;
    const authGeneration = context.authGate.current();
    const accessGate = context.chatAccessGate(key);
    const accessGeneration = accessGate.current();
    const readGate = context.chatReadGate(key);
    const readGeneration = readGate.current();
    context.chatReadInFlightByKey[key] = true;
    context.chatReadSentCandidateByKey[key] = sentCandidate;
    context.setState(current => ({
      chatReadPendingByKey: Object.assign({}, current.chatReadPendingByKey, { [key]: true })
    }));
    try {
      const response = chat.kind === 'direct'
        ? await AuthAPI.markDirectChatRead(chat.target_id, sentCandidate.id)
        : await AuthAPI.markGroupChatRead(chat.target_id, sentCandidate.id);
      if (!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
          !readGate.isCurrent(readGeneration)) return;
      context.applyChatUnreadPayload(response);
      const queued = context.state.chatReadQueuedThroughByKey[key];
      const hasNewer = queued && context.compareChatReadCandidates(queued, sentCandidate) > 0;
      const pending = Object.assign({}, context.state.chatReadPendingByKey);
      const queuedByKey = Object.assign({}, context.state.chatReadQueuedThroughByKey);
      if (!hasNewer) {
        delete pending[key];
        delete queuedByKey[key];
      }
      context.setState({
        chatReadPendingByKey: pending,
        chatReadQueuedThroughByKey: queuedByKey,
        chatReadErrorByKey: Object.assign({}, context.state.chatReadErrorByKey, { [key]: '' })
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
          !readGate.isCurrent(readGeneration)) return;
      if (chat.kind === 'group' && error && (error.status === 403 || error.status === 404)) {
        context.purgeChat(key);
        return;
      }
      const pending = Object.assign({}, context.state.chatReadPendingByKey);
      delete pending[key];
      context.setState({
        chatReadPendingByKey: pending,
        chatReadErrorByKey: Object.assign({}, context.state.chatReadErrorByKey, {
          [key]: requestErrorMessage(error, 'Could not mark conversation as read.')
        })
      });
    } finally {
      if (context.authGate.isCurrent(authGeneration) && accessGate.isCurrent(accessGeneration) &&
          readGate.isCurrent(readGeneration)) {
        delete context.chatReadInFlightByKey[key];
        const queued = context.state.chatReadQueuedThroughByKey[key];
        if (queued && context.compareChatReadCandidates(queued, sentCandidate) > 0) {
          context.drainChatReadQueue(key);
        }
      }
    }
  };

  context.applyChatUnreadPayload = function (payload) {
    const revision = Number(payload && payload.revision);
    const unreadCount = Number(payload && payload.unread_count);
    const chatUnreadCount = Number(payload && payload.chat_unread_count);
    let key;
    try {
      key = ChatModel.chatKey(payload && payload.chat && payload.chat.kind, payload && payload.chat && payload.chat.target_id);
    } catch (ignore) {
      return false;
    }
    if (!Number.isInteger(revision) || revision < Number(context.state.chatUnreadRevision || 0) ||
        !Number.isInteger(unreadCount) || unreadCount < 0 ||
        !Number.isInteger(chatUnreadCount) || chatUnreadCount < 0) return false;
    context.setState(current => {
      if (revision < Number(current.chatUnreadRevision || 0)) return {};
      const chatUnreadByKey = Object.assign({}, current.chatUnreadByKey);
      const chatsByKey = Object.assign({}, current.chatsByKey);
      const readThrough = Object.assign({}, current.chatReadThroughMessageIDByKey);
      const chat = ChatModel.parseChatKey(key);
      const canApplyKey = !!chatsByKey[key] &&
        !(chat && chat.kind === 'group' && context.groupAccessIsRevoked(chat.target_id));
      if (canApplyKey) {
        chatUnreadByKey[key] = chatUnreadCount;
        chatsByKey[key] = Object.assign({}, chatsByKey[key], { unreadCount: chatUnreadCount });
      }
      const markerID = Number(payload.read_through_message_id);
      if (Number.isInteger(markerID) && markerID > 0) readThrough[key] = markerID;
      return {
        chatsByKey, chatUnreadByKey, chatUnreadCount: unreadCount, chatUnreadRevision: revision,
        chatReadThroughMessageIDByKey: readThrough
      };
    });
    return true;
  };

  context.usesMobileChatLayout = () => (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(max-width: 600px)').matches
  );

  context.loadChats = async (reset = true, historyReason = 'background') => {
    const authGeneration = context.authGate.current();
    const generation = reset ? context.chatsGate.begin() : context.chatsGate.current();
    if (!reset && context.state.chatsPending) return;
    const cursor = reset ? null : context.state.chatsNextCursor;
    if (!reset && !cursor) return;
    context.setState({ chatsPending: true, chatsLoading: !!reset, chatsError: '' });
    try {
      const page = await AuthAPI.chats(cursor, 20);
      if (!context.authGate.isCurrent(authGeneration) || !context.chatsGate.isCurrent(generation)) return;
      const rawChats = (page.chats || []).filter(chat => (
        chat && (chat.kind !== 'group' || !context.groupAccessIsRevoked(chat.target_id))
      ));
      const rawUsers = [];
      const rawGroups = [];
      rawChats.forEach(chat => {
        if (chat.user) rawUsers.push(chat.user);
        if (chat.group) {
          rawGroups.push(chat.group);
          if (chat.group.owner) rawUsers.push(chat.group.owner);
        }
        if (chat.last_message && chat.last_message.sender) rawUsers.push(chat.last_message.sender);
      });
      const normalized = rawChats.map(ChatModel.normalizeChatSummary);
      const revision = page.revision === undefined
        ? Number(context.state.chatUnreadRevision || 0)
        : Number(page.revision);
      const unreadCount = page.unread_count === undefined
        ? Number(context.state.chatUnreadCount || 0)
        : Number(page.unread_count);
      if (!Number.isInteger(revision) || revision < 0 ||
          !Number.isInteger(unreadCount) || unreadCount < 0) {
        throw new TypeError('invalid chat page');
      }
      normalized.forEach(chat => {
        if (chat.kind === 'group' && !context.groupAccessIsRevoked(chat.targetID)) {
          context.revokedChatKeys.delete(chat.key);
        }
      });
      normalized.forEach(chat => {
        if (chat.lastMessage) context.settlePendingMessage(chat.lastMessage.clientMessageID);
      });
      const apiUsersByID = context.mergeAPIUsers(rawUsers);
      context.setState(current => {
        const chatsByKey = ChatModel.mergeChatSummaries(current.chatsByKey, normalized);
        const chatUnreadByKey = Object.assign({}, current.chatUnreadByKey);
        if (revision >= Number(current.chatUnreadRevision || 0)) {
          normalized.forEach(chat => {
            chatUnreadByKey[chat.key] = chat.unreadCount;
            if (chatsByKey[chat.key]) {
              chatsByKey[chat.key] = Object.assign({}, chatsByKey[chat.key], {
                unreadCount: chat.unreadCount
              });
            }
          });
        } else {
          normalized.forEach(chat => {
            if (chatsByKey[chat.key]) {
              chatsByKey[chat.key] = Object.assign({}, chatsByKey[chat.key], {
                unreadCount: Math.max(0, Number(chatUnreadByKey[chat.key]) || 0)
              });
            }
          });
        }
        const activeChatKey = current.activeChatKey && chatsByKey[current.activeChatKey]
          ? current.activeChatKey
          : (current.mobileChatList ? null : (ChatModel.sortedChatKeys(chatsByKey)[0] || null));
        const patch = {
          apiUsersByID,
          apiGroupsByID: context.mergeGroupResponses(rawGroups, current.apiGroupsByID),
          chatsByKey,
          chatKeys: ChatModel.sortedChatKeys(chatsByKey),
          chatUnreadByKey,
          activeChatKey,
          chatsNextCursor: page.next_cursor || null,
          chatsPending: false, chatsLoading: false, chatsError: ''
        };
        if (revision >= Number(current.chatUnreadRevision || 0)) {
          patch.chatUnreadCount = unreadCount;
          patch.chatUnreadRevision = revision;
        }
        return patch;
      }, () => {
        const activeHistory = context.state.activeChatKey
          ? context.chatMessages(context.state.activeChatKey)
          : emptyChatMessages();
        const shouldReloadHistory = context.state.activeChatKey &&
          (!activeHistory.loaded || (reset && historyReason !== 'user-open'));
        if (shouldReloadHistory) {
          context.loadChatHistory(context.state.activeChatKey, true, historyReason);
        } else if (historyReason === 'user-open' && context.state.activeChatKey) {
          context.enqueueChatRead(context.state.activeChatKey);
        }
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !context.chatsGate.isCurrent(generation)) return;
      context.setState({
        chatsPending: false, chatsLoading: false,
        chatsError: requestErrorMessage(error, 'Could not load chats. Please try again.')
      });
    }
  };

  context.loadChatHistory = async (key, reset = true, reason = 'background') => {
    const chat = ChatModel.parseChatKey(key);
    if (!chat) return;
    const authGeneration = context.authGate.current();
    const accessGate = context.chatAccessGate(key);
    const accessGeneration = accessGate.current();
    const historyGate = context.chatHistoryGate(key);
    const historyGeneration = reset ? historyGate.begin() : historyGate.current();
    const previous = context.chatMessages(key);
    if (!reset && previous.pending) return;
    const cursor = reset ? null : previous.nextCursor;
    if (!reset && !cursor) return;
    if (!reset && context.msgEl && context.state.activeChatKey === key) {
      context.chatScrollAnchor = { key, height: context.msgEl.scrollHeight, top: context.msgEl.scrollTop };
    }
    context.setState(current => {
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      messagesByChatKey[key] = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {}, {
        loading: !!reset, pending: true, error: ''
      });
      return { messagesByChatKey };
    });
    try {
      const page = chat.kind === 'direct'
        ? await AuthAPI.directMessages(chat.target_id, cursor, 20)
        : await AuthAPI.groupMessages(chat.target_id, cursor, 20);
      if (!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
          !historyGate.isCurrent(historyGeneration)) return;
      const rawMessages = page.messages || [];
      const normalized = rawMessages.map(ChatModel.normalizeMessage);
      normalized.forEach(message => context.settlePendingMessage(message.clientMessageID));
      const apiUsersByID = context.mergeAPIUsers(rawMessages.map(message => message.sender));
      if (reset && context.state.activeChatKey === key) context.scrollChatToBottom = true;
      context.setState(current => {
        const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
        const currentEntry = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {});
        messagesByChatKey[key] = Object.assign({}, currentEntry, {
          messages: ChatModel.mergeMessages(currentEntry.messages, normalized),
          nextCursor: page.next_cursor || null,
          loading: false, pending: false, error: '', loaded: true
        });
        return { apiUsersByID, messagesByChatKey };
      }, () => {
        if (reason === 'user-open') context.enqueueChatRead(key);
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
          !historyGate.isCurrent(historyGeneration)) return;
      if ((error && (error.status === 403 || error.status === 404)) && chat.kind === 'group') {
        context.purgeChat(key);
        return;
      }
      context.setState(current => {
        const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
        messagesByChatKey[key] = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {}, {
          loading: false, pending: false, loaded: true,
          error: error && error.status === 403
            ? 'You cannot access this conversation.'
            : requestErrorMessage(error, 'Could not load messages. Please try again.')
        });
        return { messagesByChatKey };
      });
    }
  };

  context.openChat = key => {
    const chat = ChatModel.parseChatKey(key);
    if (!chat || !context.state.chatsByKey[key]) return;
    if (chat.kind === 'group' && context.groupAccessIsRevoked(chat.target_id)) return;
    context.stopTyping();
    context.activeChatGate.begin();
    context.scrollChatToBottom = true;
    context.setState({
      screen: 'chat', activeChatKey: key, mobileChatList: false,
      emojiOpen: false, chatDraft: '', chatError: ''
    }, () => {
      const history = context.chatMessages(key);
      if (!history.loaded) context.loadChatHistory(key, true, 'user-open');
      else context.enqueueChatRead(key);
    });
  };

  context.openDirectChat = userID => {
    userID = Number(userID);
    if (!Number.isInteger(userID) || userID <= 0 || userID === Number(USERS.me.apiId)) return;
    if (dependencies.navigation && !dependencies.navigation.isApplying()) {
      dependencies.navigation.directChat(userID);
      return;
    }
    const key = ChatModel.chatKey('direct', userID);
    const user = context.apiUser(userID);
    context.setState(current => {
      const chatsByKey = Object.assign({}, current.chatsByKey);
      if (!chatsByKey[key]) {
        chatsByKey[key] = {
          key, kind: 'direct', targetID: userID, userID, groupID: null,
          lastMessage: null, activityAt: new Date().toISOString(), transient: true
        };
      }
      return { chatsByKey, chatKeys: ChatModel.sortedChatKeys(chatsByKey) };
    }, () => context.openChat(key));
  };

  context.openGroupChat = async groupID => {
    groupID = Number(groupID);
    if (!Number.isInteger(groupID) || groupID <= 0) return;
    if (dependencies.navigation && !dependencies.navigation.isApplying()) {
      dependencies.navigation.groupChat(groupID);
      return;
    }
    let group = context.state.apiGroupsByID[String(groupID)];
    if (!group) {
      const authGeneration = context.authGate.current();
      context.setState({ screen: 'chat', chatError: '' });
      try {
        const raw = await AuthAPI.group(groupID);
        if (!context.authGate.isCurrent(authGeneration)) return;
        group = context.mapAPIGroup(raw);
        const apiUsersByID = context.mergeAPIUsers([raw.owner]);
        context.setState(current => ({
          apiUsersByID,
          apiGroupsByID: Object.assign({}, current.apiGroupsByID, { [String(groupID)]: group })
        }));
      } catch (error) {
        if (!context.authGate.isCurrent(authGeneration)) return;
        context.setState({
          screen: 'chat',
          chatError: error && error.status === 404
            ? 'Group not found.'
            : requestErrorMessage(error, 'Could not open this group conversation.')
        });
        return;
      }
    }
    if (context.groupAccessIsRevoked(groupID) || (group.state !== 'owner' && group.state !== 'member')) {
      context.setState({ screen: 'chat', chatError: 'Only group members can open this conversation.' });
      return;
    }
    const key = ChatModel.chatKey('group', groupID);
    context.revokedChatKeys.delete(key);
    context.setState(current => {
      const chatsByKey = Object.assign({}, current.chatsByKey);
      if (!chatsByKey[key]) {
        chatsByKey[key] = {
          key, kind: 'group', targetID: groupID, userID: null, groupID,
          lastMessage: null, activityAt: new Date().toISOString(), transient: true
        };
      }
      return { chatsByKey, chatKeys: ChatModel.sortedChatKeys(chatsByKey) };
    }, () => context.openChat(key));
  };

  context.purgeChat = function (key) {
    if (!ChatModel.parseChatKey(key)) return;
    context.revokedChatKeys.add(key);
    context.chatAccessGate(key).begin();
    context.chatHistoryGate(key).begin();
    context.chatReadGate(key).begin();
    delete context.chatReadInFlightByKey[key];
    delete context.chatReadSentCandidateByKey[key];
    if (context.typingChatKey === key) context.stopTyping(false);
    const history = context.chatMessages(key);
    history.messages.forEach(message => context.settlePendingMessage(message.clientMessageID));
    context.setState(current => {
      const chatsByKey = Object.assign({}, current.chatsByKey);
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      const typingByChatKey = Object.assign({}, current.typingByChatKey);
      const chatUnreadByKey = Object.assign({}, current.chatUnreadByKey);
      const chatReadPendingByKey = Object.assign({}, current.chatReadPendingByKey);
      const chatReadErrorByKey = Object.assign({}, current.chatReadErrorByKey);
      const chatReadQueuedThroughByKey = Object.assign({}, current.chatReadQueuedThroughByKey);
      const chatReadThroughMessageIDByKey = Object.assign({}, current.chatReadThroughMessageIDByKey);
      delete chatsByKey[key];
      delete messagesByChatKey[key];
      delete typingByChatKey[key];
      delete chatUnreadByKey[key];
      delete chatReadPendingByKey[key];
      delete chatReadErrorByKey[key];
      delete chatReadQueuedThroughByKey[key];
      delete chatReadThroughMessageIDByKey[key];
      const chatKeys = ChatModel.sortedChatKeys(chatsByKey);
      return {
        chatsByKey, messagesByChatKey, typingByChatKey, chatKeys, chatUnreadByKey,
        chatReadPendingByKey, chatReadErrorByKey, chatReadQueuedThroughByKey,
        chatReadThroughMessageIDByKey,
        activeChatKey: current.activeChatKey === key ? (chatKeys[0] || null) : current.activeChatKey
      };
    });
  };

  context.handleRealtimeMessage = function (rawMessage) {
    let message;
    try { message = ChatModel.normalizeMessage(rawMessage); } catch (ignore) { return; }
    const key = message.chatKey;
    if (context.revokedChatKeys.has(key)) return;
    const wasKnown = !!context.state.chatsByKey[key];
    context.settlePendingMessage(message.clientMessageID);
    let apiUsersByID = context.state.apiUsersByID;
    if (!apiUsersByID[String(message.senderID)] && message.senderID !== Number(USERS.me.apiId)) {
      apiUsersByID = context.mergeAPIUsers([{
        id: message.senderID, first_name: message.senderName, last_name: '',
        avatar_url: message.senderAvatarURL || '/static/avatars/neutral.svg', is_private: false
      }]);
    }
    if (context.state.activeChatKey === key) context.scrollChatToBottom = true;
    context.setState(current => {
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      const history = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {});
      messagesByChatKey[key] = Object.assign({}, history, {
        messages: ChatModel.mergeMessages(history.messages, [message])
      });
      const chatsByKey = Object.assign({}, current.chatsByKey);
      const existing = chatsByKey[key];
      chatsByKey[key] = Object.assign({}, existing || {
        key, kind: message.chat.kind, targetID: message.chat.target_id,
        userID: message.chat.kind === 'direct' ? message.chat.target_id : null,
        groupID: message.chat.kind === 'group' ? message.chat.target_id : null,
        transient: true
      }, {
        lastMessage: message, activityAt: message.createdAt
      });
      return {
        apiUsersByID, messagesByChatKey, chatsByKey,
        chatKeys: ChatModel.sortedChatKeys(chatsByKey), chatError: ''
      };
    }, () => {
      if (context.state.activeChatKey === key) context.enqueueChatRead(key);
      if (!wasKnown) context.loadChats(true);
    });
  };

  context.handleRealtimeError = function (event) {
    const clientMessageID = String(event.client_message_id || '').trim().toLowerCase();
    const messageText = String(event.message || 'Could not send message.');
    if (!clientMessageID) {
      context.setState({ chatError: messageText });
      return;
    }
    context.settlePendingMessage(clientMessageID);
    context.setState(current => {
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      Object.keys(messagesByChatKey).forEach(key => {
        const history = Object.assign({}, messagesByChatKey[key]);
        let changed = false;
        history.messages = (history.messages || []).map(message => {
          if (message.clientMessageID !== clientMessageID) return message;
          changed = true;
          return Object.assign({}, message, { pending: false, failed: true, error: messageText });
        });
        if (changed) messagesByChatKey[key] = history;
      });
      return { messagesByChatKey, chatError: messageText };
    });
  };

  context.handleTypingUpdate = function (event) {
    const userID = Number(event.user && event.user.id);
    let key;
    try { key = ChatModel.chatKey(event.chat && event.chat.kind, event.chat && event.chat.target_id); } catch (ignore) { return; }
    if (!Number.isInteger(userID) || userID <= 0 || userID === Number(USERS.me.apiId)) return;
    const timerKey = key + ':' + userID;
    if (context.typingExpiryTimers[timerKey]) {
      clearTimeout(context.typingExpiryTimers[timerKey]);
      delete context.typingExpiryTimers[timerKey];
    }
    context.setState(current => {
      const typingByChatKey = Object.assign({}, current.typingByChatKey);
      const users = Object.assign({}, typingByChatKey[key] || {});
      if (event.typing === true) {
        users[String(userID)] = {
          id: userID, name: String(event.user.display_name || ('User ' + userID))
        };
      } else {
        delete users[String(userID)];
      }
      if (Object.keys(users).length) typingByChatKey[key] = users;
      else delete typingByChatKey[key];
      return { typingByChatKey };
    });
    if (event.typing === true) {
      context.typingExpiryTimers[timerKey] = setTimeout(() => {
        delete context.typingExpiryTimers[timerKey];
        context.handleTypingUpdate({
          chat: event.chat, user: event.user, typing: false
        });
      }, 6000);
    }
  };

  context.sendTypingEvent = function (type, key) {
    const chat = ChatModel.parseChatKey(key);
    if (!chat || !context.ws || context.ws.readyState !== 1) return false;
    try {
      context.ws.send(JSON.stringify({ type, chat: { kind: chat.kind, target_id: chat.target_id } }));
      return true;
    } catch (ignore) {
      return false;
    }
  };

  context.startTyping = function (key) {
    if (!key || context.state.wsStatus !== 'connected') return;
    if (context.typingChatKey && context.typingChatKey !== key) context.stopTyping();
    if (context.typingChatKey === key) return;
    if (!context.sendTypingEvent('typing:start', key)) return;
    context.typingChatKey = key;
    context.typingHeartbeatTimer = setInterval(() => {
      if (!context.typingChatKey || !context.state.chatDraft.trim()) {
        context.stopTyping();
        return;
      }
      context.sendTypingEvent('typing:heartbeat', context.typingChatKey);
    }, 2000);
  };

  context.stopTyping = function (sendEvent = true) {
    const key = context.typingChatKey;
    context.typingChatKey = null;
    if (context.typingHeartbeatTimer) {
      clearInterval(context.typingHeartbeatTimer);
      context.typingHeartbeatTimer = null;
    }
    if (sendEvent && key) context.sendTypingEvent('typing:stop', key);
  };

  context.onChatDraft = value => {
    context.setState({ chatDraft: value, chatError: '' }, () => {
      if (context.state.chatDraft.trim() && context.state.activeChatKey) context.startTyping(context.state.activeChatKey);
      else context.stopTyping();
    });
  };

  context.settlePendingMessage = function (clientMessageID) {
    clientMessageID = String(clientMessageID || '').toLowerCase();
    if (context.pendingMessageTimers[clientMessageID]) {
      clearTimeout(context.pendingMessageTimers[clientMessageID]);
      delete context.pendingMessageTimers[clientMessageID];
    }
  };

  context.armPendingMessageTimeout = function (clientMessageID, key) {
    context.settlePendingMessage(clientMessageID);
    const authGeneration = context.authGate.current();
    const accessGeneration = context.chatAccessGate(key).current();
    context.pendingMessageTimers[clientMessageID] = setTimeout(() => {
      delete context.pendingMessageTimers[clientMessageID];
      if (!context.authGate.isCurrent(authGeneration) || !context.chatAccessGate(key).isCurrent(accessGeneration)) return;
      context.setState(current => {
        const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
        const history = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {});
        history.messages = history.messages.map(message => message.clientMessageID === clientMessageID && message.pending
          ? Object.assign({}, message, {
            pending: false, failed: true,
            error: 'No response from server. Retry with the same message ID.'
          })
          : message);
        messagesByChatKey[key] = history;
        return { messagesByChatKey };
      });
    }, 15000);
  };

  context.sendPendingMessage = function (
    message,
    authGeneration = context.authGate.current(),
    accessGeneration = message && message.chatKey ? context.chatAccessGate(message.chatKey).current() : -1
  ) {
    if (!message || !message.clientMessageID || !message.chat) return false;
    if (!context.authGate.isCurrent(authGeneration) ||
        !context.chatAccessGate(message.chatKey).isCurrent(accessGeneration) ||
        context.state.authStatus !== 'authenticated') return false;
    if (!context.ws || context.ws.readyState !== 1 || context.state.wsStatus !== 'connected') {
      context.handleRealtimeError({
        client_message_id: message.clientMessageID,
        message: 'Realtime is disconnected. Reconnect and retry.'
      });
      return false;
    }
    context.setState(current => {
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      const history = Object.assign({}, emptyChatMessages(), messagesByChatKey[message.chatKey] || {});
      history.messages = history.messages.map(item => item.clientMessageID === message.clientMessageID
        ? Object.assign({}, item, { pending: true, failed: false, error: '' })
        : item);
      messagesByChatKey[message.chatKey] = history;
      return { messagesByChatKey, chatError: '' };
    });
    try {
      context.ws.send(JSON.stringify({
        type: 'chat:send',
        client_message_id: message.clientMessageID,
        chat: { kind: message.chat.kind, target_id: message.chat.target_id },
        text: message.body
      }));
      context.armPendingMessageTimeout(message.clientMessageID, message.chatKey);
      return true;
    } catch (error) {
      context.handleRealtimeError({
        client_message_id: message.clientMessageID,
        message: 'Could not send message. Please retry.'
      });
      return false;
    }
  };

  context.sendMsg = () => {
    if (context.chatSendLock) return;
    const key = context.state.activeChatKey;
    const chat = ChatModel.parseChatKey(key);
    const body = context.state.chatDraft.trim();
    if (!chat || !body) return;
    if (Array.from(body).length > 2000) {
      context.setState({ chatError: 'Messages are limited to 2000 characters.' });
      return;
    }
    const authGeneration = context.authGate.current();
    const accessGeneration = context.chatAccessGate(key).current();
    context.chatSendLock = true;
    const message = ChatModel.pendingMessage(
      createClientMessageID(), chat, USERS.me.apiId, body, new Date().toISOString()
    );
    context.stopTyping();
    context.scrollChatToBottom = true;
    context.setState(current => {
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      const history = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {});
      history.messages = ChatModel.mergeMessages(history.messages, [message]);
      messagesByChatKey[key] = history;
      return { messagesByChatKey, chatDraft: '', emojiOpen: false, chatError: '' };
    }, () => {
      context.chatSendLock = false;
      context.sendPendingMessage(message, authGeneration, accessGeneration);
    });
  };

  context.retryMessage = clientMessageID => {
    let found = null;
    Object.keys(context.state.messagesByChatKey).some(key => {
      found = (context.state.messagesByChatKey[key].messages || []).find(message => (
        message.clientMessageID === clientMessageID
      ));
      return !!found;
    });
    if (!found || found.pending || !found.failed) return;
    context.sendPendingMessage(found);
  };

  context.toggleEmoji = () => context.setState({ emojiOpen: !context.state.emojiOpen });

  context.setMessageElement = element => { context.msgEl = element; };

    return createController('chat', dependencies, {
      chatHistoryGate: context.chatHistoryGate,
      chatAccessGate: context.chatAccessGate,
      chatReadGate: context.chatReadGate,
      chatMessages: context.chatMessages,
      documentIsVisible: context.documentIsVisible,
      compareChatReadCandidates: context.compareChatReadCandidates,
      latestAuthoritativeChatCandidate: context.latestAuthoritativeChatCandidate,
      chatReadEligible: context.chatReadEligible,
      enqueueChatRead: context.enqueueChatRead,
      drainChatReadQueue: context.drainChatReadQueue,
      applyChatUnreadPayload: context.applyChatUnreadPayload,
      usesMobileChatLayout: context.usesMobileChatLayout,
      loadChats: context.loadChats,
      loadChatHistory: context.loadChatHistory,
      openChat: context.openChat,
      openDirectChat: context.openDirectChat,
      openGroupChat: context.openGroupChat,
      purgeChat: context.purgeChat,
      handleRealtimeMessage: context.handleRealtimeMessage,
      handleRealtimeError: context.handleRealtimeError,
      handleTypingUpdate: context.handleTypingUpdate,
      sendTypingEvent: context.sendTypingEvent,
      startTyping: context.startTyping,
      stopTyping: context.stopTyping,
      onChatDraft: context.onChatDraft,
      settlePendingMessage: context.settlePendingMessage,
      armPendingMessageTimeout: context.armPendingMessageTimeout,
      sendPendingMessage: context.sendPendingMessage,
      sendMsg: context.sendMsg,
      retryMessage: context.retryMessage,
      toggleEmoji: context.toggleEmoji,
      setMessageElement: context.setMessageElement
    }, function (state) {
      var me = USERS.me;

      function chatMeta(chat) {
        if (!chat) {
          return {
            title: 'Select a conversation',
            initials: '…',
            color: 'var(--text3)',
            sub: '',
            avatarUrl: '',
            hasAvatar: false,
            noAvatar: true,
            online: false
          };
        }
        if (chat.kind === 'direct') {
          var user = context.apiUser(chat.userID || chat.targetID);
          var online = !!state.onlineUserIDs[String(user.apiId)];
          return {
            title: user.name,
            initials: user.initials,
            color: user.color,
            sub: online ? 'Online now' : 'Offline',
            online: online,
            avatarUrl: user.avatarUrl,
            hasAvatar: user.hasAvatar,
            noAvatar: user.noAvatar
          };
        }
        var group = state.apiGroupsByID[
          String(chat.groupID || chat.targetID)
        ] || {
          name: 'Group ' + chat.targetID,
          members: 0,
          color: GROUP_COLORS[
            Math.abs(chat.targetID) % GROUP_COLORS.length
          ]
        };
        return {
          title: group.name,
          initials: String(group.name || 'G').slice(0, 2).toUpperCase(),
          color: group.color,
          sub: num(group.members || 0) + ' members',
          online: false,
          avatarUrl: '',
          hasAvatar: false,
          noAvatar: true
        };
      }

      var conversations = state.chatKeys.map(function (key) {
        var chat = state.chatsByKey[key];
        var meta = chatMeta(chat);
        var last = chat.lastMessage;
        var unread = Math.max(
          0,
          Number(
            state.chatUnreadByKey[key] !== undefined
              ? state.chatUnreadByKey[key]
              : chat.unreadCount
          ) || 0
        );
        return {
          title: meta.title,
          initials: meta.initials,
          color: meta.color,
          avatarUrl: meta.avatarUrl,
          hasAvatar: meta.hasAvatar,
          noAvatar: meta.noAvatar,
          preview: last
            ? (Number(last.senderID) === Number(me.apiId) ? 'You: ' : '') +
              last.body
            : 'No messages yet',
          previewColor: unread > 0 ? 'var(--text)' : 'var(--text3)',
          previewW: unread > 0 ? '750' : '500',
          hasUnread: unread > 0,
          unread: num(unread > 99 ? '99+' : unread),
          time: last ? formatPostTime(last.createdAt) : '',
          online: meta.online,
          bg: key === state.activeChatKey ? 'var(--soft)' : 'transparent',
          open: function () {
            var target = ChatModel.parseChatKey(key);
            if (target && target.kind === 'direct') {
              return context.openDirectChat(target.target_id);
            }
            if (target && target.kind === 'group') {
              return context.openGroupChat(target.target_id);
            }
          }
        };
      });
      var active = state.activeChatKey
        ? state.chatsByKey[state.activeChatKey]
        : null;
      var activeMeta = chatMeta(active);
      var activeHistory = state.activeChatKey
        ? (state.messagesByChatKey[state.activeChatKey] || emptyChatMessages())
        : emptyChatMessages();
      var activeMessages = activeHistory.messages || [];
      var messages = activeMessages.map(function (message, index) {
        var previous = activeMessages[index - 1];
        var user = context.apiUser(message.senderID);
        if (
          (!user || user.name.indexOf('User ') === 0) &&
          message.senderName
        ) {
          user = decorateUser({
            id: String(message.senderID),
            apiId: message.senderID,
            name: message.senderName,
            initials: message.senderName
              .split(/\s+/)
              .map(function (part) { return part.charAt(0); })
              .join('')
              .slice(0, 2)
              .toUpperCase() || '?',
            color: GROUP_COLORS[
              Math.abs(message.senderID) % GROUP_COLORS.length
            ],
            avatarUrl: message.senderAvatarURL ||
              '/static/avatars/neutral.svg'
          });
        }
        return {
          text: message.body,
          time: formatPostTime(message.createdAt),
          mine: Number(message.senderID) === Number(me.apiId),
          theirs: Number(message.senderID) !== Number(me.apiId),
          user: user,
          showName: active && active.kind === 'group' &&
            Number(message.senderID) !== Number(me.apiId) &&
            (!previous ||
              Number(previous.senderID) !== Number(message.senderID)),
          pending: !!message.pending,
          failed: !!message.failed,
          error: message.error || '',
          hasStatus: !!message.pending || !!message.failed,
          statusLabel: message.failed
            ? 'Failed'
            : (message.pending ? 'Sending…' : ''),
          retry: function () {
            return context.retryMessage(message.clientMessageID);
          }
        };
      });
      var typingUsers = Object.values(
        state.typingByChatKey[state.activeChatKey] || {}
      );
      var typingLabel = typingUsers.length > 1
        ? typingUsers.map(function (user) { return user.name; }).join(', ') +
          ' are typing'
        : (typingUsers[0] ? typingUsers[0].name + ' is typing' : '');
      var emojis = EMOJIS.map(function (character) {
        return {
          ch: character,
          add: function () {
            return context.onChatDraft(state.chatDraft + character);
          }
        };
      });

      return {
        convos: conversations,
        messages: messages,
        activeTitle: activeMeta.title,
        activeSub: activeMeta.sub,
        activeInitials: activeMeta.initials,
        activeColor: activeMeta.color,
        activeAvatarUrl: activeMeta.avatarUrl,
        activeHasAvatar: activeMeta.hasAvatar,
        activeNoAvatar: activeMeta.noAvatar,
        chatHasActive: !!active,
        chatHasNoActive: !active && !state.chatsLoading,
        chatLayoutClass: active ? 'chat-active' : '',
        backToChats: context.goChat,
        chatsLoading: state.chatsLoading,
        chatsHasError: !!state.chatsError,
        chatsError: state.chatsError,
        retryChats: function () { return context.loadChats(true); },
        chatsHasMore: !!state.chatsNextCursor,
        loadMoreChats: function () { return context.loadChats(false); },
        chatsLoadMoreDisabled: state.chatsPending,
        historyLoading: activeHistory.loading,
        historyHasError: !!activeHistory.error,
        historyError: activeHistory.error,
        retryHistory: function () {
          return state.activeChatKey &&
            context.loadChatHistory(state.activeChatKey, true, 'user-open');
        },
        historyHasMore: !!activeHistory.nextCursor,
        loadMoreHistory: function () {
          return state.activeChatKey &&
            context.loadChatHistory(state.activeChatKey, false);
        },
        historyLoadMoreDisabled: activeHistory.pending,
        typing: typingUsers.length > 0,
        typingLabel: typingLabel,
        chatDraft: state.chatDraft,
        onChatDraft: function (event) {
          return context.onChatDraft(event.target.value);
        },
        onChatBlur: context.stopTyping,
        onChatKey: function (event) {
          if (event.key !== 'Enter') return;
          event.preventDefault();
          return context.sendMsg();
        },
        sendMsg: context.sendMsg,
        chatSendDisabled: !active ||
          state.wsStatus !== 'connected' || !state.chatDraft.trim(),
        chatInputDisabled: !active || state.wsStatus !== 'connected',
        chatStatus: state.wsStatus === 'connected'
          ? 'Live'
          : (state.wsStatus === 'reconnecting'
            ? 'Reconnecting…'
            : 'Connecting…'),
        chatHasError: !!state.chatError,
        chatError: state.chatError,
        emojiOpen: state.emojiOpen,
        toggleEmoji: context.toggleEmoji,
        emojis: emojis,
        msgRef: context.setMessageElement
      };
    }, {});
  };
});
