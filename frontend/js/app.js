
const IC = {
  home: 'M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1Z',
  user: 'M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-7 10a7 7 0 0 1 14 0',
  users: 'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-7 10a7 7 0 0 1 14 0m1-17.5a4 4 0 0 1 0 7.1M17.8 14.6a7 7 0 0 1 4.2 6.4',
  chat: 'M21 11.5a8.5 8.5 0 0 1-8.5 8.5c-1.5 0-3-.4-4.2-1.1L3 21l2.1-5.3A8.5 8.5 0 1 1 21 11.5Z',
  bell: 'M18 8.5a6 6 0 1 0-12 0c0 7-2.5 8.5-2.5 8.5h17S18 15.5 18 8.5m-4.3 12a2 2 0 0 1-3.4 0',
  globe: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Zm-9-9h18M12 3c2.5 2.5 3.8 5.6 3.8 9s-1.3 6.5-3.8 9c-2.5-2.5-3.8-5.6-3.8-9S9.5 5.5 12 3Z',
  lock: 'M7 10.5V7a5 5 0 0 1 10 0v3.5M6 10.5h12a1 1 0 0 1 1 1V20a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1v-8.5a1 1 0 0 1 1-1Z',
  sun: 'M12 16.5a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9ZM12 2.5v2m0 15v2m-9.5-9.5h2m15 0h2M5.3 5.3l1.4 1.4m10.6 10.6 1.4 1.4m0-13.4-1.4 1.4M6.7 17.3l-1.4 1.4',
  moon: 'M20.5 13.2A8.5 8.5 0 1 1 10.8 3.5a7 7 0 0 0 9.7 9.7Z',
  plus: 'M12 5v14M5 12h14',
  cal: 'M8 2.5v3m8-3v3M3.5 9h17m-15-3.5h13a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2v-11a2 2 0 0 1 2-2Z'
};

const USERS = {
  me:    { id: 'me', name: 'Alex Morgan', handle: '@alexmorgan', initials: 'AM', color: '#6b62c9', bio: 'Product designer. Building calm interfaces.', email: 'alex@loop.social', dob: 'March 14, 1996', private: false },
  mei:   { id: 'mei', name: 'Mei Lin', handle: '@meilin', initials: 'ML', color: '#b3813f', bio: 'Design systems at Fluxo. Type nerd.', email: 'mei@fluxo.co', dob: 'June 2, 1994', private: false },
  david: { id: 'david', name: 'David Okafor', handle: '@dokafor', initials: 'DO', color: '#3f9a85', bio: 'Frontend engineer. Weekend trail runner.', email: 'david@okafor.dev', dob: 'Nov 20, 1991', private: false },
  nina:  { id: 'nina', name: 'Nina Kov\u00e1cs', handle: '@ninak', initials: 'NK', color: '#c25a83', bio: 'Brand designer. Mostly lurking.', email: 'nina@studio-nk.com', dob: 'Feb 8, 1997', private: true },
  tom:   { id: 'tom', name: 'Tom\u00e1s Rivera', handle: '@tomriv', initials: 'TR', color: '#4d84c4', bio: 'Photographer. Film only, no exceptions.', email: 'tom@rivera.photo', dob: 'Aug 30, 1993', private: true },
  sara:  { id: 'sara', name: 'Sara Bishop', handle: '@sarab', initials: 'SB', color: '#8f6cc9', bio: 'Illustrator and printmaker.', email: 'sara@bishop.ink', dob: 'Apr 17, 1995', private: false }
};

const REPLIES = ['Totally agree \ud83d\ude04', 'Ha! Send it over', 'Let\u2019s sync tomorrow?', '\ud83d\udc40 looking now', 'Love that \u2728', 'Okay that\u2019s actually great'];
const EMOJIS = ['\ud83d\ude00', '\ud83d\ude02', '\ud83d\ude0d', '\ud83d\udd25', '\ud83d\udc4d', '\ud83c\udf89', '\ud83d\ude2e', '\ud83d\ude22', '\u2764\ufe0f', '\ud83d\udc40', '\u2728', '\ud83d\ude4c'];
const GROUP_COLORS = ['#6b62c9', '#b3813f', '#3f9a85', '#c25a83', '#4d84c4', '#8f6cc9'];

function cover(color) {
  return 'linear-gradient(135deg, color-mix(in oklab, ' + color + ' 55%, var(--surface2)), color-mix(in oklab, ' + color + ' 14%, var(--surface2)))';
}
function num(x) { return String(x); }

function emptyRegistrationForm() {
  return {
    authEmail: '', authPassword: '',
    regFirstName: '', regLastName: '', regDateOfBirth: '', regGender: '',
    regNickname: '', regAboutMe: '', regAvatar: null, regAvatarName: ''
  };
}

function emptyProfileEditor() {
  return {
    profileEditOpen: false, profileEditPending: false, profileAvatarPending: false,
    profileEditError: '', editFirstName: '', editLastName: '', editDateOfBirth: '',
    editGender: '', editNickname: '', editAboutMe: '', editAvatar: null, editAvatarName: ''
  };
}

function emptyCommentState() {
  return {
    comments: [], loading: false, pending: false, error: '', nextCursor: null,
    draft: '', createPending: false, createError: '', loaded: false
  };
}

