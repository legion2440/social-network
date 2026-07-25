(function (root, factory) {
  var create = factory();
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createControllerContext = create;
})(typeof window !== 'undefined' ? window : globalThis, function () {
  function copyFunctions(target, source) {
    Object.keys(source || {}).forEach(function (name) {
      if (typeof source[name] !== 'function') {
        throw new TypeError('controller dependency "' + name + '" must be a function');
      }
      target[name] = source[name];
    });
  }

  return function createControllerContext(dependencies) {
    if (
      !dependencies ||
      !dependencies.state ||
      typeof dependencies.state.get !== 'function' ||
      typeof dependencies.state.set !== 'function'
    ) {
      throw new TypeError('controller requires a state get/set adapter');
    }

    var context = {};
    Object.defineProperty(context, 'state', {
      enumerable: true,
      get: dependencies.state.get
    });
    context.setState = dependencies.state.set;

    copyFunctions(context, dependencies.helpers);
    copyFunctions(context, dependencies.callbacks);
    Object.keys(dependencies.gates || {}).forEach(function (name) {
      context[name] = dependencies.gates[name];
    });
    Object.keys(dependencies.resources || {}).forEach(function (name) {
      context[name] = dependencies.resources[name];
    });
    Object.keys(dependencies.refs || {}).forEach(function (name) {
      var ref = dependencies.refs[name];
      if (!ref || typeof ref.get !== 'function' || typeof ref.set !== 'function') {
        throw new TypeError('controller ref "' + name + '" requires get/set functions');
      }
      Object.defineProperty(context, name, {
        enumerable: true,
        get: ref.get,
        set: ref.set
      });
    });
    return context;
  };
});
