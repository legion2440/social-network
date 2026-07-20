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
      }
    };
  }

  return { APIError: APIError, createAuthAPI: createAuthAPI };
});
