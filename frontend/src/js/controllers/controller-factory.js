(function (root, factory) {
  var createController = factory();
  if (typeof module === 'object' && module.exports) module.exports = createController;
  if (root) root.createFeatureController = createController;
})(typeof window !== 'undefined' ? window : globalThis, function () {
  return function createFeatureController(name, dependencies, actions, derived, lifecycle) {
    if (
      !name ||
      !dependencies ||
      !dependencies.state ||
      typeof dependencies.state.get !== 'function' ||
      typeof dependencies.state.set !== 'function' ||
      !dependencies.api ||
      !dependencies.models
    ) {
      throw new TypeError(name + ' controller requires state, api and models dependencies');
    }
    return Object.freeze({
      name: name,
      dependencies: Object.freeze({
        api: Object.freeze(Object.keys(dependencies.api)),
        models: Object.freeze(Object.keys(dependencies.models)),
        gates: Object.freeze(Object.keys(dependencies.gates || {})),
        helpers: Object.freeze(Object.keys(dependencies.helpers || {})),
        callbacks: Object.freeze(Object.keys(dependencies.callbacks || {})),
        refs: Object.freeze(Object.keys(dependencies.refs || {})),
        navigation: Object.freeze(Object.keys(dependencies.navigation || {})),
        session: Object.freeze(Object.keys(dependencies.session || {})),
        values: Object.freeze(Object.keys(dependencies.values || {})),
        presenters: Object.freeze(Object.keys(dependencies.presenters || {}))
      }),
      actions: Object.freeze(actions || {}),
      derived: typeof derived === 'function' ? derived : function () { return {}; },
      lifecycle: Object.freeze(lifecycle || {})
    });
  };
});
