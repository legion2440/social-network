(function (root, factory) {
  var merge = factory();
  if (typeof module === 'object' && module.exports) module.exports = merge;
  if (root) root.mergeViewModels = merge;
})(typeof window !== 'undefined' ? window : globalThis, function () {
  function allowedOwners(allowedDuplicates, key) {
    var owners = allowedDuplicates && allowedDuplicates[key];
    return Array.isArray(owners) ? owners.slice().sort().join('|') : '';
  }

  return function mergeViewModels(entries, allowedDuplicates) {
    var result = {};
    var ownersByKey = {};
    (entries || []).forEach(function (entry) {
      if (!entry || typeof entry.name !== 'string' || !entry.values) {
        throw new TypeError('view model entries require name and values');
      }
      Object.keys(entry.values).forEach(function (key) {
        var previousOwner = ownersByKey[key];
        if (previousOwner) {
          var actualOwners = [previousOwner, entry.name].sort().join('|');
          if (actualOwners !== allowedOwners(allowedDuplicates, key)) {
            throw new Error(
              'view model key "' + key + '" is owned by both "' +
              previousOwner + '" and "' + entry.name + '"'
            );
          }
        }
        ownersByKey[key] = entry.name;
        result[key] = entry.values[key];
      });
    });
    return result;
  };
});
