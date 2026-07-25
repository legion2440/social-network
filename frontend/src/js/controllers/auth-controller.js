(function (root, factory) {
  var create = factory(root && root.createFeatureController);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createAuthController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController) {
  return function createAuthController(dependencies) {
    var view = dependencies.root;
    return createController('auth', dependencies, {
      submit: function (event) { return view.submitAuth(event); },
      logout: function () { return view.logout(); },
      retry: function () { return view.loadCurrentUser(); }
    }, function (state) {
      return { authenticated: !!(state && state.currentUser), checking: !!(state && state.authChecking) };
    }, {
      start: function () { return view.loadCurrentUser(); }
    });
  };
});
