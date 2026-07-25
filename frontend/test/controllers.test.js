const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const controllerDir = path.resolve(__dirname, '..', 'src', 'js', 'controllers');

test('feature controllers expose explicit actions and keep state ownership in Root', () => {
  global.createFeatureController = require(path.join(controllerDir, 'controller-factory.js'));
  global.createAuthController = require(path.join(controllerDir, 'auth-controller.js'));
  global.createFeedController = require(path.join(controllerDir, 'feed-controller.js'));
  global.createProfileController = require(path.join(controllerDir, 'profile-controller.js'));
  global.createGroupsController = require(path.join(controllerDir, 'groups-controller.js'));
  global.createEventsController = require(path.join(controllerDir, 'events-controller.js'));
  global.createChatController = require(path.join(controllerDir, 'chat-controller.js'));
  global.createNotificationController = require(path.join(controllerDir, 'notification-controller.js'));
  global.createRealtimeController = require(path.join(controllerDir, 'realtime-controller.js'));
  global.createRouterController = require(path.join(controllerDir, 'router-controller.js'));

  const calls = [];
  const root = new Proxy({}, {
    get(target, property) {
      if (!(property in target)) target[property] = (...args) => calls.push([property, ...args]);
      return target[property];
    }
  });
  const createFeatureControllers = require(path.join(controllerDir, 'feature-controllers.js'));
  const controllers = createFeatureControllers({ root, api: {}, models: {} });

  assert.deepEqual(Object.keys(controllers), [
    'auth', 'feed', 'profile', 'groups', 'events', 'chat', 'notification', 'realtime', 'router'
  ]);
  for (const controller of Object.values(controllers)) {
    assert.equal(Object.isFrozen(controller), true);
    assert.equal(Object.isFrozen(controller.actions), true);
    assert.equal(typeof controller.derived, 'function');
    assert.equal('state' in controller, false);
  }

  controllers.feed.actions.load(true);
  controllers.router.actions.group(7);
  controllers.realtime.lifecycle.stop();
  assert.deepEqual(calls, [
    ['loadFeed', true],
    ['openGroup', 7],
    ['stopRealtime']
  ]);
});

test('controller modules do not mutate Root prototypes or merge hidden state', () => {
  for (const name of fs.readdirSync(controllerDir)) {
    if (!name.endsWith('.js')) continue;
    const source = fs.readFileSync(path.join(controllerDir, name), 'utf8');
    assert.doesNotMatch(source, /\.prototype\s*=|Object\.assign\s*\(/, name);
  }
});

test('router parses strict routes and synchronizes push, replace, and popstate navigation', () => {
  global.createFeatureController = require(path.join(controllerDir, 'controller-factory.js'));
  delete require.cache[require.resolve(path.join(controllerDir, 'router-controller.js'))];
  const createRouterController = require(path.join(controllerDir, 'router-controller.js'));
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
  const root = {
    state: { screen: 'feed' },
    go: screen => calls.push(['screen', screen]),
    openProfile: id => calls.push(['profile', id]),
    openGroup: id => calls.push(['group', id]),
    openDirectChat: id => calls.push(['direct-chat', id]),
    openGroupChat: id => calls.push(['group-chat', id])
  };
  const router = createRouterController({ root, api: {}, models: {}, environment });

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
  assert.deepEqual(calls.slice(-2), [['replace', '/'], ['screen', 'feed']]);

  router.lifecycle.stop();
  assert.equal(listeners.popstate, undefined);
});
