(function (root, factory) {
  var create = factory(root && root.createFeatureController, root && root.createControllerContext);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createProfileController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController, createContext) {
  return function createProfileController(dependencies) {
    var context = createContext(dependencies);
    var AuthAPI = dependencies.api;
    var USERS = dependencies.session.users;
    var UserModel = dependencies.models.users;
    var NotificationModel = dependencies.models.notifications;
    var emptyProfileEditor = dependencies.helpers.emptyProfileEditor;
    var requestErrorMessage = dependencies.helpers.requestErrorMessage;
    var parseDateOfBirth = dependencies.helpers.parseDateOfBirth;
    var formatDateOfBirthInput = dependencies.helpers.formatDateOfBirthInput;
    var cover = dependencies.helpers.cover;
    var num = dependencies.helpers.num;
    var IC = dependencies.values.IC;
    var presentPost = dependencies.presenters.post;

  context.loadDirectory = async (reset = true) => {
    const authGeneration = context.authGate.current();
    const generation = reset ? context.directoryGate.begin() : context.directoryGate.current();
    if (!reset && context.state.directoryLoading) return;
    const cursor = reset ? null : context.state.directoryNextCursor;
    if (!reset && !cursor) return;
    context.setState({ directoryLoading: true, directoryError: '' });
    try {
      const response = await AuthAPI.users(cursor, 20);
      if (!context.authGate.isCurrent(authGeneration) || !context.directoryGate.isCurrent(generation)) return;
      const apiUsersByID = context.mergeAPIUsers(response.users || []);
      const incomingIDs = (response.users || []).map(user => Number(user.id));
      context.setState({
        apiUsersByID,
        directoryUserIDs: reset
          ? incomingIDs
          : context.state.directoryUserIDs.concat(incomingIDs.filter(id => context.state.directoryUserIDs.indexOf(id) < 0)),
        directoryNextCursor: response.next_cursor || null,
        directoryLoading: false, directoryError: ''
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !context.directoryGate.isCurrent(generation)) return;
      context.setState({
        directoryLoading: false,
        directoryError: requestErrorMessage(error, 'Could not load user suggestions.')
      });
    }
  };

  context.loadFollowRequests = async () => {
    if (context.state.followRequestsLoading) return;
    const authGeneration = context.authGate.current();
    context.setState({ followRequestsLoading: true, followRequestsError: '' });
    try {
      const response = await AuthAPI.followRequests();
      if (!context.authGate.isCurrent(authGeneration)) return;
      const requests = response.requests || [];
      const apiUsersByID = context.mergeAPIUsers(requests.map(request => request.user));
      context.setState({ apiUsersByID, followRequests: requests, followRequestsLoading: false });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration)) return;
      context.setState({
        followRequestsLoading: false,
        followRequestsError: requestErrorMessage(error, 'Could not load follow requests.')
      });
    }
  };

  context.loadProfileConnections = async (targetUserID, generation) => {
    targetUserID = Number(targetUserID);
    const authGeneration = context.authGate.current();
    generation = generation || context.profileGate.current();
    context.setState({ profileListsLoading: true, profileListsError: '' });
    try {
      const responses = await Promise.all([
        AuthAPI.followers(targetUserID),
        AuthAPI.following(targetUserID)
      ]);
      if (
        !context.authGate.isCurrent(authGeneration) ||
        !context.profileGate.isCurrent(generation) ||
        Number(context.state.profileId) !== targetUserID
      ) return;
      const followers = responses[0].users || [];
      const following = responses[1].users || [];
      const apiUsersByID = context.mergeAPIUsers(following, context.mergeAPIUsers(followers));
      context.setState({
        apiUsersByID,
        profileFollowers: followers.map(user => Number(user.id)),
        profileFollowing: following.map(user => Number(user.id)),
        profileListsLoading: false, profileListsError: ''
      });
    } catch (error) {
      if (
        !context.authGate.isCurrent(authGeneration) ||
        !context.profileGate.isCurrent(generation) ||
        Number(context.state.profileId) !== targetUserID
      ) return;
      if (error && error.status === 403) {
        context.setState({ profileFollowers: [], profileFollowing: [], profileListsLoading: false, profileListsError: '' });
        return;
      }
      context.setState({
        profileListsLoading: false,
        profileListsError: requestErrorMessage(error, 'Could not load followers and following.')
      });
    }
  };

  context.loadProfilePosts = async (targetUserID, reset, generation) => {
    targetUserID = Number(targetUserID || context.state.profileId);
    const authGeneration = context.authGate.current();
    generation = generation || context.profileGate.current();
    if (!targetUserID || (!reset && context.state.profilePostsPending)) return;
    const cursor = reset ? null : context.state.profilePostsNextCursor;
    if (!reset && !cursor) return;
    context.setState({ profilePostsPending: true, profilePostsLoading: !!reset, profilePostsError: '' });
    try {
      const page = await AuthAPI.userPosts(targetUserID, cursor, 20);
      if (
        !context.authGate.isCurrent(authGeneration) ||
        !context.profileGate.isCurrent(generation) ||
        Number(context.state.profileId) !== targetUserID
      ) return;
      const mapped = (page.posts || []).map(post => context.mapAPIPost(post));
      const apiUsersByID = context.mergeAPIUsers((page.posts || []).map(post => post.author));
      context.setState(current => {
		const merged = context.mergePostCommentsCounts(mapped, current.posts, current.profilePosts, current.groupPosts);
        return {
          profilePosts: reset ? merged : current.profilePosts.concat(merged),
          apiUsersByID,
          profilePostsLoading: false, profilePostsPending: false,
          profilePostsNextCursor: page.next_cursor || null, profilePostsError: ''
        };
      });
    } catch (error) {
      if (
        !context.authGate.isCurrent(authGeneration) ||
        !context.profileGate.isCurrent(generation) ||
        Number(context.state.profileId) !== targetUserID
      ) return;
      if (error && error.status === 403) {
        const user = context.apiUser(targetUserID);
        user.canViewProfile = false;
        user.bio = ''; user.dob = ''; user.postsCount = 0;
        context.setState({
          apiUsersByID: Object.assign({}, context.state.apiUsersByID, { [String(targetUserID)]: user }),
          profilePosts: [], profileFollowers: [], profileFollowing: [],
          profilePostsLoading: false, profilePostsPending: false, profilePostsError: ''
        });
        return;
      }
      context.setState({
        profilePostsLoading: false, profilePostsPending: false,
        profilePostsError: requestErrorMessage(error, 'Could not load profile posts. Please try again.')
      });
    }
  };

  context.openProfile = async (targetUserID) => {
    if (targetUserID === 'me') targetUserID = USERS.me.apiId;
    targetUserID = Number(targetUserID);
    if (!Number.isInteger(targetUserID) || targetUserID <= 0) return;
    if (dependencies.navigation && !dependencies.navigation.isApplying()) {
      dependencies.navigation.profile(targetUserID);
      return;
    }
    context.stopTyping();
    const authGeneration = context.authGate.current();
    const generation = context.profileGate.begin();
    const isMe = targetUserID === USERS.me.apiId;
    context.setState({
      screen: 'profile', profileId: targetUserID, profileTab: 'posts',
      profileLoading: true, profileReady: false, profileError: '',
      profilePosts: [], profilePostsLoading: false, profilePostsPending: false,
      profilePostsError: '', profilePostsNextCursor: null,
      profileFollowers: [], profileFollowing: [], profileListsLoading: false, profileListsError: '',
      profileEditOpen: isMe ? context.state.profileEditOpen : false,
      profileEditError: isMe ? context.state.profileEditError : ''
    });
    try {
      const results = await Promise.all([
        AuthAPI.userProfile(targetUserID),
        isMe ? Promise.resolve({ status: 'none', follows_me: false }) : AuthAPI.relationship(targetUserID)
      ]);
      if (!context.authGate.isCurrent(authGeneration) || !context.profileGate.isCurrent(generation)) return;
      const rawUser = Object.assign({}, results[0], { relationship: results[1] });
      const apiUsersByID = context.mergeAPIUsers([rawUser]);
      const profileUser = apiUsersByID[String(targetUserID)];
      context.setState({ apiUsersByID, profileLoading: false, profileReady: true, profileError: '' });
      if (profileUser.canViewProfile) {
        context.loadProfilePosts(targetUserID, true, generation);
        context.loadProfileConnections(targetUserID, generation);
      }
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !context.profileGate.isCurrent(generation)) return;
      context.setState({
        profileLoading: false, profileReady: false,
        profileError: error && error.status === 404
          ? 'User not found.'
          : requestErrorMessage(error, 'Could not load this profile.')
      });
    }
  };

  context.openProfileEdit = () => {
    const me = USERS.me;
    context.setState({
      profileEditOpen: true, profileEditError: '',
      editFirstName: me.firstName || '', editLastName: me.lastName || '',
      editDateOfBirth: me.dob || '', editGender: me.gender || '',
      editNickname: me.nickname || '', editAboutMe: me.aboutMe || '',
      editAvatar: null, editAvatarName: ''
    });
  };

  context.cancelProfileEdit = () => context.setState(Object.assign({}, emptyProfileEditor()));

  context.saveProfile = async (event) => {
    if (event) event.preventDefault();
    if (context.state.profileEditPending || context.state.profileAvatarPending || context.state.profilePrivacyPending) return;
    const authGeneration = context.authGate.current();
    const s = context.state;
    if (!parseDateOfBirth(s.editDateOfBirth.trim())) {
      context.setState({ profileEditError: 'Enter a real calendar date as DD-MM-YYYY.' });
      return;
    }
    context.setState({ profileEditPending: true, profileEditError: '' });
    try {
      const user = await AuthAPI.updateProfile({
        first_name: s.editFirstName.trim(),
        last_name: s.editLastName.trim(),
        date_of_birth: s.editDateOfBirth.trim(),
        gender: s.editGender || null,
        nickname: s.editNickname,
        about_me: s.editAboutMe
      });
      if (!context.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = context.applyAuthUser(user);
      context.setState(Object.assign({
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public'
      }, emptyProfileEditor()));
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration)) return;
      context.setState({
        profileEditPending: false,
        profileEditError: requestErrorMessage(error, 'Could not update your profile. Please try again.')
      });
    }
  };

  context.pickProfileAvatar = () => {
    const input = document.getElementById('profile-avatar');
    if (input) input.click();
  };

  context.onProfileAvatar = (event) => {
    const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
    context.setState({ editAvatar: file, editAvatarName: file ? file.name : '', profileEditError: '' });
  };

  context.replaceProfileAvatar = async () => {
    if (context.state.profileAvatarPending || context.state.profileEditPending || context.state.profilePrivacyPending || !context.state.editAvatar) return;
    const authGeneration = context.authGate.current();
    const avatar = context.state.editAvatar;
    context.setState({ profileAvatarPending: true, profileEditError: '' });
    try {
      const form = new FormData();
      form.append('avatar', avatar, avatar.name);
      const user = await AuthAPI.replaceAvatar(form);
      if (!context.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = context.applyAuthUser(user);
      const input = document.getElementById('profile-avatar');
      if (input) input.value = '';
      context.setState({
        apiUsersByID,
        profileAvatarPending: false, editAvatar: null, editAvatarName: '',
        myPrivacy: user.is_private === true ? 'private' : 'public'
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration)) return;
      context.setState({
        profileAvatarPending: false,
        profileEditError: requestErrorMessage(error, 'Could not replace your avatar. Please try again.')
      });
    }
  };

  context.deleteProfileAvatar = async () => {
    if (context.state.profileAvatarPending || context.state.profileEditPending || context.state.profilePrivacyPending) return;
    const authGeneration = context.authGate.current();
    context.setState({ profileAvatarPending: true, profileEditError: '' });
    try {
      const user = await AuthAPI.deleteAvatar();
      if (!context.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = context.applyAuthUser(user);
      const input = document.getElementById('profile-avatar');
      if (input) input.value = '';
      context.setState({
        apiUsersByID,
        profileAvatarPending: false, editAvatar: null, editAvatarName: '',
        myPrivacy: user.is_private === true ? 'private' : 'public'
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration)) return;
      context.setState({
        profileAvatarPending: false,
        profileEditError: requestErrorMessage(error, 'Could not delete your avatar. Please try again.')
      });
    }
  };

  context.setProfilePrivacy = async (privacy, confirmed, triggerElement) => {
    if (
      context.state.profilePrivacyPending ||
      context.state.profileEditPending ||
      context.state.profileAvatarPending ||
      privacy === context.state.myPrivacy
    ) return false;
    if (!confirmed) {
      context.openConfirmation({
        kind: 'privacy',
        target: privacy,
        title: privacy === 'private' ? 'Make your profile private?' : 'Make your profile public?',
        message: privacy === 'private'
          ? 'Only accepted followers will be able to see your private profile details and personal posts.'
          : 'Everyone will be able to see your profile details and public posts.',
        confirmLabel: 'Change privacy'
      }, triggerElement);
      return false;
    }
    const authGeneration = context.authGate.current();
    const isPrivate = privacy === 'private';
    context.setState({ profilePrivacyPending: true, profilePrivacyError: '' });
    try {
      const user = await AuthAPI.updateProfile({ is_private: isPrivate });
      if (!context.authGate.isCurrent(authGeneration)) return;
      const apiUsersByID = context.applyAuthUser(user);
      context.setState({
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public',
        profilePrivacyPending: false,
        profilePrivacyError: ''
      });
      return true;
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration)) return false;
      context.setState({
        profilePrivacyPending: false,
        profilePrivacyError: requestErrorMessage(error, 'Could not update profile privacy. Please try again.')
      });
      return false;
    }
  };

  context.toggleFollow = async (targetUserID, confirmed, triggerElement) => {
    targetUserID = Number(targetUserID);
    if (!Number.isInteger(targetUserID) || targetUserID <= 0 || targetUserID === USERS.me.apiId) return false;
    const key = String(targetUserID);
    if (context.state.followPendingByID[key]) return false;
    const authGeneration = context.authGate.current();
    const relationshipGate = context.relationshipGeneration(targetUserID);
    const relationshipGeneration = relationshipGate.current();
    const user = context.apiUser(targetUserID);
    const status = UserModel.normalizeStatus(user.relationship && user.relationship.status);
    if (status !== 'none' && !confirmed) {
      context.openConfirmation({
        kind: 'unfollow',
        target: targetUserID,
        title: status === 'requested' ? 'Cancel follow request?' : 'Unfollow ' + user.name + '?',
        message: status === 'requested'
          ? 'This pending follow request will be cancelled.'
          : 'Posts shared with followers may disappear from your feed.',
        confirmLabel: status === 'requested' ? 'Cancel request' : 'Unfollow'
      }, triggerElement);
      return false;
    }
    context.setState({
      followPendingByID: Object.assign({}, context.state.followPendingByID, { [key]: true }),
      followErrorByID: Object.assign({}, context.state.followErrorByID, { [key]: '' }),
      appError: ''
    });
    try {
      const response = status === 'none'
        ? await AuthAPI.follow(targetUserID)
        : (await AuthAPI.unfollow(targetUserID), { status: 'none' });
      if (!context.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
      context.beginRelationshipGeneration(targetUserID);
      const apiUsersByID = context.mergeAPIUsers([{
        id: targetUserID,
        relationship: {
          status: response.status,
          follows_me: user.relationship && user.relationship.follows_me === true
        }
      }]);
      const pending = Object.assign({}, context.state.followPendingByID);
      delete pending[key];
      const inaccessiblePostIDs = status === 'none' ? [] : context.state.posts.concat(context.state.profilePosts)
        .filter(post => Number(post.apiAuthorID) === targetUserID)
        .map(post => post.id);
      context.purgeCommentStates(inaccessiblePostIDs);
      const posts = status === 'none'
        ? context.state.posts
        : context.state.posts.filter(post => Number(post.apiAuthorID) !== targetUserID);
      context.setState({ apiUsersByID, followPendingByID: pending, posts });

      context.loadDirectory();
      context.loadFeed(true);
      if (context.state.screen === 'profile' && Number(context.state.profileId) === targetUserID) context.openProfile(targetUserID);
      return true;
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return false;
      const pending = Object.assign({}, context.state.followPendingByID);
      delete pending[key];
      const message = requestErrorMessage(error, 'Could not update follow status.');
      context.setState({
        followPendingByID: pending,
        followErrorByID: Object.assign({}, context.state.followErrorByID, { [key]: message }),
        appError: message
      });
      return false;
    }
  };

  context.acceptFollowRequest = async (requestID) => {
    const key = String(requestID);
    if (context.state.followRequestPendingByID[key]) return;
    const request = context.state.followRequests.find(item => String(item.id) === key);
    if (!request) return;
    const authGeneration = context.authGate.current();
    const relationshipGate = context.relationshipGeneration(request.user.id);
    const relationshipGeneration = relationshipGate.current();
    context.setState({
      followRequestPendingByID: Object.assign({}, context.state.followRequestPendingByID, { [key]: true }),
      followRequestsError: ''
    });
    try {
      await AuthAPI.acceptFollowRequest(requestID);
      if (!context.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
      context.beginRelationshipGeneration(request.user.id);
      const user = context.apiUser(request.user.id);
      const apiUsersByID = context.mergeAPIUsers([{
        id: request.user.id,
        relationship: {
          status: user.relationship.status,
          follows_me: true
        }
      }]);
      const pending = Object.assign({}, context.state.followRequestPendingByID);
      delete pending[key];
      context.setState({
        apiUsersByID,
        followRequests: context.state.followRequests.filter(item => String(item.id) !== key),
        followRequestPendingByID: pending
      });
      context.loadPostFollowers();
      context.loadDirectory();
      context.loadFeed(true);
      if (context.state.screen === 'profile' && Number.isInteger(Number(context.state.profileId))) {
        context.openProfile(Number(context.state.profileId));
      } else {
        context.profileGate.begin();
      }
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
      const pending = Object.assign({}, context.state.followRequestPendingByID);
      delete pending[key];
      context.setState({
        followRequestPendingByID: pending,
        followRequestsError: requestErrorMessage(error, 'Could not accept follow request.')
      });
    }
  };

  context.rejectFollowRequest = async (requestID) => {
    const key = String(requestID);
    if (context.state.followRequestPendingByID[key]) return;
    const request = context.state.followRequests.find(item => String(item.id) === key);
    if (!request) return;
    const authGeneration = context.authGate.current();
    const relationshipGate = context.relationshipGeneration(request.user.id);
    const relationshipGeneration = relationshipGate.current();
    context.setState({
      followRequestPendingByID: Object.assign({}, context.state.followRequestPendingByID, { [key]: true }),
      followRequestsError: ''
    });
    try {
      await AuthAPI.rejectFollowRequest(requestID);
      if (!context.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
      context.beginRelationshipGeneration(request.user.id);
      const pending = Object.assign({}, context.state.followRequestPendingByID);
      delete pending[key];
      context.setState({
        followRequests: context.state.followRequests.filter(item => String(item.id) !== key),
        followRequestPendingByID: pending
      });
      context.loadDirectory();
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !relationshipGate.isCurrent(relationshipGeneration)) return;
      const pending = Object.assign({}, context.state.followRequestPendingByID);
      delete pending[key];
      context.setState({
        followRequestPendingByID: pending,
        followRequestsError: requestErrorMessage(error, 'Could not reject follow request.')
      });
    }
  };

  context.followBtn = function (userID) {
    const user = context.apiUser(userID);
    const model = UserModel.followButton(user, context.state.followPendingByID[String(userID)]);
    if (model.tone === 'muted') return { label: model.label, bg: 'var(--surface2)', color: 'var(--text2)', bd: 'transparent', disabled: model.disabled };
    if (model.tone === 'soft') return { label: model.label, bg: 'var(--soft)', color: 'var(--accent)', bd: 'transparent', disabled: model.disabled };
    return { label: model.label, bg: 'var(--accent)', color: '#fff', bd: 'transparent', disabled: model.disabled };
  };

  context.selectTab = tab => {
    if (['posts', 'followers', 'following'].indexOf(tab) < 0) return;
    context.setState({ profileTab: tab });
  };

  context.updateEditorField = (field, value) => {
    const allowed = {
      editFirstName: true,
      editLastName: true,
      editDateOfBirth: true,
      editGender: true,
      editNickname: true,
      editAboutMe: true
    };
    if (!allowed[field]) return;
    context.setState({
      [field]: field === 'editDateOfBirth' ? formatDateOfBirthInput(value) : value
    });
  };

    return createController('profile', dependencies, {
      loadDirectory: context.loadDirectory,
      loadFollowRequests: context.loadFollowRequests,
      loadProfileConnections: context.loadProfileConnections,
      loadProfilePosts: context.loadProfilePosts,
      openProfile: context.openProfile,
      openProfileEdit: context.openProfileEdit,
      cancelProfileEdit: context.cancelProfileEdit,
      saveProfile: context.saveProfile,
      pickProfileAvatar: context.pickProfileAvatar,
      onProfileAvatar: context.onProfileAvatar,
      replaceProfileAvatar: context.replaceProfileAvatar,
      deleteProfileAvatar: context.deleteProfileAvatar,
      setProfilePrivacy: context.setProfilePrivacy,
      toggleFollow: context.toggleFollow,
      acceptFollowRequest: context.acceptFollowRequest,
      rejectFollowRequest: context.rejectFollowRequest,
      followBtn: context.followBtn,
      selectTab: context.selectTab,
      updateEditorField: context.updateEditorField
    }, function (state) {
      var me = USERS.me;
      var profileID = Number(state.profileId || me.apiId);
      var user = context.apiUser(profileID);
      var isMe = profileID === Number(me.apiId);
      var canView = state.profileReady && user.canViewProfile !== false;

      function followButton(userID) {
        var target = context.apiUser(userID);
        var model = UserModel.followButton(
          target,
          state.followPendingByID[String(userID)]
        );
        if (model.tone === 'muted') {
          return {
            label: model.label,
            bg: 'var(--surface2)',
            color: 'var(--text2)',
            bd: 'transparent',
            disabled: model.disabled
          };
        }
        if (model.tone === 'soft') {
          return {
            label: model.label,
            bg: 'var(--soft)',
            color: 'var(--accent)',
            bd: 'transparent',
            disabled: model.disabled
          };
        }
        return {
          label: model.label,
          bg: 'var(--accent)',
          color: '#fff',
          bd: 'transparent',
          disabled: model.disabled
        };
      }

      function userRow(userID) {
        var rowUser = context.apiUser(userID);
        var button = followButton(userID);
        return {
          user: rowUser,
          showBtn: Number(userID) !== Number(me.apiId),
          btnLabel: button.label,
          btnBg: button.bg,
          btnColor: button.color,
          btnBd: button.bd,
          btnDisabled: button.disabled,
          onBtn: function (event) {
            return context.toggleFollow(userID, false, event && event.currentTarget);
          },
          message: function () { return context.openDirectChat(userID); },
          goProfile: function () { return context.openProfile(userID); }
        };
      }

      var profileTabs = [
        { key: 'posts', label: 'Posts' },
        { key: 'followers', label: 'Followers · ' + (user.followersCount || 0) },
        { key: 'following', label: 'Following · ' + (user.followingCount || 0) }
      ].map(function (tab) {
        var selected = state.profileTab === tab.key;
        return {
          label: tab.label,
          color: selected ? 'var(--text)' : 'var(--text3)',
          bd: selected ? 'var(--accent)' : 'transparent',
          pick: function () { return context.selectTab(tab.key); }
        };
      });
      var privacySegments = [
        { key: 'public', label: 'Public', icon: IC.globe },
        { key: 'private', label: 'Private', icon: IC.lock }
      ].map(function (option) {
        var selected = state.myPrivacy === option.key;
        var disabled = state.profilePrivacyPending ||
          state.profileEditPending || state.profileAvatarPending;
        return {
          label: option.label,
          icon: option.icon,
          bg: selected ? 'var(--surface)' : 'transparent',
          color: selected ? 'var(--text)' : 'var(--text3)',
          disabled: disabled,
          opacity: disabled ? '0.6' : '1',
          cursor: state.profilePrivacyPending
            ? 'wait'
            : (state.profileEditPending || state.profileAvatarPending ? 'not-allowed' : 'pointer'),
          pick: function (event) {
            return context.setProfilePrivacy(
              option.key,
              false,
              event && event.currentTarget
            );
          }
        };
      });
      var follow = followButton(profileID);
      var pendingRequestActorIDs = {};
      state.notifications.forEach(function (notification) {
        if (
          notification.type === 'follow_request' &&
          NotificationModel.isActionable(notification)
        ) {
          pendingRequestActorIDs[String(Number(notification.actorID))] = true;
        }
      });
      var suggestions = state.directoryUserIDs
        .map(function (userID) { return context.apiUser(userID); })
        .filter(function (suggestion) {
          return !suggestion.relationship ||
            suggestion.relationship.status !== 'accepted';
        })
        .filter(function (suggestion) {
          return !pendingRequestActorIDs[String(Number(suggestion.apiId))];
        })
        .map(function (suggestion) {
          var button = followButton(suggestion.apiId);
          var relationship = suggestion.relationship;
          return {
            user: suggestion,
            isPrivate: suggestion.private,
            btnLabel: button.label,
            btnBg: button.bg,
            btnColor: button.color,
            btnBd: button.bd,
            btnDisabled: button.disabled,
            onBtn: function (event) {
              return context.toggleFollow(
                suggestion.apiId,
                false,
                event && event.currentTarget
              );
            },
            canMessage: relationship &&
              (relationship.status === 'accepted' || relationship.follows_me),
            message: function () { return context.openDirectChat(suggestion.apiId); },
            goProfile: function () { return context.openProfile(suggestion.apiId); }
          };
        });

      return {
        profileError: state.profileError,
        retryProfile: function () { return context.openProfile(state.profileId); },
        pUser: user,
        pIsMe: isMe,
        pOther: !isMe,
        pCover: cover(user.color),
        pShowLock: user.private || (isMe && state.myPrivacy === 'private'),
        pCanView: canView,
        pLocked: !canView,
        pShowEmail: canView && !!user.email,
        pShowGender: canView && !!user.gender,
        pGenderLabel: user.gender === 'male'
          ? 'Male'
          : (user.gender === 'female' ? 'Female' : ''),
        pStatPosts: num(user.postsCount || 0),
        pStatFollowers: num(user.followersCount || 0),
        pStatFollowing: num(user.followingCount || 0),
        pTabs: profileTabs,
        pTabPosts: state.profileTab === 'posts',
        pTabFollowers: state.profileTab === 'followers',
        pTabFollowing: state.profileTab === 'following',
        pPosts: state.profilePosts.map(function (post) {
          return presentPost(state, post);
        }),
        pNoPosts: !state.profilePostsLoading &&
          !state.profilePostsError && state.profilePosts.length === 0,
        pPostsLoading: state.profilePostsLoading,
        pPostsHasError: !!state.profilePostsError,
        pPostsError: state.profilePostsError,
        retryProfilePosts: function () {
          return context.loadProfilePosts(state.profileId, true);
        },
        pPostsHasMore: !!state.profilePostsNextCursor,
        loadMoreProfilePosts: function () {
          return context.loadProfilePosts(state.profileId, false);
        },
        profileLoadMoreLabel: state.profilePostsPending &&
          !state.profilePostsLoading ? 'Loading…' : 'Load more',
        profileLoadMoreDisabled: state.profilePostsPending,
        pFollowers: state.profileFollowers.map(userRow),
        pFollowing: state.profileFollowing.map(userRow),
        pListsLoading: state.profileListsLoading,
        pListsHasError: !!state.profileListsError,
        pListsError: state.profileListsError,
        followLabel: follow.label,
        followBg: follow.bg,
        followColor: follow.color,
        followBd: follow.bd,
        followDisabled: follow.disabled,
        followHasError: !!state.followErrorByID[String(state.profileId)],
        followError: state.followErrorByID[String(state.profileId)] || '',
        onFollow: function (event) {
          return context.toggleFollow(
            state.profileId,
            false,
            event && event.currentTarget
          );
        },
        msgProfile: function () { return context.openDirectChat(state.profileId); },
        privacySeg: privacySegments,
        profilePrivacyHasError: isMe && !!state.profilePrivacyError,
        profilePrivacyError: state.profilePrivacyError,
        showProfileEdit: isMe && state.profileEditOpen,
        openProfileEdit: context.openProfileEdit,
        cancelProfileEdit: context.cancelProfileEdit,
        saveProfile: context.saveProfile,
        profileEditPending: state.profileEditPending ||
          state.profileAvatarPending || state.profilePrivacyPending,
        profileSaveLabel: state.profileEditPending ? 'Saving…' : 'Save changes',
        profileEditHasError: !!state.profileEditError,
        profileEditError: state.profileEditError,
        editFirstName: state.editFirstName,
        onEditFirstName: function (event) {
          return context.updateEditorField('editFirstName', event.target.value);
        },
        editLastName: state.editLastName,
        onEditLastName: function (event) {
          return context.updateEditorField('editLastName', event.target.value);
        },
        editDateOfBirth: state.editDateOfBirth,
        onEditDateOfBirth: function (event) {
          return context.updateEditorField('editDateOfBirth', event.target.value);
        },
        editGender: state.editGender,
        onEditGender: function (event) {
          return context.updateEditorField('editGender', event.target.value);
        },
        editNickname: state.editNickname,
        onEditNickname: function (event) {
          return context.updateEditorField('editNickname', event.target.value);
        },
        editAboutMe: state.editAboutMe,
        onEditAboutMe: function (event) {
          return context.updateEditorField('editAboutMe', event.target.value);
        },
        profileAvatarLabel: state.editAvatarName || 'Choose image',
        profileAvatarPending: state.profileAvatarPending ||
          state.profileEditPending || state.profilePrivacyPending,
        profileAvatarUploadDisabled: state.profileAvatarPending ||
          state.profileEditPending || state.profilePrivacyPending || !state.editAvatar,
        profileAvatarUploadOpacity: state.profileAvatarPending ||
          state.profileEditPending || state.profilePrivacyPending || !state.editAvatar
          ? '0.55'
          : '1',
        profileAvatarUploadLabel: state.profileAvatarPending ? 'Working…' : 'Upload',
        profileHasCustomAvatar: me.hasCustomAvatar,
        pickProfileAvatar: context.pickProfileAvatar,
        onProfileAvatar: context.onProfileAvatar,
        replaceProfileAvatar: context.replaceProfileAvatar,
        deleteProfileAvatar: context.deleteProfileAvatar,
        suggestions: suggestions,
        suggestionsLoading: state.directoryLoading,
        suggestionsHasError: !!state.directoryError,
        suggestionsError: state.directoryError,
        suggestionsHasMore: !!state.directoryNextCursor,
        loadMoreSuggestions: function () { return context.loadDirectory(false); }
      };
    }, {});
  };
});
