(function (root, factory) {
  var create = factory(root);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createFeatureControllers = create;
})(typeof window !== 'undefined' ? window : globalThis, function (root) {
  return function createFeatureControllers(dependencies) {
    return Object.freeze({
      auth: root.createAuthController(dependencies),
      feed: root.createFeedController(dependencies),
      profile: root.createProfileController(dependencies),
      groups: root.createGroupsController(dependencies),
      events: root.createEventsController(dependencies),
      chat: root.createChatController(dependencies),
      notification: root.createNotificationController(dependencies),
      realtime: root.createRealtimeController(dependencies),
      router: root.createRouterController(dependencies)
    });
  };
});
