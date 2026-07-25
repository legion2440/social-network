
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
function formatDateOfBirthInput(value) {
  const digits = String(value || '').replace(/\D/g, '').slice(0, 8);
  if (digits.length <= 2) return digits;
  if (digits.length <= 4) return digits.slice(0, 2) + '-' + digits.slice(2);
  return digits.slice(0, 2) + '-' + digits.slice(2, 4) + '-' + digits.slice(4);
}

function parseDateOfBirth(value) {
  const match = /^(\d{2})-(\d{2})-(\d{4})$/.exec(String(value || ''));
  if (!match) return null;
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  if (year < 1 || month < 1 || month > 12 || day < 1 || day > 31) return null;
  const result = new Date(0);
  result.setUTCFullYear(year, month - 1, day);
  result.setUTCHours(0, 0, 0, 0);
  if (
    result.getUTCFullYear() !== year ||
    result.getUTCMonth() !== month - 1 ||
    result.getUTCDate() !== day
  ) return null;
  return result;
}

function formatDateTimeInput(value) {
  let digits = String(value || '').replace(/\D/g, '').slice(0, 12);
  if (digits.length >= 2) {
    const day = Number(digits.slice(0, 2));
    if (day < 1 || day > 31) digits = digits.slice(0, 1);
  }
  if (digits.length >= 4) {
    const month = Number(digits.slice(2, 4));
    if (month < 1 || month > 12) digits = digits.slice(0, 3);
  }
  if (digits.length >= 10) {
    const hour = Number(digits.slice(8, 10));
    if (hour > 23) digits = digits.slice(0, 9);
  }
  if (digits.length >= 12) {
    const minute = Number(digits.slice(10, 12));
    if (minute > 59) digits = digits.slice(0, 11);
  }
  if (digits.length <= 2) return digits;
  if (digits.length <= 4) return digits.slice(0, 2) + '-' + digits.slice(2);
  if (digits.length <= 8) {
    return digits.slice(0, 2) + '-' + digits.slice(2, 4) + '-' + digits.slice(4);
  }
  if (digits.length <= 10) {
    return digits.slice(0, 2) + '-' + digits.slice(2, 4) + '-' + digits.slice(4, 8) +
      ' ' + digits.slice(8);
  }
  return digits.slice(0, 2) + '-' + digits.slice(2, 4) + '-' + digits.slice(4, 8) +
    ' ' + digits.slice(8, 10) + ':' + digits.slice(10);
}

function parseLocalDateTime(value) {
  const match = /^(\d{2})-(\d{2})-(\d{4}) (\d{2}):(\d{2})$/.exec(String(value || ''));
  if (!match) return null;
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const hour = Number(match[4]);
  const minute = Number(match[5]);
  if (year < 1 || month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59) {
    return null;
  }
  const result = new Date(0);
  result.setFullYear(year, month - 1, day);
  result.setHours(hour, minute, 0, 0);
  if (
    result.getFullYear() !== year || result.getMonth() !== month - 1 || result.getDate() !== day ||
    result.getHours() !== hour || result.getMinutes() !== minute
  ) return null;
  return result;
}

function emptyRegistrationForm() {
  return {
    authEmail: '', authPassword: '',
    regFirstName: '', regLastName: '', regDateOfBirth: '', regGender: '',
    regNickname: '', regAboutMe: '', regAvatar: null, regAvatarName: '',
    regAvatarPreviewURL: ''
  };
}

function emptyProfileEditor() {
  return {
    profileEditOpen: false, profileEditPending: false, profileAvatarPending: false,
    profileEditError: '', editFirstName: '', editLastName: '', editDateOfBirth: '',
    editGender: '', editNickname: '', editAboutMe: '', editAvatar: null, editAvatarName: ''
  };
}

