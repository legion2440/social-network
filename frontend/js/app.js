
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

function emptyCurrentUser() {
  return decorateUser({
    id: 'me', apiId: 0, name: '', handle: '', initials: '?', color: '#5661d8',
    bio: '', email: '', dob: '', gender: '', private: false
  });
}

const USERS = { me: emptyCurrentUser() };

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
    draft: '', mediaFile: null, mediaFileName: '', mediaPreviewURL: '',
    createPending: false, createError: '', loaded: false
  };
}

function emptyGroupPostState() {
  return {
    groupPosts: [], groupPostsNextCursor: null,
    groupPostsLoading: false, groupPostsPending: false, groupPostsError: '',
    groupPostComposerText: '', groupPostComposerFile: null, groupPostComposerFileName: '',
    groupPostComposerError: '', groupPostComposerPending: false
  };
}

function emptyGroupEventState() {
  return {
    groupEvents: [], groupEventsNextCursor: null,
    groupEventsLoading: false, groupEventsPending: false, groupEventsError: '',
    groupEventComposerOpen: false, groupEventTitle: '', groupEventDescription: '', groupEventStartsAt: '',
    groupEventCreatePending: false, groupEventCreateError: '',
    groupEventResponsePendingByID: {}, groupEventResponseErrorByID: {}
  };
}

function emptyNotificationState() {
  return {
    notifications: [], notificationsNextCursor: null,
    notificationsLoading: false, notificationsPending: false, notificationsError: '',
    notificationUnreadCount: 0, notificationRevision: 0,
    notificationReadPendingByID: {}, notificationReadErrorByID: {},
    notificationActionPendingByID: {}, notificationActionErrorByID: {},
    notificationReadAllPending: false
  };
}

function emptyChatMessages() {
  return { messages: [], nextCursor: null, loading: false, pending: false, error: '', loaded: false };
}

function emptyChatState() {
  return {
    chatsByKey: {}, chatKeys: [], chatsNextCursor: null,
    chatsLoading: false, chatsPending: false, chatsError: '',
    activeChatKey: null, messagesByChatKey: {}, onlineUserIDs: {}, typingByChatKey: {},
    chatUnreadByKey: {}, chatUnreadCount: 0, chatUnreadRevision: 0,
    chatReadPendingByKey: {}, chatReadErrorByKey: {},
    chatReadQueuedThroughByKey: {}, chatReadThroughMessageIDByKey: {},
    wsStatus: 'disconnected', wsReconnectAttempt: 0,
    chatDraft: '', chatError: '', emojiOpen: false
  };
}

