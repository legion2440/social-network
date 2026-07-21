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
global.CommentModel = require('./comment-model.js');
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
    comments_count: 0,
    created_at: '2026-07-20T12:00:00Z'
  };
}

function rawComment(id, postID, authorID, createdAt) {
  return {
    id,
    post_id: postID,
    text: 'comment ' + id,
    created_at: createdAt || '2026-07-21T12:00:00Z',
    author: rawUser(authorID)
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

function installEmptyAuthenticatedLoads() {
  global.AuthAPI.feed = async () => ({ posts: [], next_cursor: null });
  global.AuthAPI.followers = async () => ({ users: [] });
  global.AuthAPI.users = async () => ({ users: [], next_cursor: null });
  global.AuthAPI.followRequests = async () => ({ requests: [] });
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
    { id: '21', apiAuthorID: 2 },
    { id: '11', apiAuthorID: 1 }
  ];
  component.state.commentsByPostID = {
    '21': Object.assign(emptyTestCommentState(), { comments: [{ apiId: 9 }] }),
    '11': Object.assign(emptyTestCommentState(), { comments: [{ apiId: 10 }] })
  };

  const staleLoad = component.loadFeed(true);
  await component.toggleFollow(2);

  assert.deepEqual(component.state.posts.map(post => post.id), ['11']);
  assert.equal(component.state.commentsByPostID['21'], undefined);
  assert.equal(component.state.commentsByPostID['11'].comments[0].apiId, 10);
  assert.equal(feedCalls, 2);

  freshRequest.resolve({ posts: [], next_cursor: null });
  await Promise.resolve();
  oldRequest.resolve({ posts: [rawPost(77, 2)], next_cursor: null });
  await staleLoad;
  await Promise.resolve();

  assert.deepEqual(component.state.posts, []);
});

function emptyTestCommentState() {
  return {
    comments: [], loading: false, pending: false, error: '', nextCursor: null,
    draft: '', createPending: false, createError: '', loaded: false
  };
}

test('feed load has no comment N+1 and first open loads comments lazily', async () => {
  const component = createComponent();
  let commentCalls = 0;
  global.AuthAPI.feed = async () => ({ posts: [rawPost(7, 2)], next_cursor: null });
  global.AuthAPI.postComments = async () => {
    commentCalls += 1;
    return { comments: [rawComment(1, 7, 2)], next_cursor: null };
  };

  await component.loadFeed(true);
  assert.equal(commentCalls, 0);
  component.togglePostComments(7);
  await Promise.resolve();
  await Promise.resolve();

  assert.equal(commentCalls, 1);
  assert.equal(component.commentState(7).loaded, true);
  assert.deepEqual(component.commentState(7).comments.map(comment => comment.apiId), [1]);

  component.togglePostComments(7);
  component.togglePostComments(7);
  await Promise.resolve();
  assert.equal(commentCalls, 1);
});

test('comment pagination uses cursor and deduplicates page boundaries', async () => {
  const component = createComponent();
  const calls = [];
  global.AuthAPI.postComments = async (postID, cursor, limit) => {
    calls.push({ postID, cursor, limit });
    if (!cursor) {
      return {
        comments: [rawComment(1, postID, 2), rawComment(2, postID, 2)],
        next_cursor: 'next-comment-page'
      };
    }
    return {
      comments: [rawComment(2, postID, 2), rawComment(3, postID, 2, '2026-07-21T12:00:01Z')],
      next_cursor: null
    };
  };

  await component.loadComments(7, true);
  await component.loadComments(7, false);

  assert.deepEqual(calls, [
    { postID: 7, cursor: null, limit: 20 },
    { postID: 7, cursor: 'next-comment-page', limit: 20 }
  ]);
  assert.deepEqual(component.commentState(7).comments.map(comment => comment.apiId), [1, 2, 3]);
  assert.equal(component.commentState(7).nextCursor, null);
});

