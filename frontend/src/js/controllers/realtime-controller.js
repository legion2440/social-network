(function (root, factory) {
  var create = factory(root && root.createFeatureController);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createRealtimeController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController) {
  return function createRealtimeController(dependencies) {
    var view = dependencies.root;
    return createController('realtime', dependencies, {
      connect: function (generation) { return view.connectRealtime(generation); },
      dispatch: function (event) { return view.handleRealtimeEvent(event); }
    }, function (state) {
      return { status: state ? state.wsStatus : 'disconnected' };
    }, {
      stop: function () { return view.stopRealtime(); }
    });
  };
});
