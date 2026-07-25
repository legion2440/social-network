(function (root, factory) {
  var create = factory();
  if (typeof module === 'object' && module.exports) module.exports = create;
  if (root) root.createPostPresenter = create;
})(typeof window !== 'undefined' ? window : globalThis, function () {
  return function createPostPresenter(dependencies) {
    if (
      !dependencies ||
      typeof dependencies.resolveUser !== 'function' ||
      typeof dependencies.formatDate !== 'function' ||
      typeof dependencies.emptyCommentState !== 'function' ||
      !dependencies.callbacks ||
      !dependencies.icons
    ) {
      throw new TypeError('post presenter requires user, date, comment, callback and icon dependencies');
    }

    var callbacks = dependencies.callbacks;
    var icons = dependencies.icons;

    return function presentPost(state, post, options) {
      options = options || {};
      var commentState = state.commentsByPostID[String(Number(post.id))] ||
        dependencies.emptyCommentState();
      var privacyMeta = {
        public: { icon: icons.globe, label: 'Public' },
        followers: { icon: icons.users, label: 'Followers' },
        selected: { icon: icons.lock, label: 'Selected' }
      };
      var privacy = privacyMeta[post.privacy] || privacyMeta.public;
      return Object.assign({}, post, {
        delay: options.delay || '',
        user: dependencies.resolveUser(post.apiAuthorID),
        hasText: !!String(post.text || '').trim(),
        privacyIcon: privacy.icon,
        privacyLabel: privacy.label,
        hasGroup: Number.isInteger(Number(post.groupID)) && Number(post.groupID) > 0,
        groupTitle: post.groupTitle || '',
        hasImage: !!post.mediaUrl,
        mediaUrl: post.mediaUrl || '',
        commentCount: String(post.commentsCount || 0),
        showComments: !!state.openComments[post.id],
        comments: commentState.comments.map(function (comment) {
          return Object.assign({}, comment, {
            user: dependencies.resolveUser(comment.apiAuthorID),
            time: dependencies.formatDate(comment.createdAt),
            hasText: !!String(comment.text || '').trim(),
            hasMedia: !!comment.mediaUrl
          });
        }),
        draft: commentState.draft,
        commentsLoading: commentState.loading,
        commentsPending: commentState.pending,
        commentsHasError: !!commentState.error,
        commentsError: commentState.error,
        commentsHasMore: !!commentState.nextCursor,
        commentCreatePending: commentState.createPending,
        commentCreateHasError: !!commentState.createError,
        commentCreateError: commentState.createError,
        commentSendDisabled: commentState.createPending ||
          (!commentState.draft.trim() && !commentState.mediaFile),
        commentMediaInputID: callbacks.commentMediaInputID(post.id),
        commentHasMedia: !!commentState.mediaFile,
        commentMediaFileName: commentState.mediaFileName,
        commentMediaPreviewURL: commentState.mediaPreviewURL,
        commentMediaControlsDisabled: commentState.createPending,
        commentSendLabel: commentState.createPending ? '…' : 'Send',
        onToggleComments: function () { return callbacks.togglePostComments(post.id); },
        onDraft: function (event) { return callbacks.setCommentDraft(post.id, event.target.value); },
        onKey: function (event) {
          if (event.key !== 'Enter') return;
          event.preventDefault();
          return callbacks.createComment(post.id);
        },
        onSendComment: function () { return callbacks.createComment(post.id); },
        onCommentMedia: function (event) { return callbacks.selectCommentMedia(post.id, event); },
        onChooseCommentMedia: function () { return callbacks.chooseCommentMedia(post.id); },
        onRemoveCommentMedia: function () { return callbacks.removeCommentMedia(post.id); },
        loadMoreComments: function () { return callbacks.loadComments(post.id, false); },
        retryComments: function () { return callbacks.loadComments(post.id, true); },
        goProfile: function () { return callbacks.openProfile(post.apiAuthorID); },
        goGroup: function () { return callbacks.openGroup(post.groupID); }
      });
    };
  };
});
