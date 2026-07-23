(function (root, factory) {
  var library = factory();

  if (typeof module === 'object' && module.exports) {
    module.exports = library;
  }

  if (root && typeof root.fetch === 'function') {
    root.AuthAPI = library.createAuthAPI(root.fetch.bind(root));
    root.AuthAPI.APIError = library.APIError;
  }
})(typeof window !== 'undefined' ? window : null, function () {
  class APIError extends Error {
    constructor(message, status, details, cause) {
      super(message);
      this.name = 'APIError';
      this.status = status || 0;
      this.details = details || null;
      if (cause) this.cause = cause;
    }
  }

  function createAuthAPI(fetchImpl) {
    if (typeof fetchImpl !== 'function') {
      throw new TypeError('fetch implementation is required');
    }

    async function request(path, options) {
      var requestOptions = options || {};
      var headers = Object.assign({ Accept: 'application/json' }, requestOptions.headers || {});
      var init = {
        method: requestOptions.method || 'GET',
        credentials: 'same-origin',
        headers: headers
      };
      if (requestOptions.body !== undefined) init.body = requestOptions.body;

      var response;
      try {
        response = await fetchImpl(path, init);
      } catch (cause) {
        throw new APIError('Network error. Please try again.', 0, null, cause);
      }

      var data = null;
      if (response.status !== 204) {
        var contentType = response.headers && response.headers.get
          ? response.headers.get('Content-Type') || ''
          : '';
        try {
          if (contentType.toLowerCase().indexOf('application/json') >= 0) {
            data = await response.json();
          } else if (typeof response.text === 'function') {
            var text = await response.text();
            if (text) data = { error: text };
          }
        } catch (ignore) {
          data = null;
        }
      }

      if (response.status !== requestOptions.expectedStatus) {
        var message = data && typeof data.error === 'string' && data.error.trim()
          ? data.error.trim()
          : (response.ok ? 'Unexpected server response.' : 'Request failed. Please try again.');
        throw new APIError(message, response.status, data);
      }

      return data;
    }

    function pagePath(path, cursor, limit) {
      var query = [];
      if (cursor) query.push('cursor=' + encodeURIComponent(cursor));
      if (limit) query.push('limit=' + encodeURIComponent(String(limit)));
      return path + (query.length ? '?' + query.join('&') : '');
    }

    return {
      register: function (formData) {
        return request('/api/auth/register', {
          method: 'POST',
          body: formData,
          expectedStatus: 201
        });
      },
      login: function (email, password) {
        return request('/api/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email: email, password: password }),
          expectedStatus: 200
        });
      },
      logout: function () {
        return request('/api/auth/logout', {
          method: 'POST',
          expectedStatus: 204
        });
      },
      me: function () {
        return request('/api/auth/me', {
          method: 'GET',
          expectedStatus: 200
        });
      },
      updateProfile: function (profile) {
        return request('/api/profile', {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(profile),
          expectedStatus: 200
        });
      },
      replaceAvatar: function (formData) {
        return request('/api/profile/avatar', {
          method: 'PUT',
          body: formData,
          expectedStatus: 200
        });
      },
      deleteAvatar: function () {
        return request('/api/profile/avatar', {
          method: 'DELETE',
          expectedStatus: 200
        });
      },
      createPost: function (formData) {
        return request('/api/posts', {
          method: 'POST',
          body: formData,
          expectedStatus: 201
        });
      },
      feed: function (cursor, limit) {
        return request(pagePath('/api/posts/feed', cursor, limit), {
          method: 'GET',
          expectedStatus: 200
        });
      },
      userPosts: function (userID, cursor, limit) {
        return request(pagePath('/api/users/' + encodeURIComponent(String(userID)) + '/posts', cursor, limit), {
          method: 'GET',
          expectedStatus: 200
        });
      },
      groupPosts: function (groupID, cursor, limit) {
        return request(pagePath('/api/groups/' + encodeURIComponent(String(groupID)) + '/posts', cursor, limit), {
          method: 'GET',
          expectedStatus: 200
        });
      },
      createGroupPost: function (groupID, formData) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/posts', {
          method: 'POST',
          body: formData,
          expectedStatus: 201
        });
      },
      groupEvents: function (groupID, cursor, limit) {
        return request(pagePath('/api/groups/' + encodeURIComponent(String(groupID)) + '/events', cursor, limit), {
          method: 'GET',
          expectedStatus: 200
        });
      },
      createGroupEvent: function (groupID, event) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/events', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(event),
          expectedStatus: 201
        });
      },
      respondToGroupEvent: function (groupID, eventID, response) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/events/' + encodeURIComponent(String(eventID)) + '/response', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ response: response }),
          expectedStatus: 200
        });
      },
      postComments: function (postID, cursor, limit) {
        return request(pagePath('/api/posts/' + encodeURIComponent(String(postID)) + '/comments', cursor, limit), {
          method: 'GET',
          expectedStatus: 200
        });
      },
      createComment: function (postID, text) {
        return request('/api/posts/' + encodeURIComponent(String(postID)) + '/comments', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ text: text }),
          expectedStatus: 201
        });
      },
      users: function (cursor, limit) {
        return request(pagePath('/api/users', cursor, limit), {
          method: 'GET',
          expectedStatus: 200
        });
      },
      userProfile: function (userID) {
        return request('/api/users/' + encodeURIComponent(String(userID)), {
          method: 'GET',
          expectedStatus: 200
        });
      },
      relationship: function (userID) {
        return request('/api/users/' + encodeURIComponent(String(userID)) + '/follow', {
          method: 'GET',
          expectedStatus: 200
        });
      },
      follow: function (userID) {
        return request('/api/users/' + encodeURIComponent(String(userID)) + '/follow', {
          method: 'PUT',
          expectedStatus: 200
        });
      },
      unfollow: function (userID) {
        return request('/api/users/' + encodeURIComponent(String(userID)) + '/follow', {
          method: 'DELETE',
          expectedStatus: 204
        });
      },
      followers: function (userID) {
        return request('/api/users/' + encodeURIComponent(String(userID)) + '/followers', {
          method: 'GET',
          expectedStatus: 200
        });
      },
      following: function (userID) {
        return request('/api/users/' + encodeURIComponent(String(userID)) + '/following', {
          method: 'GET',
          expectedStatus: 200
        });
      },
      followRequests: function () {
        return request('/api/follow-requests', {
          method: 'GET',
          expectedStatus: 200
        });
      },
      acceptFollowRequest: function (requestID) {
        return request('/api/follow-requests/' + encodeURIComponent(String(requestID)) + '/accept', {
          method: 'POST',
          expectedStatus: 200
        });
      },
      rejectFollowRequest: function (requestID) {
        return request('/api/follow-requests/' + encodeURIComponent(String(requestID)), {
          method: 'DELETE',
          expectedStatus: 204
        });
      },
      notifications: function (cursor, limit) {
        return request(pagePath('/api/notifications', cursor, limit), {
          method: 'GET',
          expectedStatus: 200
        });
      },
      markNotificationRead: function (notificationID) {
        return request('/api/notifications/' + encodeURIComponent(String(notificationID)) + '/read', {
          method: 'PUT',
          expectedStatus: 200
        });
      },
      markAllNotificationsRead: function () {
        return request('/api/notifications/read-all', {
          method: 'PUT',
          expectedStatus: 200
        });
      },
      actOnNotification: function (notificationID, action) {
        return request('/api/notifications/' + encodeURIComponent(String(notificationID)) + '/action', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: action }),
          expectedStatus: 200
        });
      },
      groups: function (cursor, limit) {
        return request(pagePath('/api/groups', cursor, limit), {
          method: 'GET', expectedStatus: 200
        });
      },
      createGroup: function (title, description) {
        return request('/api/groups', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ title: title, description: description }),
          expectedStatus: 201
        });
      },
      group: function (groupID) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)), {
          method: 'GET', expectedStatus: 200
        });
      },
      groupMembers: function (groupID, cursor, limit) {
        return request(pagePath('/api/groups/' + encodeURIComponent(String(groupID)) + '/members', cursor, limit), {
          method: 'GET', expectedStatus: 200
        });
      },
      requestGroupJoin: function (groupID) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/join-request', {
          method: 'POST', expectedStatus: 200
        });
      },
      cancelGroupJoin: function (groupID) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/join-request', {
          method: 'DELETE', expectedStatus: 200
        });
      },
      groupJoinRequests: function (groupID, cursor, limit) {
        return request(pagePath('/api/groups/' + encodeURIComponent(String(groupID)) + '/join-requests', cursor, limit), {
          method: 'GET', expectedStatus: 200
        });
      },
      acceptGroupJoinRequest: function (groupID, userID) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/join-requests/' + encodeURIComponent(String(userID)) + '/accept', {
          method: 'POST', expectedStatus: 200
        });
      },
      rejectGroupJoinRequest: function (groupID, userID) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/join-requests/' + encodeURIComponent(String(userID)), {
          method: 'DELETE', expectedStatus: 200
        });
      },
      groupInvitations: function (groupID, cursor, limit) {
        return request(pagePath('/api/groups/' + encodeURIComponent(String(groupID)) + '/invitations', cursor, limit), {
          method: 'GET', expectedStatus: 200
        });
      },
      inviteToGroup: function (groupID, userID) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/invitations', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ user_id: userID }),
          expectedStatus: 200
        });
      },
      groupInvitationInbox: function (cursor, limit) {
        return request(pagePath('/api/group-invitations', cursor, limit), {
          method: 'GET', expectedStatus: 200
        });
      },
      acceptGroupInvitation: function (groupID) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/invitation/accept', {
          method: 'POST', expectedStatus: 200
        });
      },
      declineGroupInvitation: function (groupID) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/invitation', {
          method: 'DELETE', expectedStatus: 200
        });
      },
      leaveGroup: function (groupID) {
        return request('/api/groups/' + encodeURIComponent(String(groupID)) + '/membership', {
          method: 'DELETE', expectedStatus: 200
        });
      },
      chats: function (cursor, limit) {
        return request(pagePath('/api/chats', cursor, limit), {
          method: 'GET', expectedStatus: 200
        });
      },
      directMessages: function (userID, cursor, limit) {
        return request(pagePath('/api/chats/direct/' + encodeURIComponent(String(userID)) + '/messages', cursor, limit), {
          method: 'GET', expectedStatus: 200
        });
      },
      groupMessages: function (groupID, cursor, limit) {
        return request(pagePath('/api/groups/' + encodeURIComponent(String(groupID)) + '/chat/messages', cursor, limit), {
          method: 'GET', expectedStatus: 200
        });
      }
    };
  }

  return { APIError: APIError, createAuthAPI: createAuthAPI };
});
