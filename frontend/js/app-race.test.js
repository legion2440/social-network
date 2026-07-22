const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

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
global.ChatModel = require('./chat-model.js');
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

function rawGroupPost(id, groupID, authorID, commentsCount) {
  const post = rawPost(id, authorID);
  delete post.privacy;
  post.group_id = groupID;
  post.comments_count = commentsCount || 0;
  return post;
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

function emptyCommentStateForTest() {
  return {
    comments: [], loading: false, pending: false, error: '', nextCursor: null,
    draft: '', createPending: false, createError: '', loaded: false
  };
}

function rawGroup(id, status, members, ownerID) {
  return {
    id,
    title: 'Group ' + id,
    description: 'Description ' + id,
    created_at: '2026-07-21T12:00:00Z',
    members_count: members == null ? 1 : members,
    viewer_status: status || 'none',
    owner: rawUser(ownerID || 1)
  };
}

function rawChatMessage(id, clientMessageID, kind, targetID, senderID, createdAt) {
  return {
    id,
    client_message_id: clientMessageID,
    chat: { kind, target_id: targetID },
    sender: rawUser(senderID),
    body: 'chat message ' + id,
    created_at: createdAt || '2026-07-22T12:00:00Z'
  };
}

function rawDirectChat(userID, lastMessage) {
  return {
    kind: 'direct',
    target_id: userID,
    user: rawUser(userID),
    last_message: lastMessage || null,
    activity_at: lastMessage ? lastMessage.created_at : '2026-07-22T11:00:00Z'
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

test('pending comment create survives a reset comments request', async () => {
  const component = createComponent();
  const createResponse = deferred();
  const loadResponse = deferred();
  global.AuthAPI.createComment = () => createResponse.promise;
  global.AuthAPI.postComments = () => loadResponse.promise;
  component.state.posts = [component.mapAPIPost(Object.assign(rawPost(7, 2), { comments_count: 4 }))];
  component.state.commentsByPostID = {
    '7': Object.assign(emptyTestCommentState(), { draft: 'survives retry' })
  };

  const create = component.createComment(7);
  const retry = component.loadComments(7, true);
  loadResponse.resolve({ comments: [], next_cursor: null });
  await retry;
  assert.equal(component.commentState(7).createPending, true);

  createResponse.resolve(rawComment(9, 7, 1));
  await create;

  assert.equal(component.commentState(7).createPending, false);
  assert.equal(component.commentState(7).draft, '');
  assert.deepEqual(component.commentState(7).comments.map(comment => comment.apiId), [9]);
  assert.equal(component.state.posts[0].commentsCount, 5);
});

test('comment create does not double-count a refreshed server count', async () => {
  const component = createComponent();
  const createResponse = deferred();
  global.AuthAPI.createComment = () => createResponse.promise;
  global.AuthAPI.feed = async () => ({
    posts: [Object.assign(rawPost(7, 2), { comments_count: 5 })],
    next_cursor: null
  });
  component.state.posts = [component.mapAPIPost(Object.assign(rawPost(7, 2), { comments_count: 4 }))];
  component.state.profilePosts = [component.mapAPIPost(Object.assign(rawPost(7, 2), { comments_count: 3 }))];
  component.state.commentsByPostID = {
    '7': Object.assign(emptyTestCommentState(), { draft: 'count once' })
  };

  const create = component.createComment(7);
  await component.loadFeed(true);
  assert.equal(component.state.posts[0].commentsCount, 5);

  createResponse.resolve(rawComment(9, 7, 1));
  await create;

  assert.equal(component.state.posts[0].commentsCount, 5);
  assert.equal(component.state.profilePosts[0].commentsCount, 5);
});

test('stale feed count cannot roll back a locally completed comment create', async () => {
  const component = createComponent();
  const feedResponse = deferred();
  global.AuthAPI.feed = () => feedResponse.promise;
  global.AuthAPI.createComment = async () => rawComment(9, 7, 1);
  component.state.posts = [component.mapAPIPost(Object.assign(rawPost(7, 2), { comments_count: 4 }))];
  component.state.commentsByPostID = {
    '7': Object.assign(emptyTestCommentState(), { draft: 'monotonic count' })
  };

  const staleFeed = component.loadFeed(true);
  await component.createComment(7);
  assert.equal(component.state.posts[0].commentsCount, 5);

  feedResponse.resolve({
    posts: [Object.assign(rawPost(7, 2), { comments_count: 4 })],
    next_cursor: null
  });
  await staleFeed;

  assert.equal(component.state.posts[0].commentsCount, 5);
});

test('stale profile response cannot lower a local comments count', async () => {
  const component = createComponent();
  component.state.profileId = 2;
  component.state.profilePosts = [component.mapAPIPost(Object.assign(rawPost(7, 2), { comments_count: 5 }))];
  global.AuthAPI.userPosts = async () => ({
    posts: [Object.assign(rawPost(7, 2), { comments_count: 4 })],
    next_cursor: null
  });

  await component.loadProfilePosts(2, true, component.profileGate.current());

  assert.equal(component.state.profilePosts[0].commentsCount, 5);
});

test('current terminal comments denial invalidates a pending create', async () => {
  const component = createComponent();
  const createResponse = deferred();
  global.AuthAPI.createComment = () => createResponse.promise;
  global.AuthAPI.postComments = async () => {
    const error = new Error('forbidden');
    error.status = 403;
    throw error;
  };
  component.state.posts = [component.mapAPIPost(Object.assign(rawPost(7, 2), { comments_count: 4 }))];
  component.state.commentsByPostID = {
    '7': Object.assign(emptyTestCommentState(), { draft: 'must become stale' })
  };

  const create = component.createComment(7);
  await component.loadComments(7, true);
  createResponse.resolve(rawComment(9, 7, 1));
  await create;

  assert.equal(component.commentState(7).createPending, false);
  assert.match(component.commentState(7).error, /no longer have access/i);
  assert.deepEqual(component.commentState(7).comments, []);
  assert.equal(component.state.posts[0].commentsCount, 4);
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

test('group posts no longer retain a mock comment mutation', () => {
  const component = createComponent();
  let realCalls = 0;
  let mockCalls = 0;
  component.createComment = () => { realCalls += 1; };
  component.addGroupComment = () => { mockCalls += 1; };

  component.mapPost({ id: '7', real: true, apiAuthorID: 2, privacy: 'public', commentsCount: 0 }, false).onSendComment();
  component.mapPost({ id: 'group-post', uid: 'me', comments: [], privacy: 'public' }, true).onSendComment();

  assert.equal(realCalls, 1);
  assert.equal(mockCalls, 0);
});

test('group mutation invalidates an older directory response', async () => {
  const component = createComponent();
  const staleDirectory = deferred();
  global.AuthAPI.groups = () => staleDirectory.promise;
  global.AuthAPI.requestGroupJoin = async () => rawGroup(5, 'requested', 1, 2);
  component.state.apiGroupsByID = { '5': component.mapAPIGroup(rawGroup(5, 'none', 1, 2)) };
  component.state.groupIDs = [5];

  const staleLoad = component.loadGroups(true);
  await component.requestGroupJoin(5);
  staleDirectory.resolve({ groups: [rawGroup(5, 'none', 1, 2)], next_cursor: null });
  await staleLoad;

  assert.equal(component.state.apiGroupsByID['5'].state, 'requested');
  assert.equal(component.state.groupMutationPendingByID['5'], false);
});

test('opening group B rejects the late detail and members responses for group A', async () => {
  const component = createComponent();
  const detailA = deferred();
  const membersA = deferred();
  global.AuthAPI.group = id => id === 10 ? detailA.promise : Promise.resolve(rawGroup(20, 'none', 1, 3));
  global.AuthAPI.groupMembers = id => id === 10
    ? membersA.promise
    : Promise.resolve({ members: [{ user: rawUser(3), status: 'owner', created_at: '2026-07-21T12:00:00Z' }], next_cursor: null });

  component.openGroup(10);
  component.openGroup(20);
  await Promise.resolve();
  await Promise.resolve();
  detailA.resolve(rawGroup(10, 'owner', 4, 1));
  membersA.resolve({ members: [{ user: rawUser(2), status: 'member', created_at: '2026-07-21T12:00:00Z' }], next_cursor: null });
  await Promise.resolve();
  await Promise.resolve();

  assert.equal(component.state.groupId, 20);
  assert.equal(component.state.apiGroupsByID['10'], undefined);
  assert.deepEqual(component.state.groupMembers.map(member => member.userID), [3]);
});

test('accepted request cannot be resurrected by an older owner list response', async () => {
  const component = createComponent();
  const staleRequests = deferred();
  let requestCalls = 0;
  component.state.groupId = 5;
  component.state.apiGroupsByID = { '5': component.mapAPIGroup(rawGroup(5, 'owner', 1, 1)) };
  global.AuthAPI.groupJoinRequests = () => (++requestCalls === 1
    ? staleRequests.promise
    : Promise.resolve({ requests: [], next_cursor: null }));
  global.AuthAPI.acceptGroupJoinRequest = async () => rawGroup(5, 'owner', 2, 1);
  global.AuthAPI.groupMembers = async () => ({ members: [], next_cursor: null });
  global.AuthAPI.groupInvitations = async () => ({ invitations: [], next_cursor: null });

  const staleLoad = component.loadGroupRequests(5, true);
  await component.acceptGroupRequest(5, 2);
  staleRequests.resolve({ requests: [{ user: rawUser(2), created_at: '2026-07-21T12:00:00Z' }], next_cursor: null });
  await staleLoad;

  assert.equal(component.state.apiGroupsByID['5'].members, 2);
  assert.deepEqual(component.state.groupRequests, []);
});

test('owner request and invitation lists expose and load their second pages', async () => {
  const component = createComponent();
  component.state.groupId = 5;
  component.state.screen = 'group';
  component.state.apiGroupsByID = { '5': component.mapAPIGroup(rawGroup(5, 'owner', 1, 1)) };
  component.state.groupIDs = [5];
  component.state.directoryUserIDs = [2, 3, 4, 5, 6];
  component.state.apiUsersByID = component.mergeAPIUsers([rawUser(6)]);
  const requestCursors = [];
  const invitationCursors = [];
  global.AuthAPI.groupJoinRequests = async (groupID, cursor) => {
    assert.equal(groupID, 5);
    requestCursors.push(cursor);
    return cursor
      ? { requests: [{ user: rawUser(3), created_at: '2026-07-21T12:01:00Z' }], next_cursor: null }
      : { requests: [{ user: rawUser(2), created_at: '2026-07-21T12:00:00Z' }], next_cursor: 'requests-page-2' };
  };
  global.AuthAPI.groupInvitations = async (groupID, cursor) => {
    assert.equal(groupID, 5);
    invitationCursors.push(cursor);
    return cursor
      ? { invitations: [{ user: rawUser(5), created_at: '2026-07-21T12:03:00Z' }], next_cursor: null }
      : { invitations: [{ user: rawUser(4), created_at: '2026-07-21T12:02:00Z' }], next_cursor: 'invitations-page-2' };
  };

  await Promise.all([component.loadGroupRequests(5, true), component.loadGroupInvitations(5, true)]);
  let view = component.renderVals();
  assert.equal(view.groupRequestsHasMore, true);
  assert.equal(view.groupInvitationsHasMore, true);
  assert.equal(typeof view.loadMoreGroupRequests, 'function');
  assert.equal(typeof view.loadMoreGroupInvitations, 'function');

  await Promise.all([view.loadMoreGroupRequests(), view.loadMoreGroupInvitations()]);
  view = component.renderVals();
  assert.deepEqual(requestCursors, [null, 'requests-page-2']);
  assert.deepEqual(invitationCursors, [null, 'invitations-page-2']);
  assert.deepEqual(component.state.groupRequests.map(item => item.userID), [2, 3]);
  assert.deepEqual(component.state.groupInvitations.map(item => item.userID), [4, 5]);
  assert.equal(view.groupRequestsHasMore, false);
  assert.equal(view.groupInvitationsHasMore, false);
  assert.deepEqual(view.inviteCandidates.map(item => item.user.apiId), [6]);

  const template = fs.readFileSync(path.join(__dirname, '..', 'index.html'), 'utf8');
  const requestsSection = template.indexOf('PENDING JOIN REQUESTS');
  const requestsButton = template.indexOf('onclick="{{loadMoreGroupRequests}}"');
  const invitationsSection = template.indexOf('SENT INVITATIONS');
  const invitationsButton = template.indexOf('onclick="{{loadMoreGroupInvitations}}"');
  assert.ok(requestsSection >= 0 && requestsButton > requestsSection && requestsButton < invitationsSection);
  assert.ok(invitationsSection >= 0 && invitationsButton > invitationsSection);
});

test('accepted invitation invalidates the old inbox response', async () => {
  const component = createComponent();
  const staleInbox = deferred();
  component.state.apiGroupsByID = { '7': component.mapAPIGroup(rawGroup(7, 'invited', 1, 2)) };
  component.state.groupIDs = [7];
  global.AuthAPI.groupInvitationInbox = () => staleInbox.promise;
  global.AuthAPI.acceptGroupInvitation = async () => rawGroup(7, 'member', 2, 2);

  const staleLoad = component.loadGroupInvitationInbox(true);
  const originalReload = component.loadGroupInvitationInbox;
  component.loadGroupInvitationInbox = async () => {};
  await component.acceptGroupInvitation(7);
  staleInbox.resolve({ invitations: [{ group: rawGroup(7, 'invited', 1, 2), created_at: '2026-07-21T12:00:00Z' }], next_cursor: null });
  await staleLoad;
  component.loadGroupInvitationInbox = originalReload;

  assert.equal(component.state.apiGroupsByID['7'].state, 'member');
  assert.deepEqual(component.state.groupInvitationInbox, []);
});

test('group mutation is serialized and cannot update state after logout', async () => {
  const component = createComponent();
  const response = deferred();
  let calls = 0;
  component.state.apiGroupsByID = { '9': component.mapAPIGroup(rawGroup(9, 'none', 1, 2)) };
  component.state.groupIDs = [9];
  global.AuthAPI.requestGroupJoin = () => { calls += 1; return response.promise; };
  global.AuthAPI.logout = async () => null;

  const first = component.requestGroupJoin(9);
  const duplicate = component.requestGroupJoin(9);
  assert.equal(calls, 1);
  await component.logout();
  response.resolve(rawGroup(9, 'requested', 1, 2));
  await Promise.all([first, duplicate]);

  assert.equal(component.state.authStatus, 'anonymous');
  assert.deepEqual(component.state.apiGroupsByID, {});
  assert.deepEqual(component.state.groupMutationPendingByID, {});
});

test('failed group create and invitation preserve their form selections', async () => {
  const component = createComponent();
  component.state.ngName = 'Keep title';
  component.state.ngDesc = 'Keep description';
  global.AuthAPI.createGroup = async () => { throw new Error('create failed'); };
  await component.createGroup();
  assert.equal(component.state.ngName, 'Keep title');
  assert.equal(component.state.ngDesc, 'Keep description');

  component.state.groupId = 5;
  component.state.groupInviteUserID = '2';
  component.state.apiGroupsByID = { '5': component.mapAPIGroup(rawGroup(5, 'owner', 1, 1)) };
  global.AuthAPI.inviteToGroup = async () => { throw new Error('invite failed'); };
  await component.inviteSelectedUser();
  assert.equal(component.state.groupInviteUserID, '2');
  assert.match(component.state.groupMutationErrorByID['5'], /invite failed/);
});

test('stale invitation from an old session cannot clear the new session selection', async () => {
  const component = createComponent();
  const oldInvitation = deferred();
  component.state.groupId = 5;
  component.state.groupInviteUserID = '2';
  component.state.apiGroupsByID = { '5': component.mapAPIGroup(rawGroup(5, 'owner', 1, 1)) };
  global.AuthAPI.inviteToGroup = () => oldInvitation.promise;
  global.AuthAPI.logout = async () => null;

  const oldMutation = component.inviteSelectedUser();
  await component.logout();
  installEmptyAuthenticatedLoads();
  global.AuthAPI.login = async () => rawUser(7);
  component.state.authMode = 'login';
  component.state.authEmail = 'new-session@example.test';
  component.state.authPassword = 'password';
  await component.submitAuth();
  component.state.groupId = 8;
  component.state.groupInviteUserID = '9';
  component.state.apiGroupsByID = { '8': component.mapAPIGroup(rawGroup(8, 'owner', 1, 7)) };

  oldInvitation.resolve(rawGroup(5, 'owner', 1, 1));
  assert.equal(await oldMutation, false);
  assert.equal(component.state.groupInviteUserID, '9');
});

test('duplicate pending invitation does not clear a newer selection', async () => {
  const component = createComponent();
  const invitation = deferred();
  let calls = 0;
  component.state.groupId = 5;
  component.state.groupInviteUserID = '2';
  component.state.apiGroupsByID = { '5': component.mapAPIGroup(rawGroup(5, 'owner', 1, 1)) };
  global.AuthAPI.inviteToGroup = () => { calls += 1; return invitation.promise; };
  global.AuthAPI.groupMembers = async () => ({ members: [], next_cursor: null });
  global.AuthAPI.groupJoinRequests = async () => ({ requests: [], next_cursor: null });
  global.AuthAPI.groupInvitations = async () => ({ invitations: [], next_cursor: null });

  const first = component.inviteSelectedUser();
  component.state.groupInviteUserID = '3';
  const duplicate = component.inviteSelectedUser();
  assert.equal(await duplicate, false);
  assert.equal(calls, 1);
  assert.equal(component.state.groupInviteUserID, '3');

  invitation.resolve(rawGroup(5, 'owner', 1, 1));
  assert.equal(await first, true);
  assert.equal(component.state.groupInviteUserID, '3');
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

test('chat list maps direct and group summaries and opens real HTTP history', async () => {
  const component = createComponent();
  const directMessage = rawChatMessage(
    11, '441a59ff-4254-46d0-a9bd-ed29d8bb14ea', 'direct', 2, 2
  );
  global.AuthAPI.chats = async () => ({
    chats: [
      rawDirectChat(2, directMessage),
      {
        kind: 'group', target_id: 7, group: rawGroup(7, 'member', 3, 3),
        last_message: null, activity_at: '2026-07-22T11:30:00Z'
      }
    ],
    next_cursor: null
  });
  const histories = [];
  global.AuthAPI.directMessages = async (userID, cursor, limit) => {
    histories.push({ kind: 'direct', userID, cursor, limit });
    return { messages: [directMessage], next_cursor: null };
  };
  global.AuthAPI.groupMessages = async () => ({ messages: [], next_cursor: null });

  await component.loadChats(true);
  await Promise.resolve();

  assert.deepEqual(component.state.chatKeys, ['direct:2', 'group:7']);
  assert.equal(component.state.activeChatKey, 'direct:2');
  assert.deepEqual(histories, [{ kind: 'direct', userID: 2, cursor: null, limit: 20 }]);
  assert.deepEqual(component.chatMessages('direct:2').messages.map(message => message.apiId), [11]);
});

test('group posts expose a real composer only to active owner or member access', () => {
  const component = createComponent();
  component.state.screen = 'group';
  component.state.groupId = 7;
  component.state.apiGroupsByID = { '7': component.mapAPIGroup(rawGroup(7, 'member', 2, 2)) };

  let view = component.renderVals();
  assert.equal(view.gCanContent, true);
  assert.equal(view.gContentLocked, false);
  assert.equal(typeof view.sendGroupPost, 'function');

  component.state.apiGroupsByID['7'] = component.mapAPIGroup(rawGroup(7, 'requested', 1, 2));
  view = component.renderVals();
  assert.equal(view.gCanContent, false);
  assert.equal(view.gContentLocked, true);
});

test('group switch and reset gates reject stale group post pages', async () => {
  const component = createComponent();
  const groupA = deferred();
  const groupB = deferred();
  component.state.screen = 'group';
  component.state.groupId = 7;
  component.state.apiGroupsByID = {
    '7': component.mapAPIGroup(rawGroup(7, 'member', 2, 2)),
    '8': component.mapAPIGroup(rawGroup(8, 'member', 2, 2))
  };
  global.AuthAPI.groupPosts = groupID => groupID === 7 ? groupA.promise : groupB.promise;

  const first = component.loadGroupPosts(7, true);
  component.state.groupId = 8;
  const second = component.loadGroupPosts(8, true);
  groupB.resolve({ posts: [rawGroupPost(80, 8, 2)], next_cursor: null });
  await second;
  groupA.resolve({ posts: [rawGroupPost(70, 7, 2)], next_cursor: null });
  await first;

  assert.deepEqual(component.state.groupPosts.map(post => post.id), ['80']);
  assert.equal(component.state.groupPosts[0].groupID, 8);
});

test('authoritative group post survives an older pending page', async () => {
  const component = createComponent();
  const stalePage = deferred();
  component.state.screen = 'group';
  component.state.groupId = 7;
  component.state.groupPostComposerText = 'new group post';
  component.state.apiGroupsByID = { '7': component.mapAPIGroup(rawGroup(7, 'owner', 1, 1)) };
  global.AuthAPI.groupPosts = () => stalePage.promise;
  global.AuthAPI.createGroupPost = async () => rawGroupPost(72, 7, 1);

  const staleLoad = component.loadGroupPosts(7, true);
  await component.sendGroupPost();
  stalePage.resolve({ posts: [rawGroupPost(71, 7, 1)], next_cursor: null });
  await staleLoad;

  assert.deepEqual(component.state.groupPosts.map(post => post.id), ['72']);
  assert.equal(component.state.groupPostComposerText, '');
  assert.equal(component.state.groupPostComposerPending, false);
});

test('chat remove purges group posts, comments, drafts and pending content responses', async () => {
  const component = createComponent();
  const stalePage = deferred();
  component.state.screen = 'group';
  component.state.groupId = 7;
  component.state.apiGroupsByID = { '7': component.mapAPIGroup(rawGroup(7, 'member', 2, 2)) };
  component.state.groupPosts = [component.mapAPIPost(rawGroupPost(71, 7, 2))];
  component.state.groupPostComposerText = 'discarded draft';
  component.state.groupPostComposerFile = { name: 'discarded.png' };
  component.state.groupPostComposerFileName = 'discarded.png';
  component.state.inviteOpen = true;
  component.state.commentsByPostID = {
    '71': Object.assign({}, emptyCommentStateForTest(), { draft: 'discarded comment', loaded: true })
  };
  component.state.openComments = { '71': true };
  global.AuthAPI.groupPosts = () => stalePage.promise;

  const pending = component.loadGroupPosts(7, true);
  component.handleRealtimeEvent(JSON.stringify({ type: 'chat:remove', chat: { kind: 'group', target_id: 7 } }));
  stalePage.resolve({ posts: [rawGroupPost(72, 7, 2)], next_cursor: null });
  await pending;

  assert.deepEqual(component.state.groupPosts, []);
  assert.equal(component.state.commentsByPostID['71'], undefined);
  assert.equal(component.state.openComments['71'], undefined);
  assert.equal(component.state.groupPostComposerText, '');
  assert.equal(component.state.groupPostComposerFile, null);
  assert.equal(component.groupAccessIsRevoked(7), true);
  const view = component.renderVals();
  assert.equal(view.gCanChat, false);
  assert.equal(view.gCanContent, false);
  assert.equal(view.gIsMember, false);
  assert.equal(view.inviteOpen, false);

  const chatKey = ChatModel.chatKey('group', 7);
  component.openGroupChat(7);
  assert.equal(component.state.chatsByKey[chatKey], undefined);
  assert.equal(component.revokedChatKeys.has(chatKey), true);
});

test('chat remove for another group does not clear the active group content', () => {
  const component = createComponent();
  component.state.screen = 'group';
  component.state.groupId = 8;
  component.state.apiGroupsByID = {
    '7': component.mapAPIGroup(rawGroup(7, 'member', 2, 2)),
    '8': component.mapAPIGroup(rawGroup(8, 'member', 2, 2))
  };
  component.state.groupPosts = [component.mapAPIPost(rawGroupPost(81, 8, 2))];
  component.state.groupPostComposerText = 'keep active draft';

  component.handleRealtimeEvent(JSON.stringify({ type: 'chat:remove', chat: { kind: 'group', target_id: 7 } }));

  assert.deepEqual(component.state.groupPosts.map(post => post.id), ['81']);
  assert.equal(component.state.groupPostComposerText, 'keep active draft');
  assert.equal(component.groupAccessIsRevoked(7), true);
  assert.equal(component.groupAccessIsRevoked(8), false);
});

test('chat remove invalidates pending group detail and members without reviving access', async () => {
  const component = createComponent();
  const detail = deferred();
  const members = deferred();
  component.state.screen = 'group';
  component.state.groupId = 7;
  component.state.apiGroupsByID = { '7': component.mapAPIGroup(rawGroup(7, 'member', 2, 2)) };
  component.state.groupMembers = [{ userID: 1, status: 'member', createdAt: '2026-07-22T10:00:00Z' }];
  global.AuthAPI.group = () => detail.promise;
  global.AuthAPI.groupMembers = () => members.promise;

  const pendingDetail = component.loadGroupDetail(7);
  const pendingMembers = component.loadGroupMembers(7, true);
  component.handleRealtimeEvent(JSON.stringify({ type: 'chat:remove', chat: { kind: 'group', target_id: 7 } }));
  detail.resolve(rawGroup(7, 'member', 2, 2));
  members.resolve({ members: [{ user: rawUser(2), status: 'member', created_at: '2026-07-22T10:01:00Z' }], next_cursor: null });
  await Promise.all([pendingDetail, pendingMembers]);

  const view = component.renderVals();
  assert.equal(component.groupAccessIsRevoked(7), true);
  assert.equal(view.gCanChat, false);
  assert.equal(view.gIsMember, false);
  assert.deepEqual(component.state.groupMembers, []);
});

test('local leave purges group content and authoritative rejoin loads fresh history', async () => {
  const component = createComponent();
  component.state.screen = 'group';
  component.state.groupId = 7;
  component.state.apiGroupsByID = { '7': component.mapAPIGroup(rawGroup(7, 'member', 2, 2)) };
  component.state.groupPosts = [component.mapAPIPost(rawGroupPost(71, 7, 1))];
  component.state.commentsByPostID = {
    '71': Object.assign({}, emptyCommentStateForTest(), { draft: 'leave draft', loaded: true })
  };
  global.AuthAPI.leaveGroup = async () => rawGroup(7, 'none', 1, 2);
  global.AuthAPI.groupMembers = async () => ({ members: [], next_cursor: null });
  global.AuthAPI.chats = async () => ({ chats: [], next_cursor: null });

  await component.leaveGroup(7);
  assert.equal(component.groupAccessIsRevoked(7), true);
  assert.deepEqual(component.state.groupPosts, []);
  assert.equal(component.state.commentsByPostID['71'], undefined);

  component.state.apiGroupsByID['7'] = component.mapAPIGroup(rawGroup(7, 'invited', 1, 2));
  global.AuthAPI.acceptGroupInvitation = async () => rawGroup(7, 'member', 2, 2);
  global.AuthAPI.groupInvitationInbox = async () => ({ invitations: [], next_cursor: null });
  global.AuthAPI.groupPosts = async () => ({ posts: [rawGroupPost(72, 7, 1)], next_cursor: null });
  await component.acceptGroupInvitation(7);
  await Promise.resolve();
  await Promise.resolve();

  assert.equal(component.groupAccessIsRevoked(7), false);
  assert.deepEqual(component.state.groupPosts.map(post => post.id), ['72']);
});

test('rejected group post create cannot write an error into the next group composer', async () => {
  const component = createComponent();
  const create = deferred();
  component.state.screen = 'group';
  component.state.groupId = 7;
  component.state.groupPostComposerText = 'group A draft';
  component.state.apiGroupsByID = {
    '7': component.mapAPIGroup(rawGroup(7, 'member', 2, 2)),
    '8': component.mapAPIGroup(rawGroup(8, 'member', 2, 2))
  };
  global.AuthAPI.createGroupPost = () => create.promise;
  global.AuthAPI.group = async () => rawGroup(8, 'member', 2, 2);
  global.AuthAPI.groupMembers = async () => ({ members: [], next_cursor: null });
  global.AuthAPI.groupPosts = async () => ({ posts: [], next_cursor: null });

  const pendingCreate = component.sendGroupPost();
  component.openGroup(8);
  component.setState({
    groupPostComposerText: 'group B draft',
    groupPostComposerPending: false,
    groupPostComposerError: ''
  });
  create.reject(new Error('group A network failure'));
  await pendingCreate;

  assert.equal(component.state.groupId, 8);
  assert.equal(component.state.groupPostComposerText, 'group B draft');
  assert.equal(component.state.groupPostComposerPending, false);
  assert.equal(component.state.groupPostComposerError, '');
});

test('group comment count is monotonic across group and personal copies', async () => {
  const component = createComponent();
  const create = deferred();
  const groupPost = component.mapAPIPost(rawGroupPost(91, 7, 2, 4));
  component.state.groupPosts = [groupPost];
  component.state.posts = [Object.assign({}, groupPost, { commentsCount: 3 })];
  component.state.commentsByPostID = {
    '91': Object.assign({}, emptyCommentStateForTest(), { draft: 'new comment' })
  };
  global.AuthAPI.createComment = () => create.promise;

  const pending = component.createComment(91);
  component.state.groupPosts = [Object.assign({}, groupPost, { commentsCount: 5 })];
  create.resolve(rawComment(99, 91, 1));
  await pending;

  assert.equal(component.state.groupPosts[0].commentsCount, 5);
  assert.equal(component.state.posts[0].commentsCount, 5);
});

test('logout invalidates pending group post load and create operations', async () => {
  const component = createComponent();
  const load = deferred();
  const create = deferred();
  component.state.screen = 'group';
  component.state.groupId = 7;
  component.state.groupPostComposerText = 'pending create';
  component.state.apiGroupsByID = { '7': component.mapAPIGroup(rawGroup(7, 'member', 2, 2)) };
  global.AuthAPI.groupPosts = () => load.promise;
  global.AuthAPI.createGroupPost = () => create.promise;
  global.AuthAPI.logout = async () => null;

  const pendingLoad = component.loadGroupPosts(7, true);
  const pendingCreate = component.sendGroupPost();
  await component.logout();
  load.resolve({ posts: [rawGroupPost(71, 7, 2)], next_cursor: null });
  create.resolve(rawGroupPost(72, 7, 1));
  await Promise.all([pendingLoad, pendingCreate]);

  assert.equal(component.state.authStatus, 'anonymous');
  assert.deepEqual(component.state.groupPosts, []);
  assert.equal(component.state.groupPostComposerText, '');
});

test('late history for chat A never replaces active chat B', async () => {
  const component = createComponent();
  const historyA = deferred();
  const historyB = deferred();
  component.state.chatsByKey = {
    'direct:2': ChatModel.normalizeChatSummary(rawDirectChat(2)),
    'direct:3': ChatModel.normalizeChatSummary(rawDirectChat(3))
  };
  component.state.chatKeys = ['direct:2', 'direct:3'];
  global.AuthAPI.directMessages = userID => userID === 2 ? historyA.promise : historyB.promise;

  component.openChat('direct:2');
  component.openChat('direct:3');
  historyB.resolve({
    messages: [rawChatMessage(32, 'e62617ed-e1bb-4483-81dd-a317d59aa23a', 'direct', 3, 3)],
    next_cursor: null
  });
  await Promise.resolve();
  await Promise.resolve();
  historyA.resolve({
    messages: [rawChatMessage(22, '6bbaef99-b778-4f98-bcc6-b51f52394403', 'direct', 2, 2)],
    next_cursor: null
  });
  await Promise.resolve();
  await Promise.resolve();

  assert.equal(component.state.activeChatKey, 'direct:3');
  assert.deepEqual(component.chatMessages(component.state.activeChatKey).messages.map(message => message.apiId), [32]);
});

test('optimistic message becomes one authoritative message and HTTP copy stays deduplicated', async () => {
  const component = createComponent();
  const key = 'direct:2';
  component.state.chatsByKey = { [key]: ChatModel.normalizeChatSummary(rawDirectChat(2)) };
  component.state.chatKeys = [key];
  component.state.activeChatKey = key;
  component.state.chatDraft = 'Hello';
  component.state.wsStatus = 'connected';
  const sent = [];
  component.ws = { readyState: 1, send: payload => sent.push(JSON.parse(payload)) };

  component.sendMsg();
  assert.equal(sent.length, 1);
  const clientID = sent[0].client_message_id;
  const authoritative = rawChatMessage(90, clientID, 'direct', 2, 1);
  component.handleRealtimeMessage(authoritative);
  component.handleRealtimeMessage(authoritative);

  const messages = component.chatMessages(key).messages;
  assert.equal(messages.length, 1);
  assert.equal(messages[0].apiId, 90);
  assert.equal(messages[0].pending, false);
  component.stopRealtime(false);
});

test('failed send retries with the exact same client_message_id', () => {
  const component = createComponent();
  const key = 'direct:2';
  component.state.chatsByKey = { [key]: ChatModel.normalizeChatSummary(rawDirectChat(2)) };
  component.state.chatKeys = [key];
  component.state.activeChatKey = key;
  component.state.chatDraft = 'Retry me';
  component.state.wsStatus = 'connected';
  const sent = [];
  component.ws = { readyState: 1, send: payload => sent.push(JSON.parse(payload)) };

  component.sendMsg();
  const clientID = sent[0].client_message_id;
  component.handleRealtimeError({ client_message_id: clientID, message: 'forbidden' });
  component.retryMessage(clientID);

  assert.equal(sent.length, 2);
  assert.equal(sent[1].client_message_id, clientID);
  assert.equal(component.chatMessages(key).messages.length, 1);
  component.stopRealtime(false);
});

test('group leave purges chat and a late socket message cannot recreate it', () => {
  const component = createComponent();
  const key = 'group:7';
  component.state.chatsByKey = {
    [key]: ChatModel.normalizeChatSummary({
      kind: 'group', target_id: 7, group: rawGroup(7, 'member', 3, 2),
      last_message: null, activity_at: '2026-07-22T11:00:00Z'
    })
  };
  component.state.chatKeys = [key];
  component.state.activeChatKey = key;

  component.purgeChat(key);
  component.handleRealtimeMessage(rawChatMessage(
    77, '0932903b-c2f1-4dca-9e52-c7ca9ac4f94c', 'group', 7, 2
  ));

  assert.equal(component.state.chatsByKey[key], undefined);
  assert.equal(component.state.messagesByChatKey[key], undefined);
});

test('chat remove purges revoked group access in every tab state', () => {
  const component = createComponent();
  const removedKey = 'group:7';
  const fallbackKey = 'direct:2';
  component.state.chatsByKey = {
    [removedKey]: ChatModel.normalizeChatSummary({
      kind: 'group', target_id: 7, group: rawGroup(7, 'member', 3, 2),
      last_message: null, activity_at: '2026-07-22T11:00:01Z'
    }),
    [fallbackKey]: ChatModel.normalizeChatSummary(rawDirectChat(2))
  };
  component.state.chatKeys = [removedKey, fallbackKey];
  component.state.activeChatKey = removedKey;
  component.state.messagesByChatKey = {
    [removedKey]: { messages: [], nextCursor: null, loading: false, pending: false, loaded: true, error: '' }
  };
  component.state.typingByChatKey = { [removedKey]: { 2: { id: 2, name: 'User 2' } } };
  component.typingChatKey = removedKey;
  const accessGeneration = component.chatAccessGate(removedKey).current();

  component.handleRealtimeEvent(JSON.stringify({
    type: 'chat:remove', chat: { kind: 'group', target_id: 7 }
  }));
  component.handleRealtimeMessage(rawChatMessage(
    78, '14ecf674-cfed-48f0-8ea0-ed6c9dcd0627', 'group', 7, 2
  ));

  assert.equal(component.state.chatsByKey[removedKey], undefined);
  assert.equal(component.state.messagesByChatKey[removedKey], undefined);
  assert.equal(component.state.typingByChatKey[removedKey], undefined);
  assert.equal(component.state.activeChatKey, fallbackKey);
  assert.equal(component.typingChatKey, null);
  assert.equal(component.chatAccessGate(removedKey).isCurrent(accessGeneration), false);
});

test('old WebSocket generation cannot mutate a newly authenticated session', () => {
  const component = createComponent();
  const previousWindow = global.window;
  const previousWebSocket = global.WebSocket;
  class FakeSocket {
    constructor() {
      this.readyState = 0;
      this.sent = [];
      FakeSocket.instances.push(this);
    }
    send(payload) { this.sent.push(payload); }
    close() { this.readyState = 3; }
  }
  FakeSocket.instances = [];
  global.window = { location: { protocol: 'http:', host: 'example.test' } };
  global.WebSocket = FakeSocket;
  try {
    component.connectRealtime(component.authGate.current());
    const oldSocket = FakeSocket.instances[0];
    oldSocket.readyState = 1;
    oldSocket.onopen();
    component.stopRealtime(false);
    component.state.authStatus = 'authenticated';
    component.connectRealtime(component.authGate.current());
    const currentSocket = FakeSocket.instances[1];
    currentSocket.readyState = 1;
    currentSocket.onopen();

    oldSocket.onmessage({
      data: JSON.stringify({ type: 'presence:init', online_user_ids: [99] })
    });
    assert.deepEqual(component.state.onlineUserIDs, {});
    component.stopRealtime(false);
  } finally {
    global.window = previousWindow;
    global.WebSocket = previousWebSocket;
  }
});

test('deferred send callback cannot cross logout into another session', () => {
  const component = createComponent();
  const key = 'direct:2';
  component.state.chatsByKey = { [key]: ChatModel.normalizeChatSummary(rawDirectChat(2)) };
  component.state.chatKeys = [key];
  component.state.activeChatKey = key;
  component.state.chatDraft = 'must not cross sessions';
  component.state.wsStatus = 'connected';
  const sent = [];
  component.ws = { readyState: 1, send: payload => sent.push(payload), close() {} };

  const originalSetState = component.setState.bind(component);
  let deferredSend = null;
  component.setState = (patch, callback) => {
    originalSetState(patch);
    if (callback) deferredSend = callback;
  };
  component.sendMsg();
  assert.equal(typeof deferredSend, 'function');

  component.authGate.begin();
  component.chatAccessGate(key).begin();
  component.state.authStatus = 'anonymous';
  component.stopRealtime(false);
  deferredSend();

  assert.deepEqual(sent, []);
});

test('typing state is keyed by chat and clears on stop', () => {
  const component = createComponent();
  const event = {
    chat: { kind: 'group', target_id: 7 },
    user: { id: 2, display_name: 'User 2' },
    typing: true
  };
  component.handleTypingUpdate(event);
  assert.equal(component.state.typingByChatKey['group:7']['2'].name, 'User 2');
  component.handleTypingUpdate(Object.assign({}, event, { typing: false }));
  assert.equal(component.state.typingByChatKey['group:7'], undefined);
});
