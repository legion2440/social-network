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
      }
    };
  }

  return { APIError: APIError, createAuthAPI: createAuthAPI };
});
