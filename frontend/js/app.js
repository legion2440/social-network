
const IC = {
  home: 'M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1Z',
  user: 'M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-7 10a7 7 0 0 1 14 0',
  users: 'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-7 10a7 7 0 0 1 14 0m1-17.5a4 4 0 0 1 0 7.1M17.8 14.6a7 7 0 0 1 4.2 6.4',
  chat: 'M21 11.5a8.5 8.5 0 0 1-8.5 8.5c-1.5 0-3-.4-4.2-1.1L3 21l2.1-5.3A8.5 8.5 0 1 1 21 11.5Z',
  bell: 'M18 8.5a6 6 0 1 0-12 0c0 7-2.5 8.5-2.5 8.5h17S18 15.5 18 8.5m-4.3 12a2 2 0 0 1-3.4 0',
  globe: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Zm-9-9h18M12 3c2.5 2.5 3.8 5.6 3.8 9s-1.3 6.5-3.8 9c-2.5-2.5-3.8-5.6-3.8-9S9.5 5.5 12 3Z',
  lock: 'M7 10.5V7a5 5 0 0 1 10 0v3.5M6 10.5h12a1 1 0 0 1 1 1V20a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1v-8.5a1 1 0 0 1 1-1Z',
  sun: 'M12 16.5a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9ZM12 2.5v2m0 15v2m-9.5-9.5h2m15 0h2M5.3 5.3l1.4 1.4m10.6 10.6 1.4 1.4m0-13.4-1.4 1.4M6.7 17.3l-1.4 1.4',
  moon: 'M20.5 13.2A8.5 8.5 0 1 1 10.8 3.5a7 7 0 0 0 9.7 9.7Z',
  plus: 'M12 5v14M5 12h14',
  cal: 'M8 2.5v3m8-3v3M3.5 9h17m-15-3.5h13a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2v-11a2 2 0 0 1 2-2Z'
};

const USERS = {
  me:    { id: 'me', name: 'Alex Morgan', handle: '@alexmorgan', initials: 'AM', color: '#6b62c9', bio: 'Product designer. Building calm interfaces.', email: 'alex@loop.social', dob: 'March 14, 1996', private: false },
  mei:   { id: 'mei', name: 'Mei Lin', handle: '@meilin', initials: 'ML', color: '#b3813f', bio: 'Design systems at Fluxo. Type nerd.', email: 'mei@fluxo.co', dob: 'June 2, 1994', private: false },
  david: { id: 'david', name: 'David Okafor', handle: '@dokafor', initials: 'DO', color: '#3f9a85', bio: 'Frontend engineer. Weekend trail runner.', email: 'david@okafor.dev', dob: 'Nov 20, 1991', private: false },
  nina:  { id: 'nina', name: 'Nina Kov\u00e1cs', handle: '@ninak', initials: 'NK', color: '#c25a83', bio: 'Brand designer. Mostly lurking.', email: 'nina@studio-nk.com', dob: 'Feb 8, 1997', private: true },
  tom:   { id: 'tom', name: 'Tom\u00e1s Rivera', handle: '@tomriv', initials: 'TR', color: '#4d84c4', bio: 'Photographer. Film only, no exceptions.', email: 'tom@rivera.photo', dob: 'Aug 30, 1993', private: true },
  sara:  { id: 'sara', name: 'Sara Bishop', handle: '@sarab', initials: 'SB', color: '#8f6cc9', bio: 'Illustrator and printmaker.', email: 'sara@bishop.ink', dob: 'Apr 17, 1995', private: false }
};

const REPLIES = ['Totally agree \ud83d\ude04', 'Ha! Send it over', 'Let\u2019s sync tomorrow?', '\ud83d\udc40 looking now', 'Love that \u2728', 'Okay that\u2019s actually great'];
const EMOJIS = ['\ud83d\ude00', '\ud83d\ude02', '\ud83d\ude0d', '\ud83d\udd25', '\ud83d\udc4d', '\ud83c\udf89', '\ud83d\ude2e', '\ud83d\ude22', '\u2764\ufe0f', '\ud83d\udc40', '\u2728', '\ud83d\ude4c'];
const GROUP_COLORS = ['#6b62c9', '#b3813f', '#3f9a85', '#c25a83', '#4d84c4', '#8f6cc9'];

function cover(color) {
  return 'linear-gradient(135deg, color-mix(in oklab, ' + color + ' 55%, var(--surface2)), color-mix(in oklab, ' + color + ' 14%, var(--surface2)))';
}
function num(x) { return String(x); }

function emptyRegistrationForm() {
  return {
    authEmail: '', authPassword: '',
    regFirstName: '', regLastName: '', regDateOfBirth: '', regGender: '',
    regNickname: '', regAboutMe: '', regAvatar: null, regAvatarName: ''
  };
}

function emptyProfileEditor() {
  return {
    profileEditOpen: false, profileEditPending: false, profileAvatarPending: false,
    profileEditError: '', editFirstName: '', editLastName: '', editDateOfBirth: '',
    editGender: '', editNickname: '', editAboutMe: '', editAvatar: null, editAvatarName: ''
  };
}

