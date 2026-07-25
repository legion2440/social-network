(function (root, factory) {
  var library = factory();

  if (typeof module === 'object' && module.exports) {
    module.exports = library;
  }

  if (root) {
    root.AvatarURL = library;
  }
})(typeof window !== 'undefined' ? window : null, function () {
  function isCustomAvatarURL(value) {
    if (typeof value !== 'string') return false;
    return /^\/api\/users\/[1-9][0-9]*\/avatar(?:\?[^#]*)?$/.test(value.trim());
  }

  return { isCustomAvatarURL: isCustomAvatarURL };
});
