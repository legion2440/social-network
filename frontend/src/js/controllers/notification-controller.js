(function (root, factory) {
  var create = factory(root && root.createFeatureController, root && root.createControllerContext);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createNotificationController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController, createContext) {
  return function createNotificationController(dependencies) {
    var context = createContext(dependencies);
    var AuthAPI = dependencies.api;
    var UserModel = dependencies.models.users;
    var NotificationModel = dependencies.models.notifications;
    var requestErrorMessage = dependencies.helpers.requestErrorMessage;

  context.notificationReadGate = function (notificationID) {
    const key = String(Number(notificationID));
    if (!context.notificationReadGatesByID[key]) context.notificationReadGatesByID[key] = UserModel.createRequestGate();
    return context.notificationReadGatesByID[key];
  };

  context.notificationActionGate = function (notificationID) {
    const key = String(Number(notificationID));
    if (!context.notificationActionGatesByID[key]) context.notificationActionGatesByID[key] = UserModel.createRequestGate();
    return context.notificationActionGatesByID[key];
  };

  context.relationshipGeneration = function (userID) {
    const key = String(Number(userID));
    if (!context.relationshipGenerationsByID[key]) context.relationshipGenerationsByID[key] = UserModel.createRequestGate();
    return context.relationshipGenerationsByID[key];
  };

  context.advanceRelationshipLifecycle = function (userID) {
    const key = String(Number(userID));
    const generation = context.relationshipGeneration(userID).begin();
    if (context.state.followPendingByID[key]) {
      const pending = Object.assign({}, context.state.followPendingByID);
      delete pending[key];
      context.setState({ followPendingByID: pending });
    }
    return generation;
  };

  context.beginRelationshipGeneration = function (userID) {
    const generation = context.advanceRelationshipLifecycle(userID);
    context.directoryGate.begin();
    context.postFollowersGate.begin();
    if (Number(context.state.profileId) === Number(userID)) context.profileGate.begin();
    return generation;
  };

  context.trackNotificationLifecycles = function (notifications) {
    (notifications || []).forEach(notification => {
      const sourceKey = NotificationModel.sourceKey(notification);
      if (!sourceKey) return;
      const previousID = context.latestActionableNotificationIDBySourceKey[sourceKey];
      if (Number(previousID) === Number(notification.id)) return;
      if (notification.type === 'follow_request') {
        context.advanceRelationshipLifecycle(notification.actorID);
      } else if (notification.group && notification.group.id) {
        const groupID = Number(notification.group.id);
        context.groupGeneration(groupID).begin();
        if (Number(context.state.groupId) === groupID) {
          context.loadGroupDetail(groupID);
          context.loadGroupMembers(groupID, true);
        }
      }
      context.latestActionableNotificationIDBySourceKey[sourceKey] = notification.id;
    });
  };

  context.applyNotificationPayload = function (payload, trackLifecycle) {
    const revision = Number(payload && payload.revision);
    const unreadCount = Number(payload && payload.unread_count);
    if (!Number.isInteger(revision) || revision < 0 || !Number.isInteger(unreadCount) || unreadCount < 0) return false;
    if (revision < Number(context.state.notificationRevision || 0)) return false;
    let notification;
    try { notification = NotificationModel.normalize(payload.notification); } catch (ignore) { return false; }
    if (trackLifecycle) context.trackNotificationLifecycles([notification]);
    context.setState(current => {
      if (revision < Number(current.notificationRevision || 0)) return {};
      return {
        apiUsersByID: context.mergeAPIUsers([notification.actor], current.apiUsersByID),
        notifications: NotificationModel.merge(current.notifications, [notification]),
        notificationUnreadCount: unreadCount,
        notificationRevision: revision
      };
    });
    return true;
  };

  context.loadNotifications = async (reset = true) => {
    const authGeneration = context.authGate.current();
    const generation = reset ? context.notificationsGate.begin() : context.notificationsGate.current();
    if (!reset && context.state.notificationsPending) return;
    const cursor = reset ? null : context.state.notificationsNextCursor;
    if (!reset && !cursor) return;
    context.setState({ notificationsPending: true, notificationsLoading: !!reset, notificationsError: '' });
    try {
      const page = await AuthAPI.notifications(cursor, 20);
      if (!context.authGate.isCurrent(authGeneration) || !context.notificationsGate.isCurrent(generation)) return;
      const revision = Number(page.revision);
      const unreadCount = Number(page.unread_count);
      if (!Number.isInteger(revision) || revision < 0 || !Number.isInteger(unreadCount) || unreadCount < 0) {
        throw new TypeError('invalid notification page');
      }
      if (revision < Number(context.state.notificationRevision || 0)) {
        context.setState({ notificationsPending: false, notificationsLoading: false });
        return;
      }
      const incoming = (page.notifications || []).map(NotificationModel.normalize);
      if (reset) context.trackNotificationLifecycles(incoming);
      context.setState(current => {
        if (revision < Number(current.notificationRevision || 0)) {
          return { notificationsPending: false, notificationsLoading: false };
        }
        return {
          apiUsersByID: context.mergeAPIUsers(incoming.map(notification => notification.actor), current.apiUsersByID),
          notifications: reset ? incoming : NotificationModel.merge(current.notifications, incoming),
          notificationsNextCursor: page.next_cursor || null,
          notificationsPending: false, notificationsLoading: false, notificationsError: '',
          notificationUnreadCount: unreadCount, notificationRevision: revision
        };
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !context.notificationsGate.isCurrent(generation)) return;
      context.setState({
        notificationsPending: false, notificationsLoading: false,
        notificationsError: requestErrorMessage(error, 'Could not load notifications.')
      });
    }
  };

  context.markNotificationRead = async notificationID => {
    notificationID = Number(notificationID);
    const key = String(notificationID);
    const notification = context.state.notifications.find(item => item.id === notificationID);
    if (!notification || notification.readAt || context.state.notificationReadPendingByID[key]) return;
    const authGeneration = context.authGate.current();
    const gate = context.notificationReadGate(notificationID);
    const generation = gate.begin();
    context.setState({
      notificationReadPendingByID: Object.assign({}, context.state.notificationReadPendingByID, { [key]: true }),
      notificationReadErrorByID: Object.assign({}, context.state.notificationReadErrorByID, { [key]: '' })
    });
    try {
      const response = await AuthAPI.markNotificationRead(notificationID);
      if (!context.authGate.isCurrent(authGeneration) || !gate.isCurrent(generation)) return;
      context.applyNotificationPayload(response, false);
      const pending = Object.assign({}, context.state.notificationReadPendingByID);
      delete pending[key];
      context.setState({ notificationReadPendingByID: pending });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !gate.isCurrent(generation)) return;
      const pending = Object.assign({}, context.state.notificationReadPendingByID);
      delete pending[key];
      context.setState({
        notificationReadPendingByID: pending,
        notificationReadErrorByID: Object.assign({}, context.state.notificationReadErrorByID, {
          [key]: requestErrorMessage(error, 'Could not mark notification as read.')
        })
      });
    }
  };

  context.markAllNotificationsRead = async () => {
    if (context.state.notificationReadAllPending || context.state.notificationUnreadCount <= 0) return;
    const authGeneration = context.authGate.current();
    const generation = context.notificationReadAllGate.begin();
    context.setState({ notificationReadAllPending: true, notificationsError: '' });
    try {
      const response = await AuthAPI.markAllNotificationsRead();
      if (!context.authGate.isCurrent(authGeneration) || !context.notificationReadAllGate.isCurrent(generation)) return;
      const revision = Number(response.revision);
      if (Number.isInteger(revision) && revision >= Number(context.state.notificationRevision || 0)) {
        context.setState(current => ({
          notifications: NotificationModel.markAllRead(current.notifications, response.read_at),
          notificationUnreadCount: Number(response.unread_count) || 0,
          notificationRevision: revision,
          notificationReadAllPending: false
        }));
      } else {
        context.setState({ notificationReadAllPending: false });
      }
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !context.notificationReadAllGate.isCurrent(generation)) return;
      context.setState({
        notificationReadAllPending: false,
        notificationsError: requestErrorMessage(error, 'Could not mark notifications as read.')
      });
    }
  };

  context.applyNotificationSource = function (source, guard) {
    if (!source || !guard || !guard.gate.isCurrent(guard.generation)) return false;
    if (guard.kind === 'relationship' && source.kind === 'relationship' && Number(source.user_id) === guard.userID) {
      context.beginRelationshipGeneration(guard.userID);
      const apiUsersByID = context.mergeAPIUsers([{ id: guard.userID, relationship: source.relationship || {} }]);
      context.setState({ apiUsersByID });
      context.loadDirectory(true);
      context.loadPostFollowers();
      context.loadFeed(true);
      if (Number(context.state.profileId) === guard.userID) context.openProfile(guard.userID);
      return true;
    }
    if (guard.kind === 'group' && source.kind === 'group' && source.group && Number(source.group.id) === guard.groupID) {
      const group = context.applyAuthoritativeGroup(source.group, true);
      if (group.state === 'owner' || group.state === 'member') context.restoreGroupAccess(group);
      context.loadGroups(true);
      context.loadChats(true);
      if (Number(context.state.groupId) === guard.groupID) {
        context.loadGroupDetail(guard.groupID);
        context.loadGroupMembers(guard.groupID, true);
      }
      return true;
    }
    return false;
  };

  context.actOnNotification = async (notificationID, action) => {
    notificationID = Number(notificationID);
    const key = String(notificationID);
    const notification = context.state.notifications.find(item => item.id === notificationID);
    if (!notification || !NotificationModel.isActionable(notification) || context.state.notificationActionPendingByID[key]) return;
    const authGeneration = context.authGate.current();
    const actionGate = context.notificationActionGate(notificationID);
    const actionGeneration = actionGate.begin();
    let sourceGuard = null;
    if (notification.type === 'follow_request') {
      const gate = context.relationshipGeneration(notification.actorID);
      sourceGuard = { kind: 'relationship', userID: notification.actorID, gate, generation: gate.current() };
    } else if (notification.group && notification.group.id) {
      const groupID = Number(notification.group.id);
      const gate = context.groupGeneration(groupID);
      sourceGuard = { kind: 'group', groupID, gate, generation: gate.current() };
    }
    context.setState({
      notificationActionPendingByID: Object.assign({}, context.state.notificationActionPendingByID, { [key]: true }),
      notificationActionErrorByID: Object.assign({}, context.state.notificationActionErrorByID, { [key]: '' })
    });
    try {
      const response = await AuthAPI.actOnNotification(notificationID, action);
      if (!context.authGate.isCurrent(authGeneration) || !actionGate.isCurrent(actionGeneration)) return;
      context.applyNotificationPayload(response, false);
      context.applyNotificationSource(response.source, sourceGuard);
      const pending = Object.assign({}, context.state.notificationActionPendingByID);
      delete pending[key];
      context.setState({ notificationActionPendingByID: pending });
      context.loadNotifications(true);
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !actionGate.isCurrent(actionGeneration)) return;
      const pending = Object.assign({}, context.state.notificationActionPendingByID);
      delete pending[key];
      context.setState({
        notificationActionPendingByID: pending,
        notificationActionErrorByID: Object.assign({}, context.state.notificationActionErrorByID, {
          [key]: requestErrorMessage(error, 'Could not update notification.')
        })
      });
    }
  };

  context.openNotification = notification => {
    if (!notification) return;
    context.markNotificationRead(notification.id);
    if (notification.group && notification.group.id) {
      context.openGroup(notification.group.id);
      if (notification.type === 'group_event') context.setState({ groupTab: 'events' });
      return;
    }
    context.openProfile(notification.actorID);
  };

    return createController('notification', dependencies, {
      notificationReadGate: context.notificationReadGate,
      notificationActionGate: context.notificationActionGate,
      relationshipGeneration: context.relationshipGeneration,
      advanceRelationshipLifecycle: context.advanceRelationshipLifecycle,
      beginRelationshipGeneration: context.beginRelationshipGeneration,
      trackNotificationLifecycles: context.trackNotificationLifecycles,
      applyNotificationPayload: context.applyNotificationPayload,
      loadNotifications: context.loadNotifications,
      markNotificationRead: context.markNotificationRead,
      markAllNotificationsRead: context.markAllNotificationsRead,
      applyNotificationSource: context.applyNotificationSource,
      actOnNotification: context.actOnNotification,
      openNotification: context.openNotification
    }, function (state) {
      return { unread: state ? Number(state.notificationUnreadCount) || 0 : 0, items: state && state.notifications ? state.notifications : [] };
    }, {});
  };
});
