(function (root, factory) {
  var library = factory();

  if (typeof module === 'object' && module.exports) {
    module.exports = library;
  }
  if (root) root.PostModel = library;
})(typeof window !== 'undefined' ? window : null, function () {
  function buildCreatePostForm(draft, FormDataCtor) {
    if (!draft || typeof FormDataCtor !== 'function') throw new TypeError('draft and FormData are required');
    var form = new FormDataCtor();
    form.append('text', String(draft.text || '').trim());
    form.append('privacy', draft.privacy);

    if (draft.privacy === 'selected') {
      var seen = {};
      (draft.selectedUserIDs || []).forEach(function (value) {
        var id = String(value);
        if (!seen[id]) {
          seen[id] = true;
          form.append('selected_user_id', id);
        }
      });
    }
    if (draft.media) form.append('media', draft.media, draft.media.name);
    return form;
  }

  function normalizePostResponse(post, currentUserID) {
    if (!post || !post.author) throw new TypeError('post author is required');
    var authorID = Number(post.author.id);
    return {
      id: String(post.id),
      apiAuthorID: authorID,
      isOwn: authorID === Number(currentUserID),
      text: String(post.text || ''),
      privacy: post.privacy,
      mediaUrl: post.media_url || '',
      createdAt: post.created_at,
      author: {
        apiId: authorID,
        firstName: post.author.first_name || '',
        lastName: post.author.last_name || '',
        nickname: post.author.nickname || '',
        avatarUrl: post.author.avatar_url || '',
        isPrivate: post.author.is_private === true
      }
    };
  }

  return {
    buildCreatePostForm: buildCreatePostForm,
    normalizePostResponse: normalizePostResponse
  };
});