test('comment create prevents duplicates, preserves unloaded state and increments count', async () => {
  const component = createComponent();
  const response = deferred();
  let calls = 0;
  global.AuthAPI.createComment = () => { calls += 1; return response.promise; };
  const post = component.mapAPIPost(Object.assign(rawPost(7, 2), { comments_count: 4 }));
  component.state.posts = [post];
  component.state.commentsByPostID = {
    '7': Object.assign(emptyTestCommentState(), { draft: '  new comment  ' })
  };

  const first = component.createComment(7);
  const duplicate = component.createComment(7);
  assert.equal(calls, 1);
  response.resolve(rawComment(9, 7, 1));
  await Promise.all([first, duplicate]);

  const state = component.commentState(7);
  assert.equal(state.loaded, false);
  assert.equal(state.draft, '');
  assert.deepEqual(state.comments.map(comment => comment.apiId), [9]);
  assert.equal(component.state.posts[0].commentsCount, 5);
});

test('comment created before the first page response is retained and deduplicated', async () => {
  const component = createComponent();
  const pageResponse = deferred();
  const createResponse = deferred();
  global.AuthAPI.postComments = () => pageResponse.promise;
  global.AuthAPI.createComment = () => createResponse.promise;
  component.state.commentsByPostID = {
    '7': Object.assign(emptyTestCommentState(), { draft: 'while loading' })
  };

  const load = component.loadComments(7, true);
  const create = component.createComment(7);
  createResponse.resolve(rawComment(9, 7, 1, '2026-07-21T12:00:02Z'));
  await create;
  assert.equal(component.commentState(7).loaded, false);

  pageResponse.resolve({
    comments: [rawComment(1, 7, 2), rawComment(9, 7, 1, '2026-07-21T12:00:02Z')],
    next_cursor: null
  });
  await load;

  assert.equal(component.commentState(7).loaded, true);
  assert.deepEqual(component.commentState(7).comments.map(comment => comment.apiId), [1, 9]);
});

test('comment create error keeps the draft', async () => {
  const component = createComponent();
  global.AuthAPI.createComment = async () => { throw new Error('offline'); };
  component.state.commentsByPostID = {
    '7': Object.assign(emptyTestCommentState(), { draft: 'keep me' })
  };

  await component.createComment(7);
  assert.equal(component.commentState(7).draft, 'keep me');
  assert.equal(component.commentState(7).createPending, false);
  assert.match(component.commentState(7).createError, /offline/);
});

test('comment reset ignores an older response', async () => {
  const component = createComponent();
  const oldRequest = deferred();
  const newRequest = deferred();
  let calls = 0;
  global.AuthAPI.postComments = () => (++calls === 1 ? oldRequest.promise : newRequest.promise);

  const oldLoad = component.loadComments(7, true);
  const newLoad = component.loadComments(7, true);
  newRequest.resolve({ comments: [rawComment(2, 7, 2)], next_cursor: null });
  await newLoad;
  oldRequest.resolve({ comments: [rawComment(1, 7, 2)], next_cursor: null });
  await oldLoad;

  assert.deepEqual(component.commentState(7).comments.map(comment => comment.apiId), [2]);
});

test('pending comment load cannot update state after logout', async () => {
  const component = createComponent();
  const response = deferred();
  global.AuthAPI.postComments = () => response.promise;
  global.AuthAPI.logout = async () => null;

  const load = component.loadComments(7, true);
  await component.logout();
  response.resolve({ comments: [rawComment(1, 7, 2)], next_cursor: null });
  await load;

  assert.equal(component.state.authStatus, 'anonymous');
  assert.deepEqual(component.state.commentsByPostID, {});
});

test('pending comment create cannot update state after logout', async () => {
  const component = createComponent();
  const response = deferred();
  global.AuthAPI.createComment = () => response.promise;
  global.AuthAPI.logout = async () => null;
  component.state.commentsByPostID = {
    '7': Object.assign(emptyTestCommentState(), { draft: 'stale create' })
  };

  const create = component.createComment(7);
  await component.logout();
  response.resolve(rawComment(1, 7, 1));
  await create;

  assert.equal(component.state.authStatus, 'anonymous');
  assert.deepEqual(component.state.commentsByPostID, {});
  assert.deepEqual(component.state.posts, []);
});