function decorateUser(user) {
  const safeAvatarURL = user.avatarUrl ? String(user.avatarUrl).replace(/["\\\r\n]/g, '') : '';
  user.avatarUrl = safeAvatarURL;
  user.hasAvatar = !!safeAvatarURL;
  user.noAvatar = !safeAvatarURL;
  user.hasCustomAvatar = AvatarURL.isCustomAvatarURL(safeAvatarURL);
  return user;
}

Object.keys(USERS).forEach(uid => decorateUser(USERS[uid]));

function requestErrorMessage(error, fallback) {
  if (error && typeof error.message === 'string' && error.message.trim()) return error.message.trim();
  return fallback;
}

class Component extends DCLogic {
  constructor(props) {
    super(props);
    let saved = null;
    try { saved = localStorage.getItem('loop-theme'); } catch (e) {}
    this.state = {
      theme: saved || props.defaultTheme || 'light',
      screen: 'feed', feedLoading: true, feedPending: false, feedError: '', feedNextCursor: null,
      composerText: '', composerFile: null, composerFileName: '', composerError: '', composerPending: false,
      privacy: 'public', privacyOpen: false,
      selectedFollowers: {}, postFollowers: [], postFollowersLoading: false,
      openComments: { p1: true }, drafts: {},
      posts: [],
      mockFollow: { mei: 'accepted', david: 'accepted', nina: 'accepted', tom: 'none', sara: 'none' },
      apiUsersByID: {}, directoryUserIDs: [], directoryNextCursor: null, directoryLoading: false, directoryError: '',
      followPendingByID: {}, followErrorByID: {},
      followRequests: [], followRequestsLoading: false, followRequestsError: '', followRequestPendingByID: {},
      myPrivacy: 'public', profilePrivacyPending: false, profilePrivacyError: '',
      profileId: null, profileTab: 'posts', profileLoading: false, profileReady: false, profileError: '',
      profileFollowers: [], profileFollowing: [], profileListsLoading: false, profileListsError: '',
      profilePosts: [], profilePostsLoading: false, profilePostsPending: false,
      profilePostsError: '', profilePostsNextCursor: null,
      groups: [
        { id: 'g1', name: 'Design Systems Guild', desc: 'Tokens, components and the people who argue about them.', members: 148, color: '#6b62c9', state: 'joined', owner: 'me', memberIds: ['me', 'mei', 'david', 'nina'],
          posts: [
            { id: 'gp1', uid: 'mei', time: '3h', text: 'Poll next week: spacing scale of 4 vs 8. Prepare your arguments.', likes: 9, liked: false, comments: [] },
            { id: 'gp2', uid: 'david', time: '1d', text: 'Wrote up how we ship tokens to three platforms from one source of truth. Link in comments.', likes: 21, liked: true, comments: [ { uid: 'me', text: 'This saved our team weeks. Great write-up', time: '20h' } ] }
          ],
          events: [
            { id: 'e1', title: 'Tokens naming workshop', day: '12', mon: 'JUL', time: '18:00', desc: 'Hands-on session. Naming things is hard \u2014 we do it together.', rsvp: null, going: ['mei', 'david'] },
            { id: 'e2', title: 'Figma office hours', day: '19', mon: 'JUL', time: '17:30', desc: 'Bring your messiest file. No judgement.', rsvp: 'going', going: ['me', 'mei', 'nina'] }
          ],
          requests: [ { uid: 'sara', status: 'pending' } ] },
        { id: 'g2', name: 'Film Photography Club', desc: 'Grain is good. Weekly photo walks and darkroom nights.', members: 96, color: '#b3813f', state: 'invited', owner: 'tom', memberIds: ['tom', 'sara'], posts: [], events: [], requests: [] },
        { id: 'g3', name: 'Trail Runners ATX', desc: 'Early miles, tacos after. All paces welcome.', members: 210, color: '#3f9a85', state: 'none', owner: 'david', memberIds: ['david'], posts: [], events: [], requests: [] },
        { id: 'g4', name: 'Indie Game Devs', desc: 'Devlogs, playtests and brutally honest feedback.', members: 311, color: '#c25a83', state: 'requested', owner: 'sara', memberIds: ['sara'], posts: [], events: [], requests: [] }
      ],
      groupId: null, groupTab: 'posts', gComposer: '', inviteOpen: false, invited: {},
      newEventOpen: false, evTitle: '', evDate: '', evTime: '', evDesc: '',
      createOpen: false, ngName: '', ngDesc: '',
      convos: [
        { id: 'c1', kind: 'dm', uid: 'nina', unread: 2, typing: false, online: true, messages: [
          { from: 'nina', text: 'Did you see the moodboard I posted? \ud83d\udc40', time: '09:12' },
          { from: 'me', text: 'Yes! The serif direction is bold. I like it', time: '09:14' },
          { from: 'nina', text: 'Okay good. I was 50/50 on it \ud83d\ude05', time: '09:15' },
          { from: 'nina', text: 'Coffee this week? I want your take on the type scale', time: '09:15' } ] },
        { id: 'c2', kind: 'dm', uid: 'david', unread: 0, typing: false, online: false, messages: [
          { from: 'me', text: 'That tokens write-up is gold \ud83d\udd25', time: 'Tue' },
          { from: 'david', text: 'Ha, thanks! Took forever to edit down', time: 'Tue' } ] },
        { id: 'c3', kind: 'group', gid: 'g1', unread: 0, typing: false, online: false, messages: [
          { from: 'mei', text: 'Reminder: workshop is on the 12th \ud83c\udf89', time: 'Mon' },
          { from: 'david', text: 'I\u2019ll bring the contrast checker demo', time: 'Mon' },
          { from: 'me', text: 'Adding it to the agenda \ud83d\udc4d', time: 'Mon' } ] }
      ],
      convoId: 'c1', chatDraft: '', emojiOpen: false,
      notifs: [
        { id: 'n2', type: 'invite', uid: 'tom', gid: 'g2', time: '6h', read: false, status: 'pending' },
        { id: 'n3', type: 'request', uid: 'sara', gid: 'g1', time: '1d', read: false, status: 'pending' },
        { id: 'n4', type: 'event', uid: 'mei', gid: 'g1', time: '2d', read: true, status: 'info', extra: 'Tokens naming workshop' }
      ],
      authMode: 'login', authStatus: 'checking', authPending: false, logoutPending: false,
      authError: '', bootstrapError: '', appError: '',
      ...emptyRegistrationForm(),
      ...emptyProfileEditor()
    };
    this.msgEl = null;
    this.profileGate = UserModel.createRequestGate();
    this.feedGate = UserModel.createRequestGate();
    this.directoryGate = UserModel.createRequestGate();
    this.postFollowersGate = UserModel.createRequestGate();
  }

  componentDidMount() {
    document.documentElement.dataset.theme = this.state.theme;
    this.applyTokens();
    this.loadCurrentUser();
  }
  componentDidUpdate() {
    this.applyTokens();
    if (this.msgEl) this.msgEl.scrollTop = this.msgEl.scrollHeight;
  }
  applyTokens() {
    const el = document.documentElement;
    el.style.setProperty('--accent', this.props.accent || '#5661d8');
    el.style.setProperty('--r', (this.props.radius != null ? this.props.radius : 18) + 'px');
  }

  mergeAPIUsers(rawUsers, baseStore) {
    const currentUserID = USERS.me && USERS.me.apiId;
    const base = Object.assign({}, baseStore || this.state.apiUsersByID);
    if (currentUserID) base[String(currentUserID)] = USERS.me;
    const next = UserModel.mergeUsers(base, rawUsers, currentUserID);
    Object.keys(next).forEach(id => decorateUser(next[id]));
    return next;
  }

  applyAuthUser(user, baseStore) {
    const next = UserModel.mergeUsers(baseStore || this.state.apiUsersByID, [user], user.id);
    const me = decorateUser(next[String(user.id)]);
    USERS.me = me;
    next[String(user.id)] = me;
    return next;
  }

  apiUser(userID) {
    const id = String(Number(userID));
    if (USERS.me && String(USERS.me.apiId) === id) return USERS.me;
    return this.state.apiUsersByID[id] || decorateUser({
      id, apiId: Number(id), name: 'User ' + id, handle: 'user-' + id,
      initials: '?', color: '#5661d8', bio: '', private: false,
      relationship: { status: 'none', follows_me: false }
    });
  }

  mapAPIPost(post) {
    const normalized = PostModel.normalizePostResponse(post, USERS.me.apiId);
    return {
      id: normalized.id,
      real: true,
      apiAuthorID: normalized.apiAuthorID,
      text: normalized.text,
      privacy: normalized.privacy,
      mediaUrl: normalized.mediaUrl,
      time: this.formatPostTime(normalized.createdAt)
    };
  }

  formatPostTime(value) {
    const created = new Date(value);
    if (Number.isNaN(created.getTime())) return '';
    const seconds = Math.max(0, Math.floor((Date.now() - created.getTime()) / 1000));
    if (seconds < 60) return 'now';
    if (seconds < 3600) return Math.floor(seconds / 60) + 'm';
    if (seconds < 86400) return Math.floor(seconds / 3600) + 'h';
    if (seconds < 604800) return Math.floor(seconds / 86400) + 'd';
    return created.toLocaleDateString();
  }

  loadFeed = async (reset) => {
    const generation = reset ? this.feedGate.begin() : this.feedGate.current();
    if (!reset && this.state.feedPending) return;
    const cursor = reset ? null : this.state.feedNextCursor;
    if (!reset && !cursor) return;
    this.setState({ feedPending: true, feedLoading: !!reset, feedError: '' });
    try {
      const page = await AuthAPI.feed(cursor, 20);
      if (!this.feedGate.isCurrent(generation)) return;
      const mapped = (page.posts || []).map(post => this.mapAPIPost(post));
      const apiUsersByID = this.mergeAPIUsers((page.posts || []).map(post => post.author));
      this.setState({
        posts: reset ? mapped : this.state.posts.concat(mapped),
        apiUsersByID,
        feedLoading: false, feedPending: false,
        feedNextCursor: page.next_cursor || null, feedError: ''
      });
    } catch (error) {
      if (!this.feedGate.isCurrent(generation)) return;
      this.setState({
        feedLoading: false, feedPending: false,
        feedError: requestErrorMessage(error, 'Could not load the feed. Please try again.')
      });
    }
  };

  loadPostFollowers = async () => {
    if (!USERS.me.apiId) return;
    const generation = this.postFollowersGate.begin();
    this.setState({ postFollowersLoading: true });
    try {
      const response = await AuthAPI.followers(USERS.me.apiId);
      if (!this.postFollowersGate.isCurrent(generation)) return;
      const apiUsersByID = this.mergeAPIUsers(response.users || []);
      const followers = (response.users || []).map(user => apiUsersByID[String(user.id)]);
      this.setState({
        apiUsersByID,
        postFollowers: followers,
        postFollowersLoading: false,
        selectedFollowers: UserModel.pruneSelected(this.state.selectedFollowers, followers)
      });
    } catch (error) {
      if (!this.postFollowersGate.isCurrent(generation)) return;
      this.setState({
        postFollowersLoading: false,
        composerError: requestErrorMessage(error, 'Could not load followers for selected posts.')
      });
    }
  };

  loadDirectory = async () => {
    const generation = this.directoryGate.begin();
    this.setState({ directoryLoading: true, directoryError: '' });
    try {
      const response = await AuthAPI.users(null, 20);
      if (!this.directoryGate.isCurrent(generation)) return;
      const apiUsersByID = this.mergeAPIUsers(response.users || []);
      this.setState({
        apiUsersByID,
        directoryUserIDs: (response.users || []).map(user => Number(user.id)),
        directoryNextCursor: response.next_cursor || null,
        directoryLoading: false, directoryError: ''
      });
    } catch (error) {
      if (!this.directoryGate.isCurrent(generation)) return;
      this.setState({
        directoryLoading: false,
        directoryError: requestErrorMessage(error, 'Could not load user suggestions.')
      });
    }
  };

  loadFollowRequests = async () => {
    if (this.state.followRequestsLoading) return;
    this.setState({ followRequestsLoading: true, followRequestsError: '' });
    try {
      const response = await AuthAPI.followRequests();
      const requests = response.requests || [];
      const apiUsersByID = this.mergeAPIUsers(requests.map(request => request.user));
      this.setState({ apiUsersByID, followRequests: requests, followRequestsLoading: false });
    } catch (error) {
      this.setState({
        followRequestsLoading: false,
        followRequestsError: requestErrorMessage(error, 'Could not load follow requests.')
      });
    }
  };

  loadProfileConnections = async (targetUserID, generation) => {
    targetUserID = Number(targetUserID);
    generation = generation || this.profileGate.current();
    this.setState({ profileListsLoading: true, profileListsError: '' });
    try {
      const responses = await Promise.all([
        AuthAPI.followers(targetUserID),
        AuthAPI.following(targetUserID)
      ]);
      if (!this.profileGate.isCurrent(generation) || Number(this.state.profileId) !== targetUserID) return;
      const followers = responses[0].users || [];
      const following = responses[1].users || [];
      const apiUsersByID = this.mergeAPIUsers(following, this.mergeAPIUsers(followers));
      this.setState({
        apiUsersByID,
        profileFollowers: followers.map(user => Number(user.id)),
        profileFollowing: following.map(user => Number(user.id)),
        profileListsLoading: false, profileListsError: ''
      });
    } catch (error) {
      if (!this.profileGate.isCurrent(generation) || Number(this.state.profileId) !== targetUserID) return;
      if (error && error.status === 403) {
        this.setState({ profileFollowers: [], profileFollowing: [], profileListsLoading: false, profileListsError: '' });
        return;
      }
      this.setState({
        profileListsLoading: false,
        profileListsError: requestErrorMessage(error, 'Could not load followers and following.')
      });
    }
  };

  loadProfilePosts = async (targetUserID, reset, generation) => {
    targetUserID = Number(targetUserID || this.state.profileId);
    generation = generation || this.profileGate.current();
    if (!targetUserID || (!reset && this.state.profilePostsPending)) return;
    const cursor = reset ? null : this.state.profilePostsNextCursor;
    if (!reset && !cursor) return;
    this.setState({ profilePostsPending: true, profilePostsLoading: !!reset, profilePostsError: '' });
    try {
      const page = await AuthAPI.userPosts(targetUserID, cursor, 20);
      if (!this.profileGate.isCurrent(generation) || Number(this.state.profileId) !== targetUserID) return;
      const mapped = (page.posts || []).map(post => this.mapAPIPost(post));
      const apiUsersByID = this.mergeAPIUsers((page.posts || []).map(post => post.author));
      this.setState({
        profilePosts: reset ? mapped : this.state.profilePosts.concat(mapped),
        apiUsersByID,
        profilePostsLoading: false, profilePostsPending: false,
        profilePostsNextCursor: page.next_cursor || null, profilePostsError: ''
      });
    } catch (error) {
      if (!this.profileGate.isCurrent(generation) || Number(this.state.profileId) !== targetUserID) return;
      if (error && error.status === 403) {
        const user = this.apiUser(targetUserID);
        user.canViewProfile = false;
        user.bio = ''; user.dob = ''; user.postsCount = 0;
        this.setState({
          apiUsersByID: Object.assign({}, this.state.apiUsersByID, { [String(targetUserID)]: user }),
          profilePosts: [], profileFollowers: [], profileFollowing: [],
          profilePostsLoading: false, profilePostsPending: false, profilePostsError: ''
        });
        return;
      }
      this.setState({
        profilePostsLoading: false, profilePostsPending: false,
        profilePostsError: requestErrorMessage(error, 'Could not load profile posts. Please try again.')
      });
    }
  };

  loadCurrentUser = async () => {
    this.setState({ authStatus: 'checking', bootstrapError: '', appError: '' });
    try {
      const user = await AuthAPI.me();
      const apiUsersByID = this.applyAuthUser(user);
      this.setState({
        authStatus: 'authenticated', screen: 'feed',
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public',
        profilePrivacyPending: false, profilePrivacyError: ''
      });
      this.loadFeed(true);
      this.loadPostFollowers();
      this.loadDirectory();
      this.loadFollowRequests();
    } catch (error) {
      if (error && error.status === 401) {
        this.setState({ authStatus: 'anonymous', screen: 'auth', bootstrapError: '' });
        return;
      }
      this.setState({
        authStatus: 'error',
        bootstrapError: requestErrorMessage(error, 'Could not load your session. Please try again.')
      });
    }
  };

  setAuthMode = (mode) => this.setState({ authMode: mode, authError: '' });

  pickRegistrationAvatar = () => {
    const input = document.getElementById('registration-avatar');
    if (input) input.click();
  };

  onRegistrationAvatar = (event) => {
    const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
    this.setState({ regAvatar: file, regAvatarName: file ? file.name : '' });
  };

  submitAuth = async (event) => {
    if (event) event.preventDefault();
    if (this.state.authPending) return;

    const s = this.state;
    this.setState({ authPending: true, authError: '' });
    try {
      let user;
      if (s.authMode === 'login') {
        user = await AuthAPI.login(s.authEmail.trim(), s.authPassword);
      } else {
        const form = new FormData();
        form.append('email', s.authEmail.trim());
        form.append('password', s.authPassword);
        form.append('first_name', s.regFirstName.trim());
        form.append('last_name', s.regLastName.trim());
        form.append('date_of_birth', s.regDateOfBirth.trim());
        if (s.regGender) form.append('gender', s.regGender);
        if (s.regNickname.trim()) form.append('nickname', s.regNickname.trim());
        if (s.regAboutMe.trim()) form.append('about_me', s.regAboutMe.trim());
        if (s.regAvatar) form.append('avatar', s.regAvatar, s.regAvatar.name);
        user = await AuthAPI.register(form);
      }

      const apiUsersByID = this.applyAuthUser(user);
      const authenticatedState = {
        authStatus: 'authenticated', authPending: false, authError: '',
        authPassword: '', screen: 'feed',
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public',
        profilePrivacyPending: false, profilePrivacyError: ''
      };
      if (s.authMode === 'register') Object.assign(authenticatedState, emptyRegistrationForm());
      Object.assign(authenticatedState, emptyProfileEditor());
      this.setState(authenticatedState);
      this.loadFeed(true);
      this.loadPostFollowers();
      this.loadDirectory();
      this.loadFollowRequests();
    } catch (error) {
      this.setState({
        authPending: false,
        authError: requestErrorMessage(error, 'Authentication failed. Please try again.')
      });
    }
  };

  logout = async () => {
    if (this.state.logoutPending) return;
    this.setState({ logoutPending: true, appError: '' });
    try {
      await AuthAPI.logout();
      this.feedGate.begin();
      this.directoryGate.begin();
      this.postFollowersGate.begin();
      this.profileGate.begin();
      this.setState(Object.assign({
        authStatus: 'anonymous', logoutPending: false, authMode: 'login',
        authError: '', screen: 'auth', myPrivacy: 'public',
        profilePrivacyPending: false, profilePrivacyError: '',
        posts: [], feedLoading: true, feedPending: false, feedError: '', feedNextCursor: null,
        profilePosts: [], profilePostsLoading: false, profilePostsPending: false,
        profilePostsError: '', profilePostsNextCursor: null,
        postFollowers: [], postFollowersLoading: false, selectedFollowers: {},
        apiUsersByID: {}, directoryUserIDs: [], directoryNextCursor: null,
        directoryLoading: false, directoryError: '', followPendingByID: {}, followErrorByID: {},
        followRequests: [], followRequestsLoading: false, followRequestsError: '', followRequestPendingByID: {},
        profileId: null, profileReady: false, profileLoading: false, profileError: '',
        profileFollowers: [], profileFollowing: [], profileListsLoading: false, profileListsError: '',
        composerText: '', composerFile: null, composerFileName: '', composerError: '', composerPending: false,
        privacy: 'public', privacyOpen: false
      }, emptyRegistrationForm(), emptyProfileEditor()));
    } catch (error) {
      this.setState({
        logoutPending: false,
        appError: requestErrorMessage(error, 'Could not log out. Please try again.')
      });
    }
  };

  go = (screen) => {
    this.setState({ screen, privacyOpen: false, emojiOpen: false });
    if (screen === 'notifications') this.loadFollowRequests();
  };
  openProfile = async (targetUserID) => {
    if (targetUserID === 'me') targetUserID = USERS.me.apiId;
    targetUserID = Number(targetUserID);
    if (!Number.isInteger(targetUserID) || targetUserID <= 0) return;
    const generation = this.profileGate.begin();
    const isMe = targetUserID === USERS.me.apiId;
    this.setState({
      screen: 'profile', profileId: targetUserID, profileTab: 'posts',
      profileLoading: true, profileReady: false, profileError: '',
      profilePosts: [], profilePostsLoading: false, profilePostsPending: false,
      profilePostsError: '', profilePostsNextCursor: null,
      profileFollowers: [], profileFollowing: [], profileListsLoading: false, profileListsError: '',
      profileEditOpen: isMe ? this.state.profileEditOpen : false,
      profileEditError: isMe ? this.state.profileEditError : ''
    });
    try {
      const results = await Promise.all([
        AuthAPI.userProfile(targetUserID),
        isMe ? Promise.resolve({ status: 'none', follows_me: false }) : AuthAPI.relationship(targetUserID)
      ]);
      if (!this.profileGate.isCurrent(generation)) return;
      const rawUser = Object.assign({}, results[0], { relationship: results[1] });
      const apiUsersByID = this.mergeAPIUsers([rawUser]);
      const profileUser = apiUsersByID[String(targetUserID)];
      this.setState({ apiUsersByID, profileLoading: false, profileReady: true, profileError: '' });
      if (profileUser.canViewProfile) {
        this.loadProfilePosts(targetUserID, true, generation);
        this.loadProfileConnections(targetUserID, generation);
      }
    } catch (error) {
      if (!this.profileGate.isCurrent(generation)) return;
      this.setState({
        profileLoading: false, profileReady: false,
        profileError: error && error.status === 404
          ? 'User not found.'
          : requestErrorMessage(error, 'Could not load this profile.')
      });
    }
  };

  openProfileEdit = () => {
    const me = USERS.me;
    this.setState({
      profileEditOpen: true, profileEditError: '',
      editFirstName: me.firstName || '', editLastName: me.lastName || '',
      editDateOfBirth: me.dob || '', editGender: me.gender || '',
      editNickname: me.nickname || '', editAboutMe: me.aboutMe || '',
      editAvatar: null, editAvatarName: ''
    });
  };

  cancelProfileEdit = () => this.setState(Object.assign({}, emptyProfileEditor()));

  saveProfile = async (event) => {
    if (event) event.preventDefault();
    if (this.state.profileEditPending || this.state.profileAvatarPending || this.state.profilePrivacyPending) return;
    const s = this.state;
    this.setState({ profileEditPending: true, profileEditError: '' });
    try {
      const user = await AuthAPI.updateProfile({
        first_name: s.editFirstName.trim(),
        last_name: s.editLastName.trim(),
        date_of_birth: s.editDateOfBirth.trim(),
        gender: s.editGender || null,
        nickname: s.editNickname,
        about_me: s.editAboutMe
      });
      const apiUsersByID = this.applyAuthUser(user);
      this.setState(Object.assign({
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public'
      }, emptyProfileEditor()));
    } catch (error) {
      this.setState({
        profileEditPending: false,
        profileEditError: requestErrorMessage(error, 'Could not update your profile. Please try again.')
      });
    }
  };

  pickProfileAvatar = () => {
    const input = document.getElementById('profile-avatar');
    if (input) input.click();
  };

  onProfileAvatar = (event) => {
    const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
    this.setState({ editAvatar: file, editAvatarName: file ? file.name : '', profileEditError: '' });
  };

  replaceProfileAvatar = async () => {
    if (this.state.profileAvatarPending || this.state.profileEditPending || this.state.profilePrivacyPending || !this.state.editAvatar) return;
    const avatar = this.state.editAvatar;
    this.setState({ profileAvatarPending: true, profileEditError: '' });
    try {
      const form = new FormData();
      form.append('avatar', avatar, avatar.name);
      const user = await AuthAPI.replaceAvatar(form);
      const apiUsersByID = this.applyAuthUser(user);
      const input = document.getElementById('profile-avatar');
      if (input) input.value = '';
      this.setState({
        apiUsersByID,
        profileAvatarPending: false, editAvatar: null, editAvatarName: '',
        myPrivacy: user.is_private === true ? 'private' : 'public'
      });
    } catch (error) {
      this.setState({
        profileAvatarPending: false,
        profileEditError: requestErrorMessage(error, 'Could not replace your avatar. Please try again.')
      });
    }
  };

  deleteProfileAvatar = async () => {
    if (this.state.profileAvatarPending || this.state.profileEditPending || this.state.profilePrivacyPending) return;
    this.setState({ profileAvatarPending: true, profileEditError: '' });
    try {
      const user = await AuthAPI.deleteAvatar();
      const apiUsersByID = this.applyAuthUser(user);
      const input = document.getElementById('profile-avatar');
      if (input) input.value = '';
      this.setState({
        apiUsersByID,
        profileAvatarPending: false, editAvatar: null, editAvatarName: '',
        myPrivacy: user.is_private === true ? 'private' : 'public'
      });
    } catch (error) {
      this.setState({
        profileAvatarPending: false,
        profileEditError: requestErrorMessage(error, 'Could not delete your avatar. Please try again.')
      });
    }
  };

  setProfilePrivacy = async (privacy) => {
    if (
      this.state.profilePrivacyPending ||
      this.state.profileEditPending ||
      this.state.profileAvatarPending ||
      privacy === this.state.myPrivacy
    ) return;
    const isPrivate = privacy === 'private';
    this.setState({ profilePrivacyPending: true, profilePrivacyError: '' });
    try {
      const user = await AuthAPI.updateProfile({ is_private: isPrivate });
      const apiUsersByID = this.applyAuthUser(user);
      this.setState({
        apiUsersByID,
        myPrivacy: user.is_private === true ? 'private' : 'public',
        profilePrivacyPending: false,
        profilePrivacyError: ''
      });
    } catch (error) {
      this.setState({
        profilePrivacyPending: false,
        profilePrivacyError: requestErrorMessage(error, 'Could not update profile privacy. Please try again.')
      });
    }
  };

  toggleTheme = (e) => {
    const el = document.documentElement;
    if (e && e.clientX != null) {
      el.style.setProperty('--vt-x', e.clientX + 'px');
      el.style.setProperty('--vt-y', e.clientY + 'px');
    }
    const next = this.state.theme === 'light' ? 'dark' : 'light';
    const apply = () => {
      el.dataset.theme = next;
      this.setState({ theme: next });
      try { localStorage.setItem('loop-theme', next); } catch (err) {}
    };
    if (document.startViewTransition) {
      const vt = document.startViewTransition(apply);
      if (vt && vt.ready) vt.ready.catch(() => {});
      if (vt && vt.finished) vt.finished.catch(() => {});
    } else apply();
  };

  toggleFollow = async (targetUserID) => {
    targetUserID = Number(targetUserID);
    if (!Number.isInteger(targetUserID) || targetUserID <= 0 || targetUserID === USERS.me.apiId) return;
    const key = String(targetUserID);
    if (this.state.followPendingByID[key]) return;
    const user = this.apiUser(targetUserID);
    const status = UserModel.normalizeStatus(user.relationship && user.relationship.status);
    this.setState({
      followPendingByID: Object.assign({}, this.state.followPendingByID, { [key]: true }),
      followErrorByID: Object.assign({}, this.state.followErrorByID, { [key]: '' }),
      appError: ''
    });
    try {
      const response = status === 'none'
        ? await AuthAPI.follow(targetUserID)
        : (await AuthAPI.unfollow(targetUserID), { status: 'none' });
      const apiUsersByID = this.mergeAPIUsers([{
        id: targetUserID,
        relationship: {
          status: response.status,
          follows_me: user.relationship && user.relationship.follows_me === true
        }
      }]);
      const pending = Object.assign({}, this.state.followPendingByID);
      delete pending[key];
      const posts = status === 'none'
        ? this.state.posts
        : this.state.posts.filter(post => Number(post.apiAuthorID) !== targetUserID);
      this.setState({ apiUsersByID, followPendingByID: pending, posts });

      this.loadDirectory();
      this.loadFeed(true);
      if (this.state.screen === 'profile' && Number(this.state.profileId) === targetUserID) this.openProfile(targetUserID);
    } catch (error) {
      const pending = Object.assign({}, this.state.followPendingByID);
      delete pending[key];
      const message = requestErrorMessage(error, 'Could not update follow status.');
      this.setState({
        followPendingByID: pending,
        followErrorByID: Object.assign({}, this.state.followErrorByID, { [key]: message }),
        appError: message
      });
    }
  };

  likePost = (pid) => {
    this.setState({ posts: this.state.posts.map(p => p.id === pid ? Object.assign({}, p, { liked: !p.liked, likes: p.likes + (p.liked ? -1 : 1) }) : p) });
  };
  addComment = (pid) => {
    const text = (this.state.drafts[pid] || '').trim();
    if (!text) return;
    this.setState({
      posts: this.state.posts.map(p => p.id === pid ? Object.assign({}, p, { comments: p.comments.concat([{ uid: 'me', text, time: 'now' }]) }) : p),
      drafts: Object.assign({}, this.state.drafts, { [pid]: '' })
    });
  };
  pickComposerMedia = () => {
    const input = document.getElementById('post-media');
    if (input) input.click();
  };

  onComposerMedia = (event) => {
    const file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
    this.setState({ composerFile: file, composerFileName: file ? file.name : '', composerError: '' });
  };

  removeComposerMedia = () => {
    const input = document.getElementById('post-media');
    if (input) input.value = '';
    this.setState({ composerFile: null, composerFileName: '', composerError: '' });
  };

  sendPost = async () => {
    const s = this.state;
    if (s.composerPending || !s.composerText.trim()) return;
    const selectedUserIDs = Object.keys(s.selectedFollowers)
      .filter(id => s.selectedFollowers[id])
      .map(id => Number(id));
    this.setState({ composerPending: true, composerError: '', privacyOpen: false });
    try {
      const form = PostModel.buildCreatePostForm({
        text: s.composerText,
        privacy: s.privacy,
        selectedUserIDs,
        media: s.composerFile
      }, FormData);
      const response = await AuthAPI.createPost(form);
      const post = this.mapAPIPost(response);
      const apiUsersByID = this.mergeAPIUsers([response.author]);
      const me = apiUsersByID[String(USERS.me.apiId)];
      if (me) me.postsCount = (me.postsCount || 0) + 1;
      const input = document.getElementById('post-media');
      if (input) input.value = '';
      this.setState({
        apiUsersByID,
        posts: [post].concat(this.state.posts.filter(item => item.id !== post.id)),
        profilePosts: Number(this.state.profileId) === USERS.me.apiId
          ? [post].concat(this.state.profilePosts.filter(item => item.id !== post.id))
          : this.state.profilePosts,
        composerText: '', composerFile: null, composerFileName: '',
        composerError: '', composerPending: false,
        privacy: 'public', privacyOpen: false, selectedFollowers: {}
      });
    } catch (error) {
      this.setState({
        composerPending: false,
        composerError: requestErrorMessage(error, 'Could not create the post. Your draft was kept.')
      });
    }
  };

  likeGroupPost = (gid, pid) => {
    this.setState({ groups: this.state.groups.map(g => g.id !== gid ? g : Object.assign({}, g, { posts: g.posts.map(p => p.id === pid ? Object.assign({}, p, { liked: !p.liked, likes: p.likes + (p.liked ? -1 : 1) }) : p) })) });
  };
  addGroupComment = (gid, pid) => {
    const key = gid + ':' + pid;
    const text = (this.state.drafts[key] || '').trim();
    if (!text) return;
    this.setState({
      groups: this.state.groups.map(g => g.id !== gid ? g : Object.assign({}, g, { posts: g.posts.map(p => p.id === pid ? Object.assign({}, p, { comments: p.comments.concat([{ uid: 'me', text, time: 'now' }]) }) : p) })),
      drafts: Object.assign({}, this.state.drafts, { [key]: '' })
    });
  };

  patchGroup(gid, patch) {
    this.setState({ groups: this.state.groups.map(g => g.id === gid ? Object.assign({}, g, typeof patch === 'function' ? patch(g) : patch) : g) });
  }
  joinGroup = (gid) => {
    const g = this.state.groups.find(x => x.id === gid);
    this.patchGroup(gid, { state: g.state === 'requested' ? 'none' : 'requested' });
  };
  acceptInvite = (gid) => {
    this.patchGroup(gid, (g) => ({ state: 'joined', members: g.members + 1, memberIds: g.memberIds.concat(['me']) }));
  };
  declineInvite = (gid) => this.patchGroup(gid, { state: 'none' });
  rsvp = (gid, eid, val) => {
    this.patchGroup(gid, (g) => ({
      events: g.events.map(ev => {
        if (ev.id !== eid) return ev;
        const going = ev.going.filter(x => x !== 'me');
        return Object.assign({}, ev, { rsvp: val, going: val === 'going' ? going.concat(['me']) : going });
      })
    }));
  };
  createEvent = () => {
    const t = this.state.evTitle.trim();
    if (!t) return;
    const dateParts = (this.state.evDate.trim() || 'Jul 30').split(' ');
    const mon = (dateParts[0] || 'JUL').toUpperCase().slice(0, 3);
    const day = dateParts[1] || '30';
    this.patchGroup(this.state.groupId, (g) => ({
      events: [{ id: 'e' + Date.now(), title: t, day, mon, time: this.state.evTime.trim() || '18:00', desc: this.state.evDesc.trim() || 'Created just now.', rsvp: 'going', going: ['me'] }].concat(g.events)
    }));
    this.setState({ newEventOpen: false, evTitle: '', evDate: '', evTime: '', evDesc: '' });
  };
  createGroup = () => {
    const name = this.state.ngName.trim();
    if (!name) return;
    const g = { id: 'g' + Date.now(), name, desc: this.state.ngDesc.trim() || 'A brand new group.', members: 1, color: GROUP_COLORS[this.state.groups.length % GROUP_COLORS.length], state: 'joined', owner: 'me', memberIds: ['me'], posts: [], events: [], requests: [] };
    this.setState({ groups: [g].concat(this.state.groups), createOpen: false, ngName: '', ngDesc: '' });
  };
  postToGroup = () => {
    const text = this.state.gComposer.trim();
    if (!text) return;
    this.patchGroup(this.state.groupId, (g) => ({ posts: [{ id: 'gp' + Date.now(), uid: 'me', time: 'now', text, likes: 0, liked: false, comments: [] }].concat(g.posts) }));
    this.setState({ gComposer: '' });
  };
  handleRequest = (gid, uid, accept) => {
    this.patchGroup(gid, (g) => ({
      requests: g.requests.map(r => r.uid === uid ? Object.assign({}, r, { status: accept ? 'accepted' : 'declined' }) : r),
      members: accept ? g.members + 1 : g.members,
      memberIds: accept ? g.memberIds.concat([uid]) : g.memberIds
    }));
  };

  openConvo = (id) => {
    this.setState({ convoId: id, emojiOpen: false, convos: this.state.convos.map(c => c.id === id ? Object.assign({}, c, { unread: 0 }) : c) });
  };
  sendMsg = () => {
    const text = this.state.chatDraft.trim();
    if (!text) return;
    const id = this.state.convoId;
    const now = new Date();
    const hh = String(now.getHours()).padStart(2, '0') + ':' + String(now.getMinutes()).padStart(2, '0');
    this.setState({
      chatDraft: '', emojiOpen: false,
      convos: this.state.convos.map(c => c.id === id ? Object.assign({}, c, { messages: c.messages.concat([{ from: 'me', text, time: hh }]) }) : c)
    });
    const convo = this.state.convos.find(c => c.id === id);
    if (convo && convo.kind === 'dm') {
      setTimeout(() => this.setState({ convos: this.state.convos.map(c => c.id === id ? Object.assign({}, c, { typing: true }) : c) }), 700);
      setTimeout(() => {
        const reply = REPLIES[Math.floor(Math.random() * REPLIES.length)];
        this.setState({ convos: this.state.convos.map(c => c.id === id ? Object.assign({}, c, { typing: false, messages: c.messages.concat([{ from: convo.uid, text: reply, time: hh }]) }) : c) });
      }, 2200);
    }
  };

  acceptFollowRequest = async (requestID) => {
    const key = String(requestID);
    if (this.state.followRequestPendingByID[key]) return;
    const request = this.state.followRequests.find(item => String(item.id) === key);
    if (!request) return;
    this.setState({
      followRequestPendingByID: Object.assign({}, this.state.followRequestPendingByID, { [key]: true }),
      followRequestsError: ''
    });
    try {
      await AuthAPI.acceptFollowRequest(requestID);
      const user = this.apiUser(request.user.id);
      const apiUsersByID = this.mergeAPIUsers([{
        id: request.user.id,
        relationship: {
          status: user.relationship.status,
          follows_me: true
        }
      }]);
      const pending = Object.assign({}, this.state.followRequestPendingByID);
      delete pending[key];
      this.setState({
        apiUsersByID,
        followRequests: this.state.followRequests.filter(item => String(item.id) !== key),
        followRequestPendingByID: pending
      });
      this.loadPostFollowers();
      this.loadDirectory();
      this.loadFeed(true);
      if (this.state.screen === 'profile' && Number.isInteger(Number(this.state.profileId))) {
        this.openProfile(Number(this.state.profileId));
      } else {
        this.profileGate.begin();
      }
    } catch (error) {
      const pending = Object.assign({}, this.state.followRequestPendingByID);
      delete pending[key];
      this.setState({
        followRequestPendingByID: pending,
        followRequestsError: requestErrorMessage(error, 'Could not accept follow request.')
      });
    }
  };

  rejectFollowRequest = async (requestID) => {
    const key = String(requestID);
    if (this.state.followRequestPendingByID[key]) return;
    if (!this.state.followRequests.some(item => String(item.id) === key)) return;
    this.setState({
      followRequestPendingByID: Object.assign({}, this.state.followRequestPendingByID, { [key]: true }),
      followRequestsError: ''
    });
    try {
      await AuthAPI.rejectFollowRequest(requestID);
      const pending = Object.assign({}, this.state.followRequestPendingByID);
      delete pending[key];
      this.setState({
        followRequests: this.state.followRequests.filter(item => String(item.id) !== key),
        followRequestPendingByID: pending
      });
      this.loadDirectory();
    } catch (error) {
      const pending = Object.assign({}, this.state.followRequestPendingByID);
      delete pending[key];
      this.setState({
        followRequestPendingByID: pending,
        followRequestsError: requestErrorMessage(error, 'Could not reject follow request.')
      });
    }
  };

  acceptNotif = (nid) => {
    const n = this.state.notifs.find(x => x.id === nid);
    if (!n) return;
    if (n.type === 'invite') this.acceptInvite(n.gid);
    if (n.type === 'request') this.handleRequest(n.gid, n.uid, true);
    this.setState({ notifs: this.state.notifs.map(x => x.id === nid ? Object.assign({}, x, { status: 'accepted', read: true }) : x) });
  };
  declineNotif = (nid) => {
    const n = this.state.notifs.find(x => x.id === nid);
    if (!n) return;
    if (n.type === 'invite') this.declineInvite(n.gid);
    if (n.type === 'request') this.handleRequest(n.gid, n.uid, false);
    this.setState({ notifs: this.state.notifs.map(x => x.id === nid ? Object.assign({}, x, { status: 'declined', read: true }) : x) });
  };

  followBtn(userID) {
    const user = this.apiUser(userID);
    const model = UserModel.followButton(user, this.state.followPendingByID[String(userID)]);
    if (model.tone === 'muted') return { label: model.label, bg: 'var(--surface2)', color: 'var(--text2)', bd: 'transparent', disabled: model.disabled };
    if (model.tone === 'soft') return { label: model.label, bg: 'var(--soft)', color: 'var(--accent)', bd: 'transparent', disabled: model.disabled };
    return { label: model.label, bg: 'var(--accent)', color: '#fff', bd: 'transparent', disabled: model.disabled };
  }

  mapPost(p, inGroup) {
    const s = this.state;
    const key = inGroup ? s.groupId + ':' + p.id : p.id;
    const privacyMeta = { public: { icon: IC.globe, label: 'Public' }, followers: { icon: IC.users, label: 'Followers' }, selected: { icon: IC.lock, label: 'Selected' } };
    const pm = privacyMeta[p.privacy] || privacyMeta.public;
    const comments = p.comments || [];
    const likes = p.likes || 0;
    const user = p.real ? this.apiUser(p.apiAuthorID) : (p.user || USERS[p.uid] || USERS.me);
    return Object.assign({}, p, {
      user,
      privacyIcon: pm.icon, privacyLabel: pm.label,
      hasImage: !!p.mediaUrl || !!p.image,
      mediaUrl: p.mediaUrl || '',
      showRealMedia: !!p.mediaUrl,
      showMockImage: !p.mediaUrl && !!p.image,
      showMockActions: !p.real,
      notLiked: !p.liked,
      likeColor: p.liked ? 'var(--danger)' : 'var(--text2)',
      commentCount: num(comments.length),
      likes: num(likes),
      showComments: !!s.openComments[key],
      comments: comments.map(c => Object.assign({}, c, { user: USERS[c.uid] })),
      draft: s.drafts[key] || '',
      onLike: () => inGroup ? this.likeGroupPost(s.groupId, p.id) : this.likePost(p.id),
      onToggleComments: () => this.setState({ openComments: Object.assign({}, s.openComments, { [key]: !s.openComments[key] }) }),
      onDraft: (e) => this.setState({ drafts: Object.assign({}, this.state.drafts, { [key]: e.target.value }) }),
      onKey: (e) => { if (e.key === 'Enter') { inGroup ? this.addGroupComment(s.groupId, p.id) : this.addComment(p.id); } },
      onSendComment: () => inGroup ? this.addGroupComment(s.groupId, p.id) : this.addComment(p.id),
      goProfile: () => { if (p.real) this.openProfile(p.apiAuthorID); }
    });
  }

  renderVals() {
    const s = this.state;
    const me = USERS.me;
    const notifUnread = s.notifs.filter(n => !n.read).length + s.followRequests.length;
    const chatUnread = s.convos.reduce((a, c) => a + c.unread, 0);

    const navDefs = [
      { k: 'feed', label: 'Home', icon: IC.home, badge: 0 },
      { k: 'profile', label: 'Profile', icon: IC.user, badge: 0 },
      { k: 'groups', label: 'Groups', icon: IC.users, badge: 0 },
      { k: 'chat', label: 'Messages', icon: IC.chat, badge: chatUnread },
      { k: 'notifications', label: 'Notifications', icon: IC.bell, badge: notifUnread }
    ];
    const activeKey = s.screen === 'group' ? 'groups' : (s.screen === 'profile' && Number(s.profileId) !== me.apiId ? '' : s.screen);
    const navItems = navDefs.map(n => {
      const on = n.k === activeKey && !(n.k === 'profile' && Number(s.profileId) !== me.apiId);
      return {
        icon: n.icon, label: n.label,
        bg: on ? 'var(--soft)' : 'transparent',
        color: on ? 'var(--accent)' : 'var(--text2)',
        w: on ? '800' : '600',
        hasBadge: n.badge > 0, badge: num(n.badge),
        go: () => { if (n.k === 'profile') this.openProfile(me.apiId); else this.go(n.k); }
      };
    });

    // feed
    const feedPosts = s.posts.map((p, i) => Object.assign(this.mapPost(p, false), { delay: (i * 0.06).toFixed(2) + 's' }));
    const privacyMeta = { public: { icon: IC.globe, label: 'Public' }, followers: { icon: IC.users, label: 'Followers' }, selected: { icon: IC.lock, label: 'Selected' } };
    const privacyOptions = [
      { k: 'public', label: 'Public', desc: 'Anyone on loop can see this', icon: IC.globe },
      { k: 'followers', label: 'Followers only', desc: 'People who follow you', icon: IC.users },
      { k: 'selected', label: 'Selected followers', desc: 'Choose exactly who sees it', icon: IC.lock }
    ].map(o => ({
      label: o.label, desc: o.desc, icon: o.icon,
      isOn: s.privacy === o.k,
      bg: s.privacy === o.k ? 'var(--soft)' : 'transparent',
      pick: () => {
        this.setState({ privacy: o.k, privacyOpen: false });
        if (o.k === 'selected') this.loadPostFollowers();
      }
    }));
    const followerChips = s.postFollowers.map(u => {
      const uid = String(u.apiId);
      const on = !!s.selectedFollowers[uid];
      return {
        name: u.name.split(' ')[0], initials: u.initials, color: u.color,
        bg: on ? 'var(--soft)' : 'transparent',
        bd: on ? 'var(--accent)' : 'var(--border)',
        tc: on ? 'var(--accent)' : 'var(--text2)',
        toggle: () => this.setState({ selectedFollowers: Object.assign({}, s.selectedFollowers, { [uid]: !on }) })
      };
    });
    const composerAudienceReady = s.privacy !== 'selected' || Object.keys(s.selectedFollowers).some(id => s.selectedFollowers[id]);

    // profile
    const pUser = this.apiUser(s.profileId || me.apiId);
    const pIsMe = Number(s.profileId) === me.apiId;
    const pCanView = s.profileReady && pUser.canViewProfile !== false;
    const pPostsRaw = s.profilePosts;
    const followerIds = s.profileFollowers;
    const followingIds = s.profileFollowing;
    const mkUserRow = (userID) => {
      const u = this.apiUser(userID);
      const b = this.followBtn(userID);
      return {
        user: u, showBtn: Number(userID) !== me.apiId,
        btnLabel: b.label, btnBg: b.bg, btnColor: b.color, btnBd: b.bd,
        btnDisabled: b.disabled,
        onBtn: () => this.toggleFollow(userID),
        goProfile: () => this.openProfile(userID)
      };
    };
    const pTabs = [
      { k: 'posts', label: 'Posts' },
      { k: 'followers', label: 'Followers · ' + (pUser.followersCount || 0) },
      { k: 'following', label: 'Following · ' + (pUser.followingCount || 0) }
    ].map(t => ({
      label: t.label,
      color: s.profileTab === t.k ? 'var(--text)' : 'var(--text3)',
      bd: s.profileTab === t.k ? 'var(--accent)' : 'transparent',
      pick: () => this.setState({ profileTab: t.k })
    }));
    const fb = this.followBtn(s.profileId || 0);
    const privacySeg = [
      { k: 'public', label: 'Public', icon: IC.globe },
      { k: 'private', label: 'Private', icon: IC.lock }
    ].map(o => ({
      label: o.label, icon: o.icon,
      bg: s.myPrivacy === o.k ? 'var(--surface)' : 'transparent',
      color: s.myPrivacy === o.k ? 'var(--text)' : 'var(--text3)',
      disabled: s.profilePrivacyPending || s.profileEditPending || s.profileAvatarPending,
      opacity: s.profilePrivacyPending || s.profileEditPending || s.profileAvatarPending ? '0.6' : '1',
      cursor: s.profilePrivacyPending ? 'wait' : (s.profileEditPending || s.profileAvatarPending ? 'not-allowed' : 'pointer'),
      pick: () => this.setProfilePrivacy(o.k)
    }));

    // groups
    const groupCards = s.groups.map((g, i) => ({
      name: g.name, desc: g.desc, membersLabel: num(g.members), cover: cover(g.color),
      delay: (i * 0.05).toFixed(2) + 's',
      isJoined: g.state === 'joined', isNone: g.state === 'none', isRequested: g.state === 'requested', isInvited: g.state === 'invited',
      open: () => { if (g.state === 'joined') this.setState({ screen: 'group', groupId: g.id, groupTab: 'posts', inviteOpen: false }); },
      join: () => this.joinGroup(g.id),
      acceptInvite: () => this.acceptInvite(g.id),
      declineInvite: () => this.declineInvite(g.id)
    }));

    const g = s.groups.find(x => x.id === s.groupId) || s.groups[0];
    const gIsOwner = g.owner === 'me';
    const gTabs = [
      { k: 'posts', label: 'Posts' },
      { k: 'events', label: 'Events · ' + g.events.length },
      { k: 'members', label: 'Members' }
    ].map(t => ({
      label: t.label,
      color: s.groupTab === t.k ? 'var(--text)' : 'var(--text3)',
      bd: s.groupTab === t.k ? 'var(--accent)' : 'transparent',
      pick: () => this.setState({ groupTab: t.k })
    }));
    const gPosts = g.posts.map(p => this.mapPost(p, true));
    const gEvents = g.events.map(ev => ({
      title: ev.title, dateDay: ev.day, dateMon: ev.mon,
      timeLabel: ev.mon + ' ' + ev.day + ' · ' + ev.time, desc: ev.desc,
      goingLabel: num(ev.going.length),
      goingAvatars: ev.going.slice(0, 3).map(uid => USERS[uid]),
      goBg: ev.rsvp === 'going' ? 'var(--accent)' : 'transparent',
      goColor: ev.rsvp === 'going' ? '#fff' : 'var(--text2)',
      goBd: ev.rsvp === 'going' ? 'transparent' : 'var(--border)',
      noBg: ev.rsvp === 'not' ? 'var(--surface2)' : 'transparent',
      noColor: ev.rsvp === 'not' ? 'var(--text)' : 'var(--text2)',
      noBd: ev.rsvp === 'not' ? 'var(--text3)' : 'var(--border)',
      setGoing: () => this.rsvp(g.id, ev.id, 'going'),
      setNot: () => this.rsvp(g.id, ev.id, 'not')
    }));
    const gMembers = g.memberIds.map(uid => ({ user: USERS[uid], isOwner: uid === g.owner, goProfile: () => {} }));
    const gRequests = (gIsOwner ? g.requests : []).map(r => ({
      user: USERS[r.uid],
      pending: r.status === 'pending', done: r.status !== 'pending',
      doneLabel: r.status === 'accepted' ? 'Accepted' : 'Declined',
      accept: () => this.handleRequest(g.id, r.uid, true),
      decline: () => this.handleRequest(g.id, r.uid, false)
    }));
    const inviteChips = Object.keys(s.mockFollow).filter(uid => s.mockFollow[uid] === 'accepted' && g.memberIds.indexOf(uid) < 0).map(uid => {
      const u = USERS[uid];
      const on = !!s.invited[g.id + ':' + uid];
      return {
        label: on ? u.name.split(' ')[0] + ' · invited ✓' : u.name.split(' ')[0],
        initials: u.initials, color: u.color,
        bg: on ? 'var(--soft)' : 'transparent',
        bd: on ? 'var(--accent)' : 'var(--border)',
        tc: on ? 'var(--accent)' : 'var(--text2)',
        toggle: () => this.setState({ invited: Object.assign({}, s.invited, { [g.id + ':' + uid]: !on }) })
      };
    });

    // chat
    const convoMeta = (c) => {
      if (c.kind === 'dm') { const u = USERS[c.uid]; return { title: u.name, initials: u.initials, color: u.color, sub: c.online ? 'Online now' : 'Active recently' }; }
      const gr = s.groups.find(x => x.id === c.gid);
      return { title: gr.name, initials: gr.name.split(' ').map(w => w[0]).slice(0, 2).join(''), color: gr.color, sub: gr.members + ' members · group chat' };
    };
    const convos = s.convos.map(c => {
      const m = convoMeta(c);
      const last = c.messages[c.messages.length - 1];
      return {
        title: m.title, initials: m.initials, color: m.color,
        preview: (last.from === 'me' ? 'You: ' : '') + last.text,
        previewColor: c.unread > 0 ? 'var(--text)' : 'var(--text3)',
        previewW: c.unread > 0 ? '700' : '500',
        time: last.time,
        online: !!c.online,
        hasUnread: c.unread > 0, unread: num(c.unread),
        bg: c.id === s.convoId ? 'var(--soft)' : 'transparent',
        open: () => this.openConvo(c.id)
      };
    });
    const active = s.convos.find(c => c.id === s.convoId) || s.convos[0];
    const am = convoMeta(active);
    const messages = active.messages.map((msg, i) => {
      const prev = active.messages[i - 1];
      return {
        text: msg.text, time: msg.time,
        mine: msg.from === 'me', theirs: msg.from !== 'me',
        user: USERS[msg.from] || USERS.me,
        showName: active.kind === 'group' && msg.from !== 'me' && (!prev || prev.from !== msg.from)
      };
    });
    const emojis = EMOJIS.map(ch => ({ ch, add: () => this.setState({ chatDraft: s.chatDraft + ch }) }));

    // notifications
    const gName = (gid) => { const gr = s.groups.find(x => x.id === gid); return gr ? gr.name : ''; };
    const mockNotifItems = s.notifs.map((n, i) => {
      const meta = {
        invite: { icon: IC.users, text: 'invited you to join ' + gName(n.gid), accepted: 'Joined ' + gName(n.gid), declined: 'Invite declined' },
        request: { icon: IC.plus, text: 'asked to join your group ' + gName(n.gid), accepted: 'Added to the group', declined: 'Request declined' },
        event: { icon: IC.cal, text: 'created the event \u201c' + (n.extra || '') + '\u201d in ' + gName(n.gid), accepted: '', declined: '' }
      }[n.type];
      return {
        user: USERS[n.uid], icon: meta.icon, text: meta.text, time: n.time + ' ago',
        delay: (i * 0.06).toFixed(2) + 's',
        bg: n.read ? 'var(--surface)' : 'color-mix(in oklab, var(--accent) 5%, var(--surface))',
        unreadDot: !n.read,
        pending: n.status === 'pending',
        done: n.status === 'accepted' || n.status === 'declined',
        doneLabel: n.status === 'accepted' ? meta.accepted : meta.declined,
        accept: () => this.acceptNotif(n.id),
        decline: () => this.declineNotif(n.id),
        disabled: false,
        goProfile: () => {}
      };
    });
    const followRequestItems = s.followRequests.map((request, i) => {
      const pending = !!s.followRequestPendingByID[String(request.id)];
      return {
        user: this.apiUser(request.user.id), icon: IC.user, text: 'requested to follow you',
        time: this.formatPostTime(request.created_at), delay: (i * 0.06).toFixed(2) + 's',
        bg: 'color-mix(in oklab, var(--accent) 5%, var(--surface))', unreadDot: true,
        pending: true, done: false, doneLabel: '', disabled: pending,
        accept: () => this.acceptFollowRequest(request.id),
        decline: () => this.rejectFollowRequest(request.id),
        goProfile: () => this.openProfile(request.user.id)
      };
    });
    const notifItems = followRequestItems.concat(mockNotifItems);

    // right rail
    const suggestions = s.directoryUserIDs.map(userID => this.apiUser(userID))
      .filter(user => !user.relationship || user.relationship.status !== 'accepted')
      .map(user => {
        const b = this.followBtn(user.apiId);
        return {
          user, isPrivate: user.private,
          btnLabel: b.label, btnBg: b.bg, btnColor: b.color, btnBd: b.bd, btnDisabled: b.disabled,
          onBtn: () => this.toggleFollow(user.apiId), goProfile: () => this.openProfile(user.apiId)
        };
      });
    const railEvents = [];
    s.groups.forEach(gr => { if (gr.state === 'joined') gr.events.forEach(ev => railEvents.push({ title: ev.title, day: ev.day, mon: ev.mon, group: gr.name, open: () => this.setState({ screen: 'group', groupId: gr.id, groupTab: 'events' }) })); });

    const authTabs = [
      { k: 'login', label: 'Sign in' },
      { k: 'register', label: 'Create account' }
    ].map(t => ({
      label: t.label,
      bg: s.authMode === t.k ? 'var(--surface)' : 'transparent',
      color: s.authMode === t.k ? 'var(--text)' : 'var(--text3)',
      sh: s.authMode === t.k ? 'var(--shadow)' : 'none',
      pick: () => this.setAuthMode(t.k)
    }));

    return {
      // shell
      isAuthChecking: s.authStatus === 'checking', isAuthStartupError: s.authStatus === 'error',
      isAuth: s.authStatus === 'anonymous', isApp: s.authStatus === 'authenticated',
      isFeed: s.screen === 'feed',
      isProfile: s.screen === 'profile' && s.profileReady,
      isProfileLoading: s.screen === 'profile' && s.profileLoading,
      isProfileError: s.screen === 'profile' && !s.profileLoading && !s.profileReady && !!s.profileError,
      profileError: s.profileError,
      retryProfile: () => this.openProfile(s.profileId),
      isGroups: s.screen === 'groups',
      isGroup: s.screen === 'group', isChat: s.screen === 'chat', isNotifs: s.screen === 'notifications',
      rightRail: ['feed', 'profile', 'groups', 'notifications'].indexOf(s.screen) >= 0,
      navItems, me,
      themeIcon: s.theme === 'light' ? IC.moon : IC.sun,
      themeLabel: s.theme === 'light' ? 'Dark mode' : 'Light mode',
      toggleTheme: this.toggleTheme,
      goHome: () => this.go('feed'),
      goMyProfile: () => this.openProfile(me.apiId),
      goLogout: this.logout,
      logoutDisabled: s.logoutPending,
      appHasError: !!s.appError, appError: s.appError,
      // auth
      authTabs, authIsLogin: s.authMode === 'login', authIsReg: s.authMode === 'register',
      authCta: s.authPending ? 'Please wait…' : (s.authMode === 'login' ? 'Sign in' : 'Create account'),
      authDisabled: s.authPending,
      authButtonOpacity: s.authPending ? '0.65' : '1',
      authButtonCursor: s.authPending ? 'wait' : 'pointer',
      authHasError: !!s.authError, authError: s.authError,
      bootstrapError: s.bootstrapError, retryAuthBootstrap: this.loadCurrentUser,
      authEmail: s.authEmail, onAuthEmail: (e) => this.setState({ authEmail: e.target.value }),
      authPassword: s.authPassword, onAuthPassword: (e) => this.setState({ authPassword: e.target.value }),
      regFirstName: s.regFirstName, onRegFirstName: (e) => this.setState({ regFirstName: e.target.value }),
      regLastName: s.regLastName, onRegLastName: (e) => this.setState({ regLastName: e.target.value }),
      regDateOfBirth: s.regDateOfBirth, onRegDateOfBirth: (e) => this.setState({ regDateOfBirth: e.target.value }),
      regGender: s.regGender, onRegGender: (e) => this.setState({ regGender: e.target.value }),
      regNickname: s.regNickname, onRegNickname: (e) => this.setState({ regNickname: e.target.value }),
      regAboutMe: s.regAboutMe, onRegAboutMe: (e) => this.setState({ regAboutMe: e.target.value }),
      avatarButtonLabel: s.regAvatarName || 'avatar',
      pickRegistrationAvatar: this.pickRegistrationAvatar,
      onRegistrationAvatar: this.onRegistrationAvatar,
      submitAuth: this.submitAuth,
      // feed
      feedLoading: s.feedLoading, feedReady: !s.feedLoading,
      feedHasError: !!s.feedError, feedError: s.feedError,
      retryFeed: () => this.loadFeed(true),
      feedHasMore: !!s.feedNextCursor,
      feedLoadMore: () => this.loadFeed(false),
      feedLoadMoreLabel: s.feedPending && !s.feedLoading ? 'Loading…' : 'Load more',
      feedLoadMoreDisabled: s.feedPending,
      posts: feedPosts,
      composerText: s.composerText,
      onComposer: (e) => this.setState({ composerText: e.target.value, composerError: '' }),
      composerHasMedia: !!s.composerFile,
      composerMediaName: s.composerFileName,
      pickComposerMedia: this.pickComposerMedia,
      onComposerMedia: this.onComposerMedia,
      removeComposerMedia: this.removeComposerMedia,
      composerHasError: !!s.composerError, composerError: s.composerError,
      privacyOpen: s.privacyOpen,
      togglePrivacy: () => this.setState({ privacyOpen: !s.privacyOpen }),
      privacyIcon: privacyMeta[s.privacy].icon,
      privacyLabel: privacyMeta[s.privacy].label,
      privacyOptions,
      privacyIsSelected: s.privacy === 'selected',
      followerChips,
      selectedFollowersEmpty: s.postFollowers.length === 0 && !s.postFollowersLoading,
      postBtnOp: s.composerText.trim() && composerAudienceReady && !s.composerPending ? '1' : '0.45',
      postBtnDisabled: s.composerPending || !s.composerText.trim() || !composerAudienceReady,
      postButtonLabel: s.composerPending ? 'Posting…' : 'Post',
      sendPost: this.sendPost,
      // profile
      pUser, pIsMe, pOther: !pIsMe,
      pCover: cover(pUser.color),
      pShowLock: pUser.private || (pIsMe && s.myPrivacy === 'private'),
      pCanView, pLocked: !pCanView,
      pShowEmail: pIsMe && !!pUser.email,
      pStatPosts: num(pUser.postsCount || 0),
      pStatFollowers: num(pUser.followersCount || 0),
      pStatFollowing: num(pUser.followingCount || 0),
      pTabs,
      pTabPosts: s.profileTab === 'posts', pTabFollowers: s.profileTab === 'followers', pTabFollowing: s.profileTab === 'following',
      pPosts: pPostsRaw.map(p => this.mapPost(p, false)),
      pNoPosts: !s.profilePostsLoading && !s.profilePostsError && pPostsRaw.length === 0,
      pPostsLoading: s.profilePostsLoading,
      pPostsHasError: !!s.profilePostsError,
      pPostsError: s.profilePostsError,
      retryProfilePosts: () => this.loadProfilePosts(s.profileId, true),
      pPostsHasMore: !!s.profilePostsNextCursor,
      loadMoreProfilePosts: () => this.loadProfilePosts(s.profileId, false),
      profileLoadMoreLabel: s.profilePostsPending && !s.profilePostsLoading ? 'Loading…' : 'Load more',
      profileLoadMoreDisabled: s.profilePostsPending,
      pFollowers: followerIds.map(mkUserRow), pFollowing: followingIds.map(mkUserRow),
      pListsLoading: s.profileListsLoading,
      pListsHasError: !!s.profileListsError,
      pListsError: s.profileListsError,
      followLabel: fb.label, followBg: fb.bg, followColor: fb.color, followBd: fb.bd,
      followDisabled: fb.disabled,
      followHasError: !!s.followErrorByID[String(s.profileId)],
      followError: s.followErrorByID[String(s.profileId)] || '',
      onFollow: () => this.toggleFollow(s.profileId),
      msgProfile: () => this.setState({ appError: 'Messages are not connected to backend users yet.' }),
      privacySeg,
      profilePrivacyHasError: pIsMe && !!s.profilePrivacyError,
      profilePrivacyError: s.profilePrivacyError,
      showProfileEdit: pIsMe && s.profileEditOpen,
      openProfileEdit: this.openProfileEdit,
      cancelProfileEdit: this.cancelProfileEdit,
      saveProfile: this.saveProfile,
      profileEditPending: s.profileEditPending || s.profileAvatarPending || s.profilePrivacyPending,
      profileSaveLabel: s.profileEditPending ? 'Saving…' : 'Save changes',
      profileEditHasError: !!s.profileEditError,
      profileEditError: s.profileEditError,
      editFirstName: s.editFirstName, onEditFirstName: (e) => this.setState({ editFirstName: e.target.value }),
      editLastName: s.editLastName, onEditLastName: (e) => this.setState({ editLastName: e.target.value }),
      editDateOfBirth: s.editDateOfBirth, onEditDateOfBirth: (e) => this.setState({ editDateOfBirth: e.target.value }),
      editGender: s.editGender, onEditGender: (e) => this.setState({ editGender: e.target.value }),
      editNickname: s.editNickname, onEditNickname: (e) => this.setState({ editNickname: e.target.value }),
      editAboutMe: s.editAboutMe, onEditAboutMe: (e) => this.setState({ editAboutMe: e.target.value }),
      profileAvatarLabel: s.editAvatarName || 'Choose image',
      profileAvatarPending: s.profileAvatarPending || s.profileEditPending || s.profilePrivacyPending,
      profileAvatarUploadDisabled: s.profileAvatarPending || s.profileEditPending || s.profilePrivacyPending || !s.editAvatar,
      profileAvatarUploadOpacity: s.profileAvatarPending || s.profileEditPending || s.profilePrivacyPending || !s.editAvatar ? '0.55' : '1',
      profileAvatarUploadLabel: s.profileAvatarPending ? 'Working…' : 'Upload',
      profileHasCustomAvatar: me.hasCustomAvatar,
      pickProfileAvatar: this.pickProfileAvatar,
      onProfileAvatar: this.onProfileAvatar,
      replaceProfileAvatar: this.replaceProfileAvatar,
      deleteProfileAvatar: this.deleteProfileAvatar,
      // groups
      createOpen: s.createOpen,
      toggleCreate: () => this.setState({ createOpen: !s.createOpen }),
      ngName: s.ngName, onNgName: (e) => this.setState({ ngName: e.target.value }),
      ngDesc: s.ngDesc, onNgDesc: (e) => this.setState({ ngDesc: e.target.value }),
      createGroup: this.createGroup,
      groupCards,
      // group detail
      gName: g.name, gDesc: g.desc, gMembersLabel: num(g.members), gCover: cover(g.color), gIsOwner,
      gBack: () => this.go('groups'),
      gTabs, gTabPosts: s.groupTab === 'posts', gTabEvents: s.groupTab === 'events', gTabMembers: s.groupTab === 'members',
      gPosts, gEvents, gMembers, gRequests,
      gHasRequests: gRequests.length > 0,
      gComposer: s.gComposer,
      onGComposer: (e) => this.setState({ gComposer: e.target.value }),
      onGComposerKey: (e) => { if (e.key === 'Enter') this.postToGroup(); },
      gPost: this.postToGroup,
      inviteOpen: s.inviteOpen,
      toggleInvite: () => this.setState({ inviteOpen: !s.inviteOpen }),
      inviteChips,
      newEventOpen: s.newEventOpen,
      toggleNewEvent: () => this.setState({ newEventOpen: !s.newEventOpen }),
      evTitle: s.evTitle, onEvTitle: (e) => this.setState({ evTitle: e.target.value }),
      evDate: s.evDate, onEvDate: (e) => this.setState({ evDate: e.target.value }),
      evTime: s.evTime, onEvTime: (e) => this.setState({ evTime: e.target.value }),
      evDesc: s.evDesc, onEvDesc: (e) => this.setState({ evDesc: e.target.value }),
      createEvent: this.createEvent,
      // chat
      convos, messages,
      activeTitle: am.title, activeSub: am.sub, activeInitials: am.initials, activeColor: am.color,
      typing: active.typing,
      chatDraft: s.chatDraft,
      onChatDraft: (e) => this.setState({ chatDraft: e.target.value }),
      onChatKey: (e) => { if (e.key === 'Enter') this.sendMsg(); },
      sendMsg: this.sendMsg,
      emojiOpen: s.emojiOpen,
      toggleEmoji: () => this.setState({ emojiOpen: !s.emojiOpen }),
      emojis,
      msgRef: (el) => { this.msgEl = el; },
      // notifications
      notifItems,
      followRequestsHasError: !!s.followRequestsError,
      followRequestsError: s.followRequestsError,
      markAllRead: () => this.setState({ notifs: s.notifs.map(n => Object.assign({}, n, { read: true })) }),
      // rail
      suggestions, railEvents,
      suggestionsHasError: !!s.directoryError,
      suggestionsError: s.directoryError
    };
  }
}

if (typeof module === 'object' && module.exports) module.exports = { Component };
