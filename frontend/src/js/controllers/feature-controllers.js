(function (root, factory) {
  var create = factory(root);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createFeatureControllers = create;
})(typeof window !== 'undefined' ? window : globalThis, function (root) {
  return function createFeatureControllers(dependencies) {
    return Object.freeze({
      auth: root.createAuthController(dependencies.auth),
      feed: root.createFeedController(dependencies.feed),
      profile: root.createProfileController(dependencies.profile),
      groups: root.createGroupsController(dependencies.groups),
      events: root.createEventsController(dependencies.events),
      chat: root.createChatController(dependencies.chat),
      notification: root.createNotificationController(dependencies.notification),
      realtime: root.createRealtimeController(dependencies.realtime),
      router: root.createRouterController(dependencies.router)
    });
  };
});
