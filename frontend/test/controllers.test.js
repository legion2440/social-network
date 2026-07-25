const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const controllerDir = path.resolve(__dirname, '..', 'src', 'js', 'controllers');
const appPath = path.resolve(__dirname, '..', 'src', 'js', 'app.js');

global.createFeatureController = require(path.join(controllerDir, 'controller-factory.js'));
global.createControllerContext = require(path.join(controllerDir, 'controller-context.js'));

function requestGate() {
  let generation = 0;
  return {
    begin() { generation += 1; return generation; },
    current() { return generation; },
    isCurrent(value) { return value === generation; }
  };
}

function stateAdapter(initial) {
  const state = initial;
  return {
    state,
    adapter: {
      get: () => state,
      set(update, callback) {
        const patch = typeof update === 'function' ? update(state) : update;
        Object.assign(state, patch || {});
        if (callback) callback();
      }
    }
  };
}

function baseDependencies(initial) {
  const store = stateAdapter(initial);
  return {
    store,
    dependencies: {
      state: store.adapter,
      api: {},
      models: {},
      gates: {},
      resources: {},
      refs: {},
      helpers: {},
      callbacks: {},
      navigation: {
        isApplying: () => true,
        applyCurrent: () => {},
        profile: () => {},
        group: () => {},
        directChat: () => {},
        groupChat: () => {}
      },
      session: { users: { me: { apiId: 1 } } },
      values: { IC: { globe: '', users: '', lock: '' }, GROUP_COLORS: ['#000'] },
      presenters: {}
    }
  };
}

test('feature implementations live in controllers and cannot proxy back to Component', () => {
  const appSource = fs.readFileSync(appPath, 'utf8');
  const expectedActions = [
    'loadCurrentUser', 'loadFeed', 'createComment', 'openProfile', 'toggleFollow',
    'loadGroups', 'createGroupEvent', 'loadNotifications', 'loadChats',
    'handleRealtimeEvent', 'connectRealtime'
  ];
  for (const action of expectedActions) {
    assert.doesNotMatch(appSource, new RegExp('^\\s{2}' + action + '\\s*(?:=|\\()', 'm'), action);
  }

  for (const name of fs.readdirSync(controllerDir)) {
    if (!name.endsWith('.js')) continue;
    const source = fs.readFileSync(path.join(controllerDir, name), 'utf8');
    assert.doesNotMatch(source, /dependencies\.root\b/, name);
    assert.doesNotMatch(source, /var\s+view\s*=/, name);
  }
});

test('feed actions mutate the shared state only through the provided adapter', () => {
  const createFeedController = require(path.join(controllerDir, 'feed-controller.js'));
  const { store, dependencies } = baseDependencies({
    commentsByPostID: {},
    posts: [],
    openComments: {}
  });
  dependencies.api = {
    createComment() {}, createPost() {}, feed() {}, followers() {}, postComments() {}
  };
  dependencies.models = {
    users: { createRequestGate: requestGate },
    posts: {},
    comments: {}
  };
  dependencies.gates = {
    authGate: requestGate(),
    feedGate: requestGate(),
    postFollowersGate: requestGate()
  };
  const commentAccessGatesByPostID = {};
  const commentLoadGatesByPostID = {};
  dependencies.refs = {
    commentAccessGatesByPostID: {
      get: () => commentAccessGatesByPostID,
      set: () => { throw new Error('unexpected replacement'); }
    },
    commentLoadGatesByPostID: {
      get: () => commentLoadGatesByPostID,
      set: () => { throw new Error('unexpected replacement'); }
    }
  };
  dependencies.helpers = {
    emptyCommentState: () => ({
      comments: [], draft: '', mediaFile: null, mediaFileName: '', mediaPreviewURL: '',
      createPending: false, createError: '', loaded: false
    }),
    requestErrorMessage: (_error, fallback) => fallback,
    apiUser: id => ({ apiId: id }),
    formatPostTime: value => value,
    mapAPIPost: value => value,
    mergeAPIUsers: () => ({})
  };
  const opened = [];
  dependencies.callbacks = {
    openGroup: id => opened.push(['group', id]),
    openProfile: id => opened.push(['profile', id])
  };
  dependencies.presenters = {
    post: (_state, post) => Object.assign({}, post)
  };

  const controller = createFeedController(dependencies);
  controller.actions.setCommentDraft(7, 'hello');
  assert.equal(store.state.commentsByPostID['7'].draft, 'hello');
  assert.deepEqual(controller.dependencies.api, [
    'createComment', 'createPost', 'feed', 'followers', 'postComments'
  ]);
  assert.equal('root' in controller.dependencies, false);

  controller.actions.updateComposer('draft');
  assert.equal(store.state.composerText, 'draft');
  assert.deepEqual(opened, []);
});

