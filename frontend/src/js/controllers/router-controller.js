(function (root, factory) {
  var create = factory(
    root,
    root && root.createFeatureController,
    root && root.createControllerContext
  );
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createRouterController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (browser, createController, createContext) {
  function positiveID(value) {
    var id = Number(value);
    return Number.isInteger(id) && id > 0 ? id : null;
  }

  function queryValue(search, name) {
    var query = new URLSearchParams(String(search || '').replace(/^\?/, ''));
    return query.get(name) || '';
  }

  function parsePath(pathname, search) {
    var path = String(pathname || '/').replace(/\/+$/, '') || '/';
    if (path === '/') return { kind: 'home' };
    if (path === '/login') {
      return { kind: 'login', oauthError: queryValue(search, 'oauth_error') };
    }
    if (path === '/oauth/complete') {
      return { kind: 'oauth-complete', flow: queryValue(search, 'flow') };
    }
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
    var context = createContext(dependencies);
    var environment = dependencies.environment || browser;
    var applying = false;
    var popstate = function () { applyCurrent(); };

    function replace(path) {
      if (environment && environment.history) environment.history.replaceState({}, '', path);
    }

    function screenPath(screen) {
      if (screen === 'feed') return '/';
      if (screen === 'groups') return '/groups';
      if (screen === 'notifications') return '/notifications';
      if (screen === 'chat') return '/messages';
      return '/';
    }

    function applyScreen(screen) {
      if (screen !== 'chat') context.stopTyping();
      var nextState = { screen: screen, privacyOpen: false, emojiOpen: false };
      if (screen === 'chat') {
        nextState.mobileChatList = context.usesMobileChatLayout();
        if (nextState.mobileChatList) nextState.activeChatKey = null;
      }
      context.setState(nextState, function () {
        if (screen === 'chat') {
          if (context.state.activeChatKey) context.enqueueChatRead(context.state.activeChatKey);
          context.loadChats(true, 'user-open');
        }
        if (screen === 'notifications') context.loadNotifications(true);
        if (screen === 'groups') {
          context.loadGroups(true);
          context.loadGroupInvitationInbox(true);
        }
      });
    }

    function apply(route) {
      applying = true;
      try {
        if (
          (route.kind === 'oauth-complete' || route.kind === 'login') &&
          context.state.authStatus === 'authenticated'
        ) {
          replace('/');
          applyScreen('feed');
        } else if (route.kind === 'oauth-complete') {
          context.showOAuthCompletion(route.flow);
        } else if (route.kind === 'login') {
          context.showLoginOAuthError(route.oauthError);
        } else if (context.state.authStatus !== 'authenticated') {
          context.showLoginOAuthError('');
        } else if (route.kind === 'home') applyScreen('feed');
        else if (route.kind === 'groups') applyScreen('groups');
        else if (route.kind === 'notifications') applyScreen('notifications');
        else if (route.kind === 'messages') applyScreen('chat');
        else if (route.kind === 'profile') context.openProfile(route.id);
        else if (route.kind === 'group') context.openGroup(route.id);
        else if (route.kind === 'direct-chat') context.openDirectChat(route.id);
        else if (route.kind === 'group-chat') context.openGroupChat(route.id);
        else {
          replace('/');
          applyScreen('feed');
        }
      } finally {
        applying = false;
      }
    }

    function applyCurrent() {
      var pathname = environment && environment.location ? environment.location.pathname : '/';
      var search = environment && environment.location ? environment.location.search : '';
      apply(parsePath(pathname, search));
    }

    function navigate(path, replaceCurrent) {
      var target = new URL(String(path || '/'), 'http://router.local');
      if (!environment || !environment.history) {
        apply(parsePath(target.pathname, target.search));
        return;
      }
      if (replaceCurrent) environment.history.replaceState({}, '', path);
      else if (!environment.location || environment.location.pathname !== path) {
        environment.history.pushState({}, '', path);
      }
      apply(parsePath(target.pathname, target.search));
    }

    function go(screen) {
      if (!applying) {
        navigate(screenPath(screen), false);
        return;
      }
      applyScreen(screen);
    }

    var actions = {
      isApplying: function () { return applying; },
      parse: parsePath,
      applyCurrent: applyCurrent,
      navigate: navigate,
      go: go,
      screen: function (screen) { navigate(screenPath(screen), false); },
      profile: function (userID) { navigate('/users/' + positiveID(userID), false); },
      group: function (groupID) { navigate('/groups/' + positiveID(groupID), false); },
      directChat: function (userID) { navigate('/messages/user/' + positiveID(userID), false); },
      groupChat: function (groupID) { navigate('/messages/group/' + positiveID(groupID), false); },
      reset: function () { navigate('/', true); }
    };
    return createController('router', dependencies, actions, function (state) {
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
  };
});
