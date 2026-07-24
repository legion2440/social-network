# Social Network Frontend

Static single-page frontend for the Social Network application.

The browser application uses a custom declarative JavaScript framework named `dc-runtime`. React and ReactDOM are its rendering layer, not the application's conventional component architecture.

This document covers frontend implementation, state, HTTP/WebSocket integration, stale-response protection, media preview lifecycle, and tests. For the product overview and Docker quick start, see the [root README](../README.md). For backend API and authorization contracts, see the [backend README](../backend/README.md).

[Русская версия](README_RU.md) · [Backend documentation](../backend/README.md)

## 📋 TOC

- [🚀 Run and test](#-run-and-test)
- [🧩 Execution model](#-execution-model)
- [⚙️ dc-runtime](#️-dc-runtime)
- [📄 Template syntax](#-template-syntax)
- [📁 File responsibilities](#-file-responsibilities)
- [🗃️ Application state](#️-application-state)
- [🌐 HTTP client](#-http-client)
- [🔄 Normalization and authoritative state](#-normalization-and-authoritative-state)
- [🛡️ Request generations](#️-request-generations)
- [🔑 Authentication lifecycle](#-authentication-lifecycle)
- [📝 Posts, comments, and media](#-posts-comments-and-media)
- [👥 Groups and events](#-groups-and-events)
- [🔔 Notifications](#-notifications)
- [💬 Chat and realtime](#-chat-and-realtime)
- [🧹 Resource cleanup](#-resource-cleanup)
- [📦 Production delivery](#-production-delivery)
- [🧪 Test suite](#-test-suite)
- [📁 Structure](#-structure)

## 🚀 Run and test

The frontend is static. It is served by:

- the Go backend during local development;
- Caddy in the production Docker topology.

Start the local full-stack application from `backend/`:

```bash
cd ../backend
go run ./cmd/server
```

Open:

```text
http://127.0.0.1:8080
```

Run frontend tests from `frontend/`:

```bash
npm test
```

The command uses Node's built-in test runner. There is no production dependency install or frontend build step in `package.json`.

## 🧩 Execution model

The browser loads the application from `index.html`.

```text
index.html
  |
  +--> resources and model scripts
  +--> bundled React
  +--> bundled ReactDOM
  +--> AuthAPI
  +--> dc-runtime
  +--> application logic
```

The `<x-dc>` element contains the application template. `dc-runtime` parses it, replaces `<x-dc>` with `#dc-root`, compiles directives and interpolations, creates React elements, and mounts through ReactDOM.

The UI is not split into conventional `.jsx` React components. Most application behavior lives in one `Component extends DCLogic` class whose state, methods, and computed values are exposed to the declarative template.

## ⚙️ dc-runtime

`frontend/js/runtime.js` is a generated runtime bundle and should not be edited manually without rebuilding its source.

The runtime provides:

- `<x-dc>` document discovery;
- template parsing and compilation;
- text and attribute interpolation;
- conditional and repeated rendering;
- event attribute mapping;
- dynamic attributes and styles;
- pseudo-style directives such as `style-hover`;
- React element construction;
- ReactDOM root mounting;
- unresolved-value placeholders;
- runtime import/component hooks.

React is read from:

```text
window.React
```

ReactDOM is read from:

```text
window.ReactDOM
```

Render pipeline:

```text
template
-> parse
-> compile expressions and attributes
-> build render functions
-> create React elements
-> mount into #dc-root
```

## 📄 Template syntax

### Interpolation

```html
<span>{{user.name}}</span>
<img src="{{user.avatarUrl}}">
```

A full attribute interpolation preserves the underlying value type. Mixed interpolation produces a string.

### Conditional rendering

```html
<sc-if value="{{isAuthenticated}}">
  <main>...</main>
</sc-if>
```

A false value removes the subtree.

### Repeated rendering

```html
<sc-for list="{{posts}}" as="post">
  <article>{{post.text}}</article>
</sc-for>
```

The runtime adds `$index` to the repeated scope.

### Event handlers

```html
<button onclick="{{submitPost}}">Publish</button>
<input onchange="{{onDraftChange}}">
```

HTML event attributes are converted to React event names.

### Dynamic attributes and styles

```html
<button
  disabled="{{pending}}"
  style="background:{{buttonBackground}}"
  style-hover="filter:brightness(1.05)">
```

Values are resolved from the component scope.

### Head content

`<helmet>` is converted to the runtime head-management directive.

## 📁 File responsibilities

| File | Responsibility |
|---|---|
| `index.html` | declarative SPA template and script loading |
| `js/runtime.js` | generated `dc-runtime` bundle |
| `js/app.js` | main application state and screen orchestration |
| `js/auth-api.js` | same-origin HTTP client and `APIError` |
| `js/resources.js` | resource/bootstrap integration |
| `js/user-model.js` | user normalization, privacy cleanup, relationship state, request gates |
| `js/post-model.js` | post normalization and state helpers |
| `js/comment-model.js` | comment normalization and pagination merge |
| `js/group-event-model.js` | event normalization and RSVP state |
| `js/notification-model.js` | notification normalization, revision merges |
| `js/chat-model.js` | chat keys, history merge, optimistic/realtime reconciliation |
| `js/avatar-url.js` | controlled avatar URL classification |
| `css/styles.css` | bundled font declarations |
| `css/styles-2.css` | theme tokens, global styles, animations |
| `Caddyfile` | static delivery, backend proxy, SPA boundary |
| `Dockerfile` | non-root static Caddy image |

Production identities, posts, comments, groups, events, notifications, and chats are loaded from backend responses. `USERS.me` remains only as the current-user holder.

## 🗃️ Application state

`app.js` groups state by feature.

Main areas:

```text
theme and current screen
authentication and registration
current user and directory
feed and post composer
per-post comments
profile and profile lists
groups and membership transitions
group posts
group events and RSVP
notifications and unread revision
chats, history, typing, presence, unread revision
```

Feature-specific constructors define consistent reset state:

```text
emptyRegistrationForm
emptyProfileEditor
emptyCommentState
emptyGroupPostState
emptyGroupEventState
emptyNotificationState
emptyChatState
```

Comments are stored per post:

```text
commentsByPostID[postID]
```

Chat histories, unread values, and read markers are stored by canonical chat key.

Backend IDs are authoritative. Decorative fields such as initials, colors, labels, avatar flags, and display time are derived locally.

## 🌐 HTTP client

`js/auth-api.js` exports:

```text
APIError
createAuthAPI(fetchImpl)
```

In the browser it creates:

```text
window.AuthAPI
```

All requests use:

```text
credentials: same-origin
Accept: application/json
```

Each method checks an exact expected HTTP status.

`APIError` contains:

```text
message
status
details
cause
```

Network failures use status `0`.

The client covers:

- registration, login, logout, session restore;
- profile update and avatar replacement;
- users, relationships, followers, follow requests;
- feed, profile posts, group posts, media comments;
- groups, invitations, join requests;
- events and RSVP;
- notifications;
- chat lists, histories, read markers.

Multipart requests pass `FormData` directly and do not set a manual multipart boundary.

## 🔄 Normalization and authoritative state

Backend objects are normalized before entering UI state.

Normalization includes:

- snake_case to stable frontend properties;
- positive backend ID validation;
- canonical store keys;
- safe null handling;
- relationship state normalization;
- media and avatar URL classification;
- timestamp formatting for display;
- pagination merge and deduplication.

`UserModel.normalizeUser` removes protected data when a profile is no longer viewable:

```text
email
date of birth
gender
about
statistics
private custom avatar
```

Mutation responses are authoritative. The frontend applies returned backend state instead of reconstructing security-sensitive results from user intent.

Examples:

- follow result determines `accepted` or `pending`;
- group result determines membership;
- event response returns current RSVP counts;
- notification action returns resolution, unread state, and source;
- chat read returns authoritative marker and revision.

## 🛡️ Request generations

Async reads use gates from:

```text
UserModel.createRequestGate()
```

A gate exposes:

```text
begin()
isCurrent(generation)
current()
```

A response updates state only if its generation is still current.

Independent gates exist for:

- auth bootstrap;
- feed;
- user directory;
- profile, profile posts, profile lists;
- group directory;
- group detail and membership lists;
- group posts;
- group events and per-event RSVP;
- notifications and notification actions;
- chats and active history;
- per-chat access and read state;
- per-post comment access and comment load.

Access revocation also records explicit revoked group/chat state.

This prevents:

- old profile reads restoring private data after unfollow;
- stale group reads restoring member-only content after leave;
- stale comment creation restoring content after parent access loss;
- stale notification actions overwriting a newer lifecycle;
- old chat-list responses restoring a removed group chat;
- pre-logout requests populating the next user session.

Only a new authoritative rejoin response may clear a group revoke.

## 🔑 Authentication lifecycle

Startup state:

```text
authStatus: checking
```

The frontend calls:

```text
GET /api/auth/me
```

Possible results:

- valid session: apply current user and bootstrap application data;
- `401`: show login/registration;
- network/server error: show retryable startup error.

Registration uses `FormData`. Login uses JSON.

After authentication:

- the returned user becomes `USERS.me`;
- normalized user store is updated;
- feature data begins loading;
- WebSocket starts.

Logout:

- calls the backend logout endpoint;
- invalidates request generations;
- closes realtime;
- clears user-derived collections;
- clears drafts and selected files;
- revokes preview object URLs;
- resets the current-user holder;
- returns to the auth screen.

A failed logout does not pretend the server session was removed.

## 📝 Posts, comments, and media

### Post composers

Personal post state contains:

```text
text
optional File
privacy
selected follower IDs
pending/error
```

Group post composer uses independent state and request gates.

Both post and comment creation use `FormData`.

### Per-post comments

Every post has independent:

```text
comments
cursor
loading
load pending
load error
draft
media File
media filename
preview URL
create pending
create error
access gate
load gate
```

File input IDs are unique:

```text
comment-media-{postID}
```

Comment creation behavior:

- send requires non-empty text;
- media is optional;
- file controls are disabled while create is pending;
- validation, network, and server errors preserve text, `File`, and preview;
- `403` or `404` is treated as access revoke and clears attachment state;
- success applies the authoritative comment and clears the composer;
- late results are ignored when access generation is stale.

### Object URL lifecycle

When a file changes:

```text
old URL -> revoke
new File -> URL.createObjectURL
```

Preview URLs are revoked on:

- replacement;
- manual remove;
- successful create;
- parent access revoke;
- comment state purge;
- logout;
- component unmount.

Controls stay disabled during pending create so a later file selection cannot be erased by the earlier request's success handler.

## 👥 Groups and events

Group state is separated into:

```text
directory
invitation inbox
selected group detail
members
join requests
sent invitations
group posts
group events
RSVP pending/error per event
membership mutation pending/error per group
```

UI permissions are derived from authoritative `viewer_status`.

Owner-only controls:

- join-request management;
- sent invitation management.

Owner/member controls:

- invite;
- group posts and comments;
- event creation and RSVP;
- group chat.

Leave or realtime `chat:remove`:

- invalidates group generations;
- hides member-only controls;
- purges posts, comments, events, drafts, and previews;
- removes the group chat;
- blocks stale responses from restoring access.

## 🔔 Notifications

Notification state includes:

```text
items and cursor
loading/pagination flags
unread count
revision
per-item read state
per-item action state
mark-all state
```

HTTP and persisted revisions remain authoritative. Realtime events are upsert/refresh signals.

The frontend:

- merges items by notification ID;
- applies only non-stale revisions;
- refreshes after actions and reconnect;
- race-gates source transitions independently;
- supports mark-one, mark-all, accept, and decline;
- updates the navigation badge from persisted unread state.

## 💬 Chat and realtime

Chat state is keyed by canonical direct or group chat key.

```text
summaries
active chat
history per key
presence
typing
optimistic messages
per-chat unread
total unread and revision
read markers
WebSocket status
reconnect attempt
```

The frontend opens same-origin `/ws` after authenticated bootstrap.

Handled events:

```text
presence:init
presence:update
presence:remove
typing:update
chat:message
chat:unread
chat:remove
chat:error
notification:upsert
notifications:read-all
```

Reconnect reloads the chat list and active authoritative history.

### Optimistic send

A new message receives a client UUID and appears immediately.

States:

```text
pending
sent
failed
```

Retry reuses the same UUID. History and realtime responses deduplicate by client/server identity. Unacknowledged optimistic messages become retryable failures after a timer.

### Read behavior

A chat is marked read only when:

- Messages is the active screen;
- the conversation is active;
- authoritative history is displayed;
- the browser tab is visible.

Bootstrap, reconnect preload, background list refresh, inactive history, and older-history pagination do not mark read.

Read markers and revisions move only forward.

### Typing and presence

Typing sends:

```text
typing:start
typing:heartbeat
typing:stop
```

Heartbeat and expiry timers prevent permanent typing indicators. Visibility and access changes remove stale presence and typing state.

## 🧹 Resource cleanup

`componentWillUnmount`:

- removes the `visibilitychange` listener;
- revokes remaining comment previews;
- stops realtime.

Logout and access revoke also clear:

- pending and typing timers;
- WebSocket reconnect timer;
- active socket;
- optimistic messages and chat drafts;
- per-post files and previews;
- group content and drafts;
- notification and user-specific state;
- stale request generations.

The cleanup model prevents one user's state from leaking into a later session.

## 📦 Production delivery

The production image contains only:

```text
index.html
css/
js/
assets/
Caddyfile
```

It does not include Node, test files, or frontend build tooling.

Caddy proxies exact and wildcard paths:

```text
/api
/api/*
/ws
/ws/*
/static/avatars
/static/avatars/*
```

Removed legacy paths return `404` before SPA fallback:

```text
/uploads
/uploads/*
```

All other paths use static delivery with `index.html` fallback.

Caddy runs as a numeric non-root user. Writable config/data paths use `/tmp`, and the container root filesystem is read-only.

Build and launch commands are in the [root README](../README.md).

## 🧪 Test suite

```bash
npm test
```

Executed files:

```text
js/app-race.test.js
js/auth-api.test.js
js/avatar-url.test.js
js/chat-model.test.js
js/comment-model.test.js
js/group-event-model.test.js
js/notification-model.test.js
js/post-model.test.js
js/user-model.test.js
```

Coverage includes:

- exact HTTP methods and statuses;
- API and network errors;
- URL classification;
- response normalization;
- privacy field removal;
- request generations;
- pagination merge and deduplication;
- selected audience pruning;
- comment create and preview lifecycle;
- event and RSVP state;
- notification revisions and source races;
- optimistic chat send and retry;
- reconnect/history merge;
- unread and read-marker monotonicity;
- logout/access-revoke stale-response protection.

Accepted result:

```text
121/121 tests passed
```

## 📁 Structure

```text
frontend/
├── assets/
│   ├── fonts/
│   └── ...
├── css/
│   ├── styles.css
│   └── styles-2.css
├── js/
│   ├── app.js
│   ├── app-race.test.js
│   ├── auth-api.js
│   ├── auth-api.test.js
│   ├── avatar-url.js
│   ├── chat-model.js
│   ├── comment-model.js
│   ├── group-event-model.js
│   ├── notification-model.js
│   ├── post-model.js
│   ├── resources.js
│   ├── runtime.js
│   ├── user-model.js
│   └── vendor/
├── Caddyfile
├── Dockerfile
├── index.html
├── package.json
├── README.md
└── README_RU.md
```
