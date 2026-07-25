(function (root, factory) {
  var create = factory(root && root.createFeatureController);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createChatController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController) {
  return function createChatController(dependencies) {
    var view = dependencies.root;
    return createController('chat', dependencies, {
      load: function (reset) { return view.loadChats(reset); },
      openDirect: function (userID) { return view.openDirectChat(userID); },
      openGroup: function (groupID) { return view.openGroupChat(groupID); },
      send: function () { return view.sendMsg(); }
    }, function (state) {
      return { activeKey: state ? state.activeChatKey : null };
    });
  };
});