function createClientMessageID() {
  if (typeof globalThis !== 'undefined' && globalThis.crypto && typeof globalThis.crypto.randomUUID === 'function') {
    return globalThis.crypto.randomUUID().toLowerCase();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (character) {
    const random = Math.floor(Math.random() * 16);
    return (character === 'x' ? random : ((random & 3) | 8)).toString(16);
  });
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
      openComments: {},
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
	  ...emptyGroupPostState(),
	  ...emptyGroupEventState(),
      ...emptyNotificationState(),
      createOpen: false, ngName: '', ngDesc: '', groupCreatePending: false, groupCreateError: '',
      ...emptyChatState(),
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
	this.groupPostsGate = UserModel.createRequestGate();
    this.groupEventsGate = UserModel.createRequestGate();
    this.groupEventCreateGate = UserModel.createRequestGate();
    this.groupEventResponseGatesByID = {};
    this.notificationsGate = UserModel.createRequestGate();
    this.notificationReadAllGate = UserModel.createRequestGate();
    this.notificationReadGatesByID = {};
    this.notificationActionGatesByID = {};
    this.relationshipGenerationsByID = {};
    this.latestActionableNotificationIDBySourceKey = {};
    this.chatsGate = UserModel.createRequestGate();
    this.activeChatGate = UserModel.createRequestGate();
    this.chatHistoryGatesByKey = {};
    this.chatAccessGatesByKey = {};
    this.chatReadGatesByKey = {};
    this.chatReadInFlightByKey = {};
    this.chatReadSentCandidateByKey = {};
    this.revokedChatKeys = new Set();
    this.revokedGroupAccessIDs = new Set();
    this.ws = null;
    this.wsGeneration = 0;
    this.wsReconnectTimer = null;
    this.wsHasOpened = false;
    this.pendingMessageTimers = {};
    this.typingHeartbeatTimer = null;
    this.typingExpiryTimers = {};
    this.typingChatKey = null;
    this.chatScrollAnchor = null;
    this.scrollChatToBottom = false;
    this.chatSendLock = false;
    this.handleVisibilityChange = () => {
      if (this.documentIsVisible() && this.state && this.state.activeChatKey) {
        this.enqueueChatRead(this.state.activeChatKey);
      }
    };
  }

  componentDidMount() {
    document.documentElement.dataset.theme = this.state.theme;
    this.applyTokens();
    if (document && typeof document.addEventListener === 'function') {
      document.addEventListener('visibilitychange', this.handleVisibilityChange);
    }
    this.loadCurrentUser();
  }
  componentDidUpdate() {
    this.applyTokens();
    if (this.chatScrollAnchor && this.msgEl && this.state.activeChatKey === this.chatScrollAnchor.key) {
      this.msgEl.scrollTop = this.msgEl.scrollHeight - this.chatScrollAnchor.height + this.chatScrollAnchor.top;
      this.chatScrollAnchor = null;
    } else if (this.scrollChatToBottom && this.msgEl) {
      this.msgEl.scrollTop = this.msgEl.scrollHeight;
      this.scrollChatToBottom = false;
    }
  }
  componentWillUnmount() {
    if (typeof document !== 'undefined' && document && typeof document.removeEventListener === 'function') {
      document.removeEventListener('visibilitychange', this.handleVisibilityChange);
    }
    this.disposeAllCommentPreviews();
    this.stopRealtime();
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
      apiAuthorID: normalized.apiAuthorID,
	  groupID: normalized.groupID,
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

  revokeCommentPreview(previewURL) {
    if (
      previewURL &&
      typeof URL !== 'undefined' &&
      URL &&
      typeof URL.revokeObjectURL === 'function'
    ) {
      URL.revokeObjectURL(previewURL);
    }
  }

  disposeAllCommentPreviews() {
    const entries = (this.state && this.state.commentsByPostID) || {};
    Object.keys(entries).forEach(key => this.revokeCommentPreview(entries[key] && entries[key].mediaPreviewURL));
  }

  commentMediaInputID(postID) {
    return 'comment-media-' + String(Number(postID));
  }

  resetCommentMediaInput(postID) {
    if (typeof document === 'undefined' || !document || typeof document.getElementById !== 'function') return;
    const input = document.getElementById(this.commentMediaInputID(postID));
    if (input) input.value = '';
  }

  selectCommentMedia = (postID, event) => {
    const state = this.commentState(postID);
    if (state.createPending) return;
    const file = event && event.target && event.target.files && event.target.files[0];
    if (!file) return;
    this.revokeCommentPreview(state.mediaPreviewURL);
    const previewURL = (
      typeof URL !== 'undefined' &&
      URL &&
      typeof URL.createObjectURL === 'function'
    ) ? URL.createObjectURL(file) : '';
    this.patchCommentState(postID, {
      mediaFile: file,
      mediaFileName: file.name || 'attachment',
      mediaPreviewURL: previewURL,
      createError: ''
    });
  };

  removeCommentMedia = (postID) => {
    const state = this.commentState(postID);
    if (state.createPending) return;
    this.revokeCommentPreview(state.mediaPreviewURL);
    this.resetCommentMediaInput(postID);
    this.patchCommentState(postID, {
      mediaFile: null,
      mediaFileName: '',
      mediaPreviewURL: '',
      createError: ''
    });
  };

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
    Object.keys(removed).forEach(key => {
      const state = this.state.commentsByPostID[key];
      this.revokeCommentPreview(state && state.mediaPreviewURL);
      this.resetCommentMediaInput(key);
    });
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
        const latest = this.commentState(postID);
        this.revokeCommentPreview(latest.mediaPreviewURL);
        this.resetCommentMediaInput(postID);
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
	const countAtCreateStart = this.maxPostCommentsCount(postID, this.state.posts, this.state.profilePosts, this.state.groupPosts);
    this.patchCommentState(postID, { createPending: true, createError: '' });
    try {
      const formData = new FormData();
      formData.append('text', text);
      if (state.mediaFile) formData.append('media', state.mediaFile, state.mediaFileName || state.mediaFile.name || 'attachment');
      const response = await AuthAPI.createComment(postID, formData);
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
		  : post),
		groupPosts: current.groupPosts.map(post => Number(post.id) === postID
		  ? Object.assign({}, post, { commentsCount: Math.max(Number(post.commentsCount) || 0, countAtCreateStart + 1) })
		  : post)
      }));
      this.revokeCommentPreview(latest.mediaPreviewURL);
      this.resetCommentMediaInput(postID);
      this.patchCommentState(postID, {
        comments: CommentModel.mergeComments(latest.comments, [comment]),
        draft: '', mediaFile: null, mediaFileName: '', mediaPreviewURL: '',
        createPending: false, createError: ''
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration)) return;
      if (error && (error.status === 403 || error.status === 404)) {
        accessGate.begin();
        const latest = this.commentState(postID);
        this.revokeCommentPreview(latest.mediaPreviewURL);
        this.resetCommentMediaInput(postID);
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
		const merged = this.mergePostCommentsCounts(mapped, current.posts, current.profilePosts, current.groupPosts);
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

  notificationReadGate(notificationID) {
    const key = String(Number(notificationID));
    if (!this.notificationReadGatesByID[key]) this.notificationReadGatesByID[key] = UserModel.createRequestGate();
    return this.notificationReadGatesByID[key];
  }

  notificationActionGate(notificationID) {
    const key = String(Number(notificationID));
    if (!this.notificationActionGatesByID[key]) this.notificationActionGatesByID[key] = UserModel.createRequestGate();
    return this.notificationActionGatesByID[key];
  }

  relationshipGeneration(userID) {
    const key = String(Number(userID));
    if (!this.relationshipGenerationsByID[key]) this.relationshipGenerationsByID[key] = UserModel.createRequestGate();
    return this.relationshipGenerationsByID[key];
  }

  advanceRelationshipLifecycle(userID) {
    const key = String(Number(userID));
    const generation = this.relationshipGeneration(userID).begin();
    if (this.state.followPendingByID[key]) {
      const pending = Object.assign({}, this.state.followPendingByID);
      delete pending[key];
      this.setState({ followPendingByID: pending });
    }
    return generation;
  }

  beginRelationshipGeneration(userID) {
    const generation = this.advanceRelationshipLifecycle(userID);
    this.directoryGate.begin();
    this.postFollowersGate.begin();
    if (Number(this.state.profileId) === Number(userID)) this.profileGate.begin();
    return generation;
  }

  trackNotificationLifecycles(notifications) {
    (notifications || []).forEach(notification => {
      const sourceKey = NotificationModel.sourceKey(notification);
      if (!sourceKey) return;
      const previousID = this.latestActionableNotificationIDBySourceKey[sourceKey];
      if (Number(previousID) === Number(notification.id)) return;
      if (notification.type === 'follow_request') {
        this.advanceRelationshipLifecycle(notification.actorID);
      } else if (notification.group && notification.group.id) {
        const groupID = Number(notification.group.id);
        this.groupGeneration(groupID).begin();
        if (Number(this.state.groupId) === groupID) {
          this.loadGroupDetail(groupID);
          this.loadGroupMembers(groupID, true);
        }
      }
      this.latestActionableNotificationIDBySourceKey[sourceKey] = notification.id;
    });
  }

  applyNotificationPayload(payload, trackLifecycle) {
    const revision = Number(payload && payload.revision);
    const unreadCount = Number(payload && payload.unread_count);
    if (!Number.isInteger(revision) || revision < 0 || !Number.isInteger(unreadCount) || unreadCount < 0) return false;
    if (revision < Number(this.state.notificationRevision || 0)) return false;
    let notification;
    try { notification = NotificationModel.normalize(payload.notification); } catch (ignore) { return false; }
    if (trackLifecycle) this.trackNotificationLifecycles([notification]);
    this.setState(current => {
      if (revision < Number(current.notificationRevision || 0)) return {};
      return {
        apiUsersByID: this.mergeAPIUsers([notification.actor], current.apiUsersByID),
        notifications: NotificationModel.merge(current.notifications, [notification]),
        notificationUnreadCount: unreadCount,
        notificationRevision: revision
      };
    });
    return true;
  }

  loadNotifications = async (reset = true) => {
    const authGeneration = this.authGate.current();
    const generation = reset ? this.notificationsGate.begin() : this.notificationsGate.current();
    if (!reset && this.state.notificationsPending) return;
    const cursor = reset ? null : this.state.notificationsNextCursor;
    if (!reset && !cursor) return;
    this.setState({ notificationsPending: true, notificationsLoading: !!reset, notificationsError: '' });
    try {
      const page = await AuthAPI.notifications(cursor, 20);
      if (!this.authGate.isCurrent(authGeneration) || !this.notificationsGate.isCurrent(generation)) return;
      const revision = Number(page.revision);
      const unreadCount = Number(page.unread_count);
      if (!Number.isInteger(revision) || revision < 0 || !Number.isInteger(unreadCount) || unreadCount < 0) {
        throw new TypeError('invalid notification page');
      }
      if (revision < Number(this.state.notificationRevision || 0)) {
        this.setState({ notificationsPending: false, notificationsLoading: false });
        return;
      }
      const incoming = (page.notifications || []).map(NotificationModel.normalize);
      if (reset) this.trackNotificationLifecycles(incoming);
      this.setState(current => {
        if (revision < Number(current.notificationRevision || 0)) {
          return { notificationsPending: false, notificationsLoading: false };
        }
        return {
          apiUsersByID: this.mergeAPIUsers(incoming.map(notification => notification.actor), current.apiUsersByID),
          notifications: reset ? incoming : NotificationModel.merge(current.notifications, incoming),
          notificationsNextCursor: page.next_cursor || null,
          notificationsPending: false, notificationsLoading: false, notificationsError: '',
          notificationUnreadCount: unreadCount, notificationRevision: revision
        };
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !this.notificationsGate.isCurrent(generation)) return;
      this.setState({
        notificationsPending: false, notificationsLoading: false,
        notificationsError: requestErrorMessage(error, 'Could not load notifications.')
      });
    }
  };

  markNotificationRead = async notificationID => {
    notificationID = Number(notificationID);
    const key = String(notificationID);
    const notification = this.state.notifications.find(item => item.id === notificationID);
    if (!notification || notification.readAt || this.state.notificationReadPendingByID[key]) return;
    const authGeneration = this.authGate.current();
    const gate = this.notificationReadGate(notificationID);
    const generation = gate.begin();
    this.setState({
      notificationReadPendingByID: Object.assign({}, this.state.notificationReadPendingByID, { [key]: true }),
      notificationReadErrorByID: Object.assign({}, this.state.notificationReadErrorByID, { [key]: '' })
    });
    try {
      const response = await AuthAPI.markNotificationRead(notificationID);
      if (!this.authGate.isCurrent(authGeneration) || !gate.isCurrent(generation)) return;
      this.applyNotificationPayload(response, false);
      const pending = Object.assign({}, this.state.notificationReadPendingByID);
      delete pending[key];
      this.setState({ notificationReadPendingByID: pending });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !gate.isCurrent(generation)) return;
      const pending = Object.assign({}, this.state.notificationReadPendingByID);
      delete pending[key];
      this.setState({
        notificationReadPendingByID: pending,
        notificationReadErrorByID: Object.assign({}, this.state.notificationReadErrorByID, {
          [key]: requestErrorMessage(error, 'Could not mark notification as read.')
        })
      });
    }
  };

  markAllNotificationsRead = async () => {
    if (this.state.notificationReadAllPending || this.state.notificationUnreadCount <= 0) return;
    const authGeneration = this.authGate.current();
    const generation = this.notificationReadAllGate.begin();
    this.setState({ notificationReadAllPending: true, notificationsError: '' });
    try {
      const response = await AuthAPI.markAllNotificationsRead();
      if (!this.authGate.isCurrent(authGeneration) || !this.notificationReadAllGate.isCurrent(generation)) return;
      const revision = Number(response.revision);
      if (Number.isInteger(revision) && revision >= Number(this.state.notificationRevision || 0)) {
        this.setState(current => ({
          notifications: NotificationModel.markAllRead(current.notifications, response.read_at),
          notificationUnreadCount: Number(response.unread_count) || 0,
          notificationRevision: revision,
          notificationReadAllPending: false
        }));
      } else {
        this.setState({ notificationReadAllPending: false });
      }
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !this.notificationReadAllGate.isCurrent(generation)) return;
      this.setState({
        notificationReadAllPending: false,
        notificationsError: requestErrorMessage(error, 'Could not mark notifications as read.')
      });
    }
  };

  applyNotificationSource(source, guard) {
    if (!source || !guard || !guard.gate.isCurrent(guard.generation)) return false;
    if (guard.kind === 'relationship' && source.kind === 'relationship' && Number(source.user_id) === guard.userID) {
      this.beginRelationshipGeneration(guard.userID);
      const apiUsersByID = this.mergeAPIUsers([{ id: guard.userID, relationship: source.relationship || {} }]);
      this.setState({ apiUsersByID });
      this.loadDirectory(true);
      this.loadPostFollowers();
      this.loadFeed(true);
      if (Number(this.state.profileId) === guard.userID) this.openProfile(guard.userID);
      return true;
    }
    if (guard.kind === 'group' && source.kind === 'group' && source.group && Number(source.group.id) === guard.groupID) {
      const group = this.applyAuthoritativeGroup(source.group, true);
      if (group.state === 'owner' || group.state === 'member') this.restoreGroupAccess(group);
      this.loadGroups(true);
      this.loadChats(true);
      if (Number(this.state.groupId) === guard.groupID) {
        this.loadGroupDetail(guard.groupID);
        this.loadGroupMembers(guard.groupID, true);
      }
      return true;
    }
    return false;
  }

  actOnNotification = async (notificationID, action) => {
    notificationID = Number(notificationID);
    const key = String(notificationID);
    const notification = this.state.notifications.find(item => item.id === notificationID);
    if (!notification || !NotificationModel.isActionable(notification) || this.state.notificationActionPendingByID[key]) return;
    const authGeneration = this.authGate.current();
    const actionGate = this.notificationActionGate(notificationID);
    const actionGeneration = actionGate.begin();
    let sourceGuard = null;
    if (notification.type === 'follow_request') {
      const gate = this.relationshipGeneration(notification.actorID);
      sourceGuard = { kind: 'relationship', userID: notification.actorID, gate, generation: gate.current() };
    } else if (notification.group && notification.group.id) {
      const groupID = Number(notification.group.id);
      const gate = this.groupGeneration(groupID);
      sourceGuard = { kind: 'group', groupID, gate, generation: gate.current() };
    }
    this.setState({
      notificationActionPendingByID: Object.assign({}, this.state.notificationActionPendingByID, { [key]: true }),
      notificationActionErrorByID: Object.assign({}, this.state.notificationActionErrorByID, { [key]: '' })
    });
    try {
      const response = await AuthAPI.actOnNotification(notificationID, action);
      if (!this.authGate.isCurrent(authGeneration) || !actionGate.isCurrent(actionGeneration)) return;
      this.applyNotificationPayload(response, false);
      this.applyNotificationSource(response.source, sourceGuard);
      const pending = Object.assign({}, this.state.notificationActionPendingByID);
      delete pending[key];
      this.setState({ notificationActionPendingByID: pending });
      this.loadNotifications(true);
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !actionGate.isCurrent(actionGeneration)) return;
      const pending = Object.assign({}, this.state.notificationActionPendingByID);
      delete pending[key];
      this.setState({
        notificationActionPendingByID: pending,
        notificationActionErrorByID: Object.assign({}, this.state.notificationActionErrorByID, {
          [key]: requestErrorMessage(error, 'Could not update notification.')
        })
      });
    }
  };

  openNotification = notification => {
    if (!notification) return;
    this.markNotificationRead(notification.id);
    if (notification.group && notification.group.id) {
      this.openGroup(notification.group.id);
      if (notification.type === 'group_event') this.setState({ groupTab: 'events' });
      return;
    }
    this.openProfile(notification.actorID);
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
		const merged = this.mergePostCommentsCounts(mapped, current.posts, current.profilePosts, current.groupPosts);
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
    this.stopRealtime();
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
      }, () => this.startAuthenticatedRealtime(authGeneration));
      this.loadFeed(true);
      this.loadPostFollowers();
      this.loadDirectory();
      this.loadNotifications(true);
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      if (error && error.status === 401) {
        USERS.me = emptyCurrentUser();
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
      Object.assign(authenticatedState, emptyNotificationState());
      if (s.authMode === 'register') Object.assign(authenticatedState, emptyRegistrationForm());
      Object.assign(authenticatedState, emptyProfileEditor());
      this.setState(authenticatedState, () => this.startAuthenticatedRealtime(authGeneration));
      this.loadFeed(true);
      this.loadPostFollowers();
      this.loadDirectory();
      this.loadNotifications(true);
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
	  this.groupPostsGate.begin();
      this.groupEventsGate.begin();
      this.groupEventCreateGate.begin();
      Object.keys(this.groupEventResponseGatesByID).forEach(key => this.groupEventResponseGatesByID[key].begin());
      this.groupEventResponseGatesByID = {};
      this.notificationsGate.begin();
      this.notificationReadAllGate.begin();
      Object.keys(this.notificationReadGatesByID).forEach(key => this.notificationReadGatesByID[key].begin());
      Object.keys(this.notificationActionGatesByID).forEach(key => this.notificationActionGatesByID[key].begin());
      this.notificationReadGatesByID = {};
      this.notificationActionGatesByID = {};
      Object.keys(this.relationshipGenerationsByID).forEach(key => this.relationshipGenerationsByID[key].begin());
      this.relationshipGenerationsByID = {};
      this.latestActionableNotificationIDBySourceKey = {};
      this.chatsGate.begin();
      this.activeChatGate.begin();
      Object.keys(this.chatHistoryGatesByKey).forEach(key => this.chatHistoryGatesByKey[key].begin());
      Object.keys(this.chatAccessGatesByKey).forEach(key => this.chatAccessGatesByKey[key].begin());
      Object.keys(this.chatReadGatesByKey).forEach(key => this.chatReadGatesByKey[key].begin());
      this.chatHistoryGatesByKey = {};
      this.chatAccessGatesByKey = {};
      this.chatReadGatesByKey = {};
      this.chatReadInFlightByKey = {};
      this.chatReadSentCandidateByKey = {};
      this.revokedChatKeys.clear();
      this.revokedGroupAccessIDs.clear();
      this.stopRealtime();
      this.disposeAllCommentPreviews();
      Object.keys(this.groupGenerationsByID).forEach(key => this.groupGenerationsByID[key].begin());
      this.groupGenerationsByID = {};
      Object.keys(this.commentAccessGatesByPostID).forEach(key => this.commentAccessGatesByPostID[key].begin());
      this.commentAccessGatesByPostID = {};
      this.commentLoadGatesByPostID = {};
      USERS.me = emptyCurrentUser();
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
	  }, emptyGroupPostState(), emptyGroupEventState(), emptyNotificationState(), emptyChatState(), emptyRegistrationForm(), emptyProfileEditor()));
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration)) return;
      this.setState({
        logoutPending: false,
        appError: requestErrorMessage(error, 'Could not log out. Please try again.')
      });
    }
  };

  go = (screen) => {
    if (screen !== 'chat') this.stopTyping();
    this.setState({ screen, privacyOpen: false, emojiOpen: false }, () => {
      if (screen === 'chat') {
        if (this.state.activeChatKey) this.enqueueChatRead(this.state.activeChatKey);
        this.loadChats(true, 'user-open');
      }
      if (screen === 'notifications') this.loadNotifications(true);
      if (screen === 'groups') {
        this.loadGroups(true);
        this.loadGroupInvitationInbox(true);
      }
    });
  };
  openProfile = async (targetUserID) => {
    if (targetUserID === 'me') targetUserID = USERS.me.apiId;
    targetUserID = Number(targetUserID);
    if (!Number.isInteger(targetUserID) || targetUserID <= 0) return;
    this.stopTyping();
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
    const relationshipGate = this.relationshipGeneration(targetUserID);
    const relationshipGeneration = relationshipGate.current();
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
      if (!this.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
      this.beginRelationshipGeneration(targetUserID);
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
      if (!this.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
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

  groupEventResponseGate(eventID) {
    const key = String(Number(eventID));
    if (!this.groupEventResponseGatesByID[key]) {
      this.groupEventResponseGatesByID[key] = UserModel.createRequestGate();
    }
    return this.groupEventResponseGatesByID[key];
  }

  invalidateGroupEventResponses() {
    Object.keys(this.groupEventResponseGatesByID).forEach(key => {
      this.groupEventResponseGatesByID[key].begin();
    });
    this.groupEventResponseGatesByID = {};
  }

  groupAccessIsRevoked(groupID) {
    return this.revokedGroupAccessIDs.has(String(Number(groupID)));
  }

  revokeGroupAccess(groupID) {
    groupID = Number(groupID);
    if (!Number.isInteger(groupID) || groupID <= 0) return;
    const key = String(groupID);
    this.revokedGroupAccessIDs.add(key);
    this.groupGeneration(groupID).begin();
    this.chatsGate.begin();
    this.setState(current => ({
      groupMutationPendingByID: Object.assign({}, current.groupMutationPendingByID, { [key]: false }),
      groupMutationErrorByID: Object.assign({}, current.groupMutationErrorByID, { [key]: '' }),
      chatsPending: false,
      chatsLoading: false
    }));
    if (Number(this.state.groupId) !== groupID) return;

    this.groupDetailGate.begin();
    this.groupMembersGate.begin();
    this.groupRequestsGate.begin();
    this.groupInvitationsGate.begin();
    this.groupPostsGate.begin();
    this.groupEventsGate.begin();
    this.groupEventCreateGate.begin();
    this.invalidateGroupEventResponses();
    const postIDs = this.state.groupPosts
      .filter(post => Number(post.groupID) === groupID)
      .map(post => post.id);
    this.purgeCommentStates(postIDs);
    const input = typeof document !== 'undefined' ? document.getElementById('group-post-media') : null;
    if (input) input.value = '';
    this.setState(Object.assign({}, emptyGroupPostState(), emptyGroupEventState(), {
      inviteOpen: false,
      groupLoading: false,
      groupMembers: [], groupMembersNextCursor: null, groupMembersLoading: false, groupMembersError: '',
      groupRequests: [], groupRequestsNextCursor: null, groupRequestsLoading: false, groupRequestsError: '',
      groupInvitations: [], groupInvitationsNextCursor: null, groupInvitationsLoading: false, groupInvitationsError: ''
    }));
  }

  restoreGroupAccess(group) {
    if (!group || (group.state !== 'owner' && group.state !== 'member')) return false;
    const groupID = Number(group.id);
    if (!Number.isInteger(groupID) || groupID <= 0) return false;
    this.revokedGroupAccessIDs.delete(String(groupID));
    if (Number(this.state.groupId) !== groupID) return true;

    this.groupPostsGate.begin();
    this.groupEventsGate.begin();
    this.groupEventCreateGate.begin();
    this.invalidateGroupEventResponses();
    this.purgeCommentStates(this.state.groupPosts.map(post => post.id));
    const input = typeof document !== 'undefined' ? document.getElementById('group-post-media') : null;
    if (input) input.value = '';
    this.setState(Object.assign({}, emptyGroupPostState(), emptyGroupEventState()), () => {
      if (Number(this.state.groupId) === groupID && !this.groupAccessIsRevoked(groupID)) {
        this.loadGroupPosts(groupID, true);
        this.loadGroupEvents(groupID, true);
      }
    });
    return true;
  }

  loadGroupEvents = async (groupID, reset = true) => {
    groupID = Number(groupID);
    if (!Number.isInteger(groupID) || groupID <= 0 || this.groupAccessIsRevoked(groupID)) return;
    const group = this.state.apiGroupsByID[String(groupID)];
    if (group && group.state !== 'owner' && group.state !== 'member') return;
    const authGeneration = this.authGate.current();
    const accessGate = this.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const generation = reset ? this.groupEventsGate.begin() : this.groupEventsGate.current();
    if (!reset && this.state.groupEventsPending) return;
    const cursor = reset ? null : this.state.groupEventsNextCursor;
    if (!reset && !cursor) return;
    this.setState({ groupEventsPending: true, groupEventsLoading: !!reset, groupEventsError: '' });
    try {
      const page = await AuthAPI.groupEvents(groupID, cursor, 20);
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !this.groupEventsGate.isCurrent(generation) || this.groupAccessIsRevoked(groupID) ||
        Number(this.state.groupId) !== groupID
      ) return;
      const rawEvents = page.events || [];
      const mapped = rawEvents.map(event => GroupEventModel.normalizeEventResponse(event));
      const apiUsersByID = this.mergeAPIUsers(rawEvents.map(event => event.creator));
      this.setState(current => {
        let events = reset ? [] : current.groupEvents.slice();
        mapped.forEach(event => { events = GroupEventModel.mergeAuthoritative(events, event); });
        return {
          apiUsersByID, groupEvents: events,
          groupEventsNextCursor: page.next_cursor || null,
          groupEventsPending: false, groupEventsLoading: false, groupEventsError: ''
        };
      });
    } catch (error) {
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !this.groupEventsGate.isCurrent(generation) || this.groupAccessIsRevoked(groupID) ||
        Number(this.state.groupId) !== groupID
      ) return;
      if (error && error.status === 403) {
        this.revokeGroupAccess(groupID);
        return;
      }
      this.setState({
        groupEventsPending: false, groupEventsLoading: false,
        groupEventsError: requestErrorMessage(error, error && error.status === 404 ? 'Group not found.' : 'Could not load group events.')
      });
    }
  };

  createGroupEvent = async () => {
    const groupID = Number(this.state.groupId);
    const group = this.state.apiGroupsByID[String(groupID)];
    const startsAt = new Date(this.state.groupEventStartsAt);
    if (
      !Number.isInteger(groupID) || groupID <= 0 || this.groupAccessIsRevoked(groupID) ||
      !group || (group.state !== 'owner' && group.state !== 'member') ||
      this.state.groupEventCreatePending || !this.state.groupEventTitle.trim() ||
      !this.state.groupEventDescription.trim() || Number.isNaN(startsAt.getTime())
    ) return;
    const authGeneration = this.authGate.current();
    const accessGate = this.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const createGeneration = this.groupEventCreateGate.current();
    this.setState({ groupEventCreatePending: true, groupEventCreateError: '' });
    try {
      const raw = await AuthAPI.createGroupEvent(groupID, {
        title: this.state.groupEventTitle.trim(),
        description: this.state.groupEventDescription.trim(),
        starts_at: startsAt.toISOString()
      });
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !this.groupEventCreateGate.isCurrent(createGeneration) || this.groupAccessIsRevoked(groupID) ||
        Number(this.state.groupId) !== groupID
      ) return;
      this.groupEventsGate.begin();
      const event = GroupEventModel.normalizeEventResponse(raw);
      const apiUsersByID = this.mergeAPIUsers([raw.creator]);
      this.setState(current => ({
        apiUsersByID,
        groupEvents: GroupEventModel.mergeAuthoritative(current.groupEvents, event),
        groupEventsPending: false, groupEventsLoading: false,
        groupEventComposerOpen: false, groupEventTitle: '', groupEventDescription: '', groupEventStartsAt: '',
        groupEventCreatePending: false, groupEventCreateError: ''
      }));
    } catch (error) {
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !this.groupEventCreateGate.isCurrent(createGeneration) || this.groupAccessIsRevoked(groupID) ||
        Number(this.state.groupId) !== groupID
      ) return;
      if (error && error.status === 403) {
        this.revokeGroupAccess(groupID);
        return;
      }
      this.setState({
        groupEventCreatePending: false,
        groupEventCreateError: requestErrorMessage(error, 'Could not create the event. Your draft was kept.')
      });
    }
  };

  respondToGroupEvent = async (eventID, response) => {
    const groupID = Number(this.state.groupId);
    eventID = Number(eventID);
    const key = String(eventID);
    if (
      !Number.isInteger(groupID) || groupID <= 0 || !Number.isInteger(eventID) || eventID <= 0 ||
      (response !== 'going' && response !== 'not_going') || this.groupAccessIsRevoked(groupID) ||
      this.state.groupEventResponsePendingByID[key]
    ) return;
    const authGeneration = this.authGate.current();
    const accessGate = this.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const responseGate = this.groupEventResponseGate(eventID);
    const responseGeneration = responseGate.current();
    this.setState({
      groupEventResponsePendingByID: Object.assign({}, this.state.groupEventResponsePendingByID, { [key]: true }),
      groupEventResponseErrorByID: Object.assign({}, this.state.groupEventResponseErrorByID, { [key]: '' })
    });
    try {
      const raw = await AuthAPI.respondToGroupEvent(groupID, eventID, response);
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !responseGate.isCurrent(responseGeneration) || this.groupAccessIsRevoked(groupID) ||
        Number(this.state.groupId) !== groupID
      ) return;
      this.groupEventsGate.begin();
      const event = GroupEventModel.normalizeEventResponse(raw);
      const apiUsersByID = this.mergeAPIUsers([raw.creator]);
      this.setState(current => ({
        apiUsersByID,
        groupEvents: GroupEventModel.mergeAuthoritative(current.groupEvents, event),
        groupEventsPending: false, groupEventsLoading: false,
        groupEventResponsePendingByID: Object.assign({}, current.groupEventResponsePendingByID, { [key]: false }),
        groupEventResponseErrorByID: Object.assign({}, current.groupEventResponseErrorByID, { [key]: '' })
      }));
    } catch (error) {
      if (
        !this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !responseGate.isCurrent(responseGeneration) || this.groupAccessIsRevoked(groupID) ||
        Number(this.state.groupId) !== groupID
      ) return;
      if (error && error.status === 403) {
        this.revokeGroupAccess(groupID);
        return;
      }
      this.setState(current => ({
        groupEventResponsePendingByID: Object.assign({}, current.groupEventResponsePendingByID, { [key]: false }),
        groupEventResponseErrorByID: Object.assign({}, current.groupEventResponseErrorByID, {
          [key]: requestErrorMessage(error, 'Could not update your response.')
        })
      }));
    }
  };

  loadGroupPosts = async (groupID, reset = true) => {
	groupID = Number(groupID);
	if (!Number.isInteger(groupID) || groupID <= 0 || this.groupAccessIsRevoked(groupID)) return;
	const group = this.state.apiGroupsByID[String(groupID)];
	if (group && group.state !== 'owner' && group.state !== 'member') return;
	const authGeneration = this.authGate.current();
	const accessGate = this.groupGeneration(groupID);
	const accessGeneration = accessGate.current();
	const generation = reset ? this.groupPostsGate.begin() : this.groupPostsGate.current();
	if (!reset && this.state.groupPostsPending) return;
	const cursor = reset ? null : this.state.groupPostsNextCursor;
	if (!reset && !cursor) return;
	this.setState({ groupPostsPending: true, groupPostsLoading: !!reset, groupPostsError: '' });
	try {
	  const page = await AuthAPI.groupPosts(groupID, cursor, 20);
	  if (
		!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
		!this.groupPostsGate.isCurrent(generation) || this.groupAccessIsRevoked(groupID) ||
		Number(this.state.groupId) !== groupID
	  ) return;
	  const rawPosts = page.posts || [];
	  const mapped = rawPosts.map(post => this.mapAPIPost(post));
	  const apiUsersByID = this.mergeAPIUsers(rawPosts.map(post => post.author));
	  this.setState(current => {
		const merged = this.mergePostCommentsCounts(mapped, current.posts, current.profilePosts, current.groupPosts);
		const nextPosts = reset
		  ? merged
		  : current.groupPosts.concat(merged.filter(post => !current.groupPosts.some(item => item.id === post.id)));
		return {
		  apiUsersByID, groupPosts: nextPosts,
		  groupPostsNextCursor: page.next_cursor || null,
		  groupPostsPending: false, groupPostsLoading: false, groupPostsError: ''
		};
	  });
	} catch (error) {
	  if (
		!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
		!this.groupPostsGate.isCurrent(generation) || this.groupAccessIsRevoked(groupID) ||
		Number(this.state.groupId) !== groupID
	  ) return;
	  if (error && error.status === 403) {
		this.revokeGroupAccess(groupID);
		return;
	  }
	  this.setState({
		groupPostsPending: false, groupPostsLoading: false,
		groupPostsError: requestErrorMessage(error, error && error.status === 404 ? 'Group not found.' : 'Could not load group posts.')
	  });
	}
  };

  pickGroupPostMedia = () => {
	const input = document.getElementById('group-post-media');
	if (input) input.click();
  };

  onGroupPostMedia = (event) => {
	const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
	this.setState({
	  groupPostComposerFile: file,
	  groupPostComposerFileName: file ? file.name : '',
	  groupPostComposerError: ''
	});
  };

  removeGroupPostMedia = () => {
	const input = document.getElementById('group-post-media');
	if (input) input.value = '';
	this.setState({ groupPostComposerFile: null, groupPostComposerFileName: '', groupPostComposerError: '' });
  };

  sendGroupPost = async () => {
	const groupID = Number(this.state.groupId);
	const group = this.state.apiGroupsByID[String(groupID)];
	if (
	  !Number.isInteger(groupID) || groupID <= 0 || this.groupAccessIsRevoked(groupID) ||
	  !group || (group.state !== 'owner' && group.state !== 'member') ||
	  this.state.groupPostComposerPending || !this.state.groupPostComposerText.trim()
	) return;
	const authGeneration = this.authGate.current();
	const accessGate = this.groupGeneration(groupID);
	const accessGeneration = accessGate.current();
	const form = PostModel.buildCreateGroupPostForm({
	  text: this.state.groupPostComposerText,
	  media: this.state.groupPostComposerFile
	}, FormData);
	this.setState({ groupPostComposerPending: true, groupPostComposerError: '' });
	try {
	  const response = await AuthAPI.createGroupPost(groupID, form);
	  if (
		!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
		this.groupAccessIsRevoked(groupID) || Number(this.state.groupId) !== groupID
	  ) return;
	  this.groupPostsGate.begin();
	  const post = this.mapAPIPost(response);
	  const apiUsersByID = this.mergeAPIUsers([response.author]);
	  const input = typeof document !== 'undefined' ? document.getElementById('group-post-media') : null;
	  if (input) input.value = '';
	  this.setState(current => ({
		apiUsersByID,
		groupPosts: [post].concat(current.groupPosts.filter(item => item.id !== post.id)),
		groupPostsLoading: false, groupPostsPending: false,
		groupPostComposerText: '', groupPostComposerFile: null, groupPostComposerFileName: '',
		groupPostComposerError: '', groupPostComposerPending: false
	  }));
	} catch (error) {
	  if (
		!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
		this.groupAccessIsRevoked(groupID) || Number(this.state.groupId) !== groupID
	  ) return;
	  if (error && error.status === 403) {
		this.revokeGroupAccess(groupID);
		return;
	  }
	  this.setState({
		groupPostComposerPending: false,
		groupPostComposerError: requestErrorMessage(error, 'Could not create the group post. Your draft was kept.')
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
    this.stopTyping();
    this.groupDetailGate.begin();
    this.groupMembersGate.begin();
    this.groupRequestsGate.begin();
    this.groupInvitationsGate.begin();
	this.groupPostsGate.begin();
	this.groupEventsGate.begin();
	this.groupEventCreateGate.begin();
	this.invalidateGroupEventResponses();
	this.purgeCommentStates(this.state.groupPosts.map(post => post.id));
	this.setState(Object.assign({
      screen: 'group', groupId: groupID, groupTab: 'posts', inviteOpen: false,
      groupLoading: true, groupError: '', groupMembers: [], groupMembersNextCursor: null,
      groupMembersLoading: true, groupMembersError: '', groupRequests: [], groupRequestsNextCursor: null,
      groupRequestsLoading: false, groupRequestsError: '', groupInvitations: [], groupInvitationsNextCursor: null,
      groupInvitationsLoading: false, groupInvitationsError: '', groupInviteUserID: ''
	}, emptyGroupPostState(), emptyGroupEventState()));
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
	  if ((mapped.state === 'owner' || mapped.state === 'member') && !this.groupAccessIsRevoked(groupID)) {
		this.loadGroupPosts(groupID, true);
		this.loadGroupEvents(groupID, true);
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
    if (!Number.isInteger(groupID) || groupID <= 0 || this.state.groupMutationPendingByID[key]) return false;
    const authGeneration = this.authGate.current();
    const accessGate = this.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    this.setState({
      groupMutationPendingByID: Object.assign({}, this.state.groupMutationPendingByID, { [key]: true }),
      groupMutationErrorByID: Object.assign({}, this.state.groupMutationErrorByID, { [key]: '' })
    });
    try {
      const raw = await operation();
      const expectedRevokeAlreadyApplied = options && options.revokeGroupAccess && this.groupAccessIsRevoked(groupID);
      if (
        !this.authGate.isCurrent(authGeneration) ||
        (!accessGate.isCurrent(accessGeneration) && !expectedRevokeAlreadyApplied)
      ) return false;
      const group = this.applyAuthoritativeGroup(raw, options && options.invalidateInbox);
      if (options && options.revokeGroupAccess) this.revokeGroupAccess(groupID);
      if (options && options.restoreGroupAccess) this.restoreGroupAccess(group);
      if (Number(this.state.groupId) === groupID) {
        this.loadGroupMembers(groupID, true);
        if (group.state === 'owner') {
          this.loadGroupRequests(groupID, true);
          this.loadGroupInvitations(groupID, true);
        }
      }
      if (options && options.invalidateInbox) this.loadGroupInvitationInbox(true);
      if (options && options.purgeChat) this.purgeChat(ChatModel.chatKey('group', groupID));
      if (options && options.refreshChats) this.loadChats(true);
      return true;
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration)) return false;
      this.setState({
        groupMutationPendingByID: Object.assign({}, this.state.groupMutationPendingByID, { [key]: false }),
        groupMutationErrorByID: Object.assign({}, this.state.groupMutationErrorByID, {
          [key]: requestErrorMessage(error, 'Could not update group membership.')
        })
      });
      return false;
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
	groupID, () => AuthAPI.acceptGroupInvitation(groupID), {
	  invalidateInbox: true, refreshChats: true, restoreGroupAccess: true
	}
  );

  declineGroupInvitation = (groupID) => this.runGroupMutation(
    groupID, () => AuthAPI.declineGroupInvitation(groupID), { invalidateInbox: true }
  );

  leaveGroup = (groupID) => this.runGroupMutation(
	groupID, () => AuthAPI.leaveGroup(groupID), {
	  purgeChat: true, refreshChats: true, revokeGroupAccess: true
	}
  );

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
      const group = this.applyAuthoritativeGroup(raw, false);
      this.revokedGroupAccessIDs.delete(String(group.id));
      this.loadChats(true);
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
    return mutation.then(applied => {
      if (
        applied && Number(this.state.groupId) === groupID &&
        Number(this.state.groupInviteUserID) === userID
      ) this.setState({ groupInviteUserID: '' });
      return applied;
    });
  };

  chatHistoryGate(key) {
    if (!this.chatHistoryGatesByKey[key]) this.chatHistoryGatesByKey[key] = UserModel.createRequestGate();
    return this.chatHistoryGatesByKey[key];
  }

  chatAccessGate(key) {
    if (!this.chatAccessGatesByKey[key]) this.chatAccessGatesByKey[key] = UserModel.createRequestGate();
    return this.chatAccessGatesByKey[key];
  }

  chatReadGate(key) {
    if (!this.chatReadGatesByKey[key]) this.chatReadGatesByKey[key] = UserModel.createRequestGate();
    return this.chatReadGatesByKey[key];
  }

  chatMessages(key) {
    return this.state.messagesByChatKey[key] || emptyChatMessages();
  }

  documentIsVisible() {
    return typeof document === 'undefined' || !document || document.visibilityState === undefined ||
      document.visibilityState === 'visible';
  }

  compareChatReadCandidates(left, right) {
    if (!left) return right ? -1 : 0;
    if (!right) return 1;
    const leftTime = Date.parse(left.createdAt) || 0;
    const rightTime = Date.parse(right.createdAt) || 0;
    if (leftTime !== rightTime) return leftTime - rightTime;
    return Number(left.id) - Number(right.id);
  }

  latestAuthoritativeChatCandidate(key) {
    const messages = this.chatMessages(key).messages || [];
    let latest = null;
    messages.forEach(message => {
      const id = Number(message && message.apiId);
      if (!Number.isInteger(id) || id <= 0 || !message.createdAt) return;
      const candidate = { id, createdAt: String(message.createdAt) };
      if (!latest || this.compareChatReadCandidates(candidate, latest) > 0) latest = candidate;
    });
    return latest;
  }

  chatReadEligible(key) {
    const chat = ChatModel.parseChatKey(key);
    if (!chat || this.state.screen !== 'chat' || this.state.activeChatKey !== key ||
        !this.documentIsVisible() || !this.chatMessages(key).loaded || this.revokedChatKeys.has(key)) return false;
    return chat.kind !== 'group' || !this.groupAccessIsRevoked(chat.target_id);
  }

  enqueueChatRead(key) {
    if (!this.chatReadEligible(key)) return;
    const candidate = this.latestAuthoritativeChatCandidate(key);
    if (!candidate) return;
    const queued = this.state.chatReadQueuedThroughByKey[key];
    const sent = this.chatReadSentCandidateByKey[key];
    if (queued && this.compareChatReadCandidates(candidate, queued) <= 0) {
      if (!this.chatReadInFlightByKey[key]) {
        this.drainChatReadQueue(key);
      }
      return;
    }
    if (!queued && sent && this.compareChatReadCandidates(candidate, sent) <= 0) return;
    this.setState(current => ({
      chatReadQueuedThroughByKey: Object.assign({}, current.chatReadQueuedThroughByKey, { [key]: candidate }),
      chatReadErrorByKey: Object.assign({}, current.chatReadErrorByKey, { [key]: '' })
    }), () => this.drainChatReadQueue(key));
  }

  drainChatReadQueue = async key => {
    if (this.chatReadInFlightByKey[key] || !this.chatReadEligible(key)) return;
    const chat = ChatModel.parseChatKey(key);
    const sentCandidate = this.state.chatReadQueuedThroughByKey[key];
    if (!chat || !sentCandidate) return;
    const authGeneration = this.authGate.current();
    const accessGate = this.chatAccessGate(key);
    const accessGeneration = accessGate.current();
    const readGate = this.chatReadGate(key);
    const readGeneration = readGate.current();
    this.chatReadInFlightByKey[key] = true;
    this.chatReadSentCandidateByKey[key] = sentCandidate;
    this.setState(current => ({
      chatReadPendingByKey: Object.assign({}, current.chatReadPendingByKey, { [key]: true })
    }));
    try {
      const response = chat.kind === 'direct'
        ? await AuthAPI.markDirectChatRead(chat.target_id, sentCandidate.id)
        : await AuthAPI.markGroupChatRead(chat.target_id, sentCandidate.id);
      if (!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
          !readGate.isCurrent(readGeneration)) return;
      this.applyChatUnreadPayload(response);
      const queued = this.state.chatReadQueuedThroughByKey[key];
      const hasNewer = queued && this.compareChatReadCandidates(queued, sentCandidate) > 0;
      const pending = Object.assign({}, this.state.chatReadPendingByKey);
      const queuedByKey = Object.assign({}, this.state.chatReadQueuedThroughByKey);
      if (!hasNewer) {
        delete pending[key];
        delete queuedByKey[key];
      }
      this.setState({
        chatReadPendingByKey: pending,
        chatReadQueuedThroughByKey: queuedByKey,
        chatReadErrorByKey: Object.assign({}, this.state.chatReadErrorByKey, { [key]: '' })
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
          !readGate.isCurrent(readGeneration)) return;
      if (chat.kind === 'group' && error && (error.status === 403 || error.status === 404)) {
        this.purgeChat(key);
        return;
      }
      const pending = Object.assign({}, this.state.chatReadPendingByKey);
      delete pending[key];
      this.setState({
        chatReadPendingByKey: pending,
        chatReadErrorByKey: Object.assign({}, this.state.chatReadErrorByKey, {
          [key]: requestErrorMessage(error, 'Could not mark conversation as read.')
        })
      });
    } finally {
      if (this.authGate.isCurrent(authGeneration) && accessGate.isCurrent(accessGeneration) &&
          readGate.isCurrent(readGeneration)) {
        delete this.chatReadInFlightByKey[key];
        const queued = this.state.chatReadQueuedThroughByKey[key];
        if (queued && this.compareChatReadCandidates(queued, sentCandidate) > 0) {
          this.drainChatReadQueue(key);
        }
      }
    }
  };

  applyChatUnreadPayload(payload) {
    const revision = Number(payload && payload.revision);
    const unreadCount = Number(payload && payload.unread_count);
    const chatUnreadCount = Number(payload && payload.chat_unread_count);
    let key;
    try {
      key = ChatModel.chatKey(payload && payload.chat && payload.chat.kind, payload && payload.chat && payload.chat.target_id);
    } catch (ignore) {
      return false;
    }
    if (!Number.isInteger(revision) || revision < Number(this.state.chatUnreadRevision || 0) ||
        !Number.isInteger(unreadCount) || unreadCount < 0 ||
        !Number.isInteger(chatUnreadCount) || chatUnreadCount < 0) return false;
    this.setState(current => {
      if (revision < Number(current.chatUnreadRevision || 0)) return {};
      const chatUnreadByKey = Object.assign({}, current.chatUnreadByKey);
      const chatsByKey = Object.assign({}, current.chatsByKey);
      const readThrough = Object.assign({}, current.chatReadThroughMessageIDByKey);
      const chat = ChatModel.parseChatKey(key);
      const canApplyKey = !!chatsByKey[key] &&
        !(chat && chat.kind === 'group' && this.groupAccessIsRevoked(chat.target_id));
      if (canApplyKey) {
        chatUnreadByKey[key] = chatUnreadCount;
        chatsByKey[key] = Object.assign({}, chatsByKey[key], { unreadCount: chatUnreadCount });
      }
      const markerID = Number(payload.read_through_message_id);
      if (Number.isInteger(markerID) && markerID > 0) readThrough[key] = markerID;
      return {
        chatsByKey, chatUnreadByKey, chatUnreadCount: unreadCount, chatUnreadRevision: revision,
        chatReadThroughMessageIDByKey: readThrough
      };
    });
    return true;
  }

  startAuthenticatedRealtime(authGeneration) {
    if (!this.authGate.isCurrent(authGeneration) || this.state.authStatus !== 'authenticated') return;
    this.wsHasOpened = false;
    this.chatSendLock = false;
    this.loadChats(true);
    this.connectRealtime(authGeneration);
  }

  realtimeURL() {
    if (typeof window === 'undefined' || !window.location) return '';
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return protocol + '//' + window.location.host + '/ws';
  }

  connectRealtime(authGeneration = this.authGate.current()) {
    if (!this.authGate.isCurrent(authGeneration) || this.state.authStatus !== 'authenticated') return;
    if (this.ws && (this.ws.readyState === 0 || this.ws.readyState === 1)) return;
    if (typeof WebSocket !== 'function') {
      this.setState({ wsStatus: 'disconnected', chatError: 'Realtime is unavailable in this browser.' });
      return;
    }
    if (this.wsReconnectTimer) {
      clearTimeout(this.wsReconnectTimer);
      this.wsReconnectTimer = null;
    }
    const url = this.realtimeURL();
    if (!url) return;
    const generation = ++this.wsGeneration;
    const socket = new WebSocket(url);
    this.ws = socket;
    this.setState({ wsStatus: 'connecting' });
    socket.onopen = () => {
      if (!this.authGate.isCurrent(authGeneration) || generation !== this.wsGeneration || this.ws !== socket) {
        socket.close();
        return;
      }
      const reconnect = this.wsHasOpened;
      this.wsHasOpened = true;
      this.setState({ wsStatus: 'connected', wsReconnectAttempt: 0, chatError: '' }, () => {
        if (!reconnect) return;
        this.loadChats(true);
        this.loadNotifications(true);
        if (this.state.activeChatKey) this.loadChatHistory(this.state.activeChatKey, true);
      });
    };
    socket.onmessage = event => {
      if (!this.authGate.isCurrent(authGeneration) || generation !== this.wsGeneration || this.ws !== socket) return;
      this.handleRealtimeEvent(event && event.data);
    };
    socket.onclose = () => {
      if (!this.authGate.isCurrent(authGeneration) || generation !== this.wsGeneration || this.ws !== socket) return;
      this.ws = null;
      this.stopTyping(false);
      this.setState({ wsStatus: 'reconnecting' }, () => this.scheduleRealtimeReconnect(authGeneration));
    };
    socket.onerror = () => {};
  }

  scheduleRealtimeReconnect(authGeneration) {
    if (this.wsReconnectTimer || !this.authGate.isCurrent(authGeneration) || this.state.authStatus !== 'authenticated') return;
    const attempt = Math.max(0, Number(this.state.wsReconnectAttempt) || 0);
    const base = Math.min(15000, 500 * Math.pow(2, attempt));
    const delay = Math.round(base * (0.75 + Math.random() * 0.5));
    this.setState({ wsReconnectAttempt: attempt + 1 });
    this.wsReconnectTimer = setTimeout(() => {
      this.wsReconnectTimer = null;
      this.connectRealtime(authGeneration);
    }, delay);
  }

  stopRealtime(updateState = true) {
    this.stopTyping(false);
    this.wsGeneration += 1;
    if (this.wsReconnectTimer) {
      clearTimeout(this.wsReconnectTimer);
      this.wsReconnectTimer = null;
    }
    const socket = this.ws;
    this.ws = null;
    if (socket && typeof socket.close === 'function') {
      try { socket.close(); } catch (ignore) {}
    }
    Object.keys(this.pendingMessageTimers).forEach(id => clearTimeout(this.pendingMessageTimers[id]));
    this.pendingMessageTimers = {};
    Object.keys(this.typingExpiryTimers).forEach(id => clearTimeout(this.typingExpiryTimers[id]));
    this.typingExpiryTimers = {};
    this.wsHasOpened = false;
    if (updateState && this.state) {
      this.setState({ wsStatus: 'disconnected', wsReconnectAttempt: 0, onlineUserIDs: {}, typingByChatKey: {} });
    }
  }

  loadChats = async (reset = true, historyReason = 'background') => {
    const authGeneration = this.authGate.current();
    const generation = reset ? this.chatsGate.begin() : this.chatsGate.current();
    if (!reset && this.state.chatsPending) return;
    const cursor = reset ? null : this.state.chatsNextCursor;
    if (!reset && !cursor) return;
    this.setState({ chatsPending: true, chatsLoading: !!reset, chatsError: '' });
    try {
      const page = await AuthAPI.chats(cursor, 20);
      if (!this.authGate.isCurrent(authGeneration) || !this.chatsGate.isCurrent(generation)) return;
      const rawChats = (page.chats || []).filter(chat => (
        chat && (chat.kind !== 'group' || !this.groupAccessIsRevoked(chat.target_id))
      ));
      const rawUsers = [];
      const rawGroups = [];
      rawChats.forEach(chat => {
        if (chat.user) rawUsers.push(chat.user);
        if (chat.group) {
          rawGroups.push(chat.group);
          if (chat.group.owner) rawUsers.push(chat.group.owner);
        }
        if (chat.last_message && chat.last_message.sender) rawUsers.push(chat.last_message.sender);
      });
      const normalized = rawChats.map(ChatModel.normalizeChatSummary);
      const revision = page.revision === undefined
        ? Number(this.state.chatUnreadRevision || 0)
        : Number(page.revision);
      const unreadCount = page.unread_count === undefined
        ? Number(this.state.chatUnreadCount || 0)
        : Number(page.unread_count);
      if (!Number.isInteger(revision) || revision < 0 ||
          !Number.isInteger(unreadCount) || unreadCount < 0) {
        throw new TypeError('invalid chat page');
      }
      normalized.forEach(chat => {
        if (chat.kind === 'group' && !this.groupAccessIsRevoked(chat.targetID)) {
          this.revokedChatKeys.delete(chat.key);
        }
      });
      normalized.forEach(chat => {
        if (chat.lastMessage) this.settlePendingMessage(chat.lastMessage.clientMessageID);
      });
      const apiUsersByID = this.mergeAPIUsers(rawUsers);
      this.setState(current => {
        const chatsByKey = ChatModel.mergeChatSummaries(current.chatsByKey, normalized);
        const chatUnreadByKey = Object.assign({}, current.chatUnreadByKey);
        if (revision >= Number(current.chatUnreadRevision || 0)) {
          normalized.forEach(chat => {
            chatUnreadByKey[chat.key] = chat.unreadCount;
            if (chatsByKey[chat.key]) {
              chatsByKey[chat.key] = Object.assign({}, chatsByKey[chat.key], {
                unreadCount: chat.unreadCount
              });
            }
          });
        } else {
          normalized.forEach(chat => {
            if (chatsByKey[chat.key]) {
              chatsByKey[chat.key] = Object.assign({}, chatsByKey[chat.key], {
                unreadCount: Math.max(0, Number(chatUnreadByKey[chat.key]) || 0)
              });
            }
          });
        }
        const activeChatKey = current.activeChatKey || ChatModel.sortedChatKeys(chatsByKey)[0] || null;
        const patch = {
          apiUsersByID,
          apiGroupsByID: this.mergeGroupResponses(rawGroups, current.apiGroupsByID),
          chatsByKey,
          chatKeys: ChatModel.sortedChatKeys(chatsByKey),
          chatUnreadByKey,
          activeChatKey,
          chatsNextCursor: page.next_cursor || null,
          chatsPending: false, chatsLoading: false, chatsError: ''
        };
        if (revision >= Number(current.chatUnreadRevision || 0)) {
          patch.chatUnreadCount = unreadCount;
          patch.chatUnreadRevision = revision;
        }
        return patch;
      }, () => {
        const activeHistory = this.state.activeChatKey
          ? this.chatMessages(this.state.activeChatKey)
          : emptyChatMessages();
        const shouldReloadHistory = this.state.activeChatKey &&
          (!activeHistory.loaded || (reset && historyReason !== 'user-open'));
        if (shouldReloadHistory) {
          this.loadChatHistory(this.state.activeChatKey, true, historyReason);
        } else if (historyReason === 'user-open' && this.state.activeChatKey) {
          this.enqueueChatRead(this.state.activeChatKey);
        }
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !this.chatsGate.isCurrent(generation)) return;
      this.setState({
        chatsPending: false, chatsLoading: false,
        chatsError: requestErrorMessage(error, 'Could not load chats. Please try again.')
      });
    }
  };

  loadChatHistory = async (key, reset = true, reason = 'background') => {
    const chat = ChatModel.parseChatKey(key);
    if (!chat) return;
    const authGeneration = this.authGate.current();
    const accessGate = this.chatAccessGate(key);
    const accessGeneration = accessGate.current();
    const historyGate = this.chatHistoryGate(key);
    const historyGeneration = reset ? historyGate.begin() : historyGate.current();
    const previous = this.chatMessages(key);
    if (!reset && previous.pending) return;
    const cursor = reset ? null : previous.nextCursor;
    if (!reset && !cursor) return;
    if (!reset && this.msgEl && this.state.activeChatKey === key) {
      this.chatScrollAnchor = { key, height: this.msgEl.scrollHeight, top: this.msgEl.scrollTop };
    }
    this.setState(current => {
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      messagesByChatKey[key] = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {}, {
        loading: !!reset, pending: true, error: ''
      });
      return { messagesByChatKey };
    });
    try {
      const page = chat.kind === 'direct'
        ? await AuthAPI.directMessages(chat.target_id, cursor, 20)
        : await AuthAPI.groupMessages(chat.target_id, cursor, 20);
      if (!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
          !historyGate.isCurrent(historyGeneration)) return;
      const rawMessages = page.messages || [];
      const normalized = rawMessages.map(ChatModel.normalizeMessage);
      normalized.forEach(message => this.settlePendingMessage(message.clientMessageID));
      const apiUsersByID = this.mergeAPIUsers(rawMessages.map(message => message.sender));
      if (reset && this.state.activeChatKey === key) this.scrollChatToBottom = true;
      this.setState(current => {
        const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
        const currentEntry = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {});
        messagesByChatKey[key] = Object.assign({}, currentEntry, {
          messages: ChatModel.mergeMessages(currentEntry.messages, normalized),
          nextCursor: page.next_cursor || null,
          loading: false, pending: false, error: '', loaded: true
        });
        return { apiUsersByID, messagesByChatKey };
      }, () => {
        if (reason === 'user-open') this.enqueueChatRead(key);
      });
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
          !historyGate.isCurrent(historyGeneration)) return;
      if ((error && (error.status === 403 || error.status === 404)) && chat.kind === 'group') {
        this.purgeChat(key);
        return;
      }
      this.setState(current => {
        const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
        messagesByChatKey[key] = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {}, {
          loading: false, pending: false, loaded: true,
          error: error && error.status === 403
            ? 'You cannot access this conversation.'
            : requestErrorMessage(error, 'Could not load messages. Please try again.')
        });
        return { messagesByChatKey };
      });
    }
  };

  openChat = key => {
    const chat = ChatModel.parseChatKey(key);
    if (!chat || !this.state.chatsByKey[key]) return;
    if (chat.kind === 'group' && this.groupAccessIsRevoked(chat.target_id)) return;
    this.stopTyping();
    this.activeChatGate.begin();
    this.scrollChatToBottom = true;
    this.setState({
      screen: 'chat', activeChatKey: key, emojiOpen: false, chatDraft: '', chatError: ''
    }, () => {
      const history = this.chatMessages(key);
      if (!history.loaded) this.loadChatHistory(key, true, 'user-open');
      else this.enqueueChatRead(key);
    });
  };

  openDirectChat = userID => {
    userID = Number(userID);
    if (!Number.isInteger(userID) || userID <= 0 || userID === Number(USERS.me.apiId)) return;
    const key = ChatModel.chatKey('direct', userID);
    const user = this.apiUser(userID);
    this.setState(current => {
      const chatsByKey = Object.assign({}, current.chatsByKey);
      if (!chatsByKey[key]) {
        chatsByKey[key] = {
          key, kind: 'direct', targetID: userID, userID, groupID: null,
          lastMessage: null, activityAt: new Date().toISOString(), transient: true
        };
      }
      return { chatsByKey, chatKeys: ChatModel.sortedChatKeys(chatsByKey) };
    }, () => this.openChat(key));
  };

  openGroupChat = groupID => {
    groupID = Number(groupID);
    const group = this.state.apiGroupsByID[String(groupID)];
    if (!group || this.groupAccessIsRevoked(groupID) || (group.state !== 'owner' && group.state !== 'member')) return;
    const key = ChatModel.chatKey('group', groupID);
    this.revokedChatKeys.delete(key);
    this.setState(current => {
      const chatsByKey = Object.assign({}, current.chatsByKey);
      if (!chatsByKey[key]) {
        chatsByKey[key] = {
          key, kind: 'group', targetID: groupID, userID: null, groupID,
          lastMessage: null, activityAt: new Date().toISOString(), transient: true
        };
      }
      return { chatsByKey, chatKeys: ChatModel.sortedChatKeys(chatsByKey) };
    }, () => this.openChat(key));
  };

  purgeChat(key) {
    if (!ChatModel.parseChatKey(key)) return;
    this.revokedChatKeys.add(key);
    this.chatAccessGate(key).begin();
    this.chatHistoryGate(key).begin();
    this.chatReadGate(key).begin();
    delete this.chatReadInFlightByKey[key];
    delete this.chatReadSentCandidateByKey[key];
    if (this.typingChatKey === key) this.stopTyping(false);
    const history = this.chatMessages(key);
    history.messages.forEach(message => this.settlePendingMessage(message.clientMessageID));
    this.setState(current => {
      const chatsByKey = Object.assign({}, current.chatsByKey);
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      const typingByChatKey = Object.assign({}, current.typingByChatKey);
      const chatUnreadByKey = Object.assign({}, current.chatUnreadByKey);
      const chatReadPendingByKey = Object.assign({}, current.chatReadPendingByKey);
      const chatReadErrorByKey = Object.assign({}, current.chatReadErrorByKey);
      const chatReadQueuedThroughByKey = Object.assign({}, current.chatReadQueuedThroughByKey);
      const chatReadThroughMessageIDByKey = Object.assign({}, current.chatReadThroughMessageIDByKey);
      delete chatsByKey[key];
      delete messagesByChatKey[key];
      delete typingByChatKey[key];
      delete chatUnreadByKey[key];
      delete chatReadPendingByKey[key];
      delete chatReadErrorByKey[key];
      delete chatReadQueuedThroughByKey[key];
      delete chatReadThroughMessageIDByKey[key];
      const chatKeys = ChatModel.sortedChatKeys(chatsByKey);
      return {
        chatsByKey, messagesByChatKey, typingByChatKey, chatKeys, chatUnreadByKey,
        chatReadPendingByKey, chatReadErrorByKey, chatReadQueuedThroughByKey,
        chatReadThroughMessageIDByKey,
        activeChatKey: current.activeChatKey === key ? (chatKeys[0] || null) : current.activeChatKey
      };
    });
  }

  handleRealtimeEvent(raw) {
    let event;
    try { event = JSON.parse(String(raw || '')); } catch (ignore) { return; }
    if (!event || typeof event.type !== 'string') return;
    if (event.type === 'notification:upsert') {
      this.applyNotificationPayload(event, true);
      return;
    }
    if (event.type === 'notifications:read-all') {
      const revision = Number(event.revision);
      const unreadCount = Number(event.unread_count);
      if (!Number.isInteger(revision) || revision < Number(this.state.notificationRevision || 0) ||
          !Number.isInteger(unreadCount) || unreadCount < 0) return;
      this.setState(current => {
        if (revision < Number(current.notificationRevision || 0)) return {};
        return {
          notifications: NotificationModel.markAllRead(current.notifications, event.read_at),
          notificationUnreadCount: unreadCount,
          notificationRevision: revision
        };
      });
      return;
    }
    if (event.type === 'presence:init') {
      const onlineUserIDs = {};
      (event.online_user_ids || []).forEach(id => {
        id = Number(id);
        if (Number.isInteger(id) && id > 0) onlineUserIDs[String(id)] = true;
      });
      this.setState({ onlineUserIDs });
      return;
    }
    if (event.type === 'presence:update') {
      const userID = Number(event.user_id);
      if (!Number.isInteger(userID) || userID <= 0) return;
      this.setState(current => {
        const onlineUserIDs = Object.assign({}, current.onlineUserIDs);
        if (event.online === true) onlineUserIDs[String(userID)] = true;
        else delete onlineUserIDs[String(userID)];
        return { onlineUserIDs };
      });
      return;
    }
    if (event.type === 'presence:remove') {
      const userID = Number(event.user_id);
      this.setState(current => {
        const onlineUserIDs = Object.assign({}, current.onlineUserIDs);
        delete onlineUserIDs[String(userID)];
        return { onlineUserIDs };
      });
      return;
    }
    if (event.type === 'chat:remove') {
      if (!event.chat || event.chat.kind !== 'group') return;
	  const groupID = Number(event.chat.target_id);
	  if (!Number.isInteger(groupID) || groupID <= 0) return;
      let key;
      try { key = ChatModel.chatKey(event.chat.kind, event.chat.target_id); } catch (ignore) { return; }
	  this.revokeGroupAccess(groupID);
      this.purgeChat(key);
      return;
    }
    if (event.type === 'typing:update') {
      this.handleTypingUpdate(event);
      return;
    }
    if (event.type === 'chat:unread') {
      this.applyChatUnreadPayload(event);
      return;
    }
    if (event.type === 'chat:message' && event.message) {
      this.handleRealtimeMessage(event.message);
      return;
    }
    if (event.type === 'chat:error') this.handleRealtimeError(event);
  }

  handleRealtimeMessage(rawMessage) {
    let message;
    try { message = ChatModel.normalizeMessage(rawMessage); } catch (ignore) { return; }
    const key = message.chatKey;
    if (this.revokedChatKeys.has(key)) return;
    const wasKnown = !!this.state.chatsByKey[key];
    this.settlePendingMessage(message.clientMessageID);
    let apiUsersByID = this.state.apiUsersByID;
    if (!apiUsersByID[String(message.senderID)] && message.senderID !== Number(USERS.me.apiId)) {
      apiUsersByID = this.mergeAPIUsers([{
        id: message.senderID, first_name: message.senderName, last_name: '',
        avatar_url: message.senderAvatarURL || '/static/avatars/neutral.svg', is_private: false
      }]);
    }
    if (this.state.activeChatKey === key) this.scrollChatToBottom = true;
    this.setState(current => {
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      const history = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {});
      messagesByChatKey[key] = Object.assign({}, history, {
        messages: ChatModel.mergeMessages(history.messages, [message])
      });
      const chatsByKey = Object.assign({}, current.chatsByKey);
      const existing = chatsByKey[key];
      chatsByKey[key] = Object.assign({}, existing || {
        key, kind: message.chat.kind, targetID: message.chat.target_id,
        userID: message.chat.kind === 'direct' ? message.chat.target_id : null,
        groupID: message.chat.kind === 'group' ? message.chat.target_id : null,
        transient: true
      }, {
        lastMessage: message, activityAt: message.createdAt
      });
      return {
        apiUsersByID, messagesByChatKey, chatsByKey,
        chatKeys: ChatModel.sortedChatKeys(chatsByKey), chatError: ''
      };
    }, () => {
      if (this.state.activeChatKey === key) this.enqueueChatRead(key);
      if (!wasKnown) this.loadChats(true);
    });
  }

  handleRealtimeError(event) {
    const clientMessageID = String(event.client_message_id || '').trim().toLowerCase();
    const messageText = String(event.message || 'Could not send message.');
    if (!clientMessageID) {
      this.setState({ chatError: messageText });
      return;
    }
    this.settlePendingMessage(clientMessageID);
    this.setState(current => {
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      Object.keys(messagesByChatKey).forEach(key => {
        const history = Object.assign({}, messagesByChatKey[key]);
        let changed = false;
        history.messages = (history.messages || []).map(message => {
          if (message.clientMessageID !== clientMessageID) return message;
          changed = true;
          return Object.assign({}, message, { pending: false, failed: true, error: messageText });
        });
        if (changed) messagesByChatKey[key] = history;
      });
      return { messagesByChatKey, chatError: messageText };
    });
  }

  handleTypingUpdate(event) {
    const userID = Number(event.user && event.user.id);
    let key;
    try { key = ChatModel.chatKey(event.chat && event.chat.kind, event.chat && event.chat.target_id); } catch (ignore) { return; }
    if (!Number.isInteger(userID) || userID <= 0 || userID === Number(USERS.me.apiId)) return;
    const timerKey = key + ':' + userID;
    if (this.typingExpiryTimers[timerKey]) {
      clearTimeout(this.typingExpiryTimers[timerKey]);
      delete this.typingExpiryTimers[timerKey];
    }
    this.setState(current => {
      const typingByChatKey = Object.assign({}, current.typingByChatKey);
      const users = Object.assign({}, typingByChatKey[key] || {});
      if (event.typing === true) {
        users[String(userID)] = {
          id: userID, name: String(event.user.display_name || ('User ' + userID))
        };
      } else {
        delete users[String(userID)];
      }
      if (Object.keys(users).length) typingByChatKey[key] = users;
      else delete typingByChatKey[key];
      return { typingByChatKey };
    });
    if (event.typing === true) {
      this.typingExpiryTimers[timerKey] = setTimeout(() => {
        delete this.typingExpiryTimers[timerKey];
        this.handleTypingUpdate({
          chat: event.chat, user: event.user, typing: false
        });
      }, 6000);
    }
  }

  sendTypingEvent(type, key) {
    const chat = ChatModel.parseChatKey(key);
    if (!chat || !this.ws || this.ws.readyState !== 1) return false;
    try {
      this.ws.send(JSON.stringify({ type, chat: { kind: chat.kind, target_id: chat.target_id } }));
      return true;
    } catch (ignore) {
      return false;
    }
  }

  startTyping(key) {
    if (!key || this.state.wsStatus !== 'connected') return;
    if (this.typingChatKey && this.typingChatKey !== key) this.stopTyping();
    if (this.typingChatKey === key) return;
    if (!this.sendTypingEvent('typing:start', key)) return;
    this.typingChatKey = key;
    this.typingHeartbeatTimer = setInterval(() => {
      if (!this.typingChatKey || !this.state.chatDraft.trim()) {
        this.stopTyping();
        return;
      }
      this.sendTypingEvent('typing:heartbeat', this.typingChatKey);
    }, 2000);
  }

  stopTyping(sendEvent = true) {
    const key = this.typingChatKey;
    this.typingChatKey = null;
    if (this.typingHeartbeatTimer) {
      clearInterval(this.typingHeartbeatTimer);
      this.typingHeartbeatTimer = null;
    }
    if (sendEvent && key) this.sendTypingEvent('typing:stop', key);
  }

  onChatDraft = value => {
    this.setState({ chatDraft: value, chatError: '' }, () => {
      if (this.state.chatDraft.trim() && this.state.activeChatKey) this.startTyping(this.state.activeChatKey);
      else this.stopTyping();
    });
  };

  settlePendingMessage(clientMessageID) {
    clientMessageID = String(clientMessageID || '').toLowerCase();
    if (this.pendingMessageTimers[clientMessageID]) {
      clearTimeout(this.pendingMessageTimers[clientMessageID]);
      delete this.pendingMessageTimers[clientMessageID];
    }
  }

  armPendingMessageTimeout(clientMessageID, key) {
    this.settlePendingMessage(clientMessageID);
    const authGeneration = this.authGate.current();
    const accessGeneration = this.chatAccessGate(key).current();
    this.pendingMessageTimers[clientMessageID] = setTimeout(() => {
      delete this.pendingMessageTimers[clientMessageID];
      if (!this.authGate.isCurrent(authGeneration) || !this.chatAccessGate(key).isCurrent(accessGeneration)) return;
      this.setState(current => {
        const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
        const history = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {});
        history.messages = history.messages.map(message => message.clientMessageID === clientMessageID && message.pending
          ? Object.assign({}, message, {
            pending: false, failed: true,
            error: 'No response from server. Retry with the same message ID.'
          })
          : message);
        messagesByChatKey[key] = history;
        return { messagesByChatKey };
      });
    }, 15000);
  }

  sendPendingMessage(
    message,
    authGeneration = this.authGate.current(),
    accessGeneration = message && message.chatKey ? this.chatAccessGate(message.chatKey).current() : -1
  ) {
    if (!message || !message.clientMessageID || !message.chat) return false;
    if (!this.authGate.isCurrent(authGeneration) ||
        !this.chatAccessGate(message.chatKey).isCurrent(accessGeneration) ||
        this.state.authStatus !== 'authenticated') return false;
    if (!this.ws || this.ws.readyState !== 1 || this.state.wsStatus !== 'connected') {
      this.handleRealtimeError({
        client_message_id: message.clientMessageID,
        message: 'Realtime is disconnected. Reconnect and retry.'
      });
      return false;
    }
    this.setState(current => {
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      const history = Object.assign({}, emptyChatMessages(), messagesByChatKey[message.chatKey] || {});
      history.messages = history.messages.map(item => item.clientMessageID === message.clientMessageID
        ? Object.assign({}, item, { pending: true, failed: false, error: '' })
        : item);
      messagesByChatKey[message.chatKey] = history;
      return { messagesByChatKey, chatError: '' };
    });
    try {
      this.ws.send(JSON.stringify({
        type: 'chat:send',
        client_message_id: message.clientMessageID,
        chat: { kind: message.chat.kind, target_id: message.chat.target_id },
        text: message.body
      }));
      this.armPendingMessageTimeout(message.clientMessageID, message.chatKey);
      return true;
    } catch (error) {
      this.handleRealtimeError({
        client_message_id: message.clientMessageID,
        message: 'Could not send message. Please retry.'
      });
      return false;
    }
  }

  sendMsg = () => {
    if (this.chatSendLock) return;
    const key = this.state.activeChatKey;
    const chat = ChatModel.parseChatKey(key);
    const body = this.state.chatDraft.trim();
    if (!chat || !body) return;
    if (Array.from(body).length > 2000) {
      this.setState({ chatError: 'Messages are limited to 2000 characters.' });
      return;
    }
    const authGeneration = this.authGate.current();
    const accessGeneration = this.chatAccessGate(key).current();
    this.chatSendLock = true;
    const message = ChatModel.pendingMessage(
      createClientMessageID(), chat, USERS.me.apiId, body, new Date().toISOString()
    );
    this.stopTyping();
    this.scrollChatToBottom = true;
    this.setState(current => {
      const messagesByChatKey = Object.assign({}, current.messagesByChatKey);
      const history = Object.assign({}, emptyChatMessages(), messagesByChatKey[key] || {});
      history.messages = ChatModel.mergeMessages(history.messages, [message]);
      messagesByChatKey[key] = history;
      return { messagesByChatKey, chatDraft: '', emojiOpen: false, chatError: '' };
    }, () => {
      this.chatSendLock = false;
      this.sendPendingMessage(message, authGeneration, accessGeneration);
    });
  };

  retryMessage = clientMessageID => {
    let found = null;
    Object.keys(this.state.messagesByChatKey).some(key => {
      found = (this.state.messagesByChatKey[key].messages || []).find(message => (
        message.clientMessageID === clientMessageID
      ));
      return !!found;
    });
    if (!found || found.pending || !found.failed) return;
    this.sendPendingMessage(found);
  };

  acceptFollowRequest = async (requestID) => {
    const key = String(requestID);
    if (this.state.followRequestPendingByID[key]) return;
    const request = this.state.followRequests.find(item => String(item.id) === key);
    if (!request) return;
    const authGeneration = this.authGate.current();
    const relationshipGate = this.relationshipGeneration(request.user.id);
    const relationshipGeneration = relationshipGate.current();
    this.setState({
      followRequestPendingByID: Object.assign({}, this.state.followRequestPendingByID, { [key]: true }),
      followRequestsError: ''
    });
    try {
      await AuthAPI.acceptFollowRequest(requestID);
      if (!this.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
      this.beginRelationshipGeneration(request.user.id);
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
      if (!this.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
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
    const request = this.state.followRequests.find(item => String(item.id) === key);
    if (!request) return;
    const authGeneration = this.authGate.current();
    const relationshipGate = this.relationshipGeneration(request.user.id);
    const relationshipGeneration = relationshipGate.current();
    this.setState({
      followRequestPendingByID: Object.assign({}, this.state.followRequestPendingByID, { [key]: true }),
      followRequestsError: ''
    });
    try {
      await AuthAPI.rejectFollowRequest(requestID);
      if (!this.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
      this.beginRelationshipGeneration(request.user.id);
      const pending = Object.assign({}, this.state.followRequestPendingByID);
      delete pending[key];
      this.setState({
        followRequests: this.state.followRequests.filter(item => String(item.id) !== key),
        followRequestPendingByID: pending
      });
      this.loadDirectory();
    } catch (error) {
      if (!this.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
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
    const commentState = this.commentState(p.id);
    const comments = commentState.comments;
    const user = this.apiUser(p.apiAuthorID);
    return Object.assign({}, p, {
      user,
      privacyIcon: pm.icon, privacyLabel: pm.label,
      hasImage: !!p.mediaUrl,
      mediaUrl: p.mediaUrl || '',
      commentCount: num(p.commentsCount || 0),
      showComments: !!s.openComments[key],
      comments: comments.map(c => Object.assign({}, c, {
        user: this.apiUser(c.apiAuthorID),
        time: this.formatPostTime(c.createdAt),
        hasMedia: !!c.mediaUrl
      })),
      draft: commentState.draft,
      commentsLoading: commentState.loading,
      commentsPending: commentState.pending,
      commentsHasError: !!commentState.error,
      commentsError: commentState.error,
      commentsHasMore: !!commentState.nextCursor,
      commentCreatePending: commentState.createPending,
      commentCreateHasError: !!commentState.createError,
      commentCreateError: commentState.createError,
      commentSendDisabled: commentState.createPending || !commentState.draft.trim(),
      commentMediaInputID: this.commentMediaInputID(p.id),
      commentHasMedia: !!commentState.mediaFile,
      commentMediaFileName: commentState.mediaFileName,
      commentMediaPreviewURL: commentState.mediaPreviewURL,
      commentMediaControlsDisabled: commentState.createPending,
      commentSendLabel: commentState.createPending ? '…' : 'Send',
      onToggleComments: () => this.togglePostComments(p.id),
      onDraft: (e) => this.setCommentDraft(p.id, e.target.value),
      onKey: (e) => {
        if (e.key !== 'Enter') return;
        e.preventDefault();
        this.createComment(p.id);
      },
      onSendComment: () => this.createComment(p.id),
      onCommentMedia: (e) => this.selectCommentMedia(p.id, e),
      onChooseCommentMedia: () => {
        if (commentState.createPending || typeof document === 'undefined') return;
        const input = document.getElementById(this.commentMediaInputID(p.id));
        if (input) input.click();
      },
      onRemoveCommentMedia: () => this.removeCommentMedia(p.id),
      loadMoreComments: () => this.loadComments(p.id, false),
      retryComments: () => this.loadComments(p.id, true),
      goProfile: () => this.openProfile(p.apiAuthorID)
    });
  }

  renderVals() {
    const s = this.state;
    const me = USERS.me;
    const notifUnread = s.notificationUnreadCount;
    const chatUnread = Math.max(0, Number(s.chatUnreadCount) || 0);

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
    const feedPosts = s.posts.map((p, i) => Object.assign(this.mapPost(p), { delay: (i * 0.06).toFixed(2) + 's' }));
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
        message: () => this.openDirectChat(userID),
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
      const accessRevoked = this.groupAccessIsRevoked(g.id);
      return {
        name: g.name, desc: g.desc, membersLabel: num(g.members), cover: cover(g.color),
        owner: this.apiUser(g.ownerID),
        delay: (i * 0.05).toFixed(2) + 's', pending,
        error: s.groupMutationErrorByID[String(g.id)] || '', hasError: !!s.groupMutationErrorByID[String(g.id)],
        isJoined: !accessRevoked && (g.state === 'member' || g.state === 'owner'),
        isOwner: !accessRevoked && g.state === 'owner',
        isMember: !accessRevoked && g.state === 'member', isNone: g.state === 'none',
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
    const gAccessRevoked = this.groupAccessIsRevoked(g.id);
    const gIsOwner = !gAccessRevoked && g.state === 'owner';
    const gCanChat = !gAccessRevoked && (g.state === 'owner' || g.state === 'member');
	const gCanContent = gCanChat;
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
	const gPosts = s.groupPosts.map((post, index) => Object.assign(this.mapPost(post), {
	  delay: (index * 0.05).toFixed(2) + 's'
	}));
    const gEvents = s.groupEvents.map((event, index) => {
      const eventID = String(event.id);
      const startsAt = new Date(event.startsAt);
      const pending = !!s.groupEventResponsePendingByID[eventID];
      return {
        id: event.id,
        title: event.title,
        description: event.description,
        startsAt: Number.isNaN(startsAt.getTime()) ? event.startsAt : startsAt.toLocaleString([], {
          dateStyle: 'medium', timeStyle: 'short'
        }),
        creator: this.apiUser(event.creatorID),
        goingCount: num(event.goingCount),
        notGoingCount: num(event.notGoingCount),
        goingSelected: event.viewerResponse === 'going',
        notGoingSelected: event.viewerResponse === 'not_going',
        goingBg: event.viewerResponse === 'going' ? 'var(--accent)' : 'transparent',
        goingColor: event.viewerResponse === 'going' ? '#fff' : 'var(--text2)',
        notGoingBg: event.viewerResponse === 'not_going' ? 'var(--surface2)' : 'transparent',
        notGoingColor: event.viewerResponse === 'not_going' ? 'var(--text)' : 'var(--text2)',
        pending,
        error: s.groupEventResponseErrorByID[eventID] || '',
        hasError: !!s.groupEventResponseErrorByID[eventID],
        delay: (index * 0.05).toFixed(2) + 's',
        goProfile: () => this.openProfile(event.creatorID),
        going: () => this.respondToGroupEvent(event.id, 'going'),
        notGoing: () => this.respondToGroupEvent(event.id, 'not_going')
      };
    });
    const groupEventStartsAtDate = new Date(s.groupEventStartsAt);
    const groupEventCreateDisabled = s.groupEventCreatePending || !s.groupEventTitle.trim() ||
      !s.groupEventDescription.trim() || Number.isNaN(groupEventStartsAtDate.getTime());

    // chat
    const chatMeta = chat => {
      if (!chat) {
        return {
          title: 'Select a conversation', initials: '…', color: 'var(--text3)', sub: '',
          avatarUrl: '', hasAvatar: false, noAvatar: true, online: false
        };
      }
      if (chat.kind === 'direct') {
        const user = this.apiUser(chat.userID || chat.targetID);
        const online = !!s.onlineUserIDs[String(user.apiId)];
        return {
          title: user.name, initials: user.initials, color: user.color,
          sub: online ? 'Online now' : 'Offline', online,
          avatarUrl: user.avatarUrl, hasAvatar: user.hasAvatar, noAvatar: user.noAvatar
        };
      }
      const group = s.apiGroupsByID[String(chat.groupID || chat.targetID)] || {
        name: 'Group ' + chat.targetID, members: 0,
        color: GROUP_COLORS[Math.abs(chat.targetID) % GROUP_COLORS.length]
      };
      return {
        title: group.name, initials: String(group.name || 'G').slice(0, 2).toUpperCase(),
        color: group.color, sub: num(group.members || 0) + ' members', online: false,
        avatarUrl: '', hasAvatar: false, noAvatar: true
      };
    };
    const convos = s.chatKeys.map(key => {
      const chat = s.chatsByKey[key];
      const meta = chatMeta(chat);
      const last = chat.lastMessage;
      const unread = Math.max(0, Number(s.chatUnreadByKey[key] !== undefined
        ? s.chatUnreadByKey[key]
        : chat.unreadCount) || 0);
      return {
        title: meta.title, initials: meta.initials, color: meta.color,
        avatarUrl: meta.avatarUrl, hasAvatar: meta.hasAvatar, noAvatar: meta.noAvatar,
        preview: last
          ? (Number(last.senderID) === Number(me.apiId) ? 'You: ' : '') + last.body
          : 'No messages yet',
        previewColor: unread > 0 ? 'var(--text)' : 'var(--text3)', previewW: unread > 0 ? '750' : '500',
        hasUnread: unread > 0, unread: num(unread > 99 ? '99+' : unread),
        time: last ? this.formatPostTime(last.createdAt) : '',
        online: meta.online,
        bg: key === s.activeChatKey ? 'var(--soft)' : 'transparent',
        open: () => this.openChat(key)
      };
    });
    const active = s.activeChatKey ? s.chatsByKey[s.activeChatKey] : null;
    const am = chatMeta(active);
    const activeHistory = s.activeChatKey ? this.chatMessages(s.activeChatKey) : emptyChatMessages();
    const activeMessages = activeHistory.messages || [];
    const messages = activeMessages.map((msg, i) => {
      const prev = activeMessages[i - 1];
      let user = this.apiUser(msg.senderID);
      if ((!user || user.name.indexOf('User ') === 0) && msg.senderName) {
        user = decorateUser({
          id: String(msg.senderID), apiId: msg.senderID, name: msg.senderName,
          initials: msg.senderName.split(/\s+/).map(part => part.charAt(0)).join('').slice(0, 2).toUpperCase() || '?',
          color: GROUP_COLORS[Math.abs(msg.senderID) % GROUP_COLORS.length],
          avatarUrl: msg.senderAvatarURL || '/static/avatars/neutral.svg'
        });
      }
      return {
        text: msg.body, time: this.formatPostTime(msg.createdAt),
        mine: Number(msg.senderID) === Number(me.apiId),
        theirs: Number(msg.senderID) !== Number(me.apiId),
        user,
        showName: active && active.kind === 'group' && Number(msg.senderID) !== Number(me.apiId) &&
          (!prev || Number(prev.senderID) !== Number(msg.senderID)),
        pending: !!msg.pending, failed: !!msg.failed, error: msg.error || '',
        hasStatus: !!msg.pending || !!msg.failed,
        statusLabel: msg.failed ? 'Failed' : (msg.pending ? 'Sending…' : ''),
        retry: () => this.retryMessage(msg.clientMessageID)
      };
    });
    const activeTypingUsers = Object.values(s.typingByChatKey[s.activeChatKey] || {});
    const typingLabel = activeTypingUsers.length > 1
      ? activeTypingUsers.map(user => user.name).join(', ') + ' are typing'
      : (activeTypingUsers[0] ? activeTypingUsers[0].name + ' is typing' : '');
    const emojis = EMOJIS.map(ch => ({
      ch,
      add: () => this.onChatDraft(this.state.chatDraft + ch)
    }));

    // notifications
    const notifItems = s.notifications.map((notification, i) => {
      const key = String(notification.id);
      const actionPending = !!s.notificationActionPendingByID[key];
      const groupTitle = notification.group && notification.group.title ? notification.group.title : 'a group';
      const eventTitle = notification.event && notification.event.title ? notification.event.title : 'an event';
      const textByType = {
        follow_started: 'started following you',
        follow_request: 'requested to follow you',
        group_invitation: 'invited you to ' + groupTitle,
        group_join_request: 'requested to join ' + groupTitle,
        group_event: 'created ' + eventTitle + ' in ' + groupTitle
      };
      return {
        user: this.apiUser(notification.actorID), icon: notification.group ? IC.users : IC.user,
        text: textByType[notification.type] || 'updated something',
        time: this.formatPostTime(notification.createdAt), delay: (i * 0.04).toFixed(2) + 's',
        bg: notification.readAt ? 'var(--surface)' : 'color-mix(in oklab, var(--accent) 5%, var(--surface))',
        unreadDot: !notification.readAt,
        pending: NotificationModel.isActionable(notification),
        done: notification.resolution != null,
        doneLabel: notification.resolution === 'accepted' ? 'Accepted' :
          (notification.resolution === 'declined' ? 'Declined' : 'Cancelled'),
        disabled: actionPending,
        accept: () => this.actOnNotification(notification.id, 'accept'),
        decline: () => this.actOnNotification(notification.id, 'decline'),
        open: () => this.openNotification(notification),
        goProfile: event => {
          if (event && typeof event.stopPropagation === 'function') event.stopPropagation();
          this.markNotificationRead(notification.id);
          this.openProfile(notification.actorID);
        },
        hasError: !!s.notificationActionErrorByID[key] || !!s.notificationReadErrorByID[key],
        error: s.notificationActionErrorByID[key] || s.notificationReadErrorByID[key] || ''
      };
    });

    // right rail
    const pendingRequestActorIDs = new Set(
      s.notifications
        .filter(n => n.type === 'follow_request' && NotificationModel.isActionable(n))
        .map(n => Number(n.actorID))
    );
    const suggestions = s.directoryUserIDs.map(userID => this.apiUser(userID))
      .filter(user => !user.relationship || user.relationship.status !== 'accepted')
      .filter(user => !pendingRequestActorIDs.has(Number(user.apiId)))
      .map(user => {
        const b = this.followBtn(user.apiId);
        return {
          user, isPrivate: user.private,
          btnLabel: b.label, btnBg: b.bg, btnColor: b.color, btnBd: b.bd, btnDisabled: b.disabled,
          onBtn: () => this.toggleFollow(user.apiId),
          canMessage: user.relationship && (user.relationship.status === 'accepted' || user.relationship.follows_me),
          message: () => this.openDirectChat(user.apiId),
          goProfile: () => this.openProfile(user.apiId)
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
      postBtnDisabled: s.composerPending || !s.composerText.trim() || !composerAudienceReady,
      postBtnBg: (s.composerText.trim() && composerAudienceReady && !s.composerPending) ? 'var(--accent)' : 'var(--surface2)',
      postBtnColor: (s.composerText.trim() && composerAudienceReady && !s.composerPending) ? '#fff' : 'var(--text3)',
      postBtnCursor: s.composerPending ? 'wait' : ((s.composerText.trim() && composerAudienceReady) ? 'pointer' : 'not-allowed'),
      postButtonLabel: s.composerPending ? 'Posting…' : 'Post',
      sendPost: this.sendPost,
      // profile
      pUser, pIsMe, pOther: !pIsMe,
      pCover: cover(pUser.color),
      pShowLock: pUser.private || (pIsMe && s.myPrivacy === 'private'),
      pCanView, pLocked: !pCanView,
      pShowEmail: pCanView && !!pUser.email,
      pShowGender: pCanView && !!pUser.gender,
      pGenderLabel: pUser.gender === 'male' ? 'Male' : (pUser.gender === 'female' ? 'Female' : ''),
      pStatPosts: num(pUser.postsCount || 0),
      pStatFollowers: num(pUser.followersCount || 0),
      pStatFollowing: num(pUser.followingCount || 0),
      pTabs,
      pTabPosts: s.profileTab === 'posts', pTabFollowers: s.profileTab === 'followers', pTabFollowing: s.profileTab === 'following',
      pPosts: pPostsRaw.map(p => this.mapPost(p)),
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
      msgProfile: () => this.openDirectChat(s.profileId),
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
      gIsInvited: g.state === 'invited', gIsMember: !gAccessRevoked && g.state === 'member',
	  gCanChat, gCanContent, gContentLocked: !gCanContent,
	  gOpenChat: () => this.openGroupChat(g.id),
      gRequestJoin: () => this.requestGroupJoin(g.id),
      gAcceptInvitation: () => this.acceptGroupInvitation(g.id),
      gDeclineInvitation: () => this.declineGroupInvitation(g.id),
      gLeave: () => this.leaveGroup(g.id),
      gBack: () => this.go('groups'),
      gTabs, gTabPosts: s.groupTab === 'posts', gTabEvents: s.groupTab === 'events', gTabMembers: s.groupTab === 'members',
	  gPosts,
	  groupPostsLoading: s.groupPostsLoading,
	  groupPostsHasError: !!s.groupPostsError, groupPostsError: s.groupPostsError,
	  groupPostsEmpty: !s.groupPostsLoading && !s.groupPostsError && gPosts.length === 0,
	  groupPostsHasMore: !!s.groupPostsNextCursor,
	  groupPostsLoadMoreDisabled: s.groupPostsPending,
	  retryGroupPosts: () => this.loadGroupPosts(g.id, true),
	  loadMoreGroupPosts: () => this.loadGroupPosts(g.id, false),
	  groupPostComposerText: s.groupPostComposerText,
	  onGroupPostComposerText: (event) => this.setState({
		groupPostComposerText: event.target.value, groupPostComposerError: ''
	  }),
	  groupPostComposerFileName: s.groupPostComposerFileName,
	  groupPostComposerHasFile: !!s.groupPostComposerFile,
	  groupPostComposerPending: s.groupPostComposerPending,
	  groupPostComposerHasError: !!s.groupPostComposerError,
	  groupPostComposerError: s.groupPostComposerError,
	  groupPostComposerDisabled: s.groupPostComposerPending || !s.groupPostComposerText.trim(),
	  groupPostComposerButtonLabel: s.groupPostComposerPending ? 'Posting…' : 'Post',
	  pickGroupPostMedia: this.pickGroupPostMedia,
	  onGroupPostMedia: this.onGroupPostMedia,
	  removeGroupPostMedia: this.removeGroupPostMedia,
	  sendGroupPost: this.sendGroupPost,
      gEvents,
      groupEventsLoading: s.groupEventsLoading,
      groupEventsHasError: !!s.groupEventsError,
      groupEventsError: s.groupEventsError,
      groupEventsEmpty: !s.groupEventsLoading && !s.groupEventsError && gEvents.length === 0,
      groupEventsHasMore: !!s.groupEventsNextCursor,
      groupEventsLoadMoreDisabled: s.groupEventsPending,
      retryGroupEvents: () => this.loadGroupEvents(g.id, true),
      loadMoreGroupEvents: () => this.loadGroupEvents(g.id, false),
      groupEventComposerOpen: s.groupEventComposerOpen,
      toggleGroupEventComposer: () => this.setState({
        groupEventComposerOpen: !s.groupEventComposerOpen, groupEventCreateError: ''
      }),
      groupEventTitle: s.groupEventTitle,
      onGroupEventTitle: (event) => this.setState({ groupEventTitle: event.target.value, groupEventCreateError: '' }),
      groupEventDescription: s.groupEventDescription,
      onGroupEventDescription: (event) => this.setState({ groupEventDescription: event.target.value, groupEventCreateError: '' }),
      groupEventStartsAt: s.groupEventStartsAt,
      onGroupEventStartsAt: (event) => this.setState({ groupEventStartsAt: event.target.value, groupEventCreateError: '' }),
      groupEventCreatePending: s.groupEventCreatePending,
      groupEventCreateHasError: !!s.groupEventCreateError,
      groupEventCreateError: s.groupEventCreateError,
      groupEventCreateDisabled,
      groupEventCreateButtonLabel: s.groupEventCreatePending ? 'Creating…' : 'Create event',
      createGroupEvent: this.createGroupEvent,
      gMembers, gRequests, gInvitations,
      gHasRequests: gRequests.length > 0,
      gHasInvitations: gInvitations.length > 0,
      groupMembersLoading: s.groupMembersLoading,
      groupMembersHasError: !!s.groupMembersError, groupMembersError: s.groupMembersError,
      groupMembersHasMore: !!s.groupMembersNextCursor,
      loadMoreGroupMembers: () => this.loadGroupMembers(g.id, false),
      groupRequestsLoading: s.groupRequestsLoading,
      groupRequestsHasError: !!s.groupRequestsError, groupRequestsError: s.groupRequestsError,
      groupRequestsHasMore: !!s.groupRequestsNextCursor,
      loadMoreGroupRequests: () => this.loadGroupRequests(g.id, false),
      groupInvitationsLoading: s.groupInvitationsLoading,
      groupInvitationsHasError: !!s.groupInvitationsError, groupInvitationsError: s.groupInvitationsError,
      groupInvitationsHasMore: !!s.groupInvitationsNextCursor,
      loadMoreGroupInvitations: () => this.loadGroupInvitations(g.id, false),
      inviteOpen: s.inviteOpen && gCanChat,
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
      activeAvatarUrl: am.avatarUrl, activeHasAvatar: am.hasAvatar, activeNoAvatar: am.noAvatar,
      chatHasActive: !!active,
      chatHasNoActive: !active && !s.chatsLoading,
      chatsLoading: s.chatsLoading,
      chatsHasError: !!s.chatsError, chatsError: s.chatsError,
      retryChats: () => this.loadChats(true),
      chatsHasMore: !!s.chatsNextCursor,
      loadMoreChats: () => this.loadChats(false),
      chatsLoadMoreDisabled: s.chatsPending,
      historyLoading: activeHistory.loading,
      historyHasError: !!activeHistory.error, historyError: activeHistory.error,
      retryHistory: () => s.activeChatKey && this.loadChatHistory(s.activeChatKey, true, 'user-open'),
      historyHasMore: !!activeHistory.nextCursor,
      loadMoreHistory: () => s.activeChatKey && this.loadChatHistory(s.activeChatKey, false),
      historyLoadMoreDisabled: activeHistory.pending,
      typing: activeTypingUsers.length > 0,
      typingLabel,
      chatDraft: s.chatDraft,
      onChatDraft: (e) => this.onChatDraft(e.target.value),
      onChatBlur: () => this.stopTyping(),
      onChatKey: (e) => { if (e.key === 'Enter') { e.preventDefault(); this.sendMsg(); } },
      sendMsg: this.sendMsg,
      chatSendDisabled: !active || s.wsStatus !== 'connected' || !s.chatDraft.trim(),
      chatInputDisabled: !active || s.wsStatus !== 'connected',
      chatStatus: s.wsStatus === 'connected' ? 'Live' : (s.wsStatus === 'reconnecting' ? 'Reconnecting…' : 'Connecting…'),
      chatHasError: !!s.chatError, chatError: s.chatError,
      emojiOpen: s.emojiOpen,
      toggleEmoji: () => this.setState({ emojiOpen: !s.emojiOpen }),
      emojis,
      msgRef: (el) => { this.msgEl = el; },
      // notifications
      notifItems,
      notificationsLoading: s.notificationsLoading,
      notificationsEmpty: !s.notificationsLoading && !s.notificationsError && s.notifications.length === 0,
      notificationsHasError: !!s.notificationsError,
      notificationsError: s.notificationsError,
      retryNotifications: () => this.loadNotifications(true),
      notificationsHasMore: !!s.notificationsNextCursor,
      loadMoreNotifications: () => this.loadNotifications(false),
      notificationsLoadMoreDisabled: s.notificationsPending,
      markAllRead: this.markAllNotificationsRead,
      markAllReadDisabled: s.notificationReadAllPending || s.notificationUnreadCount <= 0,
      // rail
      suggestions, railEvents,
      suggestionsHasError: !!s.directoryError,
      suggestionsError: s.directoryError
    };
  }
}

if (typeof module === 'object' && module.exports) module.exports = { Component };
