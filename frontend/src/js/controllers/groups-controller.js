(function (root, factory) {
  var create = factory(root && root.createFeatureController);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createGroupsController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController) {
  return function createGroupsController(dependencies) {
    var view = dependencies.root;
    return createController('groups', dependencies, {
      load: function (reset) { return view.loadGroups(reset); },
      open: function (groupID) { return view.openGroup(groupID); },
      create: function () { return view.createGroup(); },
      leave: function (groupID) { return view.leaveGroup(groupID); }
    }, function (state) {
      return { groupID: state && state.groupId ? Number(state.groupId) : null };
    });
  };
});
