(function (root, factory) {
  var create = factory(root && root.createFeatureController, root && root.createControllerContext);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createGroupsController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController, createContext) {
  return function createGroupsController(dependencies) {
    var context = createContext(dependencies);
    var AuthAPI = dependencies.api;
    var UserModel = dependencies.models.users;
    var PostModel = dependencies.models.posts;
    var ChatModel = dependencies.models.chats;
    var USERS = dependencies.session.users;
    var emptyGroupPostState = dependencies.helpers.emptyGroupPostState;
    var emptyGroupEventState = dependencies.helpers.emptyGroupEventState;
    var requestErrorMessage = dependencies.helpers.requestErrorMessage;
    var cover = dependencies.helpers.cover;
    var num = dependencies.helpers.num;
    var GROUP_COLORS = dependencies.values.GROUP_COLORS;
    var presentPost = dependencies.presenters.post;

  context.groupAccessIsRevoked = function (groupID) {
    return context.revokedGroupAccessIDs.has(String(Number(groupID)));
  };

  context.revokeGroupAccess = function (groupID) {
    groupID = Number(groupID);
    if (!Number.isInteger(groupID) || groupID <= 0) return;
    const key = String(groupID);
    context.revokedGroupAccessIDs.add(key);
    context.groupGeneration(groupID).begin();
    context.chatsGate.begin();
    context.setState(current => ({
      groupMutationPendingByID: Object.assign({}, current.groupMutationPendingByID, { [key]: false }),
      groupMutationErrorByID: Object.assign({}, current.groupMutationErrorByID, { [key]: '' }),
      chatsPending: false,
      chatsLoading: false
    }));
    if (Number(context.state.groupId) !== groupID) return;

    context.groupDetailGate.begin();
    context.groupMembersGate.begin();
    context.groupRequestsGate.begin();
    context.groupInvitationsGate.begin();
    context.groupPostsGate.begin();
    context.groupEventsGate.begin();
    context.groupEventCreateGate.begin();
    context.invalidateGroupEventResponses();
    const postIDs = context.state.groupPosts
      .filter(post => Number(post.groupID) === groupID)
      .map(post => post.id);
    context.purgeCommentStates(postIDs);
    const input = typeof document !== 'undefined' ? document.getElementById('group-post-media') : null;
    if (input) input.value = '';
    context.setState(Object.assign({}, emptyGroupPostState(), emptyGroupEventState(), {
      inviteOpen: false,
      groupLoading: false,
      groupMembers: [], groupMembersNextCursor: null, groupMembersLoading: false, groupMembersError: '',
      groupRequests: [], groupRequestsNextCursor: null, groupRequestsLoading: false, groupRequestsError: '',
      groupInvitations: [], groupInvitationsNextCursor: null, groupInvitationsLoading: false, groupInvitationsError: ''
    }));
  };

  context.restoreGroupAccess = function (group) {
    if (!group || (group.state !== 'owner' && group.state !== 'member')) return false;
    const groupID = Number(group.id);
    if (!Number.isInteger(groupID) || groupID <= 0) return false;
    context.revokedGroupAccessIDs.delete(String(groupID));
    if (Number(context.state.groupId) !== groupID) return true;

    context.groupPostsGate.begin();
    context.groupEventsGate.begin();
    context.groupEventCreateGate.begin();
    context.invalidateGroupEventResponses();
    context.purgeCommentStates(context.state.groupPosts.map(post => post.id));
    const input = typeof document !== 'undefined' ? document.getElementById('group-post-media') : null;
    if (input) input.value = '';
    context.setState(Object.assign({}, emptyGroupPostState(), emptyGroupEventState()), () => {
      if (Number(context.state.groupId) === groupID && !context.groupAccessIsRevoked(groupID)) {
        context.loadGroupPosts(groupID, true);
        context.loadGroupEvents(groupID, true);
      }
    });
    return true;
  };

  context.loadGroupPosts = async (groupID, reset = true) => {
	groupID = Number(groupID);
	if (!Number.isInteger(groupID) || groupID <= 0 || context.groupAccessIsRevoked(groupID)) return;
	const group = context.state.apiGroupsByID[String(groupID)];
	if (group && group.state !== 'owner' && group.state !== 'member') return;
	const authGeneration = context.authGate.current();
	const accessGate = context.groupGeneration(groupID);
	const accessGeneration = accessGate.current();
	const generation = reset ? context.groupPostsGate.begin() : context.groupPostsGate.current();
	if (!reset && context.state.groupPostsPending) return;
	const cursor = reset ? null : context.state.groupPostsNextCursor;
	if (!reset && !cursor) return;
	context.setState({ groupPostsPending: true, groupPostsLoading: !!reset, groupPostsError: '' });
	try {
	  const page = await AuthAPI.groupPosts(groupID, cursor, 20);
	  if (
		!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
		!context.groupPostsGate.isCurrent(generation) || context.groupAccessIsRevoked(groupID) ||
		Number(context.state.groupId) !== groupID
	  ) return;
	  const rawPosts = page.posts || [];
	  const mapped = rawPosts.map(post => context.mapAPIPost(post));
	  const apiUsersByID = context.mergeAPIUsers(rawPosts.map(post => post.author));
	  context.setState(current => {
		const merged = context.mergePostCommentsCounts(mapped, current.posts, current.profilePosts, current.groupPosts);
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
		!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
		!context.groupPostsGate.isCurrent(generation) || context.groupAccessIsRevoked(groupID) ||
		Number(context.state.groupId) !== groupID
	  ) return;
	  if (error && error.status === 403) {
		context.revokeGroupAccess(groupID);
		return;
	  }
	  context.setState({
		groupPostsPending: false, groupPostsLoading: false,
		groupPostsError: requestErrorMessage(error, error && error.status === 404 ? 'Group not found.' : 'Could not load group posts.')
	  });
	}
  };

  context.pickGroupPostMedia = () => {
	const input = document.getElementById('group-post-media');
	if (input) input.click();
  };

  context.onGroupPostMedia = (event) => {
	const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
	context.setState({
	  groupPostComposerFile: file,
	  groupPostComposerFileName: file ? file.name : '',
	  groupPostComposerError: ''
	});
  };

  context.removeGroupPostMedia = () => {
	const input = document.getElementById('group-post-media');
	if (input) input.value = '';
	context.setState({ groupPostComposerFile: null, groupPostComposerFileName: '', groupPostComposerError: '' });
  };

  context.sendGroupPost = async () => {
	const groupID = Number(context.state.groupId);
	const group = context.state.apiGroupsByID[String(groupID)];
	if (
	  !Number.isInteger(groupID) || groupID <= 0 || context.groupAccessIsRevoked(groupID) ||
	  !group || (group.state !== 'owner' && group.state !== 'member') ||
	  context.state.groupPostComposerPending ||
	  (!context.state.groupPostComposerText.trim() && !context.state.groupPostComposerFile)
	) return;
	const authGeneration = context.authGate.current();
	const accessGate = context.groupGeneration(groupID);
	const accessGeneration = accessGate.current();
	const form = PostModel.buildCreateGroupPostForm({
	  text: context.state.groupPostComposerText,
	  media: context.state.groupPostComposerFile
	}, FormData);
	context.setState({ groupPostComposerPending: true, groupPostComposerError: '' });
	try {
	  const response = await AuthAPI.createGroupPost(groupID, form);
	  if (
		!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
		context.groupAccessIsRevoked(groupID) || Number(context.state.groupId) !== groupID
	  ) return;
	  context.groupPostsGate.begin();
	  const post = context.mapAPIPost(response);
	  const apiUsersByID = context.mergeAPIUsers([response.author]);
	  const input = typeof document !== 'undefined' ? document.getElementById('group-post-media') : null;
	  if (input) input.value = '';
	  context.setState(current => ({
		apiUsersByID,
		groupPosts: [post].concat(current.groupPosts.filter(item => item.id !== post.id)),
		groupPostsLoading: false, groupPostsPending: false,
		groupPostComposerText: '', groupPostComposerFile: null, groupPostComposerFileName: '',
		groupPostComposerError: '', groupPostComposerPending: false
	  }));
	} catch (error) {
	  if (
		!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
		context.groupAccessIsRevoked(groupID) || Number(context.state.groupId) !== groupID
	  ) return;
	  if (error && error.status === 403) {
		context.revokeGroupAccess(groupID);
		return;
	  }
	  context.setState({
		groupPostComposerPending: false,
		groupPostComposerError: requestErrorMessage(error, 'Could not create the group post. Your draft was kept.')
	  });
	}
  };

  context.groupGeneration = function (groupID) {
    const key = String(Number(groupID));
    if (!context.groupGenerationsByID[key]) context.groupGenerationsByID[key] = UserModel.createRequestGate();
    return context.groupGenerationsByID[key];
  };

  context.mapAPIGroup = function (raw) {
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
  };

  context.mergeGroupResponses = function (rawGroups, baseGroups) {
    const groups = Object.assign({}, baseGroups || context.state.apiGroupsByID);
    (rawGroups || []).forEach(raw => {
      const mapped = context.mapAPIGroup(raw);
      if (Number.isInteger(mapped.id) && mapped.id > 0) groups[String(mapped.id)] = mapped;
    });
    return groups;
  };

  context.loadGroups = async (reset = true) => {
    const authGeneration = context.authGate.current();
    const generation = reset ? context.groupsDirectoryGate.begin() : context.groupsDirectoryGate.current();
    if (!reset && context.state.groupsPending) return;
    const cursor = reset ? null : context.state.groupsNextCursor;
    if (!reset && !cursor) return;
    context.setState({ groupsPending: true, groupsLoading: !!reset, groupsError: '' });
    try {
      const page = await AuthAPI.groups(cursor, 20);
      if (!context.authGate.isCurrent(authGeneration) || !context.groupsDirectoryGate.isCurrent(generation)) return;
      const rawGroups = page.groups || [];
      const incomingIDs = rawGroups.map(group => Number(group.id));
      const apiUsersByID = context.mergeAPIUsers(rawGroups.map(group => group.owner));
      context.setState(current => ({
        apiUsersByID,
        apiGroupsByID: context.mergeGroupResponses(rawGroups, current.apiGroupsByID),
        groupIDs: reset
          ? incomingIDs
          : current.groupIDs.concat(incomingIDs.filter(id => current.groupIDs.indexOf(id) < 0)),
        groupsNextCursor: page.next_cursor || null,
        groupsPending: false, groupsLoading: false, groupsError: ''
      }));
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !context.groupsDirectoryGate.isCurrent(generation)) return;
      context.setState({
        groupsPending: false, groupsLoading: false,
        groupsError: requestErrorMessage(error, 'Could not load groups. Please try again.')
      });
    }
  };

  context.loadGroupInvitationInbox = async (reset = true) => {
    const authGeneration = context.authGate.current();
    const generation = reset ? context.groupInvitationInboxGate.begin() : context.groupInvitationInboxGate.current();
    if (!reset && context.state.groupInvitationInboxLoading) return;
    const cursor = reset ? null : context.state.groupInvitationInboxNextCursor;
    if (!reset && !cursor) return;
    context.setState({ groupInvitationInboxLoading: true, groupInvitationInboxError: '' });
    try {
      const page = await AuthAPI.groupInvitationInbox(cursor, 20);
      if (!context.authGate.isCurrent(authGeneration) || !context.groupInvitationInboxGate.isCurrent(generation)) return;
      const rawInvitations = page.invitations || [];
      const rawGroups = rawInvitations.map(item => item.group);
      const apiUsersByID = context.mergeAPIUsers(rawGroups.map(group => group.owner));
      const mapped = rawInvitations.map(item => ({ group: context.mapAPIGroup(item.group), createdAt: item.created_at }));
      context.setState(current => ({
        apiUsersByID,
        apiGroupsByID: context.mergeGroupResponses(rawGroups, current.apiGroupsByID),
        groupInvitationInbox: reset ? mapped : current.groupInvitationInbox.concat(mapped),
        groupInvitationInboxNextCursor: page.next_cursor || null,
        groupInvitationInboxLoading: false, groupInvitationInboxError: ''
      }));
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !context.groupInvitationInboxGate.isCurrent(generation)) return;
      context.setState({
        groupInvitationInboxLoading: false,
        groupInvitationInboxError: requestErrorMessage(error, 'Could not load group invitations.')
      });
    }
  };

  context.openGroup = (groupID) => {
    groupID = Number(groupID);
    if (!Number.isInteger(groupID) || groupID <= 0) return;
    if (dependencies.navigation && !dependencies.navigation.isApplying()) {
      dependencies.navigation.group(groupID);
      return;
    }
    context.stopTyping();
    context.groupDetailGate.begin();
    context.groupMembersGate.begin();
    context.groupRequestsGate.begin();
    context.groupInvitationsGate.begin();
	context.groupPostsGate.begin();
	context.groupEventsGate.begin();
	context.groupEventCreateGate.begin();
	context.invalidateGroupEventResponses();
	context.purgeCommentStates(context.state.groupPosts.map(post => post.id));
	context.setState(Object.assign({
      screen: 'group', groupId: groupID, groupTab: 'posts', inviteOpen: false,
      groupLoading: true, groupError: '', groupMembers: [], groupMembersNextCursor: null,
      groupMembersLoading: true, groupMembersError: '', groupRequests: [], groupRequestsNextCursor: null,
      groupRequestsLoading: false, groupRequestsError: '', groupInvitations: [], groupInvitationsNextCursor: null,
      groupInvitationsLoading: false, groupInvitationsError: '', groupInviteUserID: ''
	}, emptyGroupPostState(), emptyGroupEventState()));
    context.loadGroupDetail(groupID);
    context.loadGroupMembers(groupID, true);
  };

  context.loadGroupDetail = async (groupID) => {
    groupID = Number(groupID);
    const authGeneration = context.authGate.current();
    const accessGate = context.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const generation = context.groupDetailGate.begin();
    context.setState({ groupLoading: true, groupError: '' });
    try {
      const raw = await AuthAPI.group(groupID);
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !context.groupDetailGate.isCurrent(generation) || Number(context.state.groupId) !== groupID
      ) return;
      const apiUsersByID = context.mergeAPIUsers([raw.owner]);
      const mapped = context.mapAPIGroup(raw);
      context.setState(current => ({
        apiUsersByID,
        apiGroupsByID: Object.assign({}, current.apiGroupsByID, { [String(groupID)]: mapped }),
        groupLoading: false, groupError: ''
      }));
      if (mapped.state === 'owner') {
        context.loadGroupRequests(groupID, true);
        context.loadGroupInvitations(groupID, true);
      }
	  if ((mapped.state === 'owner' || mapped.state === 'member') && !context.groupAccessIsRevoked(groupID)) {
		context.loadGroupPosts(groupID, true);
		context.loadGroupEvents(groupID, true);
	  }
    } catch (error) {
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !context.groupDetailGate.isCurrent(generation) || Number(context.state.groupId) !== groupID
      ) return;
      context.setState({ groupLoading: false, groupError: requestErrorMessage(error, 'Could not load this group.') });
    }
  };

  context.loadGroupMembers = async (groupID, reset = true) => {
    groupID = Number(groupID);
    const authGeneration = context.authGate.current();
    const accessGate = context.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const generation = reset ? context.groupMembersGate.begin() : context.groupMembersGate.current();
    if (!reset && context.state.groupMembersLoading) return;
    const cursor = reset ? null : context.state.groupMembersNextCursor;
    if (!reset && !cursor) return;
    context.setState({ groupMembersLoading: true, groupMembersError: '' });
    try {
      const page = await AuthAPI.groupMembers(groupID, cursor, 20);
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !context.groupMembersGate.isCurrent(generation) || Number(context.state.groupId) !== groupID
      ) return;
      const rawMembers = page.members || [];
      const apiUsersByID = context.mergeAPIUsers(rawMembers.map(member => member.user));
      const mapped = rawMembers.map(member => ({
        userID: Number(member.user.id), status: member.status, createdAt: member.created_at
      }));
      context.setState(current => ({
        apiUsersByID,
        groupMembers: reset ? mapped : current.groupMembers.concat(mapped),
        groupMembersNextCursor: page.next_cursor || null,
        groupMembersLoading: false, groupMembersError: ''
      }));
    } catch (error) {
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !context.groupMembersGate.isCurrent(generation) || Number(context.state.groupId) !== groupID
      ) return;
      context.setState({ groupMembersLoading: false, groupMembersError: requestErrorMessage(error, 'Could not load members.') });
    }
  };

  context.loadGroupRequests = async (groupID, reset = true) => {
    return context.loadGroupOwnerList(groupID, reset, 'requests');
  };

  context.loadGroupInvitations = async (groupID, reset = true) => {
    return context.loadGroupOwnerList(groupID, reset, 'invitations');
  };

  context.loadGroupOwnerList = async (groupID, reset, kind) => {
    groupID = Number(groupID);
    const isRequests = kind === 'requests';
    const gate = isRequests ? context.groupRequestsGate : context.groupInvitationsGate;
    const stateKey = isRequests ? 'groupRequests' : 'groupInvitations';
    const cursorKey = stateKey + 'NextCursor';
    const loadingKey = stateKey + 'Loading';
    const errorKey = stateKey + 'Error';
    const authGeneration = context.authGate.current();
    const accessGate = context.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    const generation = reset ? gate.begin() : gate.current();
    if (!reset && context.state[loadingKey]) return;
    const cursor = reset ? null : context.state[cursorKey];
    if (!reset && !cursor) return;
    context.setState({ [loadingKey]: true, [errorKey]: '' });
    try {
      const page = isRequests
        ? await AuthAPI.groupJoinRequests(groupID, cursor, 20)
        : await AuthAPI.groupInvitations(groupID, cursor, 20);
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !gate.isCurrent(generation) || Number(context.state.groupId) !== groupID
      ) return;
      const rawItems = page[stateKey === 'groupRequests' ? 'requests' : 'invitations'] || [];
      const apiUsersByID = context.mergeAPIUsers(rawItems.map(item => item.user));
      const mapped = rawItems.map(item => ({ userID: Number(item.user.id), createdAt: item.created_at }));
      context.setState(current => ({
        apiUsersByID,
        [stateKey]: reset ? mapped : current[stateKey].concat(mapped),
        [cursorKey]: page.next_cursor || null,
        [loadingKey]: false, [errorKey]: ''
      }));
    } catch (error) {
      if (
        !context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration) ||
        !gate.isCurrent(generation) || Number(context.state.groupId) !== groupID
      ) return;
      context.setState({ [loadingKey]: false, [errorKey]: requestErrorMessage(error, 'Could not load owner controls.') });
    }
  };

  context.applyAuthoritativeGroup = function (raw, invalidateInbox) {
    const group = context.mapAPIGroup(raw);
    const key = String(group.id);
    context.groupsDirectoryGate.begin();
    context.groupGeneration(group.id).begin();
    if (invalidateInbox) context.groupInvitationInboxGate.begin();
    const apiUsersByID = context.mergeAPIUsers([raw.owner]);
    context.setState(current => ({
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
  };

  context.runGroupMutation = async (groupID, operation, options) => {
    groupID = Number(groupID);
    const key = String(groupID);
    if (!Number.isInteger(groupID) || groupID <= 0 || context.state.groupMutationPendingByID[key]) return false;
    const authGeneration = context.authGate.current();
    const accessGate = context.groupGeneration(groupID);
    const accessGeneration = accessGate.current();
    context.setState({
      groupMutationPendingByID: Object.assign({}, context.state.groupMutationPendingByID, { [key]: true }),
      groupMutationErrorByID: Object.assign({}, context.state.groupMutationErrorByID, { [key]: '' })
    });
    try {
      const raw = await operation();
      const expectedRevokeAlreadyApplied = options && options.revokeGroupAccess && context.groupAccessIsRevoked(groupID);
      if (
        !context.authGate.isCurrent(authGeneration) ||
        (!accessGate.isCurrent(accessGeneration) && !expectedRevokeAlreadyApplied)
      ) return false;
      const group = context.applyAuthoritativeGroup(raw, options && options.invalidateInbox);
      if (options && options.revokeGroupAccess) context.revokeGroupAccess(groupID);
      if (options && options.restoreGroupAccess) context.restoreGroupAccess(group);
      if (Number(context.state.groupId) === groupID) {
        context.loadGroupMembers(groupID, true);
        if (group.state === 'owner') {
          context.loadGroupRequests(groupID, true);
          context.loadGroupInvitations(groupID, true);
        }
      }
      if (options && options.invalidateInbox) context.loadGroupInvitationInbox(true);
      if (options && options.purgeChat) context.purgeChat(ChatModel.chatKey('group', groupID));
      if (options && options.refreshChats) context.loadChats(true);
      return true;
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration)) return false;
      context.setState({
        groupMutationPendingByID: Object.assign({}, context.state.groupMutationPendingByID, { [key]: false }),
        groupMutationErrorByID: Object.assign({}, context.state.groupMutationErrorByID, {
          [key]: requestErrorMessage(error, 'Could not update group membership.')
        })
      });
      return false;
    }
  };

  context.requestGroupJoin = (groupID) => {
    const group = context.state.apiGroupsByID[String(Number(groupID))];
    if (!group) return;
    return context.runGroupMutation(group.id, () => group.state === 'requested'
      ? AuthAPI.cancelGroupJoin(group.id)
      : AuthAPI.requestGroupJoin(group.id));
  };

  context.acceptGroupInvitation = (groupID) => context.runGroupMutation(
	groupID, () => AuthAPI.acceptGroupInvitation(groupID), {
	  invalidateInbox: true, refreshChats: true, restoreGroupAccess: true
	}
  );

  context.declineGroupInvitation = (groupID) => context.runGroupMutation(
    groupID, () => AuthAPI.declineGroupInvitation(groupID), { invalidateInbox: true }
  );

  context.leaveGroup = (groupID) => context.runGroupMutation(
	groupID, () => AuthAPI.leaveGroup(groupID), {
	  purgeChat: true, refreshChats: true, revokeGroupAccess: true
	}
  );

  context.acceptGroupRequest = (groupID, userID) => context.runGroupMutation(
    groupID, () => AuthAPI.acceptGroupJoinRequest(groupID, userID)
  );

  context.rejectGroupRequest = (groupID, userID) => context.runGroupMutation(
    groupID, () => AuthAPI.rejectGroupJoinRequest(groupID, userID)
  );

  context.createGroup = async () => {
    const title = context.state.ngName.trim();
    const description = context.state.ngDesc.trim();
    if (!title || !description || context.state.groupCreatePending) return;
    const authGeneration = context.authGate.current();
    context.setState({ groupCreatePending: true, groupCreateError: '' });
    try {
      const raw = await AuthAPI.createGroup(title, description);
      if (!context.authGate.isCurrent(authGeneration)) return;
      const group = context.applyAuthoritativeGroup(raw, false);
      context.revokedGroupAccessIDs.delete(String(group.id));
      context.loadChats(true);
      context.setState({
        groupCreatePending: false, groupCreateError: '', createOpen: false, ngName: '', ngDesc: ''
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration)) return;
      context.setState({
        groupCreatePending: false,
        groupCreateError: requestErrorMessage(error, 'Could not create the group. Your draft was kept.')
      });
    }
  };

  context.inviteSelectedUser = () => {
    const groupID = Number(context.state.groupId);
    const userID = Number(context.state.groupInviteUserID);
    if (!Number.isInteger(groupID) || !Number.isInteger(userID) || userID <= 0) return;
    const mutation = context.runGroupMutation(groupID, () => AuthAPI.inviteToGroup(groupID, userID));
    if (!mutation || typeof mutation.then !== 'function') return mutation;
    return mutation.then(applied => {
      if (
        applied && Number(context.state.groupId) === groupID &&
        Number(context.state.groupInviteUserID) === userID
      ) context.setState({ groupInviteUserID: '' });
      return applied;
    });
  };

  context.toggleCreate = () => context.setState({ createOpen: !context.state.createOpen });

  context.updateCreateField = (field, value) => {
    if (field !== 'ngName' && field !== 'ngDesc') return;
    context.setState({ [field]: value });
  };

  context.selectTab = tab => {
    if (['posts', 'events', 'members'].indexOf(tab) < 0) return;
    context.setState({ groupTab: tab });
  };

  context.selectInviteCandidate = userID => {
    context.setState({ groupInviteUserID: String(Number(userID)) });
  };

  context.updatePostComposer = value => {
    context.setState({ groupPostComposerText: value, groupPostComposerError: '' });
  };

  context.toggleInvite = () => {
    const opening = !context.state.inviteOpen;
    context.setState({ inviteOpen: opening });
    if (opening && !context.state.directoryUserIDs.length) context.loadDirectory(true);
  };

    return createController('groups', dependencies, {
      groupAccessIsRevoked: context.groupAccessIsRevoked,
      revokeGroupAccess: context.revokeGroupAccess,
      restoreGroupAccess: context.restoreGroupAccess,
      loadGroupPosts: context.loadGroupPosts,
      pickGroupPostMedia: context.pickGroupPostMedia,
      onGroupPostMedia: context.onGroupPostMedia,
      removeGroupPostMedia: context.removeGroupPostMedia,
      sendGroupPost: context.sendGroupPost,
      groupGeneration: context.groupGeneration,
      mapAPIGroup: context.mapAPIGroup,
      mergeGroupResponses: context.mergeGroupResponses,
      loadGroups: context.loadGroups,
      loadGroupInvitationInbox: context.loadGroupInvitationInbox,
      openGroup: context.openGroup,
      loadGroupDetail: context.loadGroupDetail,
      loadGroupMembers: context.loadGroupMembers,
      loadGroupRequests: context.loadGroupRequests,
      loadGroupInvitations: context.loadGroupInvitations,
      loadGroupOwnerList: context.loadGroupOwnerList,
      applyAuthoritativeGroup: context.applyAuthoritativeGroup,
      runGroupMutation: context.runGroupMutation,
      requestGroupJoin: context.requestGroupJoin,
      acceptGroupInvitation: context.acceptGroupInvitation,
      declineGroupInvitation: context.declineGroupInvitation,
      leaveGroup: context.leaveGroup,
      acceptGroupRequest: context.acceptGroupRequest,
      rejectGroupRequest: context.rejectGroupRequest,
      createGroup: context.createGroup,
      inviteSelectedUser: context.inviteSelectedUser,
      toggleCreate: context.toggleCreate,
      updateCreateField: context.updateCreateField,
      selectTab: context.selectTab,
      selectInviteCandidate: context.selectInviteCandidate,
      updatePostComposer: context.updatePostComposer,
      toggleInvite: context.toggleInvite
    }, function (state) {
      var me = USERS.me;
      var groupCards = state.groupIDs
        .map(function (groupID) {
          return state.apiGroupsByID[String(groupID)];
        })
        .filter(Boolean)
        .map(function (group, index) {
          var pending = !!state.groupMutationPendingByID[String(group.id)];
          var accessRevoked = context.groupAccessIsRevoked(group.id);
          return {
            name: group.name,
            desc: group.desc,
            membersLabel: num(group.members),
            cover: cover(group.color),
            owner: context.apiUser(group.ownerID),
            delay: (index * 0.05).toFixed(2) + 's',
            pending: pending,
            error: state.groupMutationErrorByID[String(group.id)] || '',
            hasError: !!state.groupMutationErrorByID[String(group.id)],
            isJoined: !accessRevoked &&
              (group.state === 'member' || group.state === 'owner'),
            isOwner: !accessRevoked && group.state === 'owner',
            isMember: !accessRevoked && group.state === 'member',
            isNone: group.state === 'none',
            isRequested: group.state === 'requested',
            isInvited: group.state === 'invited',
            open: function () { return context.openGroup(group.id); },
            join: function () { return context.requestGroupJoin(group.id); },
            leave: function () { return context.leaveGroup(group.id); },
            acceptInvite: function () {
              return context.acceptGroupInvitation(group.id);
            },
            declineInvite: function () {
              return context.declineGroupInvitation(group.id);
            }
          };
        });
      var groupInboxCards = state.groupInvitationInbox.map(function (item) {
        var group = state.apiGroupsByID[String(item.group.id)] || item.group;
        var pending = !!state.groupMutationPendingByID[String(group.id)];
        return {
          name: group.name,
          owner: context.apiUser(group.ownerID),
          pending: pending,
          accept: function () {
            return context.acceptGroupInvitation(group.id);
          },
          decline: function () {
            return context.declineGroupInvitation(group.id);
          },
          open: function () { return context.openGroup(group.id); }
        };
      });
      var group = state.apiGroupsByID[String(Number(state.groupId))] || {
        id: Number(state.groupId) || 0,
        name: '',
        desc: '',
        members: 0,
        state: 'none',
        ownerID: 0,
        color: GROUP_COLORS[0]
      };
      var accessRevoked = context.groupAccessIsRevoked(group.id);
      var isOwner = !accessRevoked && group.state === 'owner';
      var canChat = !accessRevoked &&
        (group.state === 'owner' || group.state === 'member');
      var mutationPending =
        !!state.groupMutationPendingByID[String(group.id)];
      var mutationError =
        state.groupMutationErrorByID[String(group.id)] || '';
      var tabs = [
        { key: 'posts', label: 'Posts' },
        { key: 'events', label: 'Events' },
        { key: 'members', label: 'Members' }
      ].map(function (tab) {
        var selected = state.groupTab === tab.key;
        return {
          label: tab.label,
          color: selected ? 'var(--text)' : 'var(--text3)',
          bd: selected ? 'var(--accent)' : 'transparent',
          pick: function () { return context.selectTab(tab.key); }
        };
      });
      var members = state.groupMembers.map(function (member) {
        return {
          user: context.apiUser(member.userID),
          isOwner: member.status === 'owner',
          goProfile: function () { return context.openProfile(member.userID); }
        };
      });
      var requests = (isOwner ? state.groupRequests : []).map(function (request) {
        return {
          user: context.apiUser(request.userID),
          disabled: mutationPending,
          pending: true,
          done: false,
          doneLabel: '',
          accept: function () {
            return context.acceptGroupRequest(group.id, request.userID);
          },
          decline: function () {
            return context.rejectGroupRequest(group.id, request.userID);
          }
        };
      });
      var invitations = (isOwner ? state.groupInvitations : []).map(function (invitation) {
        return { user: context.apiUser(invitation.userID) };
      });
      var excludedInviteIDs = {};
      state.groupMembers.forEach(function (item) {
        excludedInviteIDs[String(item.userID)] = true;
      });
      state.groupRequests.forEach(function (item) {
        excludedInviteIDs[String(item.userID)] = true;
      });
      state.groupInvitations.forEach(function (item) {
        excludedInviteIDs[String(item.userID)] = true;
      });
      if (me.apiId) excludedInviteIDs[String(me.apiId)] = true;
      var inviteCandidatesReady = !state.groupMembersLoading &&
        !state.groupRequestsLoading && !state.groupInvitationsLoading;
      var inviteCandidates = (inviteCandidatesReady
        ? state.directoryUserIDs
        : [])
        .filter(function (userID) {
          return !excludedInviteIDs[String(userID)];
        })
        .map(function (userID) {
          var user = context.apiUser(userID);
          var selected = String(state.groupInviteUserID) === String(userID);
          return {
            user: user,
            selected: selected,
            label: user.name,
            initials: user.initials,
            color: user.color,
            bg: selected ? 'var(--soft)' : 'transparent',
            bd: selected ? 'var(--accent)' : 'var(--border)',
            tc: selected ? 'var(--accent)' : 'var(--text2)',
            pick: function () { return context.selectInviteCandidate(userID); }
          };
        });
      var posts = state.groupPosts.map(function (post, index) {
        return presentPost(state, post, {
          delay: (index * 0.05).toFixed(2) + 's'
        });
      });

      return {
        createOpen: state.createOpen,
        toggleCreate: context.toggleCreate,
        ngName: state.ngName,
        onNgName: function (event) {
          return context.updateCreateField('ngName', event.target.value);
        },
        ngDesc: state.ngDesc,
        onNgDesc: function (event) {
          return context.updateCreateField('ngDesc', event.target.value);
        },
        createGroup: context.createGroup,
        groupCreatePending: state.groupCreatePending,
        groupCreateHasError: !!state.groupCreateError,
        groupCreateError: state.groupCreateError,
        groupCards: groupCards,
        groupsLoading: state.groupsLoading,
        groupsReady: !state.groupsLoading,
        groupsHasError: !!state.groupsError,
        groupsError: state.groupsError,
        retryGroups: function () { return context.loadGroups(true); },
        groupsHasMore: !!state.groupsNextCursor,
        loadMoreGroups: function () { return context.loadGroups(false); },
        groupsLoadMoreLabel: state.groupsPending &&
          !state.groupsLoading ? 'Loading…' : 'Load more',
        groupInboxCards: groupInboxCards,
        groupInboxLoading: state.groupInvitationInboxLoading,
        groupInboxHasError: !!state.groupInvitationInboxError,
        groupInboxError: state.groupInvitationInboxError,
        groupInboxHasItems: groupInboxCards.length > 0,
        groupInboxHasMore: !!state.groupInvitationInboxNextCursor,
        loadMoreGroupInbox: function () {
          return context.loadGroupInvitationInbox(false);
        },
        groupLoading: state.groupLoading,
        groupHasError: !!state.groupError,
        groupError: state.groupError,
        retryGroup: function () { return context.openGroup(group.id); },
        gName: group.name,
        gDesc: group.desc,
        gMembersLabel: num(group.members),
        gCover: cover(group.color),
        gIsOwner: isOwner,
        gOwner: context.apiUser(group.ownerID),
        gMutationPending: mutationPending,
        gMutationHasError: !!mutationError,
        gMutationError: mutationError,
        gIsNone: group.state === 'none',
        gIsRequested: group.state === 'requested',
        gIsInvited: group.state === 'invited',
        gIsMember: !accessRevoked && group.state === 'member',
        gCanChat: canChat,
        gCanContent: canChat,
        gContentLocked: !canChat,
        gOpenChat: function () { return context.openGroupChat(group.id); },
        gRequestJoin: function () { return context.requestGroupJoin(group.id); },
        gAcceptInvitation: function () {
          return context.acceptGroupInvitation(group.id);
        },
        gDeclineInvitation: function () {
          return context.declineGroupInvitation(group.id);
        },
        gLeave: function () { return context.leaveGroup(group.id); },
        gBack: context.goGroups,
        gTabs: tabs,
        gTabPosts: state.groupTab === 'posts',
        gTabEvents: state.groupTab === 'events',
        gTabMembers: state.groupTab === 'members',
        gPosts: posts,
        groupPostsLoading: state.groupPostsLoading,
        groupPostsHasError: !!state.groupPostsError,
        groupPostsError: state.groupPostsError,
        groupPostsEmpty: !state.groupPostsLoading &&
          !state.groupPostsError && posts.length === 0,
        groupPostsHasMore: !!state.groupPostsNextCursor,
        groupPostsLoadMoreDisabled: state.groupPostsPending,
        retryGroupPosts: function () {
          return context.loadGroupPosts(group.id, true);
        },
        loadMoreGroupPosts: function () {
          return context.loadGroupPosts(group.id, false);
        },
        groupPostComposerText: state.groupPostComposerText,
        onGroupPostComposerText: function (event) {
          return context.updatePostComposer(event.target.value);
        },
        groupPostComposerFileName: state.groupPostComposerFileName,
        groupPostComposerHasFile: !!state.groupPostComposerFile,
        groupPostComposerPending: state.groupPostComposerPending,
        groupPostComposerHasError: !!state.groupPostComposerError,
        groupPostComposerError: state.groupPostComposerError,
        groupPostComposerDisabled: state.groupPostComposerPending ||
          (!state.groupPostComposerText.trim() && !state.groupPostComposerFile),
        groupPostComposerButtonLabel: state.groupPostComposerPending
          ? 'Posting…'
          : 'Post',
        pickGroupPostMedia: context.pickGroupPostMedia,
        onGroupPostMedia: context.onGroupPostMedia,
        removeGroupPostMedia: context.removeGroupPostMedia,
        sendGroupPost: context.sendGroupPost,
        gMembers: members,
        gRequests: requests,
        gInvitations: invitations,
        gHasRequests: requests.length > 0,
        gHasInvitations: invitations.length > 0,
        groupMembersLoading: state.groupMembersLoading,
        groupMembersHasError: !!state.groupMembersError,
        groupMembersError: state.groupMembersError,
        groupMembersHasMore: !!state.groupMembersNextCursor,
        loadMoreGroupMembers: function () {
          return context.loadGroupMembers(group.id, false);
        },
        groupRequestsLoading: state.groupRequestsLoading,
        groupRequestsHasError: !!state.groupRequestsError,
        groupRequestsError: state.groupRequestsError,
        groupRequestsHasMore: !!state.groupRequestsNextCursor,
        loadMoreGroupRequests: function () {
          return context.loadGroupRequests(group.id, false);
        },
        groupInvitationsLoading: state.groupInvitationsLoading,
        groupInvitationsHasError: !!state.groupInvitationsError,
        groupInvitationsError: state.groupInvitationsError,
        groupInvitationsHasMore: !!state.groupInvitationsNextCursor,
        loadMoreGroupInvitations: function () {
          return context.loadGroupInvitations(group.id, false);
        },
        inviteOpen: state.inviteOpen && canChat,
        toggleInvite: context.toggleInvite,
        inviteCandidates: inviteCandidates,
        inviteSendDisabled: mutationPending || !state.groupInviteUserID,
        inviteSelectedUser: context.inviteSelectedUser,
        inviteLoadMore: function () { return context.loadDirectory(false); },
        inviteHasMore: !!state.directoryNextCursor,
        inviteDirectoryLoading: state.directoryLoading
      };
    }, {});
  };
});
