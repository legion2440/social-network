(function (root, factory) {
  var create = factory(root && root.createFeatureController);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createNotificationController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController) {
  return function createNotificationController(dependencies) {
    var view = dependencies.root;
    return createController('notification', dependencies, {
      load: function (reset) { return view.loadNotifications(reset); },
      readAll: function () { return view.markAllNotificationsRead(); },
      act: function (notificationID, action) { return view.actOnNotification(notificationID, action); }
    }, function (state) {
      return { unread: state ? Number(state.notificationUnreadCount) || 0 : 0 };
    });
  };
});
