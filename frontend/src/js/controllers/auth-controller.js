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
    var emptyOAuthState = dependencies.helpers.emptyOAuthState;
    var emptyProfileEditor = dependencies.helpers.emptyProfileEditor;
    var emptyConfirmationState = dependencies.helpers.emptyConfirmationState;
    var emptyGroupPostState = dependencies.helpers.emptyGroupPostState;
    var emptyGroupEventState = dependencies.helpers.emptyGroupEventState;
    var emptyNotificationState = dependencies.helpers.emptyNotificationState;
    var emptyChatState = dependencies.helpers.emptyChatState;
    var requestErrorMessage = dependencies.helpers.requestErrorMessage;
    var parseDateOfBirth = dependencies.helpers.parseDateOfBirth;
    var formatDateOfBirthInput = dependencies.helpers.formatDateOfBirthInput;
    var environment = dependencies.environment ||
      (typeof window !== 'undefined' ? window : null);

  context.oauthResetState = function () {
    var reset = emptyOAuthState();
    reset.oauthProviders = context.state.oauthProviders || [];
    reset.oauthProvidersLoading = false;
    return reset;
  };

  context.oauthErrorMessage = function (code) {
    var messages = {
      oauth_provider_unavailable: 'GitHub sign-in is not available right now.',
      oauth_provider_error: 'GitHub sign-in was cancelled or could not be completed.',
      oauth_state_invalid: 'This GitHub sign-in link is invalid or has already been used. Please try again.',
      oauth_code_missing: 'GitHub did not return an authorization code. Please try again.',
      oauth_token_exchange_failed: 'GitHub sign-in could not be completed. Please try again.',
      oauth_identity_fetch_failed: 'Your GitHub account details could not be loaded.',
      oauth_verified_email_unavailable: 'GitHub must provide a verified email address to continue.',
      oauth_email_already_registered: 'This verified email already has a local account. Sign in with email and password.',
      oauth_flow_expired: 'This GitHub registration link has expired. Please start again.',
      oauth_identity_conflict: 'This GitHub account is already connected. Please start sign-in again.'
    };
    return messages[code] || (code ? 'GitHub sign-in could not be completed. Please try again.' : '');
  };

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
        context.setState(
          { authStatus: 'anonymous', screen: 'auth', bootstrapError: '' },
          function () {
            if (dependencies.navigation) dependencies.navigation.applyCurrent();
          }
        );
        return;
      }
      context.setState({
        authStatus: 'error',
        bootstrapError: requestErrorMessage(error, 'Could not load your session. Please try again.')
      });
    }
  };

  context.loadOAuthProviders = async () => {
    context.setState({ oauthProvidersLoading: true });
    try {
      const response = await AuthAPI.oauthProviders();
      context.setState({
        oauthProviders: response && Array.isArray(response.providers) ? response.providers : [],
        oauthProvidersLoading: false
      });
    } catch (error) {
      context.setState({
        oauthProviders: [],
        oauthProvidersLoading: false
      });
    }
  };

  context.startGitHubOAuth = () => {
    if (
      environment &&
      environment.location &&
      typeof environment.location.assign === 'function'
    ) {
      environment.location.assign('/api/auth/oauth/github/start?next=/');
      return;
    }
    context.setState({ oauthError: 'GitHub sign-in could not be started.' });
  };

  context.showOAuthCompletion = token => {
    context.revokeRegistrationAvatarPreview(context.state.oauthAvatarPreviewURL);
    context.setState(Object.assign(context.oauthResetState(), {
      authStatus: 'anonymous',
      screen: 'auth',
      oauthCompletionActive: true,
      oauthFlowToken: String(token || ''),
      oauthFlowLoading: true
    }), () => context.loadOAuthFlow(token));
  };

  context.showLoginOAuthError = code => {
    context.revokeRegistrationAvatarPreview(context.state.oauthAvatarPreviewURL);
    context.setState(Object.assign(context.oauthResetState(), {
      authStatus: 'anonymous',
      screen: 'auth',
      authMode: 'login',
      oauthError: context.oauthErrorMessage(String(code || ''))
    }));
  };

  context.loadOAuthFlow = async token => {
    token = String(token || '').trim();
    if (!token) {
      context.setState({
        oauthFlowLoading: false,
        oauthCompletionError: context.oauthErrorMessage('oauth_flow_expired')
      });
      return;
    }
    context.setState({
      oauthFlowToken: token,
      oauthFlowLoading: true,
      oauthCompletionError: ''
    });
    try {
      const flow = await AuthAPI.oauthFlow(token);
      if (context.state.oauthFlowToken !== token) return;
      context.setState({
        oauthFlow: flow,
        oauthFlowLoading: false,
        oauthFirstName: flow.suggested_first_name || '',
        oauthLastName: flow.suggested_last_name || '',
        oauthNickname: flow.suggested_nickname || ''
      });
    } catch (error) {
      if (context.state.oauthFlowToken !== token) return;
      context.setState({
        oauthFlow: null,
        oauthFlowLoading: false,
        oauthCompletionError: context.oauthErrorMessage(
          error && error.message ? error.message : 'oauth_flow_expired'
        )
      });
    }
  };

  context.setAuthMode = (mode) => context.setState({ authMode: mode, authError: '' });

  context.updateField = (field, value) => {
    const allowed = {
      authEmail: true,
      authPassword: true,
      regFirstName: true,
      regLastName: true,
      regDateOfBirth: true,
      regGender: true,
      regNickname: true,
      regAboutMe: true
    };
    if (!allowed[field]) return;
    context.setState({
      [field]: field === 'regDateOfBirth' ? formatDateOfBirthInput(value) : value
    });
  };

  context.updateOAuthRegistrationField = (field, value) => {
    const allowed = {
      oauthFirstName: true,
      oauthLastName: true,
      oauthDateOfBirth: true,
      oauthNickname: true,
      oauthAboutMe: true
    };
    if (!allowed[field]) return;
    context.setState({
      [field]: field === 'oauthDateOfBirth' ? formatDateOfBirthInput(value) : value
    });
  };

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

  context.pickOAuthAvatar = () => {
    const input = document.getElementById('oauth-registration-avatar');
    if (input) input.click();
  };

  context.onOAuthAvatar = event => {
    const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
    context.revokeRegistrationAvatarPreview(context.state.oauthAvatarPreviewURL);
    const previewURL = (
      file &&
      typeof URL !== 'undefined' &&
      URL &&
      typeof URL.createObjectURL === 'function'
    ) ? URL.createObjectURL(file) : '';
    context.setState({
      oauthAvatar: file,
      oauthAvatarName: file ? file.name : '',
      oauthAvatarPreviewURL: previewURL
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
      Object.assign(authenticatedState, context.oauthResetState());
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

  context.completeOAuthRegistration = async event => {
    if (event) event.preventDefault();
    const s = context.state;
    if (s.oauthCompletionPending || !s.oauthFlow) return;
    if (!parseDateOfBirth(s.oauthDateOfBirth.trim())) {
      context.setState({
        oauthCompletionError: 'Enter a real calendar date as DD-MM-YYYY.'
      });
      return;
    }
    const authGeneration = context.authGate.begin();
    context.setState({ oauthCompletionPending: true, oauthCompletionError: '' });
    try {
      const form = new FormData();
      form.append('first_name', s.oauthFirstName.trim());
      form.append('last_name', s.oauthLastName.trim());
      form.append('date_of_birth', s.oauthDateOfBirth.trim());
      if (s.oauthNickname.trim()) form.append('nickname', s.oauthNickname.trim());
      if (s.oauthAboutMe.trim()) form.append('about_me', s.oauthAboutMe.trim());
      if (s.oauthAvatar) form.append('avatar', s.oauthAvatar, s.oauthAvatar.name);
      const result = await AuthAPI.completeOAuthRegistration(s.oauthFlowToken, form);
      if (!context.authGate.isCurrent(authGeneration)) return;
      const user = result.user;
      const next = result.next || '/';
      const apiUsersByID = context.applyAuthUser(user);
      const avatarPreviewURL = s.oauthAvatarPreviewURL;
      const authenticatedState = {
        authStatus: 'authenticated',
        authPending: false,
        authError: '',
        screen: 'feed',
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public',
        profilePrivacyPending: false,
        profilePrivacyError: ''
      };
      Object.assign(authenticatedState, emptyNotificationState());
      Object.assign(authenticatedState, emptyProfileEditor());
      Object.assign(authenticatedState, context.oauthResetState());
      context.setState(authenticatedState, () => {
        context.revokeRegistrationAvatarPreview(avatarPreviewURL);
        if (environment && environment.history) {
          environment.history.replaceState({}, '', next);
        }
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
        oauthCompletionPending: false,
        oauthCompletionError: context.oauthErrorMessage(
          error && error.message ? error.message : ''
        ) || requestErrorMessage(error, 'GitHub registration failed. Please try again.')
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
      context.revokeRegistrationAvatarPreview(context.state.oauthAvatarPreviewURL);
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
	  }, emptyGroupPostState(), emptyGroupEventState(), emptyNotificationState(), emptyChatState(), emptyRegistrationForm(), context.oauthResetState(), emptyProfileEditor(), emptyConfirmationState()));
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
      loadOAuthProviders: context.loadOAuthProviders,
      startGitHubOAuth: context.startGitHubOAuth,
      showOAuthCompletion: context.showOAuthCompletion,
      showLoginOAuthError: context.showLoginOAuthError,
      loadOAuthFlow: context.loadOAuthFlow,
      setAuthMode: context.setAuthMode,
      updateField: context.updateField,
      updateOAuthRegistrationField: context.updateOAuthRegistrationField,
      pickRegistrationAvatar: context.pickRegistrationAvatar,
      onRegistrationAvatar: context.onRegistrationAvatar,
      pickOAuthAvatar: context.pickOAuthAvatar,
      onOAuthAvatar: context.onOAuthAvatar,
      submitAuth: context.submitAuth,
      completeOAuthRegistration: context.completeOAuthRegistration,
      logout: context.logout
    }, function (state) {
      const authTabs = [
        { key: 'login', label: 'Sign in' },
        { key: 'register', label: 'Create account' }
      ].map(tab => ({
        label: tab.label,
        bg: state.authMode === tab.key ? 'var(--surface)' : 'transparent',
        color: state.authMode === tab.key ? 'var(--text)' : 'var(--text3)',
        sh: state.authMode === tab.key ? 'var(--shadow)' : 'none',
        pick: () => context.setAuthMode(tab.key)
      }));
      return {
        authIsStandard: !state.oauthCompletionActive,
        oauthIsCompletion: !!state.oauthCompletionActive,
        oauthFlowLoading: !!state.oauthFlowLoading,
        oauthFlowReady: !!state.oauthFlow && !state.oauthFlowLoading,
        oauthFlowUnavailable: !state.oauthFlow && !state.oauthFlowLoading &&
          !!state.oauthCompletionError,
        oauthHasGitHub: (state.oauthProviders || []).some(provider => provider.name === 'github'),
        oauthProvidersLoading: !!state.oauthProvidersLoading,
        oauthHasError: !!state.oauthError,
        oauthError: state.oauthError,
        startGitHubOAuth: context.startGitHubOAuth,
        authTabs,
        authIsLogin: state.authMode === 'login',
        authIsReg: state.authMode === 'register',
        authCta: state.authPending ? 'Please wait…' :
          (state.authMode === 'login' ? 'Sign in' : 'Create account'),
        authDisabled: state.authPending,
        authButtonOpacity: state.authPending ? '0.65' : '1',
        authButtonCursor: state.authPending ? 'wait' : 'pointer',
        authHasError: !!state.authError,
        authError: state.authError,
        bootstrapError: state.bootstrapError,
        retryAuthBootstrap: context.loadCurrentUser,
        authEmail: state.authEmail,
        onAuthEmail: event => context.updateField('authEmail', event.target.value),
        authPassword: state.authPassword,
        onAuthPassword: event => context.updateField('authPassword', event.target.value),
        regFirstName: state.regFirstName,
        onRegFirstName: event => context.updateField('regFirstName', event.target.value),
        regLastName: state.regLastName,
        onRegLastName: event => context.updateField('regLastName', event.target.value),
        regDateOfBirth: state.regDateOfBirth,
        onRegDateOfBirth: event => context.updateField('regDateOfBirth', event.target.value),
        regGender: state.regGender,
        onRegGender: event => context.updateField('regGender', event.target.value),
        regNickname: state.regNickname,
        onRegNickname: event => context.updateField('regNickname', event.target.value),
        regAboutMe: state.regAboutMe,
        onRegAboutMe: event => context.updateField('regAboutMe', event.target.value),
        avatarButtonLabel: state.regAvatarName || 'avatar',
        registrationAvatarHasPreview: !!state.regAvatarPreviewURL,
        registrationAvatarMissingPreview: !state.regAvatarPreviewURL,
        registrationAvatarPreviewURL: state.regAvatarPreviewURL,
        pickRegistrationAvatar: context.pickRegistrationAvatar,
        onRegistrationAvatar: context.onRegistrationAvatar,
        oauthProviderLabel: state.oauthFlow ? state.oauthFlow.provider : 'github',
        oauthEmail: state.oauthFlow ? state.oauthFlow.email : '',
        oauthGitHubUsername: state.oauthFlow ? state.oauthFlow.github_username : '',
        oauthFirstName: state.oauthFirstName,
        onOAuthFirstName: event => context.updateOAuthRegistrationField('oauthFirstName', event.target.value),
        oauthLastName: state.oauthLastName,
        onOAuthLastName: event => context.updateOAuthRegistrationField('oauthLastName', event.target.value),
        oauthDateOfBirth: state.oauthDateOfBirth,
        onOAuthDateOfBirth: event => context.updateOAuthRegistrationField('oauthDateOfBirth', event.target.value),
        oauthNickname: state.oauthNickname,
        onOAuthNickname: event => context.updateOAuthRegistrationField('oauthNickname', event.target.value),
        oauthAboutMe: state.oauthAboutMe,
        onOAuthAboutMe: event => context.updateOAuthRegistrationField('oauthAboutMe', event.target.value),
        oauthAvatarButtonLabel: state.oauthAvatarName || 'avatar',
        oauthAvatarHasPreview: !!state.oauthAvatarPreviewURL,
        oauthAvatarMissingPreview: !state.oauthAvatarPreviewURL,
        oauthAvatarPreviewURL: state.oauthAvatarPreviewURL,
        pickOAuthAvatar: context.pickOAuthAvatar,
        onOAuthAvatar: context.onOAuthAvatar,
        oauthCompletionHasError: !!state.oauthCompletionError,
        oauthCompletionError: state.oauthCompletionError,
        oauthCompletionDisabled: state.oauthCompletionPending || !state.oauthFlow,
        oauthCompletionCta: state.oauthCompletionPending ? 'Please wait…' : 'Create account',
        completeOAuthRegistration: context.completeOAuthRegistration,
        submitAuth: context.submitAuth,
        goLogout: context.logout,
        logoutDisabled: state.logoutPending
      };
    }, {
      start: function () {
        context.loadOAuthProviders();
        context.loadCurrentUser();
      }
    });
  };
});
