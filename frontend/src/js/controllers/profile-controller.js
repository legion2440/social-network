(function (root, factory) {
  var create = factory(root && root.createFeatureController);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createProfileController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController) {
  return function createProfileController(dependencies) {
    var view = dependencies.root;
    return createController('profile', dependencies, {
      open: function (userID) { return view.openProfile(userID); },
      load: function (userID) { return view.loadProfile(userID); },
      toggleFollow: function (userID) { return view.toggleFollow(userID); }
    }, function (state) {
      return { userID: state && state.profileUserId ? Number(state.profileUserId) : null };
    });
  };
});
