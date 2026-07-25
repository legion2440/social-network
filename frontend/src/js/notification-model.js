(function (root, factory) {
  var api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  if (root) root.NotificationModel = api;
})(typeof window !== 'undefined' ? window : globalThis, function () {
  var types = {
    follow_started: true,
    follow_request: true,
    group_invitation: true,
    group_join_request: true,
    group_event: true
  };
  var actionable = {
    follow_request: true,
    group_invitation: true,
    group_join_request: true
  };

  function normalize(raw) {
    var id = Number(raw && raw.id);
    var type = raw && String(raw.type || '');
    var actorID = Number(raw && raw.actor && raw.actor.id);
    if (!Number.isInteger(id) || id <= 0 || !types[type] || !Number.isInteger(actorID) || actorID <= 0) {
      throw new TypeError('invalid notification');
    }
    return {
      id: id,
      type: type,
      actor: raw.actor,
      actorID: actorID,
      followID: raw.follow_id == null ? null : Number(raw.follow_id),
      group: raw.group || null,
      event: raw.event || null,
      resolution: raw.resolution == null ? null : String(raw.resolution),
      resolvedAt: raw.resolved_at || null,
      readAt: raw.read_at || null,
      createdAt: String(raw.created_at || '')
    };
  }

  function merge(current, incoming) {
    var byID = {};
    (current || []).concat(incoming || []).forEach(function (item) {
      if (item && Number.isInteger(Number(item.id)) && Number(item.id) > 0) byID[String(Number(item.id))] = item;
    });
    return Object.keys(byID).map(function (id) { return byID[id]; }).sort(function (left, right) {
      var timeOrder = String(right.createdAt || '').localeCompare(String(left.createdAt || ''));
      return timeOrder || Number(right.id) - Number(left.id);
    });
  }

  function sourceKey(notification) {
    if (!notification || !actionable[notification.type] || notification.resolution != null) return null;
    if (notification.type === 'follow_request') return 'follow_request:' + notification.actorID;
    var groupID = Number(notification.group && notification.group.id);
    if (!Number.isInteger(groupID) || groupID <= 0) return null;
    if (notification.type === 'group_invitation') return 'group_invitation:' + groupID;
    return 'group_join_request:' + groupID + ':' + notification.actorID;
  }

  function markAllRead(notifications, readAt) {
    return (notifications || []).map(function (notification) {
      return notification.readAt ? notification : Object.assign({}, notification, { readAt: readAt });
    });
  }

  function isActionable(notification) {
    return !!(notification && actionable[notification.type] && notification.resolution == null);
  }

  return {
    normalize: normalize,
    merge: merge,
    sourceKey: sourceKey,
    markAllRead: markAllRead,
    isActionable: isActionable
  };
});
