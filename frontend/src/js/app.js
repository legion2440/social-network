
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
    const values = Object.freeze({ IC, GROUP_COLORS });
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
        purgeCommentStates: action('feed', 'purgeCommentStates'),
        relationshipGeneration: action('notification', 'relationshipGeneration'),
        stopTyping: action('chat', 'stopTyping')
      },
      groups: {
        invalidateGroupEventResponses: action('events', 'invalidateGroupEventResponses'),
        loadChats: action('chat', 'loadChats'),
        loadGroupEvents: action('events', 'loadGroupEvents'),
        mergePostCommentsCounts: action('feed', 'mergePostCommentsCounts'),
        purgeChat: action('chat', 'purgeChat'),
        purgeCommentStates: action('feed', 'purgeCommentStates'),
        stopTyping: action('chat', 'stopTyping')
      },
      events: {
        groupAccessIsRevoked: action('groups', 'groupAccessIsRevoked'),
        groupGeneration: action('groups', 'groupGeneration'),
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
      profile: ['users'],
      groups: ['users','posts','chats'],
      events: ['users','events'],
      notification: ['users','notifications'],
      chat: ['users','chats'],
      realtime: ['chats','notifications']
    };
    const helperNames = {
      auth: ['emptyCurrentUser','emptyRegistrationForm','emptyProfileEditor','emptyConfirmationState','emptyGroupPostState','emptyGroupEventState','emptyNotificationState','emptyChatState','requestErrorMessage','parseDateOfBirth','applyAuthUser'],
      feed: ['emptyCommentState','requestErrorMessage','num','apiUser','formatPostTime','mapAPIPost','mergeAPIUsers'],
      profile: ['emptyProfileEditor','requestErrorMessage','parseDateOfBirth','apiUser','applyAuthUser','mapAPIPost','mergeAPIUsers','openConfirmation'],
      groups: ['emptyGroupPostState','emptyGroupEventState','requestErrorMessage','mapAPIPost','mergeAPIUsers'],
      events: ['requestErrorMessage','parseLocalDateTime','mergeAPIUsers'],
      notification: ['requestErrorMessage','mergeAPIUsers'],
      chat: ['emptyChatMessages','createClientMessageID','requestErrorMessage','apiUser','mergeAPIUsers'],
      realtime: []
    };
    const valueNames = {
      auth: [],
      feed: ['IC'],
      profile: [],
      groups: ['GROUP_COLORS'],
      events: [],
      notification: [],
      chat: [],
      realtime: []
    };
    const sessionFeatures = {
      auth: true,
      feed: true,
      profile: true,
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
        values: selectDependencies(values, valueNames[feature])
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
      values: {}
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
    const s = this.state;
    const me = USERS.me;
    const featureViewModels = {};
    Object.keys(this.controllers || {}).forEach(name => {
      const controller = this.controllers[name];
      featureViewModels[name] = controller.derived(s);
    });
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
        go: () => { if (n.k === 'profile') this.controllers.profile.actions.openProfile(me.apiId); else this.controllers.router.actions.go(n.k); }
      };
    });

    // feed
    const feedPosts = s.posts.map((p, i) => Object.assign(this.controllers.feed.actions.mapPost(p), { delay: (i * 0.06).toFixed(2) + 's' }));
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
        if (o.k === 'selected') this.controllers.feed.actions.loadPostFollowers();
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
      const b = this.controllers.profile.actions.followBtn(userID);
      return {
        user: u, showBtn: Number(userID) !== me.apiId,
        btnLabel: b.label, btnBg: b.bg, btnColor: b.color, btnBd: b.bd,
        btnDisabled: b.disabled,
        onBtn: (event) => this.controllers.profile.actions.toggleFollow(userID, false, event && event.currentTarget),
        message: () => this.controllers.chat.actions.openDirectChat(userID),
        goProfile: () => this.controllers.profile.actions.openProfile(userID)
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
    const fb = this.controllers.profile.actions.followBtn(s.profileId || 0);
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
      pick: (event) => this.controllers.profile.actions.setProfilePrivacy(o.k, false, event && event.currentTarget)
    }));

    // groups
    const groupCards = s.groupIDs.map(groupID => s.apiGroupsByID[String(groupID)]).filter(Boolean).map((g, i) => {
      const pending = !!s.groupMutationPendingByID[String(g.id)];
      const accessRevoked = this.controllers.groups.actions.groupAccessIsRevoked(g.id);
      return {
        name: g.name, desc: g.desc, membersLabel: num(g.members), cover: cover(g.color),
        owner: this.apiUser(g.ownerID),
        delay: (i * 0.05).toFixed(2) + 's', pending,
        error: s.groupMutationErrorByID[String(g.id)] || '', hasError: !!s.groupMutationErrorByID[String(g.id)],
        isJoined: !accessRevoked && (g.state === 'member' || g.state === 'owner'),
        isOwner: !accessRevoked && g.state === 'owner',
        isMember: !accessRevoked && g.state === 'member', isNone: g.state === 'none',
        isRequested: g.state === 'requested', isInvited: g.state === 'invited',
        open: () => this.controllers.groups.actions.openGroup(g.id),
        join: () => this.controllers.groups.actions.requestGroupJoin(g.id),
        leave: () => this.controllers.groups.actions.leaveGroup(g.id),
        acceptInvite: () => this.controllers.groups.actions.acceptGroupInvitation(g.id),
        declineInvite: () => this.controllers.groups.actions.declineGroupInvitation(g.id)
      };
    });
    const groupInboxCards = s.groupInvitationInbox.map(item => {
      const g = s.apiGroupsByID[String(item.group.id)] || item.group;
      const pending = !!s.groupMutationPendingByID[String(g.id)];
      return {
        name: g.name, owner: this.apiUser(g.ownerID), pending,
        accept: () => this.controllers.groups.actions.acceptGroupInvitation(g.id),
        decline: () => this.controllers.groups.actions.declineGroupInvitation(g.id),
        open: () => this.controllers.groups.actions.openGroup(g.id)
      };
    });

    const g = s.apiGroupsByID[String(Number(s.groupId))] || {
      id: Number(s.groupId) || 0, name: '', desc: '', members: 0, state: 'none', ownerID: 0, color: GROUP_COLORS[0]
    };
    const gAccessRevoked = this.controllers.groups.actions.groupAccessIsRevoked(g.id);
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
      goProfile: () => this.controllers.profile.actions.openProfile(member.userID)
    }));
    const gRequests = (gIsOwner ? s.groupRequests : []).map(request => ({
      user: this.apiUser(request.userID), disabled: gMutationPending,
      pending: true, done: false, doneLabel: '',
      accept: () => this.controllers.groups.actions.acceptGroupRequest(g.id, request.userID),
      decline: () => this.controllers.groups.actions.rejectGroupRequest(g.id, request.userID)
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
	const gPosts = s.groupPosts.map((post, index) => Object.assign(this.controllers.feed.actions.mapPost(post), {
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
        goProfile: () => this.controllers.profile.actions.openProfile(event.creatorID),
        going: () => this.controllers.events.actions.respondToGroupEvent(event.id, 'going'),
        notGoing: () => this.controllers.events.actions.respondToGroupEvent(event.id, 'not_going')
      };
    });
    const groupEventStartsAtDate = parseLocalDateTime(s.groupEventStartsAt);
    const groupEventCreateDisabled = s.groupEventCreatePending || !s.groupEventTitle.trim() ||
      !s.groupEventDescription.trim() || !groupEventStartsAtDate;

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
        open: () => {
          const target = ChatModel.parseChatKey(key);
          if (target && target.kind === 'direct') this.controllers.chat.actions.openDirectChat(target.target_id);
          else if (target && target.kind === 'group') this.controllers.chat.actions.openGroupChat(target.target_id);
        }
      };
    });
    const active = s.activeChatKey ? s.chatsByKey[s.activeChatKey] : null;
    const am = chatMeta(active);
    const activeHistory = s.activeChatKey ? this.controllers.chat.actions.chatMessages(s.activeChatKey) : emptyChatMessages();
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
        retry: () => this.controllers.chat.actions.retryMessage(msg.clientMessageID)
      };
    });
    const activeTypingUsers = Object.values(s.typingByChatKey[s.activeChatKey] || {});
    const typingLabel = activeTypingUsers.length > 1
      ? activeTypingUsers.map(user => user.name).join(', ') + ' are typing'
      : (activeTypingUsers[0] ? activeTypingUsers[0].name + ' is typing' : '');
    const emojis = EMOJIS.map(ch => ({
      ch,
      add: () => this.controllers.chat.actions.onChatDraft(this.state.chatDraft + ch)
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
        accept: () => this.controllers.notification.actions.actOnNotification(notification.id, 'accept'),
        decline: () => this.controllers.notification.actions.actOnNotification(notification.id, 'decline'),
        open: () => this.controllers.notification.actions.openNotification(notification),
        goProfile: event => {
          if (event && typeof event.stopPropagation === 'function') event.stopPropagation();
          this.controllers.notification.actions.markNotificationRead(notification.id);
          this.controllers.profile.actions.openProfile(notification.actorID);
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
        const b = this.controllers.profile.actions.followBtn(user.apiId);
        return {
          user, isPrivate: user.private,
          btnLabel: b.label, btnBg: b.bg, btnColor: b.color, btnBd: b.bd, btnDisabled: b.disabled,
          onBtn: (event) => this.controllers.profile.actions.toggleFollow(user.apiId, false, event && event.currentTarget),
          canMessage: user.relationship && (user.relationship.status === 'accepted' || user.relationship.follows_me),
          message: () => this.controllers.chat.actions.openDirectChat(user.apiId),
          goProfile: () => this.controllers.profile.actions.openProfile(user.apiId)
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
      pick: () => this.controllers.auth.actions.setAuthMode(t.k)
    }));

    return Object.assign(
      {},
      featureViewModels.auth,
      featureViewModels.feed,
      featureViewModels.profile,
      featureViewModels.groups,
      featureViewModels.events,
      featureViewModels.chat,
      featureViewModels.notification,
      featureViewModels.realtime,
      featureViewModels.router,
      {
      // shell
      isAuthChecking: s.authStatus === 'checking', isAuthStartupError: s.authStatus === 'error',
      isAuth: s.authStatus === 'anonymous', isApp: s.authStatus === 'authenticated',
      isFeed: s.screen === 'feed',
      isProfile: s.screen === 'profile' && s.profileReady,
      isProfileLoading: s.screen === 'profile' && s.profileLoading,
      isProfileError: s.screen === 'profile' && !s.profileLoading && !s.profileReady && !!s.profileError,
      profileError: s.profileError,
      retryProfile: () => this.controllers.profile.actions.openProfile(s.profileId),
      isGroups: s.screen === 'groups',
      isGroup: s.screen === 'group', isChat: s.screen === 'chat', isNotifs: s.screen === 'notifications',
      rightRail: ['feed', 'profile', 'groups', 'notifications'].indexOf(s.screen) >= 0,
      railHeaderSpacer: ['feed', 'groups', 'notifications'].indexOf(s.screen) >= 0,
      navItems, me,
      themeIcon: s.theme === 'light' ? IC.moon : IC.sun,
      themeLabel: s.theme === 'light' ? 'Dark mode' : 'Light mode',
      toggleTheme: this.toggleTheme,
      goHome: () => this.controllers.router.actions.go('feed'),
      goMyProfile: () => this.controllers.profile.actions.openProfile(me.apiId),
      goLogout: this.controllers.auth.actions.logout,
      logoutDisabled: s.logoutPending,
      appHasError: !!s.appError, appError: s.appError,
      confirmationOpen: s.confirmationOpen,
      confirmationTitle: s.confirmationTitle,
      confirmationMessage: s.confirmationMessage,
      confirmationConfirmLabel: s.confirmationPending ? 'Please wait…' : s.confirmationConfirmLabel,
      confirmationPending: s.confirmationPending,
      cancelConfirmation: this.cancelConfirmation,
      confirmConfirmation: this.confirmConfirmation,
      confirmationKeyDown: this.handleConfirmationKeyDown,
      confirmationDialogRef: (element) => { this.confirmationDialog = element; },
      confirmationCancelRef: (element) => { this.confirmationCancelButton = element; },
      // auth
      authTabs, authIsLogin: s.authMode === 'login', authIsReg: s.authMode === 'register',
      authCta: s.authPending ? 'Please wait…' : (s.authMode === 'login' ? 'Sign in' : 'Create account'),
      authDisabled: s.authPending,
      authButtonOpacity: s.authPending ? '0.65' : '1',
      authButtonCursor: s.authPending ? 'wait' : 'pointer',
      authHasError: !!s.authError, authError: s.authError,
      bootstrapError: s.bootstrapError, retryAuthBootstrap: this.controllers.auth.actions.loadCurrentUser,
      authEmail: s.authEmail, onAuthEmail: (e) => this.setState({ authEmail: e.target.value }),
      authPassword: s.authPassword, onAuthPassword: (e) => this.setState({ authPassword: e.target.value }),
      regFirstName: s.regFirstName, onRegFirstName: (e) => this.setState({ regFirstName: e.target.value }),
      regLastName: s.regLastName, onRegLastName: (e) => this.setState({ regLastName: e.target.value }),
      regDateOfBirth: s.regDateOfBirth, onRegDateOfBirth: (e) => this.setState({ regDateOfBirth: formatDateOfBirthInput(e.target.value) }),
      regGender: s.regGender, onRegGender: (e) => this.setState({ regGender: e.target.value }),
      regNickname: s.regNickname, onRegNickname: (e) => this.setState({ regNickname: e.target.value }),
      regAboutMe: s.regAboutMe, onRegAboutMe: (e) => this.setState({ regAboutMe: e.target.value }),
      avatarButtonLabel: s.regAvatarName || 'avatar',
      registrationAvatarHasPreview: !!s.regAvatarPreviewURL,
      registrationAvatarMissingPreview: !s.regAvatarPreviewURL,
      registrationAvatarPreviewURL: s.regAvatarPreviewURL,
      pickRegistrationAvatar: this.controllers.auth.actions.pickRegistrationAvatar,
      onRegistrationAvatar: this.controllers.auth.actions.onRegistrationAvatar,
      submitAuth: this.controllers.auth.actions.submitAuth,
      // feed
      feedLoading: s.feedLoading, feedReady: !s.feedLoading,
      feedHasError: !!s.feedError, feedError: s.feedError,
      retryFeed: () => this.controllers.feed.actions.loadFeed(true),
      feedHasMore: !!s.feedNextCursor,
      feedLoadMore: () => this.controllers.feed.actions.loadFeed(false),
      feedLoadMoreLabel: s.feedPending && !s.feedLoading ? 'Loading…' : 'Load more',
      feedLoadMoreDisabled: s.feedPending,
      posts: feedPosts,
      composerText: s.composerText,
      onComposer: (e) => this.setState({ composerText: e.target.value, composerError: '' }),
      composerHasMedia: !!s.composerFile,
      composerMediaName: s.composerFileName,
      pickComposerMedia: this.controllers.feed.actions.pickComposerMedia,
      onComposerMedia: this.controllers.feed.actions.onComposerMedia,
      removeComposerMedia: this.controllers.feed.actions.removeComposerMedia,
      composerHasError: !!s.composerError, composerError: s.composerError,
      privacyOpen: s.privacyOpen,
      togglePrivacy: () => this.setState({ privacyOpen: !s.privacyOpen }),
      privacyIcon: privacyMeta[s.privacy].icon,
      privacyLabel: privacyMeta[s.privacy].label,
      privacyOptions,
      privacyIsSelected: s.privacy === 'selected',
      followerChips,
      selectedFollowersEmpty: s.postFollowers.length === 0 && !s.postFollowersLoading,
      postBtnDisabled: s.composerPending || (!s.composerText.trim() && !s.composerFile) || !composerAudienceReady,
      postBtnBg: ((s.composerText.trim() || s.composerFile) && composerAudienceReady && !s.composerPending) ? 'var(--accent)' : 'var(--surface2)',
      postBtnColor: ((s.composerText.trim() || s.composerFile) && composerAudienceReady && !s.composerPending) ? '#fff' : 'var(--text3)',
      postBtnCursor: s.composerPending ? 'wait' : (((s.composerText.trim() || s.composerFile) && composerAudienceReady) ? 'pointer' : 'not-allowed'),
      postButtonLabel: s.composerPending ? 'Posting…' : 'Post',
      sendPost: this.controllers.feed.actions.sendPost,
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
      pPosts: pPostsRaw.map(p => this.controllers.feed.actions.mapPost(p)),
      pNoPosts: !s.profilePostsLoading && !s.profilePostsError && pPostsRaw.length === 0,
      pPostsLoading: s.profilePostsLoading,
      pPostsHasError: !!s.profilePostsError,
      pPostsError: s.profilePostsError,
      retryProfilePosts: () => this.controllers.profile.actions.loadProfilePosts(s.profileId, true),
      pPostsHasMore: !!s.profilePostsNextCursor,
      loadMoreProfilePosts: () => this.controllers.profile.actions.loadProfilePosts(s.profileId, false),
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
      onFollow: (event) => this.controllers.profile.actions.toggleFollow(s.profileId, false, event && event.currentTarget),
      msgProfile: () => this.controllers.chat.actions.openDirectChat(s.profileId),
      privacySeg,
      profilePrivacyHasError: pIsMe && !!s.profilePrivacyError,
      profilePrivacyError: s.profilePrivacyError,
      showProfileEdit: pIsMe && s.profileEditOpen,
      openProfileEdit: this.controllers.profile.actions.openProfileEdit,
      cancelProfileEdit: this.controllers.profile.actions.cancelProfileEdit,
      saveProfile: this.controllers.profile.actions.saveProfile,
      profileEditPending: s.profileEditPending || s.profileAvatarPending || s.profilePrivacyPending,
      profileSaveLabel: s.profileEditPending ? 'Saving…' : 'Save changes',
      profileEditHasError: !!s.profileEditError,
      profileEditError: s.profileEditError,
      editFirstName: s.editFirstName, onEditFirstName: (e) => this.setState({ editFirstName: e.target.value }),
      editLastName: s.editLastName, onEditLastName: (e) => this.setState({ editLastName: e.target.value }),
      editDateOfBirth: s.editDateOfBirth, onEditDateOfBirth: (e) => this.setState({ editDateOfBirth: formatDateOfBirthInput(e.target.value) }),
      editGender: s.editGender, onEditGender: (e) => this.setState({ editGender: e.target.value }),
      editNickname: s.editNickname, onEditNickname: (e) => this.setState({ editNickname: e.target.value }),
      editAboutMe: s.editAboutMe, onEditAboutMe: (e) => this.setState({ editAboutMe: e.target.value }),
      profileAvatarLabel: s.editAvatarName || 'Choose image',
      profileAvatarPending: s.profileAvatarPending || s.profileEditPending || s.profilePrivacyPending,
      profileAvatarUploadDisabled: s.profileAvatarPending || s.profileEditPending || s.profilePrivacyPending || !s.editAvatar,
      profileAvatarUploadOpacity: s.profileAvatarPending || s.profileEditPending || s.profilePrivacyPending || !s.editAvatar ? '0.55' : '1',
      profileAvatarUploadLabel: s.profileAvatarPending ? 'Working…' : 'Upload',
      profileHasCustomAvatar: me.hasCustomAvatar,
      pickProfileAvatar: this.controllers.profile.actions.pickProfileAvatar,
      onProfileAvatar: this.controllers.profile.actions.onProfileAvatar,
      replaceProfileAvatar: this.controllers.profile.actions.replaceProfileAvatar,
      deleteProfileAvatar: this.controllers.profile.actions.deleteProfileAvatar,
      // groups
      createOpen: s.createOpen,
      toggleCreate: () => this.setState({ createOpen: !s.createOpen }),
      ngName: s.ngName, onNgName: (e) => this.setState({ ngName: e.target.value }),
      ngDesc: s.ngDesc, onNgDesc: (e) => this.setState({ ngDesc: e.target.value }),
      createGroup: this.controllers.groups.actions.createGroup,
      groupCreatePending: s.groupCreatePending,
      groupCreateHasError: !!s.groupCreateError, groupCreateError: s.groupCreateError,
      groupCards, groupsLoading: s.groupsLoading, groupsReady: !s.groupsLoading,
      groupsHasError: !!s.groupsError, groupsError: s.groupsError,
      retryGroups: () => this.controllers.groups.actions.loadGroups(true), groupsHasMore: !!s.groupsNextCursor,
      loadMoreGroups: () => this.controllers.groups.actions.loadGroups(false),
      groupsLoadMoreLabel: s.groupsPending && !s.groupsLoading ? 'Loading…' : 'Load more',
      groupInboxCards,
      groupInboxLoading: s.groupInvitationInboxLoading,
      groupInboxHasError: !!s.groupInvitationInboxError,
      groupInboxError: s.groupInvitationInboxError,
      groupInboxHasItems: groupInboxCards.length > 0,
      groupInboxHasMore: !!s.groupInvitationInboxNextCursor,
      loadMoreGroupInbox: () => this.controllers.groups.actions.loadGroupInvitationInbox(false),
      // group detail
      groupLoading: s.groupLoading, groupHasError: !!s.groupError, groupError: s.groupError,
      retryGroup: () => this.controllers.groups.actions.openGroup(g.id),
      gName: g.name, gDesc: g.desc, gMembersLabel: num(g.members), gCover: cover(g.color), gIsOwner,
      gOwner: this.apiUser(g.ownerID), gMutationPending,
      gMutationHasError: !!gMutationError, gMutationError,
      gIsNone: g.state === 'none', gIsRequested: g.state === 'requested',
      gIsInvited: g.state === 'invited', gIsMember: !gAccessRevoked && g.state === 'member',
	  gCanChat, gCanContent, gContentLocked: !gCanContent,
	  gOpenChat: () => this.controllers.chat.actions.openGroupChat(g.id),
      gRequestJoin: () => this.controllers.groups.actions.requestGroupJoin(g.id),
      gAcceptInvitation: () => this.controllers.groups.actions.acceptGroupInvitation(g.id),
      gDeclineInvitation: () => this.controllers.groups.actions.declineGroupInvitation(g.id),
      gLeave: () => this.controllers.groups.actions.leaveGroup(g.id),
      gBack: () => this.controllers.router.actions.go('groups'),
      gTabs, gTabPosts: s.groupTab === 'posts', gTabEvents: s.groupTab === 'events', gTabMembers: s.groupTab === 'members',
	  gPosts,
	  groupPostsLoading: s.groupPostsLoading,
	  groupPostsHasError: !!s.groupPostsError, groupPostsError: s.groupPostsError,
	  groupPostsEmpty: !s.groupPostsLoading && !s.groupPostsError && gPosts.length === 0,
	  groupPostsHasMore: !!s.groupPostsNextCursor,
	  groupPostsLoadMoreDisabled: s.groupPostsPending,
	  retryGroupPosts: () => this.controllers.groups.actions.loadGroupPosts(g.id, true),
	  loadMoreGroupPosts: () => this.controllers.groups.actions.loadGroupPosts(g.id, false),
	  groupPostComposerText: s.groupPostComposerText,
	  onGroupPostComposerText: (event) => this.setState({
		groupPostComposerText: event.target.value, groupPostComposerError: ''
	  }),
	  groupPostComposerFileName: s.groupPostComposerFileName,
	  groupPostComposerHasFile: !!s.groupPostComposerFile,
	  groupPostComposerPending: s.groupPostComposerPending,
	  groupPostComposerHasError: !!s.groupPostComposerError,
	  groupPostComposerError: s.groupPostComposerError,
	  groupPostComposerDisabled: s.groupPostComposerPending || (!s.groupPostComposerText.trim() && !s.groupPostComposerFile),
	  groupPostComposerButtonLabel: s.groupPostComposerPending ? 'Posting…' : 'Post',
	  pickGroupPostMedia: this.controllers.groups.actions.pickGroupPostMedia,
	  onGroupPostMedia: this.controllers.groups.actions.onGroupPostMedia,
	  removeGroupPostMedia: this.controllers.groups.actions.removeGroupPostMedia,
	  sendGroupPost: this.controllers.groups.actions.sendGroupPost,
      gEvents,
      groupEventsLoading: s.groupEventsLoading,
      groupEventsHasError: !!s.groupEventsError,
      groupEventsError: s.groupEventsError,
      groupEventsEmpty: !s.groupEventsLoading && !s.groupEventsError && gEvents.length === 0,
      groupEventsHasMore: !!s.groupEventsNextCursor,
      groupEventsLoadMoreDisabled: s.groupEventsPending,
      retryGroupEvents: () => this.controllers.events.actions.loadGroupEvents(g.id, true),
      loadMoreGroupEvents: () => this.controllers.events.actions.loadGroupEvents(g.id, false),
      groupEventComposerOpen: s.groupEventComposerOpen,
      toggleGroupEventComposer: () => this.setState({
        groupEventComposerOpen: !s.groupEventComposerOpen, groupEventCreateError: ''
      }),
      groupEventTitle: s.groupEventTitle,
      onGroupEventTitle: (event) => this.setState({ groupEventTitle: event.target.value, groupEventCreateError: '' }),
      groupEventDescription: s.groupEventDescription,
      onGroupEventDescription: (event) => this.setState({ groupEventDescription: event.target.value, groupEventCreateError: '' }),
      groupEventStartsAt: s.groupEventStartsAt,
      onGroupEventStartsAt: (event) => this.setState({
        groupEventStartsAt: formatDateTimeInput(event.target.value),
        groupEventCreateError: ''
      }),
      groupEventCreatePending: s.groupEventCreatePending,
      groupEventCreateHasError: !!s.groupEventCreateError,
      groupEventCreateError: s.groupEventCreateError,
      groupEventCreateDisabled,
      groupEventCreateButtonLabel: s.groupEventCreatePending ? 'Creating…' : 'Create event',
      createGroupEvent: this.controllers.events.actions.createGroupEvent,
      gMembers, gRequests, gInvitations,
      gHasRequests: gRequests.length > 0,
      gHasInvitations: gInvitations.length > 0,
      groupMembersLoading: s.groupMembersLoading,
      groupMembersHasError: !!s.groupMembersError, groupMembersError: s.groupMembersError,
      groupMembersHasMore: !!s.groupMembersNextCursor,
      loadMoreGroupMembers: () => this.controllers.groups.actions.loadGroupMembers(g.id, false),
      groupRequestsLoading: s.groupRequestsLoading,
      groupRequestsHasError: !!s.groupRequestsError, groupRequestsError: s.groupRequestsError,
      groupRequestsHasMore: !!s.groupRequestsNextCursor,
      loadMoreGroupRequests: () => this.controllers.groups.actions.loadGroupRequests(g.id, false),
      groupInvitationsLoading: s.groupInvitationsLoading,
      groupInvitationsHasError: !!s.groupInvitationsError, groupInvitationsError: s.groupInvitationsError,
      groupInvitationsHasMore: !!s.groupInvitationsNextCursor,
      loadMoreGroupInvitations: () => this.controllers.groups.actions.loadGroupInvitations(g.id, false),
      inviteOpen: s.inviteOpen && gCanChat,
      toggleInvite: () => {
        const opening = !s.inviteOpen;
        this.setState({ inviteOpen: opening });
        if (opening && !s.directoryUserIDs.length) this.controllers.profile.actions.loadDirectory(true);
      },
      inviteCandidates,
      inviteSendDisabled: gMutationPending || !s.groupInviteUserID,
      inviteSelectedUser: this.controllers.groups.actions.inviteSelectedUser,
      inviteLoadMore: () => this.controllers.profile.actions.loadDirectory(false),
      inviteHasMore: !!s.directoryNextCursor,
      inviteDirectoryLoading: s.directoryLoading,
      // chat
      convos, messages,
      activeTitle: am.title, activeSub: am.sub, activeInitials: am.initials, activeColor: am.color,
      activeAvatarUrl: am.avatarUrl, activeHasAvatar: am.hasAvatar, activeNoAvatar: am.noAvatar,
      chatHasActive: !!active,
      chatHasNoActive: !active && !s.chatsLoading,
      chatLayoutClass: active ? 'chat-active' : '',
      backToChats: () => this.controllers.router.actions.go('chat'),
      chatsLoading: s.chatsLoading,
      chatsHasError: !!s.chatsError, chatsError: s.chatsError,
      retryChats: () => this.controllers.chat.actions.loadChats(true),
      chatsHasMore: !!s.chatsNextCursor,
      loadMoreChats: () => this.controllers.chat.actions.loadChats(false),
      chatsLoadMoreDisabled: s.chatsPending,
      historyLoading: activeHistory.loading,
      historyHasError: !!activeHistory.error, historyError: activeHistory.error,
      retryHistory: () => s.activeChatKey && this.controllers.chat.actions.loadChatHistory(s.activeChatKey, true, 'user-open'),
      historyHasMore: !!activeHistory.nextCursor,
      loadMoreHistory: () => s.activeChatKey && this.controllers.chat.actions.loadChatHistory(s.activeChatKey, false),
      historyLoadMoreDisabled: activeHistory.pending,
      typing: activeTypingUsers.length > 0,
      typingLabel,
      chatDraft: s.chatDraft,
      onChatDraft: (e) => this.controllers.chat.actions.onChatDraft(e.target.value),
      onChatBlur: () => this.controllers.chat.actions.stopTyping(),
      onChatKey: (e) => { if (e.key === 'Enter') { e.preventDefault(); this.controllers.chat.actions.sendMsg(); } },
      sendMsg: this.controllers.chat.actions.sendMsg,
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
      retryNotifications: () => this.controllers.notification.actions.loadNotifications(true),
      notificationsHasMore: !!s.notificationsNextCursor,
      loadMoreNotifications: () => this.controllers.notification.actions.loadNotifications(false),
      notificationsLoadMoreDisabled: s.notificationsPending,
      markAllRead: this.controllers.notification.actions.markAllNotificationsRead,
      markAllReadDisabled: s.notificationReadAllPending || s.notificationUnreadCount <= 0,
      // rail
      suggestions, railEvents,
      suggestionsHasError: !!s.directoryError,
      suggestionsError: s.directoryError
      }
    );
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
