(function (root, factory) {
  var create = factory(root && root.createFeatureController, root && root.createControllerContext);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createAuthController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController, createContext) {
  return function createAuthController(dependencies) {
    var context = createContext(dependencies);
    var AuthAPI = dependencies.api;
    var USERS = dependencies.session.users;
    var emptyCurrentUser = dependencies.helpers.emptyCurrentUser;
    var emptyRegistrationForm = dependencies.helpers.emptyRegistrationForm;
    var emptyProfileEditor = dependencies.helpers.emptyProfileEditor;
    var emptyConfirmationState = dependencies.helpers.emptyConfirmationState;
    var emptyGroupPostState = dependencies.helpers.emptyGroupPostState;
    var emptyGroupEventState = dependencies.helpers.emptyGroupEventState;
    var emptyNotificationState = dependencies.helpers.emptyNotificationState;
    var emptyChatState = dependencies.helpers.emptyChatState;
    var requestErrorMessage = dependencies.helpers.requestErrorMessage;
    var parseDateOfBirth = dependencies.helpers.parseDateOfBirth;

  context.revokeRegistrationAvatarPreview = function (previewURL) {
    if (
      previewURL &&
      typeof URL !== 'undefined' &&
      URL &&
      typeof URL.revokeObjectURL === 'function'
    ) {
      URL.revokeObjectURL(previewURL);
    }
  };

  context.loadCurrentUser = async () => {
    const authGeneration = context.authGate.begin();
    context.stopRealtime();
    context.setState({ authStatus: 'checking', bootstrapError: '', appError: '' });
    try {
      const user = await AuthAPI.me();
      if (!context.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = context.applyAuthUser(user);
      context.setState({
        authStatus: 'authenticated', screen: 'feed',
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public',
        profilePrivacyPending: false, profilePrivacyError: ''
      }, () => {
        context.startAuthenticatedRealtime(authGeneration);
        if (dependencies.navigation) dependencies.navigation.applyCurrent();
      });
      context.loadFeed(true);
      context.loadPostFollowers();
      context.loadDirectory();
      context.loadNotifications(true);
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration)) return;
      if (error && error.status === 401) {
        USERS.me = emptyCurrentUser();
        context.setState({ authStatus: 'anonymous', screen: 'auth', bootstrapError: '' });
        return;
      }
      context.setState({
        authStatus: 'error',
        bootstrapError: requestErrorMessage(error, 'Could not load your session. Please try again.')
      });
    }
  };

  context.setAuthMode = (mode) => context.setState({ authMode: mode, authError: '' });

  context.pickRegistrationAvatar = () => {
    const input = document.getElementById('registration-avatar');
    if (input) input.click();
  };

  context.onRegistrationAvatar = (event) => {
    const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
    context.revokeRegistrationAvatarPreview(context.state.regAvatarPreviewURL);
    const previewURL = (
      file &&
      typeof URL !== 'undefined' &&
      URL &&
      typeof URL.createObjectURL === 'function'
    ) ? URL.createObjectURL(file) : '';
    context.setState({
      regAvatar: file,
      regAvatarName: file ? file.name : '',
      regAvatarPreviewURL: previewURL
    });
  };

  context.submitAuth = async (event) => {
    if (event) event.preventDefault();
    if (context.state.authPending) return;

    const s = context.state;
    if (s.authMode === 'register' && !parseDateOfBirth(s.regDateOfBirth.trim())) {
      context.setState({ authError: 'Enter a real calendar date as DD-MM-YYYY.' });
      return;
    }
    const authGeneration = context.authGate.begin();
    context.setState({ authPending: true, authError: '' });
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

      if (!context.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = context.applyAuthUser(user);
      const authenticatedState = {
        authStatus: 'authenticated', authPending: false, authError: '',
        authPassword: '', screen: 'feed',
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public',
        profilePrivacyPending: false, profilePrivacyError: ''
      };
      Object.assign(authenticatedState, emptyNotificationState());
      const registrationAvatarPreviewURL = s.authMode === 'register' ? s.regAvatarPreviewURL : '';
      if (s.authMode === 'register') Object.assign(authenticatedState, emptyRegistrationForm());
      Object.assign(authenticatedState, emptyProfileEditor());
      context.setState(authenticatedState, () => {
        context.revokeRegistrationAvatarPreview(registrationAvatarPreviewURL);
        context.startAuthenticatedRealtime(authGeneration);
        if (dependencies.navigation) dependencies.navigation.applyCurrent();
      });
      context.loadFeed(true);
      context.loadPostFollowers();
      context.loadDirectory();
      context.loadNotifications(true);
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration)) return;
      context.setState({
        authPending: false,
        authError: requestErrorMessage(error, 'Authentication failed. Please try again.')
      });
    }
  };

  context.logout = async () => {
    if (context.state.logoutPending) return;
    const authGeneration = context.authGate.current();
    context.setState({ logoutPending: true, appError: '' });
    try {
      await AuthAPI.logout();
      if (!context.authGate.isCurrent(authGeneration)) return;
      context.authGate.begin();
      context.feedGate.begin();
      context.directoryGate.begin();
      context.postFollowersGate.begin();
      context.profileGate.begin();
      context.groupsDirectoryGate.begin();
      context.groupInvitationInboxGate.begin();
      context.groupDetailGate.begin();
      context.groupMembersGate.begin();
      context.groupRequestsGate.begin();
      context.groupInvitationsGate.begin();
	  context.groupPostsGate.begin();
      context.groupEventsGate.begin();
      context.groupEventCreateGate.begin();
      Object.keys(context.groupEventResponseGatesByID).forEach(key => context.groupEventResponseGatesByID[key].begin());
      context.groupEventResponseGatesByID = {};
      context.notificationsGate.begin();
      context.notificationReadAllGate.begin();
      Object.keys(context.notificationReadGatesByID).forEach(key => context.notificationReadGatesByID[key].begin());
      Object.keys(context.notificationActionGatesByID).forEach(key => context.notificationActionGatesByID[key].begin());
      context.notificationReadGatesByID = {};
      context.notificationActionGatesByID = {};
      Object.keys(context.relationshipGenerationsByID).forEach(key => context.relationshipGenerationsByID[key].begin());
      context.relationshipGenerationsByID = {};
      context.latestActionableNotificationIDBySourceKey = {};
      context.chatsGate.begin();
      context.activeChatGate.begin();
      Object.keys(context.chatHistoryGatesByKey).forEach(key => context.chatHistoryGatesByKey[key].begin());
      Object.keys(context.chatAccessGatesByKey).forEach(key => context.chatAccessGatesByKey[key].begin());
      Object.keys(context.chatReadGatesByKey).forEach(key => context.chatReadGatesByKey[key].begin());
      context.chatHistoryGatesByKey = {};
      context.chatAccessGatesByKey = {};
      context.chatReadGatesByKey = {};
      context.chatReadInFlightByKey = {};
      context.chatReadSentCandidateByKey = {};
      context.revokedChatKeys.clear();
      context.revokedGroupAccessIDs.clear();
      context.stopRealtime();
      context.revokeRegistrationAvatarPreview(context.state.regAvatarPreviewURL);
      context.disposeAllCommentPreviews();
      Object.keys(context.groupGenerationsByID).forEach(key => context.groupGenerationsByID[key].begin());
      context.groupGenerationsByID = {};
      Object.keys(context.commentAccessGatesByPostID).forEach(key => context.commentAccessGatesByPostID[key].begin());
      context.commentAccessGatesByPostID = {};
      context.commentLoadGatesByPostID = {};
      USERS.me = emptyCurrentUser();
      context.setState(Object.assign({
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
	  }, emptyGroupPostState(), emptyGroupEventState(), emptyNotificationState(), emptyChatState(), emptyRegistrationForm(), emptyProfileEditor(), emptyConfirmationState()));
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration)) return;
      context.setState({
        logoutPending: false,
        appError: requestErrorMessage(error, 'Could not log out. Please try again.')
      });
    }
  };

    return createController('auth', dependencies, {
      revokeRegistrationAvatarPreview: context.revokeRegistrationAvatarPreview,
      loadCurrentUser: context.loadCurrentUser,
      setAuthMode: context.setAuthMode,
      pickRegistrationAvatar: context.pickRegistrationAvatar,
      onRegistrationAvatar: context.onRegistrationAvatar,
      submitAuth: context.submitAuth,
      logout: context.logout
    }, function (state) {
      return { authenticated: state && state.authStatus === 'authenticated', checking: !state || state.authStatus === 'checking' };
    }, { start: context.loadCurrentUser });
  };
});
