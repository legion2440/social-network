(function (root, factory) {
  var library = factory();

  if (typeof module === 'object' && module.exports) {
    module.exports = library;
  }
  if (root) root.GroupEventModel = library;
})(typeof window !== 'undefined' ? window : null, function () {
  function normalizeEventResponse(event) {
    if (!event || !event.creator) throw new TypeError('event creator is required');
    var response = event.viewer_response;
    if (response !== 'going' && response !== 'not_going') response = null;
    return {
      id: Number(event.id),
      groupID: Number(event.group_id),
      creatorID: Number(event.creator.id),
      title: String(event.title || ''),
      description: String(event.description || ''),
      startsAt: String(event.starts_at || ''),
      createdAt: String(event.created_at || ''),
      goingCount: Math.max(0, Number(event.going_count) || 0),
      notGoingCount: Math.max(0, Number(event.not_going_count) || 0),
      viewerResponse: response
    };
  }

  function compareEvents(first, second) {
    var firstTime = Date.parse(first && first.startsAt);
    var secondTime = Date.parse(second && second.startsAt);
    if (!Number.isFinite(firstTime)) firstTime = 0;
    if (!Number.isFinite(secondTime)) secondTime = 0;
    if (firstTime !== secondTime) return firstTime - secondTime;
    return Number(first && first.id) - Number(second && second.id);
  }

  function mergeAuthoritative(events, authoritative) {
    var next = (events || []).filter(function (event) {
      return Number(event.id) !== Number(authoritative.id);
    });
    next.push(authoritative);
    next.sort(compareEvents);
    return next;
  }

  return {
    normalizeEventResponse: normalizeEventResponse,
    compareEvents: compareEvents,
    mergeAuthoritative: mergeAuthoritative
  };
});
