(function (root, factory) {
  var library = factory();
  if (typeof module === 'object' && module.exports) module.exports = library;
  if (root) root.UserModel = library;
})(typeof window !== 'undefined' ? window : null, function () {
  var COLORS = ['#5661d8', '#3f9a85', '#b3813f', '#c25a83', '#4d84c4', '#8f6cc9'];

  function own(object, key) {
    return Object.prototype.hasOwnProperty.call(object || {}, key);
  }

  function value(raw, snake, camel) {
    if (own(raw, snake)) return raw[snake];
    if (camel && own(raw, camel)) return raw[camel];
    return undefined;
  }

  function normalizeStatus(status) {
    return status === 'pending' || status === 'accepted' ? status : 'none';
  }

  function relationshipFrom(raw, previous) {
    var source = raw && raw.relationship ? raw.relationship : null;
    var fallback = previous || { status: 'none', follows_me: false };
    return {
      status: normalizeStatus(source && source.status !== undefined ? source.status : fallback.status),
      follows_me: source && source.follows_me !== undefined ? source.follows_me === true : fallback.follows_me === true
    };
  }

  function colorForID(id) {
    var numeric = Number(id);
    if (!Number.isFinite(numeric)) numeric = 0;
    return COLORS[Math.abs(numeric) % COLORS.length];
  }

  function isStaticAvatar(url) {
    return typeof url === 'string' && url.indexOf('/static/avatars/') === 0;
  }

  function normalizeUser(raw, previous, currentUserID) {
    raw = raw || {};
    var id = Number(value(raw, 'id', 'apiId'));
    if (!Number.isInteger(id) || id <= 0) throw new TypeError('positive backend user id is required');

    var user = previous || {};
    var firstName = value(raw, 'first_name', 'firstName');
    var lastName = value(raw, 'last_name', 'lastName');
    var nickname = value(raw, 'nickname', 'nickname');
    var isPrivate = value(raw, 'is_private', 'isPrivate');
    var avatarURL = value(raw, 'avatar_url', 'avatarUrl');
    var canView = value(raw, 'can_view_profile', 'canViewProfile');
    var relationship = relationshipFrom(raw, user.relationship);

    if (firstName !== undefined) user.firstName = firstName || '';
    if (lastName !== undefined) user.lastName = lastName || '';
    if (nickname !== undefined) user.nickname = nickname || '';
    if (isPrivate !== undefined) user.private = isPrivate === true;
    if (avatarURL !== undefined) user.rawAvatarUrl = avatarURL || '';
    if (canView !== undefined) user.canViewProfile = canView === true;

    user.id = String(id);
    user.apiId = id;
    user.relationship = relationship;
    user.name = ((user.firstName || '') + ' ' + (user.lastName || '')).trim() || 'User ' + id;
    user.initials = (((user.firstName || '').charAt(0) + (user.lastName || '').charAt(0)).toUpperCase()) || '?';
    user.handle = user.nickname
      ? (user.nickname.charAt(0) === '@' ? user.nickname : '@' + user.nickname)
      : (value(raw, 'email', 'email') || user.email || 'user-' + id);
    user.color = user.color || colorForID(id);
    if (user.bio === undefined) user.bio = '';
    if (user.aboutMe === undefined) user.aboutMe = '';
    if (user.dob === undefined) user.dob = '';

    if (own(raw, 'email')) user.email = raw.email || '';
    if (own(raw, 'date_of_birth')) user.dob = raw.date_of_birth || '';
    if (own(raw, 'gender')) user.gender = raw.gender || '';
    if (own(raw, 'about_me')) {
      user.aboutMe = raw.about_me || '';
      user.bio = raw.about_me || '';
    }
    if (own(raw, 'posts_count')) user.postsCount = Number(raw.posts_count) || 0;
    if (own(raw, 'followers_count')) user.followersCount = Number(raw.followers_count) || 0;
    if (own(raw, 'following_count')) user.followingCount = Number(raw.following_count) || 0;

    var isCurrent = Number(currentUserID) === id;
    var canViewPrivate = isCurrent || !user.private || relationship.status === 'accepted';
    if (user.canViewProfile === false || !canViewPrivate) {
      user.email = '';
      user.dob = '';
      user.gender = '';
      user.aboutMe = '';
      user.bio = '';
      user.postsCount = 0;
      user.followersCount = 0;
      user.followingCount = 0;
    }

    var rawURL = user.rawAvatarUrl || '';
    if (!canViewPrivate) user.canViewProfile = false;
    user.avatarUrl = rawURL;
    user.hasAvatar = !!user.avatarUrl;
    user.noAvatar = !user.avatarUrl;
    user.hasCustomAvatar = !!user.avatarUrl && !isStaticAvatar(user.avatarUrl);
    return user;
  }

  function mergeUsers(store, rawUsers, currentUserID) {
    var next = Object.assign({}, store || {});
    (rawUsers || []).forEach(function (raw) {
      if (!raw) return;
      var id = Number(value(raw, 'id', 'apiId'));
      if (!Number.isInteger(id) || id <= 0) return;
      next[String(id)] = normalizeUser(raw, next[String(id)], currentUserID);
    });
    return next;
  }

  function followButton(user, pending) {
    var status = normalizeStatus(user && user.relationship && user.relationship.status);
    if (status === 'accepted') return { label: 'Following', tone: 'muted', disabled: !!pending };
    if (status === 'pending') return { label: 'Requested', tone: 'soft', disabled: !!pending };
    return { label: user && user.private ? 'Request' : 'Follow', tone: 'accent', disabled: !!pending };
  }

  function pruneSelected(selected, acceptedFollowers) {
    var accepted = {};
    (acceptedFollowers || []).forEach(function (user) { accepted[String(user.apiId)] = true; });
    var next = {};
    Object.keys(selected || {}).forEach(function (id) {
      if (selected[id] && accepted[id]) next[id] = true;
    });
    return next;
  }

  function createRequestGate() {
    var current = 0;
    return {
      begin: function () { current += 1; return current; },
      isCurrent: function (generation) { return generation === current; },
      current: function () { return current; }
    };
  }

  return {
    normalizeStatus: normalizeStatus,
    normalizeUser: normalizeUser,
    mergeUsers: mergeUsers,
    followButton: followButton,
    pruneSelected: pruneSelected,
    createRequestGate: createRequestGate
  };
});
