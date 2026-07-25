(function (root, factory) {
  var create = factory(root && root.createFeatureController);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createFeedController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController) {
  return function createFeedController(dependencies) {
    var view = dependencies.root;
    return createController('feed', dependencies, {
      load: function (reset) { return view.loadFeed(reset); },
      createPost: function () { return view.sendPost(); },
      loadComments: function (postID, reset) { return view.loadComments(postID, reset); },
      createComment: function (postID) { return view.sendComment(postID); }
    }, function (state) {
      return { postCount: state && state.postIDs ? state.postIDs.length : 0 };
    });
  };
});