function emptyConfirmationState() {
  return {
    confirmationOpen: false,
    confirmationKind: '',
    confirmationTarget: null,
    confirmationTitle: '',
    confirmationMessage: '',
    confirmationConfirmLabel: 'Confirm',
    confirmationPending: false
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
    activeChatKey: null, mobileChatList: false, messagesByChatKey: {}, onlineUserIDs: {}, typingByChatKey: {},
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

function selectDependencies(source, names) {
  const selected = {};
  names.forEach(name => { selected[name] = source[name]; });
  return selected;
}

function selectAPI(source, names) {
  const selected = {};
  names.forEach(name => {
    selected[name] = (...args) => source[name](...args);
  });
  return selected;
}

function controllerRefs(source, names) {
  const selected = {};
  names.forEach(name => {
    selected[name] = {
      get: () => source[name],
      set: value => { source[name] = value; }
    };
  });
  return selected;
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
      ...emptyProfileEditor(),
      ...emptyConfirmationState()
    };
    this.msgEl = null;
    this.confirmationDialog = null;
    this.confirmationCancelButton = null;
    this.confirmationReturnFocus = null;
    this.confirmationNeedsFocus = false;
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
      if (this.controllers.chat.actions.documentIsVisible() && this.state && this.state.activeChatKey) {
        this.controllers.chat.actions.enqueueChatRead(this.state.activeChatKey);
      }
    };
    this.controllers = typeof createFeatureControllers === 'function'
      ? createFeatureControllers(this.controllerDependencies())
      : null;
  }

  controllerDependencies() {
    const state = Object.freeze({
      get: () => this.state,
      set: (update, callback) => this.setState(update, callback)
    });
    const session = Object.freeze({ users: USERS });
    const values = Object.freeze({ IC, GROUP_COLORS, EMOJIS });
    const helpers = Object.freeze({
      emptyCurrentUser,
      emptyRegistrationForm,
      emptyProfileEditor,
      emptyConfirmationState,
      emptyCommentState,
      emptyGroupPostState,
      emptyGroupEventState,
      emptyNotificationState,
      emptyChatMessages,
      emptyChatState,
      createClientMessageID,
      decorateUser,
      requestErrorMessage,
      formatDateOfBirthInput,
      formatDateTimeInput,
      parseDateOfBirth,
      parseLocalDateTime,
      cover,
      num,
      mergeAPIUsers: (...args) => this.mergeAPIUsers(...args),
      applyAuthUser: (...args) => this.applyAuthUser(...args),
      apiUser: (...args) => this.apiUser(...args),
      mapAPIPost: (...args) => this.mapAPIPost(...args),
      formatPostTime: (...args) => this.formatPostTime(...args),
      openConfirmation: (...args) => this.openConfirmation(...args)
    });
    const action = (feature, name) => (...args) => {
      if (!this.controllers) return undefined;
      return this.controllers[feature].actions[name](...args);
    };
    const callbacks = {
      auth: {
        disposeAllCommentPreviews: action('feed', 'disposeAllCommentPreviews'),
        loadDirectory: action('profile', 'loadDirectory'),
        loadFeed: action('feed', 'loadFeed'),
        loadNotifications: action('notification', 'loadNotifications'),
        loadPostFollowers: action('feed', 'loadPostFollowers'),
        startAuthenticatedRealtime: action('realtime', 'startAuthenticatedRealtime'),
        stopRealtime: action('realtime', 'stopRealtime')
      },
      feed: {
        openGroup: action('groups', 'openGroup'),
        openProfile: action('profile', 'openProfile')
      },
      profile: {
        beginRelationshipGeneration: action('notification', 'beginRelationshipGeneration'),
        loadFeed: action('feed', 'loadFeed'),
        loadPostFollowers: action('feed', 'loadPostFollowers'),
        mergePostCommentsCounts: action('feed', 'mergePostCommentsCounts'),
        openDirectChat: action('chat', 'openDirectChat'),
        purgeCommentStates: action('feed', 'purgeCommentStates'),
        relationshipGeneration: action('notification', 'relationshipGeneration'),
        stopTyping: action('chat', 'stopTyping')
      },
      groups: {
        goGroups: action('router', 'go').bind(null, 'groups'),
        invalidateGroupEventResponses: action('events', 'invalidateGroupEventResponses'),
        loadChats: action('chat', 'loadChats'),
        loadDirectory: action('profile', 'loadDirectory'),
        loadGroupEvents: action('events', 'loadGroupEvents'),
        mergePostCommentsCounts: action('feed', 'mergePostCommentsCounts'),
        openGroupChat: action('chat', 'openGroupChat'),
        openProfile: action('profile', 'openProfile'),
        purgeChat: action('chat', 'purgeChat'),
        purgeCommentStates: action('feed', 'purgeCommentStates'),
        stopTyping: action('chat', 'stopTyping')
      },
      events: {
        groupAccessIsRevoked: action('groups', 'groupAccessIsRevoked'),
        groupGeneration: action('groups', 'groupGeneration'),
        openProfile: action('profile', 'openProfile'),
        revokeGroupAccess: action('groups', 'revokeGroupAccess')
      },
      notification: {
        applyAuthoritativeGroup: action('groups', 'applyAuthoritativeGroup'),
        groupGeneration: action('groups', 'groupGeneration'),
        loadChats: action('chat', 'loadChats'),
        loadDirectory: action('profile', 'loadDirectory'),
        loadFeed: action('feed', 'loadFeed'),
        loadGroupDetail: action('groups', 'loadGroupDetail'),
        loadGroupMembers: action('groups', 'loadGroupMembers'),
        loadGroups: action('groups', 'loadGroups'),
        loadPostFollowers: action('feed', 'loadPostFollowers'),
        openGroup: action('groups', 'openGroup'),
        openProfile: action('profile', 'openProfile'),
        restoreGroupAccess: action('groups', 'restoreGroupAccess')
      },
      chat: {
        goChat: action('router', 'go').bind(null, 'chat'),
        groupAccessIsRevoked: action('groups', 'groupAccessIsRevoked'),
        mapAPIGroup: action('groups', 'mapAPIGroup'),
        mergeGroupResponses: action('groups', 'mergeGroupResponses')
      },
      realtime: {
        applyChatUnreadPayload: action('chat', 'applyChatUnreadPayload'),
        applyNotificationPayload: action('notification', 'applyNotificationPayload'),
        handleRealtimeError: action('chat', 'handleRealtimeError'),
        handleRealtimeMessage: action('chat', 'handleRealtimeMessage'),
        handleTypingUpdate: action('chat', 'handleTypingUpdate'),
        loadChatHistory: action('chat', 'loadChatHistory'),
        loadChats: action('chat', 'loadChats'),
        loadNotifications: action('notification', 'loadNotifications'),
        purgeChat: action('chat', 'purgeChat'),
        revokeGroupAccess: action('groups', 'revokeGroupAccess'),
        stopTyping: action('chat', 'stopTyping')
      }
    };
    const navigation = Object.freeze({
      isApplying: action('router', 'isApplying'),
      applyCurrent: action('router', 'applyCurrent'),
      profile: action('router', 'profile'),
      group: action('router', 'group'),
      directChat: action('router', 'directChat'),
      groupChat: action('router', 'groupChat')
    });
    const navigationNames = {
      auth: ['applyCurrent'],
      feed: [],
      profile: ['isApplying','profile'],
      groups: ['isApplying','group'],
      events: [],
      notification: [],
      chat: ['isApplying','directChat','groupChat'],
      realtime: []
    };
    const gateNames = {
      auth: ['authGate','feedGate','directoryGate','postFollowersGate','profileGate','groupsDirectoryGate','groupInvitationInboxGate','groupDetailGate','groupMembersGate','groupRequestsGate','groupInvitationsGate','groupPostsGate','groupEventsGate','groupEventCreateGate','notificationReadAllGate','notificationsGate','chatsGate','activeChatGate'],
      feed: ['authGate','feedGate','postFollowersGate'],
      profile: ['authGate','directoryGate','profileGate'],
      groups: ['authGate','groupsDirectoryGate','groupInvitationInboxGate','groupDetailGate','groupMembersGate','groupRequestsGate','groupInvitationsGate','groupPostsGate','groupEventsGate','groupEventCreateGate','chatsGate'],
      events: ['authGate','groupEventsGate','groupEventCreateGate'],
      notification: ['authGate','directoryGate','postFollowersGate','profileGate','notificationsGate','notificationReadAllGate'],
      chat: ['authGate','chatsGate','activeChatGate'],
      realtime: ['authGate']
    };
    const resourceNames = {
      auth: ['chatAccessGatesByKey','chatHistoryGatesByKey','chatReadGatesByKey','chatReadInFlightByKey','chatReadSentCandidateByKey','commentAccessGatesByPostID','commentLoadGatesByPostID','groupEventResponseGatesByID','groupGenerationsByID','latestActionableNotificationIDBySourceKey','notificationActionGatesByID','notificationReadGatesByID','relationshipGenerationsByID','revokedChatKeys','revokedGroupAccessIDs'],
      feed: ['commentAccessGatesByPostID','commentLoadGatesByPostID'],
      profile: [],
      groups: ['groupGenerationsByID','revokedGroupAccessIDs'],
      events: ['groupEventResponseGatesByID'],
      notification: ['latestActionableNotificationIDBySourceKey','notificationActionGatesByID','notificationReadGatesByID','relationshipGenerationsByID'],
      chat: ['chatAccessGatesByKey','chatHistoryGatesByKey','chatReadGatesByKey','chatReadInFlightByKey','chatReadSentCandidateByKey','pendingMessageTimers','revokedChatKeys','typingExpiryTimers'],
      realtime: ['pendingMessageTimers','typingExpiryTimers']
    };
    const refNames = {
      auth: [],
      feed: [],
      profile: [],
      groups: [],
      events: [],
      notification: [],
      chat: ['chatScrollAnchor','chatSendLock','msgEl','scrollChatToBottom','typingChatKey','typingHeartbeatTimer','ws'],
      realtime: ['chatSendLock','ws','wsGeneration','wsHasOpened','wsReconnectTimer']
    };
    const apiNames = {
      auth: ['login','logout','me','register'],
      feed: ['createComment','createPost','feed','followers','postComments'],
      profile: ['acceptFollowRequest','deleteAvatar','follow','followRequests','followers','following','rejectFollowRequest','relationship','replaceAvatar','unfollow','updateProfile','userPosts','userProfile','users'],
      groups: ['acceptGroupInvitation','acceptGroupJoinRequest','cancelGroupJoin','createGroup','createGroupPost','declineGroupInvitation','group','groupInvitationInbox','groupInvitations','groupJoinRequests','groupMembers','groupPosts','groups','inviteToGroup','leaveGroup','rejectGroupJoinRequest','requestGroupJoin'],
      events: ['createGroupEvent','groupEvents','respondToGroupEvent'],
      notification: ['actOnNotification','markAllNotificationsRead','markNotificationRead','notifications'],
      chat: ['chats','directMessages','group','groupMessages','markDirectChatRead','markGroupChatRead'],
      realtime: []
    };
    const modelNames = {
      auth: [],
      feed: ['users','posts','comments'],
      profile: ['users','notifications'],
      groups: ['users','posts','chats'],
      events: ['users','events'],
      notification: ['users','notifications'],
      chat: ['users','chats'],
      realtime: ['chats','notifications']
    };
    const helperNames = {
      auth: ['emptyCurrentUser','emptyRegistrationForm','emptyProfileEditor','emptyConfirmationState','emptyGroupPostState','emptyGroupEventState','emptyNotificationState','emptyChatState','requestErrorMessage','formatDateOfBirthInput','parseDateOfBirth','applyAuthUser'],
      feed: ['emptyCommentState','requestErrorMessage','apiUser','formatPostTime','mapAPIPost','mergeAPIUsers'],
      profile: ['emptyProfileEditor','requestErrorMessage','formatDateOfBirthInput','parseDateOfBirth','cover','num','apiUser','applyAuthUser','mapAPIPost','mergeAPIUsers','openConfirmation'],
      groups: ['emptyGroupPostState','emptyGroupEventState','requestErrorMessage','cover','num','apiUser','mapAPIPost','mergeAPIUsers'],
      events: ['requestErrorMessage','formatDateTimeInput','parseLocalDateTime','num','apiUser','mergeAPIUsers'],
      notification: ['requestErrorMessage','formatPostTime','apiUser','mergeAPIUsers'],
      chat: ['emptyChatMessages','createClientMessageID','requestErrorMessage','decorateUser','formatPostTime','num','apiUser','mergeAPIUsers'],
      realtime: []
    };
    const valueNames = {
      auth: [],
      feed: ['IC'],
      profile: ['IC'],
      groups: ['GROUP_COLORS'],
      events: [],
      notification: ['IC'],
      chat: ['GROUP_COLORS','EMOJIS'],
      realtime: []
    };
    const sessionFeatures = {
      auth: true,
      feed: true,
      profile: true,
      groups: true,
      chat: true
    };
    const models = {
      users: UserModel,
      posts: PostModel,
      comments: CommentModel,
      chats: ChatModel,
      events: GroupEventModel,
      notifications: NotificationModel
    };
    const postPresenter = createPostPresenter({
      resolveUser: (...args) => this.apiUser(...args),
      formatDate: (...args) => this.formatPostTime(...args),
      emptyCommentState,
      icons: {
        globe: IC.globe,
        users: IC.users,
        lock: IC.lock
      },
      callbacks: {
        commentMediaInputID: action('feed', 'commentMediaInputID'),
        togglePostComments: action('feed', 'togglePostComments'),
        setCommentDraft: action('feed', 'setCommentDraft'),
        createComment: action('feed', 'createComment'),
        selectCommentMedia: action('feed', 'selectCommentMedia'),
        chooseCommentMedia: action('feed', 'chooseCommentMedia'),
        removeCommentMedia: action('feed', 'removeCommentMedia'),
        loadComments: action('feed', 'loadComments'),
        openProfile: action('profile', 'openProfile'),
        openGroup: action('groups', 'openGroup')
      }
    });
    const dependencies = {};
    Object.keys(apiNames).forEach(feature => {
      dependencies[feature] = {
        state,
        api: selectAPI(AuthAPI, apiNames[feature]),
        models: selectDependencies(models, modelNames[feature]),
        gates: selectDependencies(this, gateNames[feature]),
        resources: {},
        refs: controllerRefs(this, resourceNames[feature].concat(refNames[feature])),
        helpers: selectDependencies(helpers, helperNames[feature]),
        callbacks: callbacks[feature],
        navigation: selectDependencies(navigation, navigationNames[feature]),
        session: sessionFeatures[feature] ? session : {},
        values: selectDependencies(values, valueNames[feature]),
        presenters: ['feed', 'profile', 'groups'].indexOf(feature) >= 0
          ? { post: postPresenter }
          : {}
      };
    });
    dependencies.router = {
      state,
      api: {},
      models: {},
      gates: {},
      resources: {},
      refs: {},
      helpers: {
        usesMobileChatLayout: action('chat', 'usesMobileChatLayout')
      },
      callbacks: {
        stopTyping: action('chat', 'stopTyping'),
        enqueueChatRead: action('chat', 'enqueueChatRead'),
        loadChats: action('chat', 'loadChats'),
        loadNotifications: action('notification', 'loadNotifications'),
        loadGroups: action('groups', 'loadGroups'),
        loadGroupInvitationInbox: action('groups', 'loadGroupInvitationInbox'),
        openProfile: action('profile', 'openProfile'),
        openGroup: action('groups', 'openGroup'),
        openDirectChat: action('chat', 'openDirectChat'),
        openGroupChat: action('chat', 'openGroupChat')
      },
      session: {},
      values: {},
      presenters: {}
    };
    return dependencies;
  }

  componentDidMount() {
    document.documentElement.dataset.theme = this.state.theme;
    this.applyTokens();
    if (document && typeof document.addEventListener === 'function') {
      document.addEventListener('visibilitychange', this.handleVisibilityChange);
    }
    if (this.controllers) this.controllers.router.lifecycle.start();
    if (this.controllers) this.controllers.auth.lifecycle.start();
  }
  componentDidUpdate() {
    this.applyTokens();
    if (this.state.confirmationOpen && this.confirmationNeedsFocus && this.confirmationCancelButton) {
      this.confirmationNeedsFocus = false;
      this.confirmationCancelButton.focus();
    }
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
    this.controllers.auth.actions.revokeRegistrationAvatarPreview(this.state && this.state.regAvatarPreviewURL);
    this.controllers.feed.actions.disposeAllCommentPreviews();
    if (this.controllers) this.controllers.router.lifecycle.stop();
    if (this.controllers) this.controllers.realtime.lifecycle.stop();
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
      groupTitle: normalized.groupTitle,
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

  openConfirmation = (details, triggerElement) => {
    if (!details || this.state.confirmationPending) return;
    this.confirmationReturnFocus = triggerElement ||
      (typeof document !== 'undefined' ? document.activeElement : null);
    this.confirmationNeedsFocus = true;
    this.setState({
      confirmationOpen: true,
      confirmationKind: details.kind,
      confirmationTarget: details.target,
      confirmationTitle: details.title,
      confirmationMessage: details.message,
      confirmationConfirmLabel: details.confirmLabel || 'Confirm',
      confirmationPending: false
    });
  };

  closeConfirmation = () => {
    const returnFocus = this.confirmationReturnFocus;
    this.confirmationReturnFocus = null;
    this.confirmationNeedsFocus = false;
    this.setState(emptyConfirmationState(), () => {
      if (returnFocus && typeof returnFocus.focus === 'function') returnFocus.focus();
    });
  };

  cancelConfirmation = () => {
    if (!this.state.confirmationPending) this.closeConfirmation();
  };

  confirmConfirmation = async () => {
    if (!this.state.confirmationOpen || this.state.confirmationPending) return;
    const kind = this.state.confirmationKind;
    const target = this.state.confirmationTarget;
    this.setState({ confirmationPending: true });
    const succeeded = kind === 'unfollow'
      ? await this.controllers.profile.actions.toggleFollow(target, true)
      : kind === 'privacy'
        ? await this.controllers.profile.actions.setProfilePrivacy(target, true)
        : false;
    if (succeeded) this.closeConfirmation();
    else if (this.state.confirmationOpen) this.setState({ confirmationPending: false });
  };

  handleConfirmationKeyDown = (event) => {
    if (!event || !this.state.confirmationOpen) return;
    if (event.key === 'Escape') {
      event.preventDefault();
      this.cancelConfirmation();
      return;
    }
    if (event.key !== 'Tab' || !this.confirmationDialog) return;
    const focusable = Array.from(this.confirmationDialog.querySelectorAll(
      'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
    ));
    if (focusable.length === 0) {
      event.preventDefault();
      this.confirmationDialog.focus();
      return;
    }
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
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

  renderVals() {
    const state = this.state;
    const me = USERS.me;
    const notificationUnread = state.notificationUnreadCount;
    const chatUnread = Math.max(0, Number(state.chatUnreadCount) || 0);
    const navDefinitions = [
      { key: 'feed', label: 'Home', icon: IC.home, badge: 0 },
      { key: 'profile', label: 'Profile', icon: IC.user, badge: 0 },
      { key: 'groups', label: 'Groups', icon: IC.users, badge: 0 },
      { key: 'chat', label: 'Messages', icon: IC.chat, badge: chatUnread },
      {
        key: 'notifications',
        label: 'Notifications',
        icon: IC.bell,
        badge: notificationUnread
      }
    ];
    const activeKey = state.screen === 'group'
      ? 'groups'
      : (
        state.screen === 'profile' && Number(state.profileId) !== me.apiId
          ? ''
          : state.screen
      );
    const navItems = navDefinitions.map(definition => {
      const active = definition.key === activeKey &&
        !(
          definition.key === 'profile' &&
          Number(state.profileId) !== me.apiId
        );
      return {
        icon: definition.icon,
        label: definition.label,
        bg: active ? 'var(--soft)' : 'transparent',
        color: active ? 'var(--accent)' : 'var(--text2)',
        w: active ? '800' : '600',
        hasBadge: definition.badge > 0,
        badge: num(definition.badge),
        go: () => {
          if (definition.key === 'profile') {
            return this.controllers.profile.actions.openProfile(me.apiId);
          }
          return this.controllers.router.actions.go(definition.key);
        }
      };
    });
    const shell = {
      isAuthChecking: state.authStatus === 'checking',
      isAuthStartupError: state.authStatus === 'error',
      isAuth: state.authStatus === 'anonymous',
      isApp: state.authStatus === 'authenticated',
      isFeed: state.screen === 'feed',
      isProfile: state.screen === 'profile' && state.profileReady,
      isProfileLoading: state.screen === 'profile' && state.profileLoading,
      isProfileError: state.screen === 'profile' &&
        !state.profileLoading && !state.profileReady && !!state.profileError,
      isGroups: state.screen === 'groups',
      isGroup: state.screen === 'group',
      isChat: state.screen === 'chat',
      isNotifs: state.screen === 'notifications',
      rightRail: ['feed', 'profile', 'groups', 'notifications']
        .indexOf(state.screen) >= 0,
      railHeaderSpacer: ['feed', 'groups', 'notifications']
        .indexOf(state.screen) >= 0,
      navItems: navItems,
      me: me,
      themeIcon: state.theme === 'light' ? IC.moon : IC.sun,
      themeLabel: state.theme === 'light' ? 'Dark mode' : 'Light mode',
      toggleTheme: this.toggleTheme,
      goHome: () => this.controllers.router.actions.go('feed'),
      goMyProfile: () => {
        return this.controllers.profile.actions.openProfile(me.apiId);
      },
      appHasError: !!state.appError,
      appError: state.appError,
      confirmationOpen: state.confirmationOpen,
      confirmationTitle: state.confirmationTitle,
      confirmationMessage: state.confirmationMessage,
      confirmationConfirmLabel: state.confirmationPending
        ? 'Please wait…'
        : state.confirmationConfirmLabel,
      confirmationPending: state.confirmationPending,
      cancelConfirmation: this.cancelConfirmation,
      confirmConfirmation: this.confirmConfirmation,
      confirmationKeyDown: this.handleConfirmationKeyDown,
      confirmationDialogRef: element => {
        this.confirmationDialog = element;
      },
      confirmationCancelRef: element => {
        this.confirmationCancelButton = element;
      }
    };

    return mergeViewModels([
      { name: 'shell', values: shell },
      { name: 'auth', values: this.controllers.auth.derived(state) },
      { name: 'feed', values: this.controllers.feed.derived(state) },
      { name: 'profile', values: this.controllers.profile.derived(state) },
      { name: 'groups', values: this.controllers.groups.derived(state) },
      { name: 'events', values: this.controllers.events.derived(state) },
      { name: 'chat', values: this.controllers.chat.derived(state) },
      {
        name: 'notifications',
        values: this.controllers.notification.derived(state)
      },
      { name: 'realtime', values: this.controllers.realtime.derived(state) }
    ], {});
  }
}

if (typeof module === 'object' && module.exports) {
  module.exports = {
    Component,
    formatDateOfBirthInput,
    formatDateTimeInput,
    parseDateOfBirth,
    parseLocalDateTime
  };
}
