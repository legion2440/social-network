(function (root, factory) {
  var library = factory();
  if (typeof module === 'object' && module.exports) module.exports = library;
  if (root) root.CommentModel = library;
})(typeof window !== 'undefined' ? window : null, function () {
  function normalizeCommentResponse(comment) {
    if (!comment || !comment.author) throw new TypeError('comment author is required');
    var id = Number(comment.id);
    var postID = Number(comment.post_id);
    var authorID = Number(comment.author.id);
    if (!Number.isInteger(id) || id <= 0 || !Number.isInteger(postID) || postID <= 0 || !Number.isInteger(authorID) || authorID <= 0) {
      throw new TypeError('positive backend comment, post, and author ids are required');
    }
    return {
      id: String(id),
      apiId: id,
      postID: postID,
      apiAuthorID: authorID,
      text: String(comment.text || ''),
      createdAt: comment.created_at
    };
  }

  function mergeComments(existing, incoming) {
    var byID = {};
    (existing || []).concat(incoming || []).forEach(function (comment) {
      if (comment && comment.apiId > 0) byID[String(comment.apiId)] = comment;
    });
    return Object.keys(byID).map(function (id) { return byID[id]; }).sort(function (left, right) {
      var leftTime = Date.parse(left.createdAt);
      var rightTime = Date.parse(right.createdAt);
      if (leftTime !== rightTime) return leftTime - rightTime;
      return left.apiId - right.apiId;
    });
  }

  return {
    normalizeCommentResponse: normalizeCommentResponse,
    mergeComments: mergeComments
  };
});
