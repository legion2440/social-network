(function (root, factory) {
  var create = factory(root, root && root.createFeatureController);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createRouterController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (browser, createController) {
  function positiveID(value) {
    var id = Number(value);
    return Number.isInteger(id) && id > 0 ? id : null;
  }

  function parsePath(pathname) {
    var path = String(pathname || '/').replace(/\/+$/, '') || '/';
    if (path === '/') return { kind: 'home' };
    if (path === '/groups') return { kind: 'groups' };
    if (path === '/notifications') return { kind: 'notifications' };
    if (path === '/messages') return { kind: 'messages' };

    var match = path.match(/^\/users\/(\d+)$/);
    if (match && positiveID(match[1])) return { kind: 'profile', id: positiveID(match[1]) };
    match = path.match(/^\/groups\/(\d+)$/);
    if (match && positiveID(match[1])) return { kind: 'group', id: positiveID(match[1]) };
    match = path.match(/^\/messages\/user\/(\d+)$/);
    if (match && positiveID(match[1])) return { kind: 'direct-chat', id: positiveID(match[1]) };
    match = path.match(/^\/messages\/group\/(\d+)$/);
    if (match && positiveID(match[1])) return { kind: 'group-chat', id: positiveID(match[1]) };
    return { kind: 'fallback' };
  }

  return function createRouterController(dependencies) {
    var view = dependencies.root;
    var environment = dependencies.environment || browser;
    var applying = false;
    var popstate = function () { applyCurrent(); };

    function replace(path) {
      if (environment && environment.history) environment.history.replaceState({}, '', path);
    }

    function apply(route) {
      applying = true;
      try {
        if (route.kind === 'home') view.go('feed');
        else if (route.kind === 'groups') view.go('groups');
        else if (route.kind === 'notifications') view.go('notifications');
        else if (route.kind === 'messages') view.go('chat');
        else if (route.kind === 'profile') view.openProfile(route.id);
        else if (route.kind === 'group') view.openGroup(route.id);
        else if (route.kind === 'direct-chat') view.openDirectChat(route.id);
        else if (route.kind === 'group-chat') view.openGroupChat(route.id);
        else {
          replace('/');
          view.go('feed');
        }
      } finally {
        applying = false;
      }
    }

    function applyCurrent() {
      var pathname = environment && environment.location ? environment.location.pathname : '/';
      apply(parsePath(pathname));
    }

    function navigate(path, replaceCurrent) {
      if (!environment || !environment.history) {
        apply(parsePath(path));
        return;
      }
      if (replaceCurrent) environment.history.replaceState({}, '', path);
      else if (!environment.location || environment.location.pathname !== path) environment.history.pushState({}, '', path);
      apply(parsePath(path));
    }

    function screenPath(screen) {
      if (screen === 'feed') return '/';
      if (screen === 'groups') return '/groups';
      if (screen === 'notifications') return '/notifications';
      if (screen === 'chat') return '/messages';
      return '/';
    }

    var controller = createController('router', dependencies, {
      isApplying: function () { return applying; },
      parse: parsePath,
      applyCurrent: applyCurrent,
      navigate: navigate,
      screen: function (screen) { navigate(screenPath(screen), false); },
      profile: function (userID) { navigate('/users/' + positiveID(userID), false); },
      group: function (groupID) { navigate('/groups/' + positiveID(groupID), false); },
      directChat: function (userID) { navigate('/messages/user/' + positiveID(userID), false); },
      groupChat: function (groupID) { navigate('/messages/group/' + positiveID(groupID), false); },
      reset: function () { navigate('/', true); }
    }, function (state) {
      return { screen: state ? state.screen : 'feed' };
    }, {
      start: function () {
        if (environment && typeof environment.addEventListener === 'function') {
          environment.addEventListener('popstate', popstate);
        }
      },
      stop: function () {
        if (environment && typeof environment.removeEventListener === 'function') {
          environment.removeEventListener('popstate', popstate);
        }
      }
    });
    return controller;
  };
});