test('real posts use backend comment handler while group posts keep mock handler', () => {
  const component = createComponent();
  let realCalls = 0;
  let mockCalls = 0;
  component.createComment = () => { realCalls += 1; };
  component.addGroupComment = () => { mockCalls += 1; };

  component.mapPost({ id: '7', real: true, apiAuthorID: 2, privacy: 'public', commentsCount: 0 }, false).onSendComment();
  component.mapPost({ id: 'group-post', uid: 'me', comments: [], privacy: 'public' }, true).onSendComment();

  assert.equal(realCalls, 1);
  assert.equal(mockCalls, 1);
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

test('pending follow requests cannot leak from the logged-out user into a new login', async () => {
  const component = createComponent();
  const oldRequests = deferred();
  global.AuthAPI.followRequests = () => oldRequests.promise;
  global.AuthAPI.logout = async () => null;

  const staleLoad = component.loadFollowRequests();
  await component.logout();

  installEmptyAuthenticatedLoads();
  global.AuthAPI.login = async () => rawUser(9);
  component.state.authMode = 'login';
  component.state.authEmail = 'user9@example.com';
  component.state.authPassword = 'secret';
  await component.submitAuth();

  oldRequests.resolve({
    requests: [{ id: 55, user: rawUser(2, 'none', { followsMe: true }) }]
  });
  await staleLoad;
  await Promise.resolve();

  assert.equal(component.state.authStatus, 'authenticated');
  assert.equal(component.state.apiUsersByID['9'].apiId, 9);
  assert.equal(component.state.apiUsersByID['2'], undefined);
  assert.deepEqual(component.state.followRequests, []);
});

test('pending follow mutation cannot update state or refresh data after logout', async () => {
  const component = createComponent();
  const mutationResponse = deferred();
  component.state.apiUsersByID = component.mergeAPIUsers([rawUser(2, 'none')]);
  global.AuthAPI.follow = () => mutationResponse.promise;
  global.AuthAPI.logout = async () => null;
  let refreshes = 0;
  component.loadDirectory = () => { refreshes += 1; };
  component.loadFeed = () => { refreshes += 1; };

  const mutation = component.toggleFollow(2);
  await component.logout();
  mutationResponse.resolve({ status: 'accepted' });
  await mutation;

  assert.equal(component.state.authStatus, 'anonymous');
  assert.deepEqual(component.state.apiUsersByID, {});
  assert.deepEqual(component.state.followPendingByID, {});
  assert.equal(refreshes, 0);
});

test('pending accept mutation cannot update state or refresh data after logout', async () => {
  const component = createComponent();
  const mutationResponse = deferred();
  component.state.followRequests = [{ id: 41, user: rawUser(2, 'none') }];
  component.state.apiUsersByID = component.mergeAPIUsers([rawUser(2, 'none')]);
  global.AuthAPI.acceptFollowRequest = () => mutationResponse.promise;
  global.AuthAPI.logout = async () => null;
  let refreshes = 0;
  component.loadPostFollowers = () => { refreshes += 1; };
  component.loadDirectory = () => { refreshes += 1; };
  component.loadFeed = () => { refreshes += 1; };
  component.openProfile = () => { refreshes += 1; };

  const mutation = component.acceptFollowRequest(41);
  await component.logout();
  mutationResponse.resolve({ status: 'accepted' });
  await mutation;

  assert.equal(component.state.authStatus, 'anonymous');
  assert.deepEqual(component.state.apiUsersByID, {});
  assert.deepEqual(component.state.followRequests, []);
  assert.deepEqual(component.state.followRequestPendingByID, {});
  assert.equal(refreshes, 0);
});

test('pending reject mutation cannot update state or refresh data after logout', async () => {
  const component = createComponent();
  const mutationResponse = deferred();
  component.state.followRequests = [{ id: 41, user: rawUser(2, 'none') }];
  global.AuthAPI.rejectFollowRequest = () => mutationResponse.promise;
  global.AuthAPI.logout = async () => null;
  let refreshes = 0;
  component.loadDirectory = () => { refreshes += 1; };

  const mutation = component.rejectFollowRequest(41);
  await component.logout();
  mutationResponse.resolve(null);
  await mutation;

  assert.equal(component.state.authStatus, 'anonymous');
  assert.deepEqual(component.state.followRequests, []);
  assert.deepEqual(component.state.followRequestPendingByID, {});
  assert.equal(refreshes, 0);
});
