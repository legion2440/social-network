(function (root, factory) {
  var create = factory(root && root.createFeatureController, root && root.createControllerContext);
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createFeedController = create;
})(typeof window !== 'undefined' ? window : globalThis, function (createController, createContext) {
  return function createFeedController(dependencies) {
    var context = createContext(dependencies);
    var AuthAPI = dependencies.api;
    var USERS = dependencies.session.users;
    var UserModel = dependencies.models.users;
    var PostModel = dependencies.models.posts;
    var CommentModel = dependencies.models.comments;
    var emptyCommentState = dependencies.helpers.emptyCommentState;
    var requestErrorMessage = dependencies.helpers.requestErrorMessage;
    var num = dependencies.helpers.num;
    var IC = dependencies.values.IC;

  context.commentState = function (postID) {
    return context.state.commentsByPostID[String(Number(postID))] || emptyCommentState();
  };

  context.commentAccessGate = function (postID) {
    const key = String(Number(postID));
    if (!context.commentAccessGatesByPostID[key]) context.commentAccessGatesByPostID[key] = UserModel.createRequestGate();
    return context.commentAccessGatesByPostID[key];
  };

  context.commentLoadGate = function (postID) {
    const key = String(Number(postID));
    if (!context.commentLoadGatesByPostID[key]) context.commentLoadGatesByPostID[key] = UserModel.createRequestGate();
    return context.commentLoadGatesByPostID[key];
  };

  context.maxPostCommentsCount = function (postID, ...collections) {
    postID = Number(postID);
    return collections.reduce((maximum, posts) => (posts || []).reduce((currentMaximum, post) => (
      Number(post.id) === postID ? Math.max(currentMaximum, Number(post.commentsCount) || 0) : currentMaximum
    ), maximum), 0);
  };

  context.mergePostCommentsCounts = function (incoming, ...localCollections) {
    return (incoming || []).map(post => Object.assign({}, post, {
      commentsCount: Math.max(
        Number(post.commentsCount) || 0,
        context.maxPostCommentsCount(post.id, ...localCollections)
      )
    }));
  };

  context.patchCommentState = function (postID, patch) {
    const key = String(Number(postID));
    context.setState(state => {
      const entries = Object.assign({}, state.commentsByPostID);
      entries[key] = Object.assign({}, emptyCommentState(), entries[key] || {}, patch || {});
      return { commentsByPostID: entries };
    });
  };

  context.revokeCommentPreview = function (previewURL) {
    if (
      previewURL &&
      typeof URL !== 'undefined' &&
      URL &&
      typeof URL.revokeObjectURL === 'function'
    ) {
      URL.revokeObjectURL(previewURL);
    }
  };

  context.disposeAllCommentPreviews = function () {
    const entries = (context.state && context.state.commentsByPostID) || {};
    Object.keys(entries).forEach(key => context.revokeCommentPreview(entries[key] && entries[key].mediaPreviewURL));
  };

  context.commentMediaInputID = function (postID) {
    return 'comment-media-' + String(Number(postID));
  };

  context.resetCommentMediaInput = function (postID) {
    if (typeof document === 'undefined' || !document || typeof document.getElementById !== 'function') return;
    const input = document.getElementById(context.commentMediaInputID(postID));
    if (input) input.value = '';
  };

  context.selectCommentMedia = (postID, event) => {
    const state = context.commentState(postID);
    if (state.createPending) return;
    const file = event && event.target && event.target.files && event.target.files[0];
    if (!file) return;
    context.revokeCommentPreview(state.mediaPreviewURL);
    const previewURL = (
      typeof URL !== 'undefined' &&
      URL &&
      typeof URL.createObjectURL === 'function'
    ) ? URL.createObjectURL(file) : '';
    context.patchCommentState(postID, {
      mediaFile: file,
      mediaFileName: file.name || 'attachment',
      mediaPreviewURL: previewURL,
      createError: ''
    });
  };

  context.removeCommentMedia = (postID) => {
    const state = context.commentState(postID);
    if (state.createPending) return;
    context.revokeCommentPreview(state.mediaPreviewURL);
    context.resetCommentMediaInput(postID);
    context.patchCommentState(postID, {
      mediaFile: null,
      mediaFileName: '',
      mediaPreviewURL: '',
      createError: ''
    });
  };

  context.purgeCommentStates = function (postIDs) {
    const removed = {};
    (postIDs || []).forEach(postID => {
      const key = String(Number(postID));
      if (key !== 'NaN') {
        removed[key] = true;
        context.commentAccessGate(key).begin();
      }
    });
    if (!Object.keys(removed).length) return;
    Object.keys(removed).forEach(key => {
      const state = context.state.commentsByPostID[key];
      context.revokeCommentPreview(state && state.mediaPreviewURL);
      context.resetCommentMediaInput(key);
    });
    context.setState(state => {
      const entries = Object.assign({}, state.commentsByPostID);
      const openComments = Object.assign({}, state.openComments);
      Object.keys(removed).forEach(key => {
        delete entries[key];
        delete openComments[key];
      });
      return { commentsByPostID: entries, openComments };
    });
  };

  context.loadComments = async (postID, reset) => {
    postID = Number(postID);
    if (!Number.isInteger(postID) || postID <= 0) return;
    const authGeneration = context.authGate.current();
    const accessGate = context.commentAccessGate(postID);
    const accessGeneration = accessGate.current();
    const loadGate = context.commentLoadGate(postID);
    const loadGeneration = reset ? loadGate.begin() : loadGate.current();
    const state = context.commentState(postID);
    if (!reset && state.pending) return;
    const cursor = reset ? null : state.nextCursor;
    if (!reset && !cursor) return;
    context.patchCommentState(postID, { pending: true, loading: !!reset, error: '' });
    try {
      const page = await AuthAPI.postComments(postID, cursor, 20);
      if (
        !context.authGate.isCurrent(authGeneration) ||
        !accessGate.isCurrent(accessGeneration) ||
        !loadGate.isCurrent(loadGeneration)
      ) return;
      const rawComments = page.comments || [];
      const incoming = rawComments.map(CommentModel.normalizeCommentResponse);
      const apiUsersByID = context.mergeAPIUsers(rawComments.map(comment => comment.author));
      const latest = context.commentState(postID);
      context.setState({ apiUsersByID });
      context.patchCommentState(postID, {
        comments: CommentModel.mergeComments(latest.comments, incoming),
        pending: false, loading: false, error: '', loaded: true,
        nextCursor: page.next_cursor || null
      });
    } catch (error) {
      if (
        !context.authGate.isCurrent(authGeneration) ||
        !accessGate.isCurrent(accessGeneration) ||
        !loadGate.isCurrent(loadGeneration)
      ) return;
      if (error && (error.status === 403 || error.status === 404)) {
        accessGate.begin();
        const latest = context.commentState(postID);
        context.revokeCommentPreview(latest.mediaPreviewURL);
        context.resetCommentMediaInput(postID);
        context.patchCommentState(postID, Object.assign(emptyCommentState(), {
          error: error.status === 403 ? 'You no longer have access to these comments.' : 'Post not found.'
        }));
        return;
      }
      context.patchCommentState(postID, {
        pending: false, loading: false,
        error: requestErrorMessage(error, 'Could not load comments. Please try again.')
      });
    }
  };

  context.togglePostComments = (postID) => {
    const key = String(Number(postID));
    const opening = !context.state.openComments[key];
    context.setState({ openComments: Object.assign({}, context.state.openComments, { [key]: opening }) });
    const state = context.commentState(postID);
    if (opening && !state.loaded && !state.pending) context.loadComments(postID, true);
  };

  context.setCommentDraft = (postID, value) => {
    context.patchCommentState(postID, { draft: value, createError: '' });
  };

  context.createComment = async (postID) => {
    postID = Number(postID);
    const state = context.commentState(postID);
    const text = state.draft.trim();
    if ((!text && !state.mediaFile) || state.createPending) return;
    const authGeneration = context.authGate.current();
    const accessGate = context.commentAccessGate(postID);
    const accessGeneration = accessGate.current();
	const countAtCreateStart = context.maxPostCommentsCount(postID, context.state.posts, context.state.profilePosts, context.state.groupPosts);
    context.patchCommentState(postID, { createPending: true, createError: '' });
    try {
      const formData = new FormData();
      formData.append('text', text);
      if (state.mediaFile) formData.append('media', state.mediaFile, state.mediaFileName || state.mediaFile.name || 'attachment');
      const response = await AuthAPI.createComment(postID, formData);
      if (!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration)) return;
      const comment = CommentModel.normalizeCommentResponse(response);
      const apiUsersByID = context.mergeAPIUsers([response.author]);
      const latest = context.commentState(postID);
      context.setState(current => ({
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
      context.revokeCommentPreview(latest.mediaPreviewURL);
      context.resetCommentMediaInput(postID);
      context.patchCommentState(postID, {
        comments: CommentModel.mergeComments(latest.comments, [comment]),
        draft: '', mediaFile: null, mediaFileName: '', mediaPreviewURL: '',
        createPending: false, createError: ''
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !accessGate.isCurrent(accessGeneration)) return;
      if (error && (error.status === 403 || error.status === 404)) {
        accessGate.begin();
        const latest = context.commentState(postID);
        context.revokeCommentPreview(latest.mediaPreviewURL);
        context.resetCommentMediaInput(postID);
        context.patchCommentState(postID, Object.assign(emptyCommentState(), {
          draft: text,
          createError: error.status === 403 ? 'You no longer have access to this post.' : 'Post not found.'
        }));
        return;
      }
      context.patchCommentState(postID, {
        createPending: false,
        createError: requestErrorMessage(error, 'Could not send the comment. Your draft was kept.')
      });
    }
  };

  context.loadFeed = async (reset) => {
    const authGeneration = context.authGate.current();
    const generation = reset ? context.feedGate.begin() : context.feedGate.current();
    if (!reset && context.state.feedPending) return;
    const cursor = reset ? null : context.state.feedNextCursor;
    if (!reset && !cursor) return;
    context.setState({ feedPending: true, feedLoading: !!reset, feedError: '' });
    try {
      const page = await AuthAPI.feed(cursor, 20);
      if (!context.authGate.isCurrent(authGeneration) || !context.feedGate.isCurrent(generation)) return;
      const mapped = (page.posts || []).map(post => context.mapAPIPost(post));
      const apiUsersByID = context.mergeAPIUsers((page.posts || []).map(post => post.author));
      context.setState(current => {
		const merged = context.mergePostCommentsCounts(mapped, current.posts, current.profilePosts, current.groupPosts);
        return {
          posts: reset ? merged : current.posts.concat(merged),
          apiUsersByID,
          feedLoading: false, feedPending: false,
          feedNextCursor: page.next_cursor || null, feedError: ''
        };
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !context.feedGate.isCurrent(generation)) return;
      context.setState({
        feedLoading: false, feedPending: false,
        feedError: requestErrorMessage(error, 'Could not load the feed. Please try again.')
      });
    }
  };

  context.loadPostFollowers = async () => {
    if (!USERS.me.apiId) return;
    const authGeneration = context.authGate.current();
    const generation = context.postFollowersGate.begin();
    context.setState({ postFollowersLoading: true });
    try {
      const response = await AuthAPI.followers(USERS.me.apiId);
      if (!context.authGate.isCurrent(authGeneration) || !context.postFollowersGate.isCurrent(generation)) return;
      const apiUsersByID = context.mergeAPIUsers(response.users || []);
      const followers = (response.users || []).map(user => apiUsersByID[String(user.id)]);
      context.setState({
        apiUsersByID,
        postFollowers: followers,
        postFollowersLoading: false,
        selectedFollowers: UserModel.pruneSelected(context.state.selectedFollowers, followers)
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration) || !context.postFollowersGate.isCurrent(generation)) return;
      context.setState({
        postFollowersLoading: false,
        composerError: requestErrorMessage(error, 'Could not load followers for selected posts.')
      });
    }
  };

  context.pickComposerMedia = () => {
    const input = document.getElementById('post-media');
    if (input) input.click();
  };

  context.onComposerMedia = (event) => {
    const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
    context.setState({ composerFile: file, composerFileName: file ? file.name : '', composerError: '' });
  };

  context.removeComposerMedia = () => {
    const input = document.getElementById('post-media');
    if (input) input.value = '';
    context.setState({ composerFile: null, composerFileName: '', composerError: '' });
  };

  context.sendPost = async () => {
    const s = context.state;
    if (s.composerPending || (!s.composerText.trim() && !s.composerFile)) return;
    const authGeneration = context.authGate.current();
    const selectedUserIDs = Object.keys(s.selectedFollowers)
      .filter(id => s.selectedFollowers[id])
      .map(id => Number(id));
    context.setState({ composerPending: true, composerError: '', privacyOpen: false });
    try {
      const form = PostModel.buildCreatePostForm({
        text: s.composerText,
        privacy: s.privacy,
        selectedUserIDs,
        media: s.composerFile
      }, FormData);
      const response = await AuthAPI.createPost(form);
      if (!context.authGate.isCurrent(authGeneration)) return;
      const post = context.mapAPIPost(response);
      const apiUsersByID = context.mergeAPIUsers([response.author]);
      const me = apiUsersByID[String(USERS.me.apiId)];
      if (me) me.postsCount = (me.postsCount || 0) + 1;
      const input = document.getElementById('post-media');
      if (input) input.value = '';
      context.setState({
        apiUsersByID,
        posts: [post].concat(context.state.posts.filter(item => item.id !== post.id)),
        profilePosts: Number(context.state.profileId) === USERS.me.apiId
          ? [post].concat(context.state.profilePosts.filter(item => item.id !== post.id))
          : context.state.profilePosts,
        composerText: '', composerFile: null, composerFileName: '',
        composerError: '', composerPending: false,
        privacy: 'public', privacyOpen: false, selectedFollowers: {}
      });
    } catch (error) {
      if (!context.authGate.isCurrent(authGeneration)) return;
      context.setState({
        composerPending: false,
        composerError: requestErrorMessage(error, 'Could not create the post. Your draft was kept.')
      });
    }
  };

  context.mapPost = function (p) {
    const s = context.state;
    const key = p.id;
    const privacyMeta = { public: { icon: IC.globe, label: 'Public' }, followers: { icon: IC.users, label: 'Followers' }, selected: { icon: IC.lock, label: 'Selected' } };
    const pm = privacyMeta[p.privacy] || privacyMeta.public;
    const commentState = context.commentState(p.id);
    const comments = commentState.comments;
    const user = context.apiUser(p.apiAuthorID);
    return Object.assign({}, p, {
      user,
      hasText: !!String(p.text || '').trim(),
      privacyIcon: pm.icon, privacyLabel: pm.label,
      hasGroup: Number.isInteger(Number(p.groupID)) && Number(p.groupID) > 0,
      groupTitle: p.groupTitle || '',
      hasImage: !!p.mediaUrl,
      mediaUrl: p.mediaUrl || '',
      commentCount: num(p.commentsCount || 0),
      showComments: !!s.openComments[key],
      comments: comments.map(c => Object.assign({}, c, {
        user: context.apiUser(c.apiAuthorID),
        time: context.formatPostTime(c.createdAt),
        hasText: !!String(c.text || '').trim(),
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
      commentSendDisabled: commentState.createPending || (!commentState.draft.trim() && !commentState.mediaFile),
      commentMediaInputID: context.commentMediaInputID(p.id),
      commentHasMedia: !!commentState.mediaFile,
      commentMediaFileName: commentState.mediaFileName,
      commentMediaPreviewURL: commentState.mediaPreviewURL,
      commentMediaControlsDisabled: commentState.createPending,
      commentSendLabel: commentState.createPending ? '…' : 'Send',
      onToggleComments: () => context.togglePostComments(p.id),
      onDraft: (e) => context.setCommentDraft(p.id, e.target.value),
      onKey: (e) => {
        if (e.key !== 'Enter') return;
        e.preventDefault();
        context.createComment(p.id);
      },
      onSendComment: () => context.createComment(p.id),
      onCommentMedia: (e) => context.selectCommentMedia(p.id, e),
      onChooseCommentMedia: () => {
        if (commentState.createPending || typeof document === 'undefined') return;
        const input = document.getElementById(context.commentMediaInputID(p.id));
        if (input) input.click();
      },
      onRemoveCommentMedia: () => context.removeCommentMedia(p.id),
      loadMoreComments: () => context.loadComments(p.id, false),
      retryComments: () => context.loadComments(p.id, true),
      goProfile: () => context.openProfile(p.apiAuthorID),
      goGroup: () => context.openGroup(p.groupID)
    });
  };

    return createController('feed', dependencies, {
      commentState: context.commentState,
      commentAccessGate: context.commentAccessGate,
      commentLoadGate: context.commentLoadGate,
      maxPostCommentsCount: context.maxPostCommentsCount,
      mergePostCommentsCounts: context.mergePostCommentsCounts,
      patchCommentState: context.patchCommentState,
      revokeCommentPreview: context.revokeCommentPreview,
      disposeAllCommentPreviews: context.disposeAllCommentPreviews,
      commentMediaInputID: context.commentMediaInputID,
      resetCommentMediaInput: context.resetCommentMediaInput,
      selectCommentMedia: context.selectCommentMedia,
      removeCommentMedia: context.removeCommentMedia,
      purgeCommentStates: context.purgeCommentStates,
      loadComments: context.loadComments,
      togglePostComments: context.togglePostComments,
      setCommentDraft: context.setCommentDraft,
      createComment: context.createComment,
      loadFeed: context.loadFeed,
      loadPostFollowers: context.loadPostFollowers,
      pickComposerMedia: context.pickComposerMedia,
      onComposerMedia: context.onComposerMedia,
      removeComposerMedia: context.removeComposerMedia,
      sendPost: context.sendPost,
      mapPost: context.mapPost
    }, function (state) {
      return { postCount: state && state.posts ? state.posts.length : 0, posts: state && state.posts ? state.posts.map(context.mapPost) : [] };
    }, { stop: context.disposeAllCommentPreviews });
  };
});
