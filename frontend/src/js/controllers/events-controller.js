(function (root, factory) {
  var create = factory(root && root.createFeatureController);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createEventsController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController) {
  return function createEventsController(dependencies) {
    var view = dependencies.root;
    return createController('events', dependencies, {
      load: function (groupID, reset) { return view.loadGroupEvents(groupID, reset); },
      create: function () { return view.createGroupEvent(); },
      respond: function (eventID, response) { return view.respondToGroupEvent(eventID, response); }
    }, function (state) {
      return { eventCount: state && state.groupEventIDs ? state.groupEventIDs.length : 0 };
    });
  };
});
