(function (root, factory) {
  var library = factory();
  if (typeof module === 'object' && module.exports) module.exports = library;
  if (root) root.ChatModel = library;
})(typeof window !== 'undefined' ? window : null, function () {
  function validKind(kind) {
    return kind === 'direct' || kind === 'group';
  }

  function chatKey(kind, targetID) {
    targetID = Number(targetID);
    if (!validKind(kind) || !Number.isInteger(targetID) || targetID <= 0) {
      throw new TypeError('valid chat kind and positive target id are required');
    }
    return kind + ':' + targetID;
  }

  function parseChatKey(key) {
    var parts = String(key || '').split(':');
    var targetID = Number(parts[1]);
    if (parts.length !== 2 || !validKind(parts[0]) || !Number.isInteger(targetID) || targetID <= 0) {
      return null;
    }
    return { kind: parts[0], target_id: targetID };
  }

  function normalizeMessage(raw) {
    if (!raw || !raw.chat || !raw.sender) throw new TypeError('message chat and sender are required');
    var id = Number(raw.id);
    var senderID = Number(raw.sender.id);
    var targetID = Number(raw.chat.target_id);
    var clientMessageID = String(raw.client_message_id || '').trim().toLowerCase();
    if (!Number.isInteger(id) || id <= 0 || !Number.isInteger(senderID) || senderID <= 0 ||
        !validKind(raw.chat.kind) || !Number.isInteger(targetID) || targetID <= 0 || !clientMessageID) {
      throw new TypeError('invalid authoritative chat message');
    }
    return {
      id: String(id), apiId: id, clientMessageID: clientMessageID,
      chat: { kind: raw.chat.kind, target_id: targetID },
      chatKey: chatKey(raw.chat.kind, targetID), senderID: senderID,
      senderName: String(raw.sender.display_name ||
        ((raw.sender.first_name || '') + ' ' + (raw.sender.last_name || '')).trim() || ('User ' + senderID)),
      senderAvatarURL: String(raw.sender.avatar_url || ''),
      body: String(raw.body || ''), createdAt: String(raw.created_at || ''),
      pending: false, failed: false, error: ''
    };
  }

  function pendingMessage(clientMessageID, chat, senderID, body, createdAt) {
    var key = chatKey(chat && chat.kind, chat && chat.target_id);
    return {
      id: 'pending:' + clientMessageID, apiId: null,
      clientMessageID: String(clientMessageID || '').trim().toLowerCase(),
      chat: { kind: chat.kind, target_id: Number(chat.target_id) }, chatKey: key,
      senderID: Number(senderID), senderName: '', senderAvatarURL: '',
      body: String(body || ''), createdAt: String(createdAt || new Date().toISOString()),
      pending: true, failed: false, error: ''
    };
  }

  function compareMessages(left, right) {
    var leftTime = Date.parse(left && left.createdAt) || 0;
    var rightTime = Date.parse(right && right.createdAt) || 0;
    if (leftTime !== rightTime) return leftTime - rightTime;
    var leftID = Number(left && left.apiId) || Number.MAX_SAFE_INTEGER;
    var rightID = Number(right && right.apiId) || Number.MAX_SAFE_INTEGER;
    if (leftID !== rightID) return leftID - rightID;
    return String(left && left.clientMessageID).localeCompare(String(right && right.clientMessageID));
  }

  function mergeMessages(existing, incoming) {
    var byServerID = {};
    var byClientID = {};
    var result = [];
    (existing || []).concat(incoming || []).forEach(function (message) {
      if (!message || !message.clientMessageID) return;
      var serverKey = message.apiId ? String(message.apiId) : '';
      var clientKey = String(message.clientMessageID).toLowerCase();
      var index = serverKey && byServerID[serverKey] !== undefined
        ? byServerID[serverKey]
        : byClientID[clientKey];
      if (index === undefined) {
        index = result.length;
        result.push(message);
      } else {
        var previous = result[index];
        result[index] = message.apiId || !previous.apiId ? message : previous;
      }
      if (result[index].apiId) byServerID[String(result[index].apiId)] = index;
      byClientID[clientKey] = index;
    });
    return result.sort(compareMessages);
  }

  function normalizeChatSummary(raw) {
    if (!raw || !validKind(raw.kind)) throw new TypeError('valid chat summary kind is required');
    var targetID = Number(raw.target_id);
    if (!Number.isInteger(targetID) || targetID <= 0) throw new TypeError('positive chat target id is required');
    if (raw.kind === 'direct' && !raw.user) throw new TypeError('direct chat user is required');
    if (raw.kind === 'group' && !raw.group) throw new TypeError('group chat is required');
    return {
      key: chatKey(raw.kind, targetID), kind: raw.kind, targetID: targetID,
      userID: raw.kind === 'direct' ? Number(raw.user.id) : null,
      groupID: raw.kind === 'group' ? Number(raw.group.id) : null,
      lastMessage: raw.last_message ? normalizeMessage(raw.last_message) : null,
      activityAt: String(raw.activity_at || ''), transient: false
    };
  }

  function compareChats(left, right) {
    var leftTime = Date.parse(left && left.activityAt) || 0;
    var rightTime = Date.parse(right && right.activityAt) || 0;
    if (leftTime !== rightTime) return rightTime - leftTime;
    var leftRank = left && left.kind === 'direct' ? 0 : 1;
    var rightRank = right && right.kind === 'direct' ? 0 : 1;
    if (leftRank !== rightRank) return leftRank - rightRank;
    return Number(right && right.targetID) - Number(left && left.targetID);
  }

  function mergeChatSummaries(store, incoming) {
    var next = Object.assign({}, store || {});
    (incoming || []).forEach(function (chat) {
      if (!chat || !chat.key) return;
      var previous = next[chat.key];
      if (previous && (Date.parse(previous.activityAt) || 0) > (Date.parse(chat.activityAt) || 0)) {
        next[chat.key] = Object.assign({}, chat, {
          activityAt: previous.activityAt,
          lastMessage: previous.lastMessage || chat.lastMessage,
          transient: previous.transient && chat.transient
        });
      } else {
        next[chat.key] = Object.assign({}, previous || {}, chat);
      }
    });
    return next;
  }

  function sortedChatKeys(store) {
    return Object.keys(store || {}).sort(function (left, right) {
      return compareChats(store[left], store[right]);
    });
  }

  return {
    chatKey: chatKey,
    parseChatKey: parseChatKey,
    normalizeMessage: normalizeMessage,
    pendingMessage: pendingMessage,
    mergeMessages: mergeMessages,
    normalizeChatSummary: normalizeChatSummary,
    mergeChatSummaries: mergeChatSummaries,
    sortedChatKeys: sortedChatKeys
  };
});
