const test = require('node:test');
const assert = require('node:assert/strict');

global.localStorage = {
  getItem: () => null,
  setItem: () => {}
};
global.DCLogic = class {
  constructor(props) {
    this.props = props || {};
    this.state = {};
  }

  setState(update, callback) {
    const patch = typeof update === 'function' ? update(this.state, this.props) : update;
    Object.assign(this.state, patch || {});
    if (callback) callback();
  }
};
global.AvatarURL = require('./avatar-url.js');
global.UserModel = require('./user-model.js');
global.PostModel = require('./post-model.js');
global.AuthAPI = {};

const { Component } = require('./app.js');

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((onResolve, onReject) => {
    resolve = onResolve;
    reject = onReject;
  });
  return { promise, resolve, reject };
}

function rawUser(id, status, options) {
  options = options || {};
  return {
    id,
    first_name: options.firstName || 'User',
    last_name: options.lastName || String(id),
    nickname: options.nickname || 'user' + id,
    avatar_url: options.avatarURL || '/static/avatars/neutral.svg',
    is_private: options.isPrivate === true,
    can_view_profile: options.canView !== false,
    relationship: {
      status: status || 'none',
      follows_me: options.followsMe === true
    }
  };
}

function rawPost(id, authorID) {
  return {
    id,
    author: rawUser(authorID),
    text: 'post ' + id,
    privacy: 'public',
    media_url: null,
    created_at: '2026-07-20T12:00:00Z'
  };
}

function createComponent() {
  Object.keys(global.AuthAPI).forEach(key => delete global.AuthAPI[key]);
  const component = new Component({ defaultTheme: 'light' });
  component.state.apiUsersByID = component.applyAuthUser(rawUser(1));
  component.state.authStatus = 'authenticated';
  component.state.screen = 'feed';
  return component;
}

test('feed reset starts a new generation while the previous request is pending', async () => {
  const component = createComponent();
  const oldRequest = deferred();
  const newRequest = deferred();
  let calls = 0;
  global.AuthAPI.feed = () => (++calls === 1 ? oldRequest.promise : newRequest.promise);

  const oldLoad = component.loadFeed(true);
  const newLoad = component.loadFeed(true);
  assert.equal(calls, 2);

  newRequest.resolve({ posts: [rawPost(22, 3)], next_cursor: null });
  await newLoad;
  oldRequest.resolve({ posts: [rawPost(11, 2)], next_cursor: null });
  await oldLoad;

  assert.deepEqual(component.state.posts.map(post => post.id), ['22']);
  assert.equal(component.state.feedPending, false);
});

test('directory ignores an older relationship response', async () => {
  const component = createComponent();
  const oldRequest = deferred();
  const newRequest = deferred();
  let calls = 0;
  global.AuthAPI.users = () => (++calls === 1 ? oldRequest.promise : newRequest.promise);

  const oldLoad = component.loadDirectory();
  const newLoad = component.loadDirectory();
  assert.equal(calls, 2);

  newRequest.resolve({ users: [rawUser(2, 'accepted')], next_cursor: null });
  await newLoad;
  oldRequest.resolve({ users: [rawUser(2, 'none')], next_cursor: null });
  await oldLoad;

  assert.equal(component.state.apiUsersByID['2'].relationship.status, 'accepted');
  assert.deepEqual(component.state.directoryUserIDs, [2]);
  assert.equal(component.state.directoryLoading, false);
});

test('selected followers ignore an older response and prune the audience', async () => {
  const component = createComponent();
  const oldRequest = deferred();
  const newRequest = deferred();
  let calls = 0;
  global.AuthAPI.followers = () => (++calls === 1 ? oldRequest.promise : newRequest.promise);
  component.state.selectedFollowers = { 2: true, 3: true };

  const oldLoad = component.loadPostFollowers();
  const newLoad = component.loadPostFollowers();
  assert.equal(calls, 2);

  newRequest.resolve({ users: [rawUser(3, 'none', { followsMe: true })] });
  await newLoad;
  oldRequest.resolve({ users: [rawUser(2, 'none', { followsMe: true })] });
  await oldLoad;

  assert.deepEqual(component.state.postFollowers.map(user => user.apiId), [3]);
  assert.deepEqual(component.state.selectedFollowers, { 3: true });
  assert.equal(component.state.postFollowersLoading, false);
});

test('unfollow purges target posts immediately and stale feed cannot restore them', async () => {
  const component = createComponent();
  const oldRequest = deferred();
  const freshRequest = deferred();
  let feedCalls = 0;
  global.AuthAPI.feed = () => (++feedCalls === 1 ? oldRequest.promise : freshRequest.promise);
  global.AuthAPI.unfollow = async () => null;
  global.AuthAPI.users = async () => ({ users: [rawUser(2, 'none')], next_cursor: null });

  component.state.apiUsersByID = component.mergeAPIUsers([rawUser(2, 'accepted', { isPrivate: true })]);
  component.state.posts = [
    { id: 'target-post', apiAuthorID: 2 },
    { id: 'own-post', apiAuthorID: 1 }
  ];

  const staleLoad = component.loadFeed(true);
  await component.toggleFollow(2);

  assert.deepEqual(component.state.posts.map(post => post.id), ['own-post']);
  assert.equal(feedCalls, 2);

  freshRequest.resolve({ posts: [], next_cursor: null });
  await Promise.resolve();
  oldRequest.resolve({ posts: [rawPost(77, 2)], next_cursor: null });
  await staleLoad;
  await Promise.resolve();

  assert.deepEqual(component.state.posts, []);
});

test('accept invalidates feed, directory, selected followers and current profile', async () => {
  const component = createComponent();
  component.state.followRequests = [{ id: 41, user: rawUser(2, 'none') }];
  component.state.apiUsersByID = component.mergeAPIUsers([rawUser(2, 'none')]);
  component.state.screen = 'profile';
  component.state.profileId = 2;

  global.AuthAPI.acceptFollowRequest = async () => ({ status: 'accepted' });
  let feedCalls = 0;
  let directoryCalls = 0;
  let followerCalls = 0;
  const openedProfiles = [];
  component.loadFeed = reset => { assert.equal(reset, true); feedCalls += 1; };
  component.loadDirectory = () => { directoryCalls += 1; };
  component.loadPostFollowers = () => { followerCalls += 1; };
  component.openProfile = id => { openedProfiles.push(id); };

  await component.acceptFollowRequest(41);

  assert.equal(feedCalls, 1);
  assert.equal(directoryCalls, 1);
  assert.equal(followerCalls, 1);
  assert.deepEqual(openedProfiles, [2]);
  assert.equal(component.state.followRequests.length, 0);
  assert.equal(component.state.apiUsersByID['2'].relationship.follows_me, true);
});