test('derived contracts use the real shared state field names', () => {
  const createAuthController = require(path.join(controllerDir, 'auth-controller.js'));
  const createProfileController = require(path.join(controllerDir, 'profile-controller.js'));
  const authBase = baseDependencies({ authStatus: 'authenticated' });
  authBase.dependencies.models = {};
  authBase.dependencies.helpers = {
    emptyCurrentUser: () => ({}),
    emptyRegistrationForm: () => ({}),
    emptyProfileEditor: () => ({}),
    emptyConfirmationState: () => ({}),
    emptyGroupPostState: () => ({}),
    emptyGroupEventState: () => ({}),
    emptyNotificationState: () => ({}),
    emptyChatState: () => ({}),
    requestErrorMessage: (_error, fallback) => fallback,
    formatDateOfBirthInput: value => value,
    parseDateOfBirth: () => true,
    applyAuthUser: () => ({})
  };
  authBase.dependencies.callbacks = {
    disposeAllCommentPreviews() {}, loadDirectory() {}, loadFeed() {},
    loadNotifications() {}, loadPostFollowers() {},
    startAuthenticatedRealtime() {}, stopRealtime() {}
  };
  authBase.dependencies.gates = {
    authGate: requestGate(), feedGate: requestGate(), directoryGate: requestGate(),
    postFollowersGate: requestGate(), profileGate: requestGate(),
    groupsDirectoryGate: requestGate(), groupInvitationInboxGate: requestGate(),
    groupDetailGate: requestGate(), groupMembersGate: requestGate(),
    groupRequestsGate: requestGate(), groupInvitationsGate: requestGate(),
    groupPostsGate: requestGate(), groupEventsGate: requestGate(),
    groupEventCreateGate: requestGate(), notificationReadAllGate: requestGate(),
    notificationsGate: requestGate(), chatsGate: requestGate(), activeChatGate: requestGate()
  };
  const disposable = {};
  [
    'chatAccessGatesByKey', 'chatHistoryGatesByKey', 'chatReadGatesByKey',
    'chatReadInFlightByKey', 'chatReadSentCandidateByKey', 'commentAccessGatesByPostID',
    'commentLoadGatesByPostID', 'groupEventResponseGatesByID', 'groupGenerationsByID',
    'latestActionableNotificationIDBySourceKey', 'notificationActionGatesByID',
    'notificationReadGatesByID', 'relationshipGenerationsByID'
  ].forEach(name => {
    disposable[name] = {};
    authBase.dependencies.refs[name] = {
      get: () => disposable[name],
      set: value => { disposable[name] = value; }
    };
  });
  ['revokedChatKeys', 'revokedGroupAccessIDs'].forEach(name => {
    disposable[name] = new Set();
    authBase.dependencies.refs[name] = {
      get: () => disposable[name],
      set: value => { disposable[name] = value; }
    };
  });
  const auth = createAuthController(authBase.dependencies);
  const viewModel = auth.derived(Object.assign({
    authMode: 'login',
    authPending: false,
    authError: '',
    bootstrapError: '',
    regAvatarName: '',
    regAvatarPreviewURL: '',
    logoutPending: false
  }, authBase.store.state));
  assert.equal(viewModel.authIsLogin, true);
  assert.equal(viewModel.authCta, 'Sign in');
  assert.equal(typeof viewModel.onAuthEmail, 'function');
  assert.equal('currentUser' in viewModel, false);
  assert.equal('authChecking' in viewModel, false);
});

test('router parses strict routes and dispatches only named feature callbacks', () => {
  const createRouterController = require(path.join(controllerDir, 'router-controller.js'));
  const { store, dependencies } = baseDependencies({ screen: 'feed', activeChatKey: null });
  const calls = [];
  const listeners = {};
  const environment = {
    location: { pathname: '/' },
    history: {
      pushState(_state, _title, pathname) {
        environment.location.pathname = pathname;
        calls.push(['push', pathname]);
      },
      replaceState(_state, _title, pathname) {
        environment.location.pathname = pathname;
        calls.push(['replace', pathname]);
      }
    },
    addEventListener(type, listener) { listeners[type] = listener; },
    removeEventListener(type, listener) {
      if (listeners[type] === listener) delete listeners[type];
    }
  };
  dependencies.environment = environment;
  dependencies.helpers = { usesMobileChatLayout: () => false };
  dependencies.callbacks = {
    stopTyping: () => {},
    enqueueChatRead: () => {},
    loadChats: () => {},
    loadNotifications: () => {},
    loadGroups: () => {},
    loadGroupInvitationInbox: () => {},
    openProfile: id => calls.push(['profile', id]),
    openGroup: id => calls.push(['group', id]),
    openDirectChat: id => calls.push(['direct-chat', id]),
    openGroupChat: id => calls.push(['group-chat', id])
  };
  const router = createRouterController(dependencies);

  assert.deepEqual(router.actions.parse('/users/14'), { kind: 'profile', id: 14 });
  assert.deepEqual(router.actions.parse('/users/0'), { kind: 'fallback' });
  assert.deepEqual(router.actions.parse('/messages/group/nope'), { kind: 'fallback' });

  router.lifecycle.start();
  router.actions.profile(14);
  assert.deepEqual(calls.slice(-2), [['push', '/users/14'], ['profile', 14]]);

  environment.location.pathname = '/messages/group/7';
  listeners.popstate();
  assert.deepEqual(calls.at(-1), ['group-chat', 7]);

  environment.location.pathname = '/unknown';
  listeners.popstate();
  assert.equal(store.state.screen, 'feed');
  assert.deepEqual(calls.at(-1), ['replace', '/']);

  router.lifecycle.stop();
  assert.equal(listeners.popstate, undefined);
});