function decorateUser(user) {
  const safeAvatarURL = user.avatarUrl ? String(user.avatarUrl).replace(/["\\\r\n]/g, '') : '';
  user.avatarUrl = safeAvatarURL;
  user.hasAvatar = !!safeAvatarURL;
  user.noAvatar = !safeAvatarURL;
  user.hasCustomAvatar = AvatarURL.isCustomAvatarURL(safeAvatarURL);
  return user;
}

Object.keys(USERS).forEach(uid => decorateUser(USERS[uid]));

function requestErrorMessage(error, fallback) {
  if (error && typeof error.message === 'string' && error.message.trim()) return error.message.trim();
  return fallback;
}

class Component extends DCLogic {
  constructor(props) {
    super(props);
    let saved = null;
    try { saved = localStorage.getItem('loop-theme'); } catch (e) {}
    this.state = {
      theme: saved || props.defaultTheme || 'light',
      screen: 'feed', feedLoading: true, feedPending: false, feedError: '', feedNextCursor: null,
      composerText: '', composerFile: null, composerFileName: '', composerError: '', composerPending: false,
      privacy: 'public', privacyOpen: false,
      selectedFollowers: {}, postFollowers: [], postFollowersLoading: false,
      openComments: { p1: true }, drafts: {},
      commentsByPostID: {},
      posts: [],
      apiUsersByID: {}, directoryUserIDs: [], directoryNextCursor: null, directoryLoading: false, directoryError: '',
      followPendingByID: {}, followErrorByID: {},
      followRequests: [], followRequestsLoading: false, followRequestsError: '', followRequestPendingByID: {},
      myPrivacy: 'public', profilePrivacyPending: false, profilePrivacyError: '',
      profileId: null, profileTab: 'posts', profileLoading: false, profileReady: false, profileError: '',
      profileFollowers: [], profileFollowing: [], profileListsLoading: false, profileListsError: '',
      profilePosts: [], profilePostsLoading: false, profilePostsPending: false,
      profilePostsError: '', profilePostsNextCursor: null,
      apiGroupsByID: {}, groupIDs: [], groupsNextCursor: null,
      groupsLoading: false, groupsPending: false, groupsError: '',
      groupInvitationInbox: [], groupInvitationInboxNextCursor: null,
      groupInvitationInboxLoading: false, groupInvitationInboxError: '',
      groupId: null, groupTab: 'posts', groupLoading: false, groupError: '',
      groupMembers: [], groupMembersNextCursor: null, groupMembersLoading: false, groupMembersError: '',
      groupRequests: [], groupRequestsNextCursor: null, groupRequestsLoading: false, groupRequestsError: '',
      groupInvitations: [], groupInvitationsNextCursor: null, groupInvitationsLoading: false, groupInvitationsError: '',
      groupMutationPendingByID: {}, groupMutationErrorByID: {},
      inviteOpen: false, groupInviteUserID: '',
      createOpen: false, ngName: '', ngDesc: '', groupCreatePending: false, groupCreateError: '',
      convos: [
        { id: 'c1', kind: 'dm', uid: 'nina', unread: 2, typing: false, online: true, messages: [
          { from: 'nina', text: 'Did you see the moodboard I posted? \ud83d\udc40', time: '09:12' },
          { from: 'me', text: 'Yes! The serif direction is bold. I like it', time: '09:14' },
          { from: 'nina', text: 'Okay good. I was 50/50 on it \ud83d\ude05', time: '09:15' },
          { from: 'nina', text: 'Coffee this week? I want your take on the type scale', time: '09:15' } ] },
        { id: 'c2', kind: 'dm', uid: 'david', unread: 0, typing: false, online: false, messages: [
          { from: 'me', text: 'That tokens write-up is gold \ud83d\udd25', time: 'Tue' },
          { from: 'david', text: 'Ha, thanks! Took forever to edit down', time: 'Tue' } ] }
      ],
      convoId: 'c1', chatDraft: '', emojiOpen: false,
      notifs: [],
      authMode: 'login', authStatus: 'checking', authPending: false, logoutPending: false,
      authError: '', bootstrapError: '', appError: '',
      ...emptyRegistrationForm(),
      ...emptyProfileEditor()
    };
    this.msgEl = null;
    this.authGate = UserModel.createRequestGate();
    this.profileGate = UserModel.createRequestGate();
    this.feedGate = UserModel.createRequestGate();
    this.directoryGate = UserModel.createRequestGate();
    this.postFollowersGate = UserModel.createRequestGate();
    this.commentAccessGatesByPostID = {};
    this.commentLoadGatesByPostID = {};
    this.groupsDirectoryGate = UserModel.createRequestGate();
    this.groupInvitationInboxGate = UserModel.createRequestGate();
    this.groupGenerationsByID = {};
    this.groupDetailGate = UserModel.createRequestGate();
    this.groupMembersGate = UserModel.createRequestGate();
    this.groupRequestsGate = UserModel.createRequestGate();
    this.groupInvitationsGate = UserModel.createRequestGate();
  }

  componentDidMount() {
    document.documentElement.dataset.theme = this.state.theme;
    this.applyTokens();
    this.loadCurrentUser();
  }
  componentDidUpdate() {
    this.applyTokens();
    if (this.msgEl) this.msgEl.scrollTop = this.msgEl.scrollHeight;
  }
  applyTokens() {
    const el = document.documentElement;
    el.style.setProperty('--accent', this.props.accent || '#5661d8');
    el.style.setProperty('--r', (this.props.radius != null ? this.props.radius : 18) + 'px');
  }

  mergeAPIUsers(rawUsers, baseStore) {
    const currentUserID = USERS.me && USERS.me.apiId;
    const base = Object.assign({}, baseStore || this.state.apiUsersByID);
    if (currentUserID) base[String(currentUserID)] = USERS.me;
    const next = UserModel.mergeUsers(base, rawUsers, currentUserID);
    Object.keys(next).forEach(id => decorateUser(next[id]));
    return next;
  }

  applyAuthUser(user, baseStore) {
    const next = UserModel.mergeUsers(baseStore || this.state.apiUsersByID, [user], user.id);
    const me = decorateUser(next[String(user.id)]);
    USERS.me = me;
    next[String(user.id)] = me;
    return next;
  }

  apiUser(userID) {
    const id = String(Number(userID));
    if (USERS.me && String(USERS.me.apiId) === id) return USERS.me;
    return this.state.apiUsersByID[id] || decorateUser({
      id, apiId: Number(id), name: 'User ' + id, handle: 'user-' + id,
      initials: '?', color: '#5661d8', bio: '', private: false,
      relationship: { status: 'none', follows_me: false }
    });
  }

  mapAPIPost(post) {
    const normalized = PostModel.normalizePostResponse(post, USERS.me.apiId);
    return {
      id: normalized.id,
      real: true,
      apiAuthorID: normalized.apiAuthorID,
      text: normalized.text,
      privacy: normalized.privacy,
      mediaUrl: normalized.mediaUrl,
      commentsCount: normalized.commentsCount,
      time: this.formatPostTime(normalized.createdAt)
    };
  }

  formatPostTime(value) {
    const created = new Date(value);
    if (Number.isNaN(created.getTime())) return '';
    const seconds = Math.max(0, Math.floor((Date.now() - created.getTime()) / 1000));
    if (seconds < 60) return 'now';
    if (seconds < 3600) return Math.floor(seconds / 60) + 'm';
    if (seconds < 86400) return Math.floor(seconds / 3600) + 'h';
    if (seconds < 604800) return Math.floor(seconds / 86400) + 'd';
    return created.toLocaleDateString();
  }

  commentState(postID) {
    return this.state.commentsByPostID[String(Number(postID))] || emptyCommentState();
  }

  commentAccessGate(postID) {
    const key = String(Number(postID));
    if (!this.commentAccessGatesByPostID[key]) this.commentAccessGatesByPostID[key] = UserModel.createRequestGate();
    return this.commentAccessGatesByPostID[key];
  }

  commentLoadGate(postID) {
    const key = String(Number(postID));
    if (!this.commentLoadGatesByPostID[key]) this.commentLoadGatesByPostID[key] = UserModel.createRequestGate();
    return this.commentLoadGatesByPostID[key];
  }

  maxPostCommentsCount(postID, ...collections) {
    postID = Number(postID);
    return collections.reduce((maximum, posts) => (posts || []).reduce((currentMaximum, post) => (
      Number(post.id) === postID ? Math.max(currentMaximum, Number(post.commentsCount) || 0) : currentMaximum
    ), maximum), 0);
  }

  mergePostCommentsCounts(incoming, ...localCollections) {
    return (incoming || []).map(post => Object.assign({}, post, {
      commentsCount: Math.max(
        Number(post.commentsCount) || 0,
        this.maxPostCommentsCount(post.id, ...localCollections)
      )
    }));
  }

  patchCommentState(postID, patch) {
    const key = String(Number(postID));
    this.setState(state => {
      const entries = Object.assign({}, state.commentsByPostID);
      entries[key] = Object.assign({}, emptyCommentState(), entries[key] || {}, patch || {});
      return { commentsByPostID: entries };
    });
  }

  purgeCommentStates(postIDs) {
    const removed = {};
    (postIDs || []).forEach(postID => {
      const key = String(Number(postID));
      if (key !== 'NaN') {
        removed[key] = true;
        this.commentAccessGate(key).begin();
      }
    });
    if (!Object.keys(removed).length) return;
    this.setState(state => {
      const entries = Object.assign({}, state.commentsByPostID);
      const openComments = Object.assign({}, state.openComments);
      Object.keys(removed).forEach(key => {
        delete entries[key];
        delete openComments[key];
      });
      return { commentsByPostID: entries, openComments };
    });
  }

  loadComments = async (postID, reset) => {
    postID = Number(postID);
    if (!Number.isInteger(postID) || postID <= 0) return;
    const authGeneration = this.authGate.current();
    const accessGate = this.commentAccessGate(postID);
    const accessGeneration = accessGate.current();
    const loadGate = this.commentLoadGate(postID);
    const loadGeneration = reset ? loadGate.begin() : loadGate.current();
    const state = this.commentState(postID);
    if (!reset && state.pending) return;
    const cursor = reset ? null : state.nextCursor;
    if (!reset && !cursor) return;
    this.patchCommentState(postID, { pending: true, loading: !!reset, error: '' });
    try {
      const page = await AuthAPI.postComments(postID, cursor, 20);
      if (
        !this.authGate.isCurrent(authGeneration) ||
        !accessGate.isCurrent(accessGeneration) ||
        !loadGate.isCurrent(loadGeneration)
      ) return;
      const rawComments = page.comments || [];
      const incoming = rawComments.map(CommentModel.normalizeCommentResponse);
      const apiUsersByID = this.mergeAPIUsers(rawComments.map(comment => comment.author));
      const latest = this.commentState(postID);
      this.setState({ apiUsersByID });
      this.patchCommentState(postID, {
        comments: CommentModel.mergeComments(latest.comments, incoming),
        pending: false, loading: false, error: '', loaded: true,
        nextCursor: page.next_cursor || null
      });
    } catch (error) {
      if (
        !this.authGate.isCurrent(authGeneration) ||
        !accessGate.isCurrent(accessGeneration) ||
        !loadGate.isCurrent(loadGeneration)
      ) return;
      if (error && (error.status === 403 || error.status === 404)) {
        accessGate.begin();
        this.patchCommentState(postID, Object.assign(emptyCommentState(), {
          error: error.status === 403 ? 'You no longer have access to these comments.' : 'Post not found.'
        }));
        return;
      }
      this.patchCommentState(postID, {
        pending: false, loading: false,
        error: requestErrorMessage(error, 'Could not load comments. Please try again.')
      });
    }
  };

  togglePostComments = (postID) => {
    const key = String(Number(postID));
    const opening = !this.state.openComments[key];
    this.setState({ openComments: Object.assign({}, this.state.openComments, { [key]: opening }) });
    const state = this.commentState(postID);
    if (opening && !state.loaded && !state.pending) this.loadComments(postID, true);
  };

  setCommentDraft = (postID, value) => {
    this.patchCommentState(postID, { draft: value, createError: '' });
  };

  createComment = async (postID) => {
    postID = Number(postID);
    const state = this.commentState(postID);
    const text = state.draft.trim();
    if (!text || state.createPending) return;
    const authGeneration = this.authGate.current();
    const accessGate = this.commentAccessGate(postID);
    const accessGeneration = accessGate.current();
    const countAtCreateStart = this.maxPostCommentsCount(postID, this.state.posts, this.state.profilePosts);
    this.patchCommentState(postID, { createPending: true, createError: '' });
    try {
      const response = await AuthAPI.createComment(postID, text);
      if (!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration)) return;
      const comment = CommentModel.normalizeCommentResponse(response);
      const apiUsersByID = this.mergeAPIUsers([response.author]);
      const latest = this.commentState(postID);
      this.setState(current => ({
        apiUsersByID,
        posts: current.posts.map(post => Number(post.id) === postID
          ? Object.assign({}, post, { commentsCount: Math.max(Number(post.commentsCount) || 0, countAtCreateStart + 1) })
          : post),
        profilePosts: current.profilePosts.map(post => Number(post.id) === postID
          ? Object.assign({}, post, { commentsCount: Math.max(Number(post.commentsCount) || 0, countAtCreateStart + 1) })
          : post)
      }));
      this.patchCommentState(postID, {
        comments: CommentModel.mergeComments(latest.comments, [comment]),
        draft: '', createPending: false, createError: ''
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration)) return;
      if (error && (error.status === 403 || error.status === 404)) {
        accessGate.begin();
        this.patchCommentState(postID, Object.assign(emptyCommentState(), {
          draft: text,
          createError: error.status === 403 ? 'You no longer have access to this post.' : 'Post not found.'
        }));
        return;
      }
      this.patchCommentState(postID, {
        createPending: false,
        createError: requestErrorMessage(error, 'Could not send the comment. Your draft was kept.')
      });
    }
  };

  loadFeed = async (reset) => {
    const authGeneration = this.authGate.current();
    const generation = reset ? this.feedGate.begin() : this.feedGate.current();
    if (!reset && this.state.feedPending) return;
    const cursor = reset ? null : this.state.feedNextCursor;
    if (!reset && !cursor) return;
    this.setState({ feedPending: true, feedLoading: !!reset, feedError: '' });
    try {
      const page = await AuthAPI.feed(cursor, 20);
      if (!this.authGate.isCurrent(authGeneration) || !this.feedGate.isCurrent(generation)) return;
      const mapped = (page.posts || []).map(post => this.mapAPIPost(post));
      const apiUsersByID = this.mergeAPIUsers((page.posts || []).map(post => post.author));
      this.setState(current => {
        const merged = this.mergePostCommentsCounts(mapped, current.posts, current.profilePosts);
        return {
          posts: reset ? merged : current.posts.concat(merged),
          apiUsersByID,
          feedLoading: false, feedPending: false,
          feedNextCursor: page.next_cursor || null, feedError: ''
        };
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !this.feedGate.isCurrent(generation)) return;
      this.setState({
        feedLoading: false, feedPending: false,
        feedError: requestErrorMessage(error, 'Could not load the feed. Please try again.')
      });
    }
  };

  loadPostFollowers = async () => {
    if (!USERS.me.apiId) return;
    const authGeneration = this.authGate.current();
    const generation = this.postFollowersGate.begin();
    this.setState({ postFollowersLoading: true });
    try {
      const response = await AuthAPI.followers(USERS.me.apiId);
      if (!this.authGate.isCurrent(authGeneration) || !this.postFollowersGate.isCurrent(generation)) return;
      const apiUsersByID = this.mergeAPIUsers(response.users || []);
      const followers = (response.users || []).map(user => apiUsersByID[String(user.id)]);
      this.setState({
        apiUsersByID,
        postFollowers: followers,
        postFollowersLoading: false,
        selectedFollowers: UserModel.pruneSelected(this.state.selectedFollowers, followers)
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !this.postFollowersGate.isCurrent(generation)) return;
      this.setState({
        postFollowersLoading: false,
        composerError: requestErrorMessage(error, 'Could not load followers for selected posts.')
      });
    }
  };

  loadDirectory = async (reset = true) => {
    const authGeneration = this.authGate.current();
    const generation = reset ? this.directoryGate.begin() : this.directoryGate.current();
    if (!reset && this.state.directoryLoading) return;
    const cursor = reset ? null : this.state.directoryNextCursor;
    if (!reset && !cursor) return;
    this.setState({ directoryLoading: true, directoryError: '' });
    try {
      const response = await AuthAPI.users(cursor, 20);
      if (!this.authGate.isCurrent(authGeneration) || !this.directoryGate.isCurrent(generation)) return;
      const apiUsersByID = this.mergeAPIUsers(response.users || []);
      const incomingIDs = (response.users || []).map(user => Number(user.id));
      this.setState({
        apiUsersByID,
        directoryUserIDs: reset
          ? incomingIDs
          : this.state.directoryUserIDs.concat(incomingIDs.filter(id => this.state.directoryUserIDs.indexOf(id) < 0)),
        directoryNextCursor: response.next_cursor || null,
        directoryLoading: false, directoryError: ''
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !this.directoryGate.isCurrent(generation)) return;
      this.setState({
        directoryLoading: false,
        directoryError: requestErrorMessage(error, 'Could not load user suggestions.')
      });
    }
  };

  loadFollowRequests = async () => {
    if (this.state.followRequestsLoading) return;
    const authGeneration = this.authGate.current();
    this.setState({ followRequestsLoading: true, followRequestsError: '' });
    try {
      const response = await AuthAPI.followRequests();
      if (!this.authGate.isCurrent(authGeneration)) return;
      const requests = response.requests || [];
      const apiUsersByID = this.mergeAPIUsers(requests.map(request => request.user));
      this.setState({ apiUsersByID, followRequests: requests, followRequestsLoading: false });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.setState({
        followRequestsLoading: false,
        followRequestsError: requestErrorMessage(error, 'Could not load follow requests.')
      });
    }
  };

  loadProfileConnections = async (targetUserID, generation) => {
    targetUserID = Number(targetUserID);
    const authGeneration = this.authGate.current();
    generation = generation || this.profileGate.current();
    this.setState({ profileListsLoading: true, profileListsError: '' });
    try {
      const responses = await Promise.all([
        AuthAPI.followers(targetUserID),
        AuthAPI.following(targetUserID)
      ]);
      if (
        !this.authGate.isCurrent(authGeneration) ||
        !this.profileGate.isCurrent(generation) ||
        Number(this.state.profileId) !== targetUserID
      ) return;
      const followers = responses[0].users || [];
      const following = responses[1].users || [];
      const apiUsersByID = this.mergeAPIUsers(following, this.mergeAPIUsers(followers));
      this.setState({
        apiUsersByID,
        profileFollowers: followers.map(user => Number(user.id)),
        profileFollowing: following.map(user => Number(user.id)),
        profileListsLoading: false, profileListsError: ''
      });
    } catch (error) {
      if (
        !this.authGate.isCurrent(authGeneration) ||
        !this.profileGate.isCurrent(generation) ||
        Number(this.state.profileId) !== targetUserID
      ) return;
      if (error && error.status === 403) {
        this.setState({ profileFollowers: [], profileFollowing: [], profileListsLoading: false, profileListsError: '' });
        return;
      }
      this.setState({
        profileListsLoading: false,
        profileListsError: requestErrorMessage(error, 'Could not load followers and following.')
      });
    }
  };

  loadProfilePosts = async (targetUserID, reset, generation) => {
    targetUserID = Number(targetUserID || this.state.profileId);
    const authGeneration = this.authGate.current();
    generation = generation || this.profileGate.current();
    if (!targetUserID || (!reset && this.state.profilePostsPending)) return;
    const cursor = reset ? null : this.state.profilePostsNextCursor;
    if (!reset && !cursor) return;
    this.setState({ profilePostsPending: true, profilePostsLoading: !!reset, profilePostsError: '' });
    try {
      const page = await AuthAPI.userPosts(targetUserID, cursor, 20);
      if (
        !this.authGate.isCurrent(authGeneration) ||
        !this.profileGate.isCurrent(generation) ||
        Number(this.state.profileId) !== targetUserID
      ) return;
      const mapped = (page.posts || []).map(post => this.mapAPIPost(post));
      const apiUsersByID = this.mergeAPIUsers((page.posts || []).map(post => post.author));
      this.setState(current => {
        const merged = this.mergePostCommentsCounts(mapped, current.posts, current.profilePosts);
        return {
          profilePosts: reset ? merged : current.profilePosts.concat(merged),
          apiUsersByID,
          profilePostsLoading: false, profilePostsPending: false,
          profilePostsNextCursor: page.next_cursor || null, profilePostsError: ''
        };
      });
    } catch (error) {
      if (
        !this.authGate.isCurrent(authGeneration) ||
        !this.profileGate.isCurrent(generation) ||
        Number(this.state.profileId) !== targetUserID
      ) return;
      if (error && error.status === 403) {
        const user = this.apiUser(targetUserID);
        user.canViewProfile = false;
        user.bio = ''; user.dob = ''; user.postsCount = 0;
        this.setState({
          apiUsersByID: Object.assign({}, this.state.apiUsersByID, { [String(targetUserID)]: user }),
          profilePosts: [], profileFollowers: [], profileFollowing: [],
          profilePostsLoading: false, profilePostsPending: false, profilePostsError: ''
        });
        return;
      }
      this.setState({
        profilePostsLoading: false, profilePostsPending: false,
        profilePostsError: requestErrorMessage(error, 'Could not load profile posts. Please try again.')
      });
    }
  };

  loadCurrentUser = async () => {
    const authGeneration = this.authGate.begin();
    this.setState({ authStatus: 'checking', bootstrapError: '', appError: '' });
    try {
      const user = await AuthAPI.me();
      if (!this.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = this.applyAuthUser(user);
      this.setState({
        authStatus: 'authenticated', screen: 'feed',
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public',
        profilePrivacyPending: false, profilePrivacyError: ''
      });
      this.loadFeed(true);
      this.loadPostFollowers();
      this.loadDirectory();
      this.loadFollowRequests();
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      if (error && error.status === 401) {
        this.setState({ authStatus: 'anonymous', screen: 'auth', bootstrapError: '' });
        return;
      }
      this.setState({
        authStatus: 'error',
        bootstrapError: requestErrorMessage(error, 'Could not load your session. Please try again.')
      });
    }
  };

  setAuthMode = (mode) => this.setState({ authMode: mode, authError: '' });

  pickRegistrationAvatar = () => {
    const input = document.getElementById('registration-avatar');
    if (input) input.click();
  };

  onRegistrationAvatar = (event) => {
    const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
    this.setState({ regAvatar: file, regAvatarName: file ? file.name : '' });
  };

  submitAuth = async (event) => {
    if (event) event.preventDefault();
    if (this.state.authPending) return;

    const authGeneration = this.authGate.begin();
    const s = this.state;
    this.setState({ authPending: true, authError: '' });
    try {
      let user;
      if (s.authMode === 'login') {
        user = await AuthAPI.login(s.authEmail.trim(), s.authPassword);
      } else {
        const form = new FormData();
        form.append('email', s.authEmail.trim());
        form.append('password', s.authPassword);
        form.append('first_name', s.regFirstName.trim());
        form.append('last_name', s.regLastName.trim());
        form.append('date_of_birth', s.regDateOfBirth.trim());
        if (s.regGender) form.append('gender', s.regGender);
        if (s.regNickname.trim()) form.append('nickname', s.regNickname.trim());
        if (s.regAboutMe.trim()) form.append('about_me', s.regAboutMe.trim());
        if (s.regAvatar) form.append('avatar', s.regAvatar, s.regAvatar.name);
        user = await AuthAPI.register(form);
      }

      if (!this.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = this.applyAuthUser(user);
      const authenticatedState = {
        authStatus: 'authenticated', authPending: false, authError: '',
        authPassword: '', screen: 'feed',
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public',
        profilePrivacyPending: false, profilePrivacyError: ''
      };
      if (s.authMode === 'register') Object.assign(authenticatedState, emptyRegistrationForm());
      Object.assign(authenticatedState, emptyProfileEditor());
      this.setState(authenticatedState);
      this.loadFeed(true);
      this.loadPostFollowers();
      this.loadDirectory();
      this.loadFollowRequests();
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.setState({
        authPending: false,
        authError: requestErrorMessage(error, 'Authentication failed. Please try again.')
      });
    }
  };

  logout = async () => {
    if (this.state.logoutPending) return;
    const authGeneration = this.authGate.current();
    this.setState({ logoutPending: true, appError: '' });
    try {
      await AuthAPI.logout();
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.authGate.begin();
      this.feedGate.begin();
      this.directoryGate.begin();
      this.postFollowersGate.begin();
      this.profileGate.begin();
      this.groupsDirectoryGate.begin();
      this.groupInvitationInboxGate.begin();
      this.groupDetailGate.begin();
      this.groupMembersGate.begin();
      this.groupRequestsGate.begin();
      this.groupInvitationsGate.begin();
      Object.keys(this.groupGenerationsByID).forEach(key => this.groupGenerationsByID[key].begin());
      this.groupGenerationsByID = {};
      Object.keys(this.commentAccessGatesByPostID).forEach(key => this.commentAccessGatesByPostID[key].begin());
      this.commentAccessGatesByPostID = {};
      this.commentLoadGatesByPostID = {};
      this.setState(Object.assign({
        authStatus: 'anonymous', logoutPending: false, authMode: 'login',
        authError: '', screen: 'auth', myPrivacy: 'public',
        profilePrivacyPending: false, profilePrivacyError: '',
        posts: [], feedLoading: true, feedPending: false, feedError: '', feedNextCursor: null,
        profilePosts: [], profilePostsLoading: false, profilePostsPending: false,
        profilePostsError: '', profilePostsNextCursor: null,
        postFollowers: [], postFollowersLoading: false, selectedFollowers: {},
        commentsByPostID: {}, openComments: {},
        apiUsersByID: {}, directoryUserIDs: [], directoryNextCursor: null,
        directoryLoading: false, directoryError: '', followPendingByID: {}, followErrorByID: {},
        followRequests: [], followRequestsLoading: false, followRequestsError: '', followRequestPendingByID: {},
        profileId: null, profileReady: false, profileLoading: false, profileError: '',
        profileFollowers: [], profileFollowing: [], profileListsLoading: false, profileListsError: '',
        apiGroupsByID: {}, groupIDs: [], groupsNextCursor: null,
        groupsLoading: false, groupsPending: false, groupsError: '',
        groupInvitationInbox: [], groupInvitationInboxNextCursor: null,
        groupInvitationInboxLoading: false, groupInvitationInboxError: '',
        groupId: null, groupLoading: false, groupError: '', groupMembers: [], groupMembersNextCursor: null,
        groupMembersLoading: false, groupMembersError: '', groupRequests: [], groupRequestsNextCursor: null,
        groupRequestsLoading: false, groupRequestsError: '', groupInvitations: [], groupInvitationsNextCursor: null,
        groupInvitationsLoading: false, groupInvitationsError: '', groupMutationPendingByID: {},
        groupMutationErrorByID: {}, groupInviteUserID: '', inviteOpen: false,
        createOpen: false, ngName: '', ngDesc: '', groupCreatePending: false, groupCreateError: '',
        composerText: '', composerFile: null, composerFileName: '', composerError: '', composerPending: false,
        privacy: 'public', privacyOpen: false
      }, emptyRegistrationForm(), emptyProfileEditor()));
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.setState({
        logoutPending: false,
        appError: requestErrorMessage(error, 'Could not log out. Please try again.')
      });
    }
  };

  go = (screen) => {
    this.setState({ screen, privacyOpen: false, emojiOpen: false });
    if (screen === 'notifications') this.loadFollowRequests();
    if (screen === 'groups') {
      this.loadGroups(true);
      this.loadGroupInvitationInbox(true);
    }
  };
  openProfile = async (targetUserID) => {
    if (targetUserID === 'me') targetUserID = USERS.me.apiId;
    targetUserID = Number(targetUserID);
    if (!Number.isInteger(targetUserID) || targetUserID <= 0) return;
    const authGeneration = this.authGate.current();
    const generation = this.profileGate.begin();
    const isMe = targetUserID === USERS.me.apiId;
    this.setState({
      screen: 'profile', profileId: targetUserID, profileTab: 'posts',
      profileLoading: true, profileReady: false, profileError: '',
      profilePosts: [], profilePostsLoading: false, profilePostsPending: false,
      profilePostsError: '', profilePostsNextCursor: null,
      profileFollowers: [], profileFollowing: [], profileListsLoading: false, profileListsError: '',
      profileEditOpen: isMe ? this.state.profileEditOpen : false,
      profileEditError: isMe ? this.state.profileEditError : ''
    });
    try {
      const results = await Promise.all([
        AuthAPI.userProfile(targetUserID),
        isMe ? Promise.resolve({ status: 'none', follows_me: false }) : AuthAPI.relationship(targetUserID)
      ]);
      if (!this.authGate.isCurrent(authGeneration) || !this.profileGate.isCurrent(generation)) return;
      const rawUser = Object.assign({}, results[0], { relationship: results[1] });
      const apiUsersByID = this.mergeAPIUsers([rawUser]);
      const profileUser = apiUsersByID[String(targetUserID)];
      this.setState({ apiUsersByID, profileLoading: false, profileReady: true, profileError: '' });
      if (profileUser.canViewProfile) {
        this.loadProfilePosts(targetUserID, true, generation);
        this.loadProfileConnections(targetUserID, generation);
      }
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !this.profileGate.isCurrent(generation)) return;
      this.setState({
        profileLoading: false, profileReady: false,
        profileError: error && error.status === 404
          ? 'User not found.'
          : requestErrorMessage(error, 'Could not load this profile.')
      });
    }
  };

  openProfileEdit = () => {
    const me = USERS.me;
    this.setState({
      profileEditOpen: true, profileEditError: '',
      editFirstName: me.firstName || '', editLastName: me.lastName || '',
      editDateOfBirth: me.dob || '', editGender: me.gender || '',
      editNickname: me.nickname || '', editAboutMe: me.aboutMe || '',
      editAvatar: null, editAvatarName: ''
    });
  };

  cancelProfileEdit = () => this.setState(Object.assign({}, emptyProfileEditor()));

  saveProfile = async (event) => {
    if (event) event.preventDefault();
    if (this.state.profileEditPending || this.state.profileAvatarPending || this.state.profilePrivacyPending) return;
    const authGeneration = this.authGate.current();
    const s = this.state;
    this.setState({ profileEditPending: true, profileEditError: '' });
    try {
      const user = await AuthAPI.updateProfile({
        first_name: s.editFirstName.trim(),
        last_name: s.editLastName.trim(),
        date_of_birth: s.editDateOfBirth.trim(),
        gender: s.editGender || null,
        nickname: s.editNickname,
        about_me: s.editAboutMe
      });
      if (!this.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = this.applyAuthUser(user);
      this.setState(Object.assign({
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public'
      }, emptyProfileEditor()));
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.setState({
        profileEditPending: false,
        profileEditError: requestErrorMessage(error, 'Could not update your profile. Please try again.')
      });
    }
  };

  pickProfileAvatar = () => {
    const input = document.getElementById('profile-avatar');
    if (input) input.click();
  };

  onProfileAvatar = (event) => {
    const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
    this.setState({ editAvatar: file, editAvatarName: file ? file.name : '', profileEditError: '' });
  };

  replaceProfileAvatar = async () => {
    if (this.state.profileAvatarPending || this.state.profileEditPending || this.state.profilePrivacyPending || !this.state.editAvatar) return;
    const authGeneration = this.authGate.current();
    const avatar = this.state.editAvatar;
    this.setState({ profileAvatarPending: true, profileEditError: '' });
    try {
      const form = new FormData();
      form.append('avatar', avatar, avatar.name);
      const user = await AuthAPI.replaceAvatar(form);
      if (!this.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = this.applyAuthUser(user);
      const input = document.getElementById('profile-avatar');
      if (input) input.value = '';
      this.setState({
        apiUsersByID,
        profileAvatarPending: false, editAvatar: null, editAvatarName: '',
        myPrivacy: user.is_private === true ? 'private' : 'public'
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.setState({
        profileAvatarPending: false,
        profileEditError: requestErrorMessage(error, 'Could not replace your avatar. Please try again.')
      });
    }
  };

  deleteProfileAvatar = async () => {
    if (this.state.profileAvatarPending || this.state.profileEditPending || this.state.profilePrivacyPending) return;
    const authGeneration = this.authGate.current();
    this.setState({ profileAvatarPending: true, profileEditError: '' });
    try {
      const user = await AuthAPI.deleteAvatar();
      if (!this.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = this.applyAuthUser(user);
      const input = document.getElementById('profile-avatar');
      if (input) input.value = '';
      this.setState({
        apiUsersByID,
        profileAvatarPending: false, editAvatar: null, editAvatarName: '',
        myPrivacy: user.is_private === true ? 'private' : 'public'
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.setState({
        profileAvatarPending: false,
        profileEditError: requestErrorMessage(error, 'Could not delete your avatar. Please try again.')
      });
    }
  };

  setProfilePrivacy = async (privacy) => {
    if (
      this.state.profilePrivacyPending ||
      this.state.profileEditPending ||
      this.state.profileAvatarPending ||
      privacy === this.state.myPrivacy
    ) return;
    const authGeneration = this.authGate.current();
    const isPrivate = privacy === 'private';
    this.setState({ profilePrivacyPending: true, profilePrivacyError: '' });
    try {
      const user = await AuthAPI.updateProfile({ is_private: isPrivate });
      if (!this.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = this.applyAuthUser(user);
      this.setState({
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public',
        profilePrivacyPending: false,
        profilePrivacyError: ''
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.setState({
        profilePrivacyPending: false,
        profilePrivacyError: requestErrorMessage(error, 'Could not update profile privacy. Please try again.')
      });
    }
  };

  toggleTheme = (e) => {
    const el = document.documentElement;
    if (e && e.clientX != null) {
      el.style.setProperty('--vt-x', e.clientX + 'px');
      el.style.setProperty('--vt-y', e.clientY + 'px');
    }
    const next = this.state.theme === 'light' ? 'dark' : 'light';
    const apply = () => {
      el.dataset.theme = next;
      this.setState({ theme: next });
      try { localStorage.setItem('loop-theme', next); } catch (err) {}
    };
    if (document.startViewTransition) {
      const vt = document.startViewTransition(apply);
      if (vt && vt.ready) vt.ready.catch(() => {});
      if (vt && vt.finished) vt.finished.catch(() => {});
    } else apply();
  };

  toggleFollow = async (targetUserID) => {
    targetUserID = Number(targetUserID);
    if (!Number.isInteger(targetUserID) || targetUserID <= 0 || targetUserID === USERS.me.apiId) return;
    const key = String(targetUserID);
    if (this.state.followPendingByID[key]) return;
    const authGeneration = this.authGate.current();
    const user = this.apiUser(targetUserID);
    const status = UserModel.normalizeStatus(user.relationship && user.relationship.status);
    this.setState({
      followPendingByID: Object.assign({}, this.state.followPendingByID, { [key]: true }),
      followErrorByID: Object.assign({}, this.state.followErrorByID, { [key]: '' }),
      appError: ''
    });
    try {
      const response = status === 'none'
        ? await AuthAPI.follow(targetUserID)
        : (await AuthAPI.unfollow(targetUserID), { status: 'none' });
      if (!this.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = this.mergeAPIUsers([{
        id: targetUserID,
        relationship: {
          status: response.status,
          follows_me: user.relationship && user.relationship.follows_me === true
        }
      }]);
      const pending = Object.assign({}, this.state.followPendingByID);
      delete pending[key];
      const inaccessiblePostIDs = status === 'none' ? [] : this.state.posts.concat(this.state.profilePosts)
        .filter(post => Number(post.apiAuthorID) === targetUserID)
        .map(post => post.id);
      this.purgeCommentStates(inaccessiblePostIDs);
      const posts = status === 'none'
        ? this.state.posts
        : this.state.posts.filter(post => Number(post.apiAuthorID) !== targetUserID);
      this.setState({ apiUsersByID, followPendingByID: pending, posts });

      this.loadDirectory();
      this.loadFeed(true);
      if (this.state.screen === 'profile' && Number(this.state.profileId) === targetUserID) this.openProfile(targetUserID);
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      const pending = Object.assign({}, this.state.followPendingByID);
      delete pending[key];
      const message = requestErrorMessage(error, 'Could not update follow status.');
      this.setState({
        followPendingByID: pending,
        followErrorByID: Object.assign({}, this.state.followErrorByID, { [key]: message }),
        appError: message
      });
    }
  };

  likePost = (pid) => {
    this.setState({ posts: this.state.posts.map(p => p.id === pid ? Object.assign({}, p, { liked: !p.liked, likes: p.likes + (p.liked ? -1 : 1) }) : p) });
  };
  addComment = (pid) => {
    const text = (this.state.drafts[pid] || '').trim();
    if (!text) return;
    this.setState({
      posts: this.state.posts.map(p => p.id === pid ? Object.assign({}, p, { comments: p.comments.concat([{ uid: 'me', text, time: 'now' }]) }) : p),
      drafts: Object.assign({}, this.state.drafts, { [pid]: '' })
    });
  };
  pickComposerMedia = () => {
    const input = document.getElementById('post-media');
    if (input) input.click();
  };

  onComposerMedia = (event) => {
    const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
    this.setState({ composerFile: file, composerFileName: file ? file.name : '', composerError: '' });
  };

  removeComposerMedia = () => {
    const input = document.getElementById('post-media');
    if (input) input.value = '';
    this.setState({ composerFile: null, composerFileName: '', composerError: '' });
  };

  sendPost = async () => {
    const s = this.state;
    if (s.composerPending || !s.composerText.trim()) return;
    const authGeneration = this.authGate.current();
    const selectedUserIDs = Object.keys(s.selectedFollowers)
      .filter(id => s.selectedFollowers[id])
      .map(id => Number(id));
    this.setState({ composerPending: true, composerError: '', privacyOpen: false });
    try {
      const form = PostModel.buildCreatePostForm({
        text: s.composerText,
        privacy: s.privacy,
        selectedUserIDs,
        media: s.composerFile
      }, FormData);
      const response = await AuthAPI.createPost(form);
      if (!this.authGate.isCurrent(authGeneration)) return;
      const post = this.mapAPIPost(response);
      const apiUsersByID = this.mergeAPIUsers([response.author]);
      const me = apiUsersByID[String(USERS.me.apiId)];
      if (me) me.postsCount = (me.postsCount || 0) + 1;
      const input = document.getElementById('post-media');
      if (input) input.value = '';
      this.setState({
        apiUsersByID,
        posts: [post].concat(this.state.posts.filter(item => item.id !== post.id)),
        profilePosts: Number(this.state.profileId) === USERS.me.apiId
          ? [post].concat(this.state.profilePosts.filter(item => item.id !== post.id))
          : this.state.profilePosts,
        composerText: '', composerFile: null, composerFileName: '',
        composerError: '', composerPending: false,
        privacy: 'public', privacyOpen: false, selectedFollowers: {}
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.setState({
        composerPending: false,
        composerError: requestErrorMessage(error, 'Could not create the post. Your draft was kept.')
      });
    }
  };

  groupGeneration(groupID) {
    const key = String(Number(groupID));
    if (!this.groupGenerationsByID[key]) this.groupGenerationsByID[key] = UserModel.createRequestGate();
    return this.groupGenerationsByID[key];
  }

  mapAPIGroup(raw) {
    const id = Number(raw && raw.id);
    return {
      id,
      name: raw && typeof raw.title === 'string' ? raw.title : '',
      desc: raw && typeof raw.description === 'string' ? raw.description : '',
      members: Math.max(0, Number(raw && raw.members_count) || 0),
      state: raw && ['none', 'requested', 'invited', 'member', 'owner'].indexOf(raw.viewer_status) >= 0
        ? raw.viewer_status : 'none',
      ownerID: Number(raw && raw.owner && raw.owner.id) || 0,
      createdAt: raw && raw.created_at ? String(raw.created_at) : '',
      color: GROUP_COLORS[Math.abs(id || 0) % GROUP_COLORS.length]
    };
  }

  mergeGroupResponses(rawGroups, baseGroups) {
    const groups = Object.assign({}, baseGroups || this.state.apiGroupsByID);
    (rawGroups || []).forEach(raw => {
      const mapped = this.mapAPIGroup(raw);
      if (Number.isInteger(mapped.id) && mapped.id > 0) groups[String(mapped.id)] = mapped;
    });
    return groups;
  }

  loadGroups = async (reset = true) => {
    const authGeneration = this.authGate.current();
    const generation = reset ? this.groupsDirectoryGate.begin() : this.groupsDirectoryGate.current();
    if (!reset && this.state.groupsPending) return;
    const cursor = reset ? null : this.state.groupsNextCursor;
    if (!reset && !cursor) return;
    this.setState({ groupsPending: true, groupsLoading: !!reset, groupsError: '' });
    try {
      const page = await AuthAPI.groups(cursor, 20);
      if (!this.authGate.isCurrent(authGeneration) || !this.groupsDirectoryGate.isCurrent(generation)) return;
      const rawGroups = page.groups || [];
      const incomingIDs = rawGroups.map(group => Number(group.id));
      const apiUsersByID = this.mergeAPIUsers(rawGroups.map(group => group.owner));
      this.setState(current => ({
        apiUsersByID,
        apiGroupsByID: this.mergeGroupResponses(rawGroups, current.apiGroupsByID),
        groupIDs: reset
          ? incomingIDs
          : current.groupIDs.concat(incomingIDs.filter(id => current.groupIDs.indexOf(id) < 0)),
        groupsNextCursor: page.next_cursor || null,
        groupsPending: false, groupsLoading: false, groupsError: ''
      }));
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !this.groupsDirectoryGate.isCurrent(generation)) return;
      this.setState({
        groupsPending: false, groupsLoading: false,
        groupsError: requestErrorMessage(error, 'Could not load groups. Please try again.')
      });
    }
  };

  loadGroupInvitationInbox = async (reset = true) => {
    const authGeneration = this.authGate.current();
    const generation = reset ? this.groupInvitationInboxGate.begin() : this.groupInvitationInboxGate.current();
    if (!reset && this.state.groupInvitationInboxLoading) return;
    const cursor = reset ? null : this.state.groupInvitationInboxNextCursor;
    if (!reset && !cursor) return;
    this.setState({ groupInvitationInboxLoading: true, groupInvitationInboxError: '' });
    try {
      const page = await AuthAPI.groupInvitationInbox(cursor, 20);
      if (!this.authGate.isCurrent(authGeneration) || !this.groupInvitationInboxGate.isCurrent(generation)) return;
      const rawInvitations = page.invitations || [];
      const rawGroups = rawInvitations.map(item => item.group);
      const apiUsersByID = this.mergeAPIUsers(rawGroups.map(group => group.owner));
      const mapped = rawInvitations.map(item => ({ group: this.mapAPIGroup(item.group), createdAt: item.created_at }));
      this.setState(current => ({
        apiUsersByID,
        apiGroupsByID: this.mergeGroupResponses(rawGroups, current.apiGroupsByID),
        groupInvitationInbox: reset ? mapped : current.groupInvitationInbox.concat(mapped),
        groupInvitationInboxNextCursor: page.next_cursor || null,
        groupInvitationInboxLoading: false, groupInvitationInboxError: ''
      }));
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !this.groupInvitationInboxGate.isCurrent(generation)) return;
      this.setState({
        groupInvitationInboxLoading: false,
        groupInvitationInboxError: requestErrorMessage(error, 'Could not load group invitations.')
      });
    }
  };

  openGroup = (groupID) => {
    groupID = Number(groupID);
    if (!Number.isInteger(groupID) || groupID <= 0) return;
    this.groupDetailGate.begin();
    this.groupMembersGate.begin();
    this.groupRequestsGate.begin();
    this.groupInvitationsGate.begin();
    this.setState({
      screen: 'group', groupId: groupID, groupTab: 'posts', inviteOpen: false,
      groupLoading: true, groupError: '', groupMembers: [], groupMembersNextCursor: null,
      groupMembersLoading: true, groupMembersError: '', groupRequests: [], groupRequestsNextCursor: null,
      groupRequestsLoading: false, groupRequestsError: '', groupInvitations: [], groupInvitationsNextCursor: null,
      groupInvitationsLoading: false, groupInvitationsError: '', groupInviteUserID: ''
    });
    this.loadGroupDetail(groupID);
    this.loadGroupMembers(groupID, true);
  };

  loadGroupDetail = async (groupID) => {
    groupID = Number(groupID);
    const authGeneration = this.authGate.current();
    const accessGate = this.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const generation = this.groupDetailGate.begin();
    this.setState({ groupLoading: true, groupError: '' });
    try {
      const raw = await AuthAPI.group(groupID);
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !this.groupDetailGate.isCurrent(generation) || Number(this.state.groupId) !== groupID
      ) return;
      const apiUsersByID = this.mergeAPIUsers([raw.owner]);
      const mapped = this.mapAPIGroup(raw);
      this.setState(current => ({
        apiUsersByID,
        apiGroupsByID: Object.assign({}, current.apiGroupsByID, { [String(groupID)]: mapped }),
        groupLoading: false, groupError: ''
      }));
      if (mapped.state === 'owner') {
        this.loadGroupRequests(groupID, true);
        this.loadGroupInvitations(groupID, true);
      }
    } catch (error) {
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !this.groupDetailGate.isCurrent(generation) || Number(this.state.groupId) !== groupID
      ) return;
      this.setState({ groupLoading: false, groupError: requestErrorMessage(error, 'Could not load this group.') });
    }
  };

  loadGroupMembers = async (groupID, reset = true) => {
    groupID = Number(groupID);
    const authGeneration = this.authGate.current();
    const accessGate = this.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const generation = reset ? this.groupMembersGate.begin() : this.groupMembersGate.current();
    if (!reset && this.state.groupMembersLoading) return;
    const cursor = reset ? null : this.state.groupMembersNextCursor;
    if (!reset && !cursor) return;
    this.setState({ groupMembersLoading: true, groupMembersError: '' });
    try {
      const page = await AuthAPI.groupMembers(groupID, cursor, 20);
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !this.groupMembersGate.isCurrent(generation) || Number(this.state.groupId) !== groupID
      ) return;
      const rawMembers = page.members || [];
      const apiUsersByID = this.mergeAPIUsers(rawMembers.map(member => member.user));
      const mapped = rawMembers.map(member => ({
        userID: Number(member.user.id), status: member.status, createdAt: member.created_at
      }));
      this.setState(current => ({
        apiUsersByID,
        groupMembers: reset ? mapped : current.groupMembers.concat(mapped),
        groupMembersNextCursor: page.next_cursor || null,
        groupMembersLoading: false, groupMembersError: ''
      }));
    } catch (error) {
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !this.groupMembersGate.isCurrent(generation) || Number(this.state.groupId) !== groupID
      ) return;
      this.setState({ groupMembersLoading: false, groupMembersError: requestErrorMessage(error, 'Could not load members.') });
    }
  };

  loadGroupRequests = async (groupID, reset = true) => {
    return this.loadGroupOwnerList(groupID, reset, 'requests');
  };

  loadGroupInvitations = async (groupID, reset = true) => {
    return this.loadGroupOwnerList(groupID, reset, 'invitations');
  };

  loadGroupOwnerList = async (groupID, reset, kind) => {
    groupID = Number(groupID);
    const isRequests = kind === 'requests';
    const gate = isRequests ? this.groupRequestsGate : this.groupInvitationsGate;
    const stateKey = isRequests ? 'groupRequests' : 'groupInvitations';
    const cursorKey = stateKey + 'NextCursor';
    const loadingKey = stateKey + 'Loading';
    const errorKey = stateKey + 'Error';
    const authGeneration = this.authGate.current();
    const accessGate = this.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const generation = reset ? gate.begin() : gate.current();
    if (!reset && this.state[loadingKey]) return;
    const cursor = reset ? null : this.state[cursorKey];
    if (!reset && !cursor) return;
    this.setState({ [loadingKey]: true, [errorKey]: '' });
    try {
      const page = isRequests
        ? await AuthAPI.groupJoinRequests(groupID, cursor, 20)
        : await AuthAPI.groupInvitations(groupID, cursor, 20);
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !gate.isCurrent(generation) || Number(this.state.groupId) !== groupID
      ) return;
      const rawItems = page[stateKey === 'groupRequests' ? 'requests' : 'invitations'] || [];
      const apiUsersByID = this.mergeAPIUsers(rawItems.map(item => item.user));
      const mapped = rawItems.map(item => ({ userID: Number(item.user.id), createdAt: item.created_at }));
      this.setState(current => ({
        apiUsersByID,
        [stateKey]: reset ? mapped : current[stateKey].concat(mapped),
        [cursorKey]: page.next_cursor || null,
        [loadingKey]: false, [errorKey]: ''
      }));
    } catch (error) {
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !gate.isCurrent(generation) || Number(this.state.groupId) !== groupID
      ) return;
      this.setState({ [loadingKey]: false, [errorKey]: requestErrorMessage(error, 'Could not load owner controls.') });
    }
  };

  applyAuthoritativeGroup(raw, invalidateInbox) {
    const group = this.mapAPIGroup(raw);
    const key = String(group.id);
    this.groupsDirectoryGate.begin();
    this.groupGeneration(group.id).begin();
    if (invalidateInbox) this.groupInvitationInboxGate.begin();
    const apiUsersByID = this.mergeAPIUsers([raw.owner]);
    this.setState(current => ({
      apiUsersByID,
      apiGroupsByID: Object.assign({}, current.apiGroupsByID, { [key]: group }),
      groupIDs: current.groupIDs.indexOf(group.id) >= 0 ? current.groupIDs : [group.id].concat(current.groupIDs),
      groupsPending: false, groupsLoading: false,
      groupInvitationInbox: invalidateInbox
        ? current.groupInvitationInbox.filter(item => Number(item.group.id) !== group.id)
        : current.groupInvitationInbox,
      groupMutationPendingByID: Object.assign({}, current.groupMutationPendingByID, { [key]: false }),
      groupMutationErrorByID: Object.assign({}, current.groupMutationErrorByID, { [key]: '' }),
      groupRequests: Number(current.groupId) === group.id && group.state !== 'owner' ? [] : current.groupRequests,
      groupInvitations: Number(current.groupId) === group.id && group.state !== 'owner' ? [] : current.groupInvitations,
      groupLoading: Number(current.groupId) === group.id ? false : current.groupLoading,
      groupError: Number(current.groupId) === group.id ? '' : current.groupError
    }));
    return group;
  }

  runGroupMutation = async (groupID, operation, options) => {
    groupID = Number(groupID);
    const key = String(groupID);
    if (!Number.isInteger(groupID) || groupID <= 0 || this.state.groupMutationPendingByID[key]) return;
    const authGeneration = this.authGate.current();
    const accessGate = this.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    this.setState({
      groupMutationPendingByID: Object.assign({}, this.state.groupMutationPendingByID, { [key]: true }),
      groupMutationErrorByID: Object.assign({}, this.state.groupMutationErrorByID, { [key]: '' })
    });
    try {
      const raw = await operation();
      if (!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration)) return;
      const group = this.applyAuthoritativeGroup(raw, options && options.invalidateInbox);
      if (Number(this.state.groupId) === groupID) {
        this.loadGroupMembers(groupID, true);
        if (group.state === 'owner') {
          this.loadGroupRequests(groupID, true);
          this.loadGroupInvitations(groupID, true);
        }
      }
      if (options && options.invalidateInbox) this.loadGroupInvitationInbox(true);
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration)) return;
      this.setState({
        groupMutationPendingByID: Object.assign({}, this.state.groupMutationPendingByID, { [key]: false }),
        groupMutationErrorByID: Object.assign({}, this.state.groupMutationErrorByID, {
          [key]: requestErrorMessage(error, 'Could not update group membership.')
        })
      });
    }
  };

  requestGroupJoin = (groupID) => {
    const group = this.state.apiGroupsByID[String(Number(groupID))];
    if (!group) return;
    return this.runGroupMutation(group.id, () => group.state === 'requested'
      ? AuthAPI.cancelGroupJoin(group.id)
      : AuthAPI.requestGroupJoin(group.id));
  };

  acceptGroupInvitation = (groupID) => this.runGroupMutation(
    groupID, () => AuthAPI.acceptGroupInvitation(groupID), { invalidateInbox: true }
  );

  declineGroupInvitation = (groupID) => this.runGroupMutation(
    groupID, () => AuthAPI.declineGroupInvitation(groupID), { invalidateInbox: true }
  );

  leaveGroup = (groupID) => this.runGroupMutation(groupID, () => AuthAPI.leaveGroup(groupID));

  acceptGroupRequest = (groupID, userID) => this.runGroupMutation(
    groupID, () => AuthAPI.acceptGroupJoinRequest(groupID, userID)
  );

  rejectGroupRequest = (groupID, userID) => this.runGroupMutation(
    groupID, () => AuthAPI.rejectGroupJoinRequest(groupID, userID)
  );

  createGroup = async () => {
    const title = this.state.ngName.trim();
    const description = this.state.ngDesc.trim();
    if (!title || !description || this.state.groupCreatePending) return;
    const authGeneration = this.authGate.current();
    this.setState({ groupCreatePending: true, groupCreateError: '' });
    try {
      const raw = await AuthAPI.createGroup(title, description);
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.applyAuthoritativeGroup(raw, false);
      this.setState({
        groupCreatePending: false, groupCreateError: '', createOpen: false, ngName: '', ngDesc: ''
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.setState({
        groupCreatePending: false,
        groupCreateError: requestErrorMessage(error, 'Could not create the group. Your draft was kept.')
      });
    }
  };

  inviteSelectedUser = () => {
    const groupID = Number(this.state.groupId);
    const userID = Number(this.state.groupInviteUserID);
    if (!Number.isInteger(groupID) || !Number.isInteger(userID) || userID <= 0) return;
    const mutation = this.runGroupMutation(groupID, () => AuthAPI.inviteToGroup(groupID, userID));
    if (!mutation || typeof mutation.then !== 'function') return mutation;
    return mutation.then(() => {
      if (!this.state.groupMutationErrorByID[String(groupID)]) this.setState({ groupInviteUserID: '' });
    });
  };

  openConvo = (id) => {
    this.setState({ convoId: id, emojiOpen: false, convos: this.state.convos.map(c => c.id === id ? Object.assign({}, c, { unread: 0 }) : c) });
  };
  sendMsg = () => {
    const text = this.state.chatDraft.trim();
    if (!text) return;
    const id = this.state.convoId;
    const now = new Date();
    const hh = String(now.getHours()).padStart(2, '0') + ':' + String(now.getMinutes()).padStart(2, '0');
    this.setState({
      chatDraft: '', emojiOpen: false,
      convos: this.state.convos.map(c => c.id === id ? Object.assign({}, c, { messages: c.messages.concat([{ from: 'me', text, time: hh }]) }) : c)
    });
    const convo = this.state.convos.find(c => c.id === id);
    if (convo && convo.kind === 'dm') {
      setTimeout(() => this.setState({ convos: this.state.convos.map(c => c.id === id ? Object.assign({}, c, { typing: true }) : c) }), 700);
      setTimeout(() => {
        const reply = REPLIES[Math.floor(Math.random() * REPLIES.length)];
        this.setState({ convos: this.state.convos.map(c => c.id === id ? Object.assign({}, c, { typing: false, messages: c.messages.concat([{ from: convo.uid, text: reply, time: hh }]) }) : c) });
      }, 2200);
    }
  };

  acceptFollowRequest = async (requestID) => {
    const key = String(requestID);
    if (this.state.followRequestPendingByID[key]) return;
    const request = this.state.followRequests.find(item => String(item.id) === key);
    if (!request) return;
    const authGeneration = this.authGate.current();
    this.setState({
      followRequestPendingByID: Object.assign({}, this.state.followRequestPendingByID, { [key]: true }),
      followRequestsError: ''
    });
    try {
      await AuthAPI.acceptFollowRequest(requestID);
      if (!this.authGate.isCurrent(authGeneration)) return;
      const user = this.apiUser(request.user.id);
      const apiUsersByID = this.mergeAPIUsers([{
        id: request.user.id,
        relationship: {
          status: user.relationship.status,
          follows_me: true
        }
      }]);
      const pending = Object.assign({}, this.state.followRequestPendingByID);
      delete pending[key];
      this.setState({
        apiUsersByID,
        followRequests: this.state.followRequests.filter(item => String(item.id) !== key),
        followRequestPendingByID: pending
      });
      this.loadPostFollowers();
      this.loadDirectory();
      this.loadFeed(true);
      if (this.state.screen === 'profile' && Number.isInteger(Number(this.state.profileId))) {
        this.openProfile(Number(this.state.profileId));
      } else {
        this.profileGate.begin();
      }
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      const pending = Object.assign({}, this.state.followRequestPendingByID);
      delete pending[key];
      this.setState({
        followRequestPendingByID: pending,
        followRequestsError: requestErrorMessage(error, 'Could not accept follow request.')
      });
    }
  };

  rejectFollowRequest = async (requestID) => {
    const key = String(requestID);
    if (this.state.followRequestPendingByID[key]) return;
    if (!this.state.followRequests.some(item => String(item.id) === key)) return;
    const authGeneration = this.authGate.current();
    this.setState({
      followRequestPendingByID: Object.assign({}, this.state.followRequestPendingByID, { [key]: true }),
      followRequestsError: ''
    });
    try {
      await AuthAPI.rejectFollowRequest(requestID);
      if (!this.authGate.isCurrent(authGeneration)) return;
      const pending = Object.assign({}, this.state.followRequestPendingByID);
      delete pending[key];
      this.setState({
        followRequests: this.state.followRequests.filter(item => String(item.id) !== key),
        followRequestPendingByID: pending
      });
      this.loadDirectory();
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      const pending = Object.assign({}, this.state.followRequestPendingByID);
      delete pending[key];
      this.setState({
        followRequestPendingByID: pending,
        followRequestsError: requestErrorMessage(error, 'Could not reject follow request.')
      });
    }
  };

  followBtn(userID) {
    const user = this.apiUser(userID);
    const model = UserModel.followButton(user, this.state.followPendingByID[String(userID)]);
    if (model.tone === 'muted') return { label: model.label, bg: 'var(--surface2)', color: 'var(--text2)', bd: 'transparent', disabled: model.disabled };
    if (model.tone === 'soft') return { label: model.label, bg: 'var(--soft)', color: 'var(--accent)', bd: 'transparent', disabled: model.disabled };
    return { label: model.label, bg: 'var(--accent)', color: '#fff', bd: 'transparent', disabled: model.disabled };
  }

  mapPost(p) {
    const s = this.state;
    const key = p.id;
    const privacyMeta = { public: { icon: IC.globe, label: 'Public' }, followers: { icon: IC.users, label: 'Followers' }, selected: { icon: IC.lock, label: 'Selected' } };
    const pm = privacyMeta[p.privacy] || privacyMeta.public;
    const commentState = p.real ? this.commentState(p.id) : null;
    const comments = p.real ? commentState.comments : (p.comments || []);
    const likes = p.likes || 0;
    const user = p.real ? this.apiUser(p.apiAuthorID) : (p.user || USERS[p.uid] || USERS.me);
    return Object.assign({}, p, {
      user,
      privacyIcon: pm.icon, privacyLabel: pm.label,
      hasImage: !!p.mediaUrl || !!p.image,
      mediaUrl: p.mediaUrl || '',
      showRealMedia: !!p.mediaUrl,
      showMockImage: !p.mediaUrl && !!p.image,
      showMockActions: !p.real,
      notLiked: !p.liked,
      likeColor: p.liked ? 'var(--danger)' : 'var(--text2)',
      commentCount: num(p.real ? (p.commentsCount || 0) : comments.length),
      likes: num(likes),
      showComments: !!s.openComments[key],
      comments: comments.map(c => Object.assign({}, c, {
        user: p.real ? this.apiUser(c.apiAuthorID) : USERS[c.uid],
        time: p.real ? this.formatPostTime(c.createdAt) : c.time
      })),
      draft: p.real ? commentState.draft : (s.drafts[key] || ''),
      commentsLoading: p.real && commentState.loading,
      commentsPending: p.real && commentState.pending,
      commentsHasError: p.real && !!commentState.error,
      commentsError: p.real ? commentState.error : '',
      commentsHasMore: p.real && !!commentState.nextCursor,
      commentCreatePending: p.real && commentState.createPending,
      commentCreateHasError: p.real && !!commentState.createError,
      commentCreateError: p.real ? commentState.createError : '',
      commentSendDisabled: p.real && (commentState.createPending || !commentState.draft.trim()),
      commentSendLabel: p.real && commentState.createPending ? '…' : 'Send',
      onLike: () => this.likePost(p.id),
      onToggleComments: () => p.real
        ? this.togglePostComments(p.id)
        : this.setState({ openComments: Object.assign({}, s.openComments, { [key]: !s.openComments[key] }) }),
      onDraft: (e) => p.real
        ? this.setCommentDraft(p.id, e.target.value)
        : this.setState({ drafts: Object.assign({}, this.state.drafts, { [key]: e.target.value }) }),
      onKey: (e) => {
        if (e.key !== 'Enter') return;
        if (p.real) { e.preventDefault(); this.createComment(p.id); }
      },
      onSendComment: () => { if (p.real) this.createComment(p.id); },
      loadMoreComments: () => this.loadComments(p.id, false),
      retryComments: () => this.loadComments(p.id, true),
      goProfile: () => { if (p.real) this.openProfile(p.apiAuthorID); }
    });
  }

  renderVals() {
    const s = this.state;
    const me = USERS.me;
    const notifUnread = s.notifs.filter(n => !n.read).length + s.followRequests.length;
    const chatUnread = s.convos.reduce((a, c) => a + c.unread, 0);

    const navDefs = [
      { k: 'feed', label: 'Home', icon: IC.home, badge: 0 },
      { k: 'profile', label: 'Profile', icon: IC.user, badge: 0 },
      { k: 'groups', label: 'Groups', icon: IC.users, badge: 0 },
      { k: 'chat', label: 'Messages', icon: IC.chat, badge: chatUnread },
      { k: 'notifications', label: 'Notifications', icon: IC.bell, badge: notifUnread }
    ];
    const activeKey = s.screen === 'group' ? 'groups' : (s.screen === 'profile' && Number(s.profileId) !== me.apiId ? '' : s.screen);
    const navItems = navDefs.map(n => {
      const on = n.k === activeKey && !(n.k === 'profile' && Number(s.profileId) !== me.apiId);
      return {
        icon: n.icon, label: n.label,
        bg: on ? 'var(--soft)' : 'transparent',
        color: on ? 'var(--accent)' : 'var(--text2)',
        w: on ? '800' : '600',
        hasBadge: n.badge > 0, badge: num(n.badge),
        go: () => { if (n.k === 'profile') this.openProfile(me.apiId); else this.go(n.k); }
      };
    });

    // feed
    const feedPosts = s.posts.map((p, i) => Object.assign(this.mapPost(p, false), { delay: (i * 0.06).toFixed(2) + 's' }));
    const privacyMeta = { public: { icon: IC.globe, label: 'Public' }, followers: { icon: IC.users, label: 'Followers' }, selected: { icon: IC.lock, label: 'Selected' } };
    const privacyOptions = [
      { k: 'public', label: 'Public', desc: 'Anyone on loop can see this', icon: IC.globe },
      { k: 'followers', label: 'Followers only', desc: 'People who follow you', icon: IC.users },
      { k: 'selected', label: 'Selected followers', desc: 'Choose exactly who sees it', icon: IC.lock }
    ].map(o => ({
      label: o.label, desc: o.desc, icon: o.icon,
      isOn: s.privacy === o.k,
      bg: s.privacy === o.k ? 'var(--soft)' : 'transparent',
      pick: () => {
        this.setState({ privacy: o.k, privacyOpen: false });
        if (o.k === 'selected') this.loadPostFollowers();
      }
    }));
    const followerChips = s.postFollowers.map(u => {
      const uid = String(u.apiId);
      const on = !!s.selectedFollowers[uid];
      return {
        name: u.name.split(' ')[0], initials: u.initials, color: u.color,
        bg: on ? 'var(--soft)' : 'transparent',
        bd: on ? 'var(--accent)' : 'var(--border)',
        tc: on ? 'var(--accent)' : 'var(--text2)',
        toggle: () => this.setState({ selectedFollowers: Object.assign({}, s.selectedFollowers, { [uid]: !on }) })
      };
    });
    const composerAudienceReady = s.privacy !== 'selected' || Object.keys(s.selectedFollowers).some(id => s.selectedFollowers[id]);

    // profile
    const pUser = this.apiUser(s.profileId || me.apiId);
    const pIsMe = Number(s.profileId) === me.apiId;
    const pCanView = s.profileReady && pUser.canViewProfile !== false;
    const pPostsRaw = s.profilePosts;
    const followerIds = s.profileFollowers;
    const followingIds = s.profileFollowing;
    const mkUserRow = (userID) => {
      const u = this.apiUser(userID);
      const b = this.followBtn(userID);
      return {
        user: u, showBtn: Number(userID) !== me.apiId,
        btnLabel: b.label, btnBg: b.bg, btnColor: b.color, btnBd: b.bd,
        btnDisabled: b.disabled,
        onBtn: () => this.toggleFollow(userID),
        goProfile: () => this.openProfile(userID)
      };
    };
    const pTabs = [
      { k: 'posts', label: 'Posts' },
      { k: 'followers', label: 'Followers · ' + (pUser.followersCount || 0) },
      { k: 'following', label: 'Following · ' + (pUser.followingCount || 0) }
    ].map(t => ({
      label: t.label,
      color: s.profileTab === t.k ? 'var(--text)' : 'var(--text3)',
      bd: s.profileTab === t.k ? 'var(--accent)' : 'transparent',
      pick: () => this.setState({ profileTab: t.k })
    }));
    const fb = this.followBtn(s.profileId || 0);
    const privacySeg = [
      { k: 'public', label: 'Public', icon: IC.globe },
      { k: 'private', label: 'Private', icon: IC.lock }
    ].map(o => ({
      label: o.label, icon: o.icon,
      bg: s.myPrivacy === o.k ? 'var(--surface)' : 'transparent',
      color: s.myPrivacy === o.k ? 'var(--text)' : 'var(--text3)',
      disabled: s.profilePrivacyPending || s.profileEditPending || s.profileAvatarPending,
      opacity: s.profilePrivacyPending || s.profileEditPending || s.profileAvatarPending ? '0.6' : '1',
      cursor: s.profilePrivacyPending ? 'wait' : (s.profileEditPending || s.profileAvatarPending ? 'not-allowed' : 'pointer'),
      pick: () => this.setProfilePrivacy(o.k)
    }));

    // groups
    const groupCards = s.groupIDs.map(groupID => s.apiGroupsByID[String(groupID)]).filter(Boolean).map((g, i) => {
      const pending = !!s.groupMutationPendingByID[String(g.id)];
      return {
        name: g.name, desc: g.desc, membersLabel: num(g.members), cover: cover(g.color),
        owner: this.apiUser(g.ownerID),
        delay: (i * 0.05).toFixed(2) + 's', pending,
        error: s.groupMutationErrorByID[String(g.id)] || '', hasError: !!s.groupMutationErrorByID[String(g.id)],
        isJoined: g.state === 'member' || g.state === 'owner', isOwner: g.state === 'owner',
        isMember: g.state === 'member', isNone: g.state === 'none',
        isRequested: g.state === 'requested', isInvited: g.state === 'invited',
        open: () => this.openGroup(g.id),
        join: () => this.requestGroupJoin(g.id),
        leave: () => this.leaveGroup(g.id),
        acceptInvite: () => this.acceptGroupInvitation(g.id),
        declineInvite: () => this.declineGroupInvitation(g.id)
      };
    });
    const groupInboxCards = s.groupInvitationInbox.map(item => {
      const g = s.apiGroupsByID[String(item.group.id)] || item.group;
      const pending = !!s.groupMutationPendingByID[String(g.id)];
      return {
        name: g.name, owner: this.apiUser(g.ownerID), pending,
        accept: () => this.acceptGroupInvitation(g.id),
        decline: () => this.declineGroupInvitation(g.id),
        open: () => this.openGroup(g.id)
      };
    });

    const g = s.apiGroupsByID[String(Number(s.groupId))] || {
      id: Number(s.groupId) || 0, name: '', desc: '', members: 0, state: 'none', ownerID: 0, color: GROUP_COLORS[0]
    };
    const gIsOwner = g.state === 'owner';
    const gMutationPending = !!s.groupMutationPendingByID[String(g.id)];
    const gMutationError = s.groupMutationErrorByID[String(g.id)] || '';
    const gTabs = [
      { k: 'posts', label: 'Posts' },
      { k: 'events', label: 'Events' },
      { k: 'members', label: 'Members' }
    ].map(t => ({
      label: t.label,
      color: s.groupTab === t.k ? 'var(--text)' : 'var(--text3)',
      bd: s.groupTab === t.k ? 'var(--accent)' : 'transparent',
      pick: () => this.setState({ groupTab: t.k })
    }));
    const gMembers = s.groupMembers.map(member => ({
      user: this.apiUser(member.userID), isOwner: member.status === 'owner',
      goProfile: () => this.openProfile(member.userID)
    }));
    const gRequests = (gIsOwner ? s.groupRequests : []).map(request => ({
      user: this.apiUser(request.userID), disabled: gMutationPending,
      pending: true, done: false, doneLabel: '',
      accept: () => this.acceptGroupRequest(g.id, request.userID),
      decline: () => this.rejectGroupRequest(g.id, request.userID)
    }));
    const gInvitations = (gIsOwner ? s.groupInvitations : []).map(invitation => ({
      user: this.apiUser(invitation.userID)
    }));
    const excludedInviteIDs = {};
    s.groupMembers.forEach(item => { excludedInviteIDs[String(item.userID)] = true; });
    s.groupRequests.forEach(item => { excludedInviteIDs[String(item.userID)] = true; });
    s.groupInvitations.forEach(item => { excludedInviteIDs[String(item.userID)] = true; });
    if (me.apiId) excludedInviteIDs[String(me.apiId)] = true;
    const inviteCandidatesReady = !s.groupMembersLoading && !s.groupRequestsLoading && !s.groupInvitationsLoading;
    const inviteCandidates = (inviteCandidatesReady ? s.directoryUserIDs : []).filter(id => !excludedInviteIDs[String(id)]).map(id => {
      const user = this.apiUser(id);
      return {
        user,
        selected: String(s.groupInviteUserID) === String(id),
        label: user.name,
        initials: user.initials, color: user.color,
        bg: String(s.groupInviteUserID) === String(id) ? 'var(--soft)' : 'transparent',
        bd: String(s.groupInviteUserID) === String(id) ? 'var(--accent)' : 'var(--border)',
        tc: String(s.groupInviteUserID) === String(id) ? 'var(--accent)' : 'var(--text2)',
        pick: () => this.setState({ groupInviteUserID: String(id) })
      };
    });

    // chat
    const convoMeta = (c) => {
      if (c.kind === 'dm') { const u = USERS[c.uid]; return { title: u.name, initials: u.initials, color: u.color, sub: c.online ? 'Online now' : 'Active recently' }; }
      return { title: 'Group chat', initials: 'GC', color: GROUP_COLORS[0], sub: 'Not implemented' };
    };
    const convos = s.convos.map(c => {
      const m = convoMeta(c);
      const last = c.messages[c.messages.length - 1];
      return {
        title: m.title, initials: m.initials, color: m.color,
        preview: (last.from === 'me' ? 'You: ' : '') + last.text,
        previewColor: c.unread > 0 ? 'var(--text)' : 'var(--text3)',
        previewW: c.unread > 0 ? '700' : '500',
        time: last.time,
        online: !!c.online,
        hasUnread: c.unread > 0, unread: num(c.unread),
        bg: c.id === s.convoId ? 'var(--soft)' : 'transparent',
        open: () => this.openConvo(c.id)
      };
    });
    const active = s.convos.find(c => c.id === s.convoId) || s.convos[0];
    const am = convoMeta(active);
    const messages = active.messages.map((msg, i) => {
      const prev = active.messages[i - 1];
      return {
        text: msg.text, time: msg.time,
        mine: msg.from === 'me', theirs: msg.from !== 'me',
        user: USERS[msg.from] || USERS.me,
        showName: active.kind === 'group' && msg.from !== 'me' && (!prev || prev.from !== msg.from)
      };
    });
    const emojis = EMOJIS.map(ch => ({ ch, add: () => this.setState({ chatDraft: s.chatDraft + ch }) }));

    // notifications
    const followRequestItems = s.followRequests.map((request, i) => {
      const pending = !!s.followRequestPendingByID[String(request.id)];
      return {
        user: this.apiUser(request.user.id), icon: IC.user, text: 'requested to follow you',
        time: this.formatPostTime(request.created_at), delay: (i * 0.06).toFixed(2) + 's',
        bg: 'color-mix(in oklab, var(--accent) 5%, var(--surface))', unreadDot: true,
        pending: true, done: false, doneLabel: '', disabled: pending,
        accept: () => this.acceptFollowRequest(request.id),
        decline: () => this.rejectFollowRequest(request.id),
        goProfile: () => this.openProfile(request.user.id)
      };
    });
    const notifItems = followRequestItems;

    // right rail
    const suggestions = s.directoryUserIDs.map(userID => this.apiUser(userID))
      .filter(user => !user.relationship || user.relationship.status !== 'accepted')
      .map(user => {
        const b = this.followBtn(user.apiId);
        return {
          user, isPrivate: user.private,
          btnLabel: b.label, btnBg: b.bg, btnColor: b.color, btnBd: b.bd, btnDisabled: b.disabled,
          onBtn: () => this.toggleFollow(user.apiId), goProfile: () => this.openProfile(user.apiId)
        };
      });
    const railEvents = [];

    const authTabs = [
      { k: 'login', label: 'Sign in' },
      { k: 'register', label: 'Create account' }
    ].map(t => ({
      label: t.label,
      bg: s.authMode === t.k ? 'var(--surface)' : 'transparent',
      color: s.authMode === t.k ? 'var(--text)' : 'var(--text3)',
      sh: s.authMode === t.k ? 'var(--shadow)' : 'none',
      pick: () => this.setAuthMode(t.k)
    }));

    return {
      // shell
      isAuthChecking: s.authStatus === 'checking', isAuthStartupError: s.authStatus === 'error',
      isAuth: s.authStatus === 'anonymous', isApp: s.authStatus === 'authenticated',
      isFeed: s.screen === 'feed',
      isProfile: s.screen === 'profile' && s.profileReady,
      isProfileLoading: s.screen === 'profile' && s.profileLoading,
      isProfileError: s.screen === 'profile' && !s.profileLoading && !s.profileReady && !!s.profileError,
      profileError: s.profileError,
      retryProfile: () => this.openProfile(s.profileId),
      isGroups: s.screen === 'groups',
      isGroup: s.screen === 'group', isChat: s.screen === 'chat', isNotifs: s.screen === 'notifications',
      rightRail: ['feed', 'profile', 'groups', 'notifications'].indexOf(s.screen) >= 0,
      navItems, me,
      themeIcon: s.theme === 'light' ? IC.moon : IC.sun,
      themeLabel: s.theme === 'light' ? 'Dark mode' : 'Light mode',
      toggleTheme: this.toggleTheme,
      goHome: () => this.go('feed'),
      goMyProfile: () => this.openProfile(me.apiId),
      goLogout: this.logout,
      logoutDisabled: s.logoutPending,
      appHasError: !!s.appError, appError: s.appError,
      // auth
      authTabs, authIsLogin: s.authMode === 'login', authIsReg: s.authMode === 'register',
      authCta: s.authPending ? 'Please wait…' : (s.authMode === 'login' ? 'Sign in' : 'Create account'),
      authDisabled: s.authPending,
      authButtonOpacity: s.authPending ? '0.65' : '1',
      authButtonCursor: s.authPending ? 'wait' : 'pointer',
      authHasError: !!s.authError, authError: s.authError,
      bootstrapError: s.bootstrapError, retryAuthBootstrap: this.loadCurrentUser,
      authEmail: s.authEmail, onAuthEmail: (e) => this.setState({ authEmail: e.target.value }),
      authPassword: s.authPassword, onAuthPassword: (e) => this.setState({ authPassword: e.target.value }),
      regFirstName: s.regFirstName, onRegFirstName: (e) => this.setState({ regFirstName: e.target.value }),
      regLastName: s.regLastName, onRegLastName: (e) => this.setState({ regLastName: e.target.value }),
      regDateOfBirth: s.regDateOfBirth, onRegDateOfBirth: (e) => this.setState({ regDateOfBirth: e.target.value }),
      regGender: s.regGender, onRegGender: (e) => this.setState({ regGender: e.target.value }),
      regNickname: s.regNickname, onRegNickname: (e) => this.setState({ regNickname: e.target.value }),
      regAboutMe: s.regAboutMe, onRegAboutMe: (e) => this.setState({ regAboutMe: e.target.value }),
      avatarButtonLabel: s.regAvatarName || 'avatar',
      pickRegistrationAvatar: this.pickRegistrationAvatar,
      onRegistrationAvatar: this.onRegistrationAvatar,
      submitAuth: this.submitAuth,
      // feed
      feedLoading: s.feedLoading, feedReady: !s.feedLoading,
      feedHasError: !!s.feedError, feedError: s.feedError,
      retryFeed: () => this.loadFeed(true),
      feedHasMore: !!s.feedNextCursor,
      feedLoadMore: () => this.loadFeed(false),
      feedLoadMoreLabel: s.feedPending && !s.feedLoading ? 'Loading…' : 'Load more',
      feedLoadMoreDisabled: s.feedPending,
      posts: feedPosts,
      composerText: s.composerText,
      onComposer: (e) => this.setState({ composerText: e.target.value, composerError: '' }),
      composerHasMedia: !!s.composerFile,
      composerMediaName: s.composerFileName,
      pickComposerMedia: this.pickComposerMedia,
      onComposerMedia: this.onComposerMedia,
      removeComposerMedia: this.removeComposerMedia,
      composerHasError: !!s.composerError, composerError: s.composerError,
      privacyOpen: s.privacyOpen,
      togglePrivacy: () => this.setState({ privacyOpen: !s.privacyOpen }),
      privacyIcon: privacyMeta[s.privacy].icon,
      privacyLabel: privacyMeta[s.privacy].label,
      privacyOptions,
      privacyIsSelected: s.privacy === 'selected',
      followerChips,
      selectedFollowersEmpty: s.postFollowers.length === 0 && !s.postFollowersLoading,
      postBtnOp: s.composerText.trim() && composerAudienceReady && !s.composerPending ? '1' : '0.45',
      postBtnDisabled: s.composerPending || !s.composerText.trim() || !composerAudienceReady,
      postButtonLabel: s.composerPending ? 'Posting…' : 'Post',
      sendPost: this.sendPost,
      // profile
      pUser, pIsMe, pOther: !pIsMe,
      pCover: cover(pUser.color),
      pShowLock: pUser.private || (pIsMe && s.myPrivacy === 'private'),
      pCanView, pLocked: !pCanView,
      pShowEmail: pIsMe && !!pUser.email,
      pStatPosts: num(pUser.postsCount || 0),
      pStatFollowers: num(pUser.followersCount || 0),
      pStatFollowing: num(pUser.followingCount || 0),
      pTabs,
      pTabPosts: s.profileTab === 'posts', pTabFollowers: s.profileTab === 'followers', pTabFollowing: s.profileTab === 'following',
      pPosts: pPostsRaw.map(p => this.mapPost(p, false)),
      pNoPosts: !s.profilePostsLoading && !s.profilePostsError && pPostsRaw.length === 0,
      pPostsLoading: s.profilePostsLoading,
      pPostsHasError: !!s.profilePostsError,
      pPostsError: s.profilePostsError,
      retryProfilePosts: () => this.loadProfilePosts(s.profileId, true),
      pPostsHasMore: !!s.profilePostsNextCursor,
      loadMoreProfilePosts: () => this.loadProfilePosts(s.profileId, false),
      profileLoadMoreLabel: s.profilePostsPending && !s.profilePostsLoading ? 'Loading…' : 'Load more',
      profileLoadMoreDisabled: s.profilePostsPending,
      pFollowers: followerIds.map(mkUserRow), pFollowing: followingIds.map(mkUserRow),
      pListsLoading: s.profileListsLoading,
      pListsHasError: !!s.profileListsError,
      pListsError: s.profileListsError,
      followLabel: fb.label, followBg: fb.bg, followColor: fb.color, followBd: fb.bd,
      followDisabled: fb.disabled,
      followHasError: !!s.followErrorByID[String(s.profileId)],
      followError: s.followErrorByID[String(s.profileId)] || '',
      onFollow: () => this.toggleFollow(s.profileId),
      msgProfile: () => this.setState({ appError: 'Messages are not connected to backend users yet.' }),
      privacySeg,
      profilePrivacyHasError: pIsMe && !!s.profilePrivacyError,
      profilePrivacyError: s.profilePrivacyError,
      showProfileEdit: pIsMe && s.profileEditOpen,
      openProfileEdit: this.openProfileEdit,
      cancelProfileEdit: this.cancelProfileEdit,
      saveProfile: this.saveProfile,
      profileEditPending: s.profileEditPending || s.profileAvatarPending || s.profilePrivacyPending,
      profileSaveLabel: s.profileEditPending ? 'Saving…' : 'Save changes',
      profileEditHasError: !!s.profileEditError,
      profileEditError: s.profileEditError,
      editFirstName: s.editFirstName, onEditFirstName: (e) => this.setState({ editFirstName: e.target.value }),
      editLastName: s.editLastName, onEditLastName: (e) => this.setState({ editLastName: e.target.value }),
      editDateOfBirth: s.editDateOfBirth, onEditDateOfBirth: (e) => this.setState({ editDateOfBirth: e.target.value }),
      editGender: s.editGender, onEditGender: (e) => this.setState({ editGender: e.target.value }),
      editNickname: s.editNickname, onEditNickname: (e) => this.setState({ editNickname: e.target.value }),
      editAboutMe: s.editAboutMe, onEditAboutMe: (e) => this.setState({ editAboutMe: e.target.value }),
      profileAvatarLabel: s.editAvatarName || 'Choose image',
      profileAvatarPending: s.profileAvatarPending || s.profileEditPending || s.profilePrivacyPending,
      profileAvatarUploadDisabled: s.profileAvatarPending || s.profileEditPending || s.profilePrivacyPending || !s.editAvatar,
      profileAvatarUploadOpacity: s.profileAvatarPending || s.profileEditPending || s.profilePrivacyPending || !s.editAvatar ? '0.55' : '1',
      profileAvatarUploadLabel: s.profileAvatarPending ? 'Working…' : 'Upload',
      profileHasCustomAvatar: me.hasCustomAvatar,
      pickProfileAvatar: this.pickProfileAvatar,
      onProfileAvatar: this.onProfileAvatar,
      replaceProfileAvatar: this.replaceProfileAvatar,
      deleteProfileAvatar: this.deleteProfileAvatar,
      // groups
      createOpen: s.createOpen,
      toggleCreate: () => this.setState({ createOpen: !s.createOpen }),
      ngName: s.ngName, onNgName: (e) => this.setState({ ngName: e.target.value }),
      ngDesc: s.ngDesc, onNgDesc: (e) => this.setState({ ngDesc: e.target.value }),
      createGroup: this.createGroup,
      groupCreatePending: s.groupCreatePending,
      groupCreateHasError: !!s.groupCreateError, groupCreateError: s.groupCreateError,
      groupCards, groupsLoading: s.groupsLoading, groupsReady: !s.groupsLoading,
      groupsHasError: !!s.groupsError, groupsError: s.groupsError,
      retryGroups: () => this.loadGroups(true), groupsHasMore: !!s.groupsNextCursor,
      loadMoreGroups: () => this.loadGroups(false),
      groupsLoadMoreLabel: s.groupsPending && !s.groupsLoading ? 'Loading…' : 'Load more',
      groupInboxCards,
      groupInboxLoading: s.groupInvitationInboxLoading,
      groupInboxHasError: !!s.groupInvitationInboxError,
      groupInboxError: s.groupInvitationInboxError,
      groupInboxHasItems: groupInboxCards.length > 0,
      groupInboxHasMore: !!s.groupInvitationInboxNextCursor,
      loadMoreGroupInbox: () => this.loadGroupInvitationInbox(false),
      // group detail
      groupLoading: s.groupLoading, groupHasError: !!s.groupError, groupError: s.groupError,
      retryGroup: () => this.openGroup(g.id),
      gName: g.name, gDesc: g.desc, gMembersLabel: num(g.members), gCover: cover(g.color), gIsOwner,
      gOwner: this.apiUser(g.ownerID), gMutationPending,
      gMutationHasError: !!gMutationError, gMutationError,
      gIsNone: g.state === 'none', gIsRequested: g.state === 'requested',
      gIsInvited: g.state === 'invited', gIsMember: g.state === 'member',
      gRequestJoin: () => this.requestGroupJoin(g.id),
      gAcceptInvitation: () => this.acceptGroupInvitation(g.id),
      gDeclineInvitation: () => this.declineGroupInvitation(g.id),
      gLeave: () => this.leaveGroup(g.id),
      gBack: () => this.go('groups'),
      gTabs, gTabPosts: s.groupTab === 'posts', gTabEvents: s.groupTab === 'events', gTabMembers: s.groupTab === 'members',
      gMembers, gRequests, gInvitations,
      gHasRequests: gRequests.length > 0,
      gHasInvitations: gInvitations.length > 0,
      groupMembersLoading: s.groupMembersLoading,
      groupMembersHasError: !!s.groupMembersError, groupMembersError: s.groupMembersError,
      groupMembersHasMore: !!s.groupMembersNextCursor,
      loadMoreGroupMembers: () => this.loadGroupMembers(g.id, false),
      groupRequestsLoading: s.groupRequestsLoading,
      groupRequestsHasError: !!s.groupRequestsError, groupRequestsError: s.groupRequestsError,
      groupInvitationsLoading: s.groupInvitationsLoading,
      groupInvitationsHasError: !!s.groupInvitationsError, groupInvitationsError: s.groupInvitationsError,
      inviteOpen: s.inviteOpen && gIsOwner,
      toggleInvite: () => {
        const opening = !s.inviteOpen;
        this.setState({ inviteOpen: opening });
        if (opening && !s.directoryUserIDs.length) this.loadDirectory(true);
      },
      inviteCandidates,
      inviteSendDisabled: gMutationPending || !s.groupInviteUserID,
      inviteSelectedUser: this.inviteSelectedUser,
      inviteLoadMore: () => this.loadDirectory(false),
      inviteHasMore: !!s.directoryNextCursor,
      inviteDirectoryLoading: s.directoryLoading,
      // chat
      convos, messages,
      activeTitle: am.title, activeSub: am.sub, activeInitials: am.initials, activeColor: am.color,
      typing: active.typing,
      chatDraft: s.chatDraft,
      onChatDraft: (e) => this.setState({ chatDraft: e.target.value }),
      onChatKey: (e) => { if (e.key === 'Enter') this.sendMsg(); },
      sendMsg: this.sendMsg,
      emojiOpen: s.emojiOpen,
      toggleEmoji: () => this.setState({ emojiOpen: !s.emojiOpen }),
      emojis,
      msgRef: (el) => { this.msgEl = el; },
      // notifications
      notifItems,
      followRequestsHasError: !!s.followRequestsError,
      followRequestsError: s.followRequestsError,
      markAllRead: () => this.setState({ notifs: s.notifs.map(n => Object.assign({}, n, { read: true })) }),
      // rail
      suggestions, railEvents,
      suggestionsHasError: !!s.directoryError,
      suggestionsError: s.directoryError
    };
  }
}

if (typeof module === 'object' && module.exports) module.exports = { Component };
