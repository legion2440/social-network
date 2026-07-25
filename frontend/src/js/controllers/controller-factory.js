(function (root, factory) {
  var createController = factory();
  if (typeof module === 'object' && module.exports) module.exports = createController;
  if (root) root.createFeatureController = createController;
})(typeof window !== 'undefined' ? window : globalThis, function () {
  return function createFeatureController(name, dependencies, actions, derived, lifecycle) {
    if (!name || !dependencies || !dependencies.root || !dependencies.api || !dependencies.models) {
      throw new TypeError(name + ' controller requires root, api and models dependencies');
    }
    return Object.freeze({
      name: name,
      dependencies: Object.freeze({
        api: dependencies.api,
        models: dependencies.models
      }),
      actions: Object.freeze(actions || {}),
      derived: typeof derived === 'function' ? derived : function () { return {}; },
      lifecycle: Object.freeze(lifecycle || {})
    });
  };
});
