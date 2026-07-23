# Social Network Backend

Minimal Go backend foundation adapted from the tested `forum` project.

## Run

From this `backend` directory:

```bash
go test ./...
go run ./cmd/server
```

The dependency-free frontend API-client tests use Node's built-in test runner:

```bash
cd ../frontend
npm test
```

The SQLite driver is `github.com/mattn/go-sqlite3`, so CGO and a working C
compiler are required. On Windows, run from Git Bash and verify that
`go env CGO_ENABLED` prints `1` before building.

The default address is `http://127.0.0.1:8080`. Runtime paths such as
`var/social-network.db` and `var/uploads` are relative to the backend working
directory. For local development the same Go server also serves the frontend
from `../frontend` by default, so browser requests and the session cookie stay
same-origin.

## Database migrations

Versioned SQLite migrations are embedded from
`internal/repo/sqlite/migrations`. Every version has an `.up.sql` and a
`.down.sql` file. `golang-migrate` applies pending migrations automatically
while the database is opened, before the HTTP server is created. The current
version and dirty state are stored in `schema_migrations`.

Current migrations:

- `000001_create_users`
- `000002_create_media`
- `000003_add_user_avatar_media`
- `000004_create_sessions`
- `000005_add_user_privacy`
- `000006_create_follows`
- `000007_create_posts`
- `000008_create_post_comments`
- `000009_create_groups`
- `000010_create_chats`
- `000011_add_group_posts`
- `000012_create_group_events`
- `000013_create_notifications`
- `000014_create_chat_read_states`

Migration `000011` rebuilds `posts` so personal and group publications share
one table while keeping existing post IDs, selected audiences, comments,
media links, timestamps, and the saved `sqlite_sequence` values for posts and
comments, including IDs above the current maximum that belonged to deleted
rows. Its down migration preserves the same AUTOINCREMENT state and is
allowed only while no group posts exist; the application rejects that down
operation before `golang-migrate` can mark the database dirty.

To inspect the current version with the SQLite CLI:

```bash
sqlite3 var/social-network.db "SELECT version, dirty FROM schema_migrations;"
```

The previous `schema.sql` bootstrap database is disposable and is not
automatically converted. If `var/social-network.db` was created by that old
bootstrap, delete it once before starting this version; the migrations will
create a fresh database. Runtime database and upload files remain ignored by
Git.

`date_of_birth` is stored as SQLite `TEXT` and exposed in JSON strictly as
`DD-MM-YYYY`. It must be a real calendar date, so values such as `31-02-1992`
are rejected. User audit timestamps, session timestamps, and media timestamps
continue to use UTC Unix seconds in SQLite.

The user model contains email, password hash, first and last name, date of
birth, optional `male`/`female` gender, optional nickname/about text, an
optional custom avatar media reference, and created/updated timestamps. Gender
accepts only `NULL`, `male`, or `female`; every other value is an error.
Profiles are public by default; `is_private` is stored as a constrained SQLite
integer and exposed as a JSON boolean. Deleting a user cascades to owned media
metadata, sessions, and follow relations; deleting avatar media sets the user's
optional avatar reference to `NULL`.

## Authentication

Registration uses one `multipart/form-data` request with required `email`,
`password`, `first_name`, `last_name`, and `date_of_birth` fields. Optional
fields are `gender`, `nickname`, `about_me`, and the `avatar` file. Omitting
`gender` stores `NULL`; an empty or unsupported value is rejected.

Avatar files are staged and detected from their contents before the SQLite
registration transaction starts. JPEG, PNG, GIF, and WebP are accepted up to
20 MB. The transaction creates the user, optional media row, avatar relation,
and session together. The file is moved to its final name before commit and is
removed if the transaction fails. The session cookie is set only after a
successful commit.

Users without custom media receive one of three static SVG placeholders based
on gender. Every full or summary user response exposes the computed
`avatar_url`. Custom avatars use the controlled
`/api/users/{userID}/avatar?v={mediaID}` route; the media ID is a cache-busting
version and changes on replacement. Placeholder URLs remain under
`/static/avatars/`.

Login accepts JSON with `email` and `password`. Authentication currently uses
an HttpOnly, SameSite=Lax session cookie. Request token extraction is separate
from cookie transport so a Bearer source can be added later. Multiple sessions
per user are allowed, and logout removes only the current session. Missing
tokens and already-absent sessions are successful no-ops; session storage
failures return `500`.

## Own profile

`PATCH /api/profile` accepts a partial JSON object containing `first_name`,
`last_name`, `date_of_birth`, `gender`, `nickname`, `about_me`, or `is_private`.
Omitted fields are unchanged. First/last name and `date_of_birth` cannot be
empty or `null`; dates keep the strict real-date `DD-MM-YYYY` contract. `gender`
accepts `male`, `female`, or JSON `null` to clear it. Empty nickname/about
values and JSON `null` clear those optional fields. `is_private` accepts only a
JSON boolean; `null`, strings, and numbers return `400`. Empty objects, unknown
fields, and unsupported values also return `400`.

Changing a public profile to private keeps existing accepted followers.
Changing a private profile to public does not automatically accept pending
requests. A pending request is promoted to accepted when its author explicitly
follows the now-public profile again, or when the profile owner accepts it.
The existing frontend Public/Private profile switch persists this field through
the same endpoint and keeps its previous state when the request fails.

`PUT /api/profile/avatar` replaces the current avatar from the multipart field
`avatar`; `DELETE /api/profile/avatar` removes it and returns the gender-based
placeholder. Both return the same full user representation as auth endpoints.
Replacement uses staging plus one SQL transaction for the media row and user
relation. A failed transaction removes the new file and preserves the old
avatar. After a successful commit, the replaced media file is removed.

## Custom avatar delivery

`GET /api/users/{id}/avatar` requires authentication and serves only the
user's current custom avatar. The query string is ignored. Public-profile
avatars are available to every authenticated user. Private-profile avatars are
available only to their owner and accepted followers; pending followers and
outsiders receive `403`. The current privacy and follow relation are checked in
the same read transaction as the user and media rows on every request.

Users without custom media keep their gender placeholder in `avatar_url`; an
authorized request to their controlled avatar route returns `404`. Missing
users, media rows, foreign-owned media, and missing physical files also return
`404`. Successful image responses include the stored MIME, actual file length,
`X-Content-Type-Options: nosniff`, and `Cache-Control: private, no-store`.

The legacy `/uploads/{mediaID}` route remains owner-only for general media and
is no longer emitted as a user avatar URL. The frontend recognizes only the
controlled user-avatar URL as custom, so replace keeps the Remove action and
delete switches it off when the response returns a placeholder.

## Followers

All follower endpoints require authentication. `PUT /api/users/{id}/follow`
creates an accepted relation for a public profile or a pending request for a
private profile. It returns `{"status":"accepted"}` or
`{"status":"pending"}`. Repeating the operation never demotes an accepted
relation. `DELETE /api/users/{id}/follow` removes the current relation and is
idempotent.

`GET /api/users/{id}/follow` returns the current relation as `none`, `pending`,
or `accepted`, plus `follows_me` for the accepted reverse relation.
`GET /api/users/{id}/followers` and `/following` list accepted relations only.
Every list row also contains the current viewer's `relationship` to that row,
so the frontend can render follow controls without an N+1 request pattern.
Lists for a public profile are available to every authenticated user. Lists for
a private profile are available only to its owner and accepted followers;
pending followers and outsiders receive `403`. Pending requests are available
to their target through `GET /api/follow-requests`; the owner accepts one with
`POST /api/follow-requests/{id}/accept` or rejects it with
`DELETE /api/follow-requests/{id}`. Pending relations never count as followers.

## User profiles and directory

`GET /api/users/{id}` returns an authenticated user's safe profile card and
never exposes email. Public profiles, owners, and accepted followers receive
`can_view_profile: true`, the full profile fields, accepted follower/following
counts, and a post count filtered through the same current post access policy
used by profile posts. Pending followers and outsiders still receive the basic
card for a private profile, but `can_view_profile` is false and sensitive
fields and counts are omitted. Unknown users return `404`.

`GET /api/users` is the authenticated discovery directory. It excludes the
current user, orders by `(created_at DESC, id DESC)`, and returns viewer-aware
user summaries plus an opaque `next_cursor`. The default limit is 20 and the
maximum is 50. This endpoint is a directory for the current frontend step; it
does not implement text search.

## Posts

`POST /api/posts` creates a post from one strict `multipart/form-data`
request. `text` and `privacy` must each occur exactly once. Text is trimmed,
must contain 1–5000 Unicode code points, and is always required even when media
is attached. Privacy is `public`, `followers`, or `selected`. Selected posts
accept repeated `selected_user_id` values; duplicates are normalized and the
result must contain 1–100 current accepted followers other than the author.
Audience values on other privacy modes, unknown fields, and duplicate scalar
fields return `400`.

The optional `media` field accepts one JPEG, PNG, GIF, or WebP file up to 20
MB. Media is staged before the SQL transaction. The transaction rechecks the
selected followers, creates the media, post, and audience rows, then finalizes
the file before commit. Any failure, including a commit failure after the move,
rolls back every row and removes the staged or final file.

`GET /api/posts/feed` returns the current user's posts plus posts from users
they currently follow with an accepted relation. It is not a global public
discovery feed. `GET /api/users/{id}/posts` returns accessible posts for one
profile; a private-profile outsider receives `403`, while an unknown profile
returns `404`. Both endpoints order by `(created_at DESC, id DESC)`, accept an
opaque `cursor`, default to `limit=20`, enforce a maximum of 50, and return
`posts` plus nullable `next_cursor`.

Post access is filtered in SQLite before cursor pagination and LIMIT. Authors
always see their own posts. For everyone else, a private author profile first
requires a current accepted follow. `followers` posts require that relation,
and `selected` posts additionally require the persisted audience row. Audience
rows survive unfollow, so access disappears immediately and returns after the
accepted relation is restored.

Every personal post response contains the post ID, safe author summary,
trimmed text, privacy, nullable `media_url`, `comments_count`, and creation
time. Personal feed, profile-post queries, selected audiences, and profile
post counts explicitly exclude group posts.
`GET /api/posts/{id}/media` rechecks the same current access policy on every
authenticated request. It returns `403` for an existing but forbidden post and
`404` for absent media, ownership mismatch, missing metadata, or a missing
physical file. Successful responses use the detected MIME, actual file length,
`X-Content-Type-Options: nosniff`, and `Cache-Control: private, no-store`.

## Post comments

`GET /api/posts/{id}/comments` lists comments in `(created_at ASC, id ASC)`
order. It uses an opaque cursor, defaults to `limit=20`, and enforces a maximum
of 50. `POST /api/posts/{id}/comments` accepts strict `application/json` with
exactly one `text` field. The request body is limited to 64 KiB; comment text is
trimmed and must contain 1–5000 Unicode code points. Unknown or duplicate JSON
fields and invalid values return `400`; an oversized body returns `413` and an
unsupported content type returns `415`.

Both endpoints resolve access by post kind. Personal posts use the current
profile/follow/audience policy; group posts require a current `owner` or
`member` membership. Existing inaccessible posts return `403`, while unknown
posts return `404`. Comments are created inside a SQL transaction, and a
failed commit leaves no comment row. Comment author summaries are mapped for
the current viewer, so group membership never exposes an inaccessible private
custom avatar. Editing, deletion, replies, likes, and realtime delivery are
outside this step.

## Groups

Groups are discoverable by every authenticated user. `POST /api/groups`
accepts strict JSON with exactly one `title` and one `description`; both values
are trimmed and required. Titles contain at most 100 Unicode code points and
descriptions at most 2000. The request body is limited to 64 KiB. Group creation
and the creator's `owner` membership are committed in one SQLite transaction.

`group_memberships` stores one state per group/user pair: `owner`, `member`,
`invited`, or `requested`. The external `viewer_status` additionally uses
`none`. Membership changes use conditional updates or deletes against the
expected current state. A repeated or stale transition returns `409`; owner-only
management attempted by another user returns `403`. Owners cannot leave because
group deletion and ownership transfer are outside this step.

The group catalog is ordered by `(created_at DESC, id DESC)`. Members are
ordered by `(owner_rank ASC, updated_at ASC, user_id ASC)`, with the owner first.
Join requests and sent invitations use `(created_at ASC, user_id ASC)`. The
current user's invitation inbox uses `(created_at ASC, group_id ASC)`. All lists
use opaque cursors, default to 20 items, and enforce a maximum of 50.

Catalog, group detail, and member lists are available to every authenticated
user. Join requests and the sent-invitation list are owner-only; both owners and
members may send an invitation. Invitations may target any existing user;
follow relationships are irrelevant. Group membership does not expand access
to private custom avatars: safe owner/member summaries emit a static gender
placeholder whenever the viewer could not open the controlled avatar route
under the existing profile privacy rules.

`POST /api/groups/{id}/posts` creates a group post from strict multipart data:
exactly one non-empty `text` field and at most one JPEG, PNG, GIF, or WebP
`media` file up to 20 MB. Privacy, selected audiences, duplicate scalar fields,
and unknown fields are rejected. `GET /api/groups/{id}/posts` uses the shared
opaque post cursor and `(created_at DESC, id DESC)` order. Group responses
contain `group_id` and omit `privacy`; personal responses do the reverse.
Creation, listing, comments, and `/api/posts/{postID}/media` all require a
current owner/member membership. Leaving immediately removes access, including
for the post author, while rejoining restores the existing history.

`GET|POST /api/groups/{id}/events` lists or creates group events for current
owners and members. Create accepts strict JSON containing `title`,
`description`, and RFC3339 `starts_at`; text is trimmed and limited to 100 and
2000 Unicode code points respectively, and the start must be in the future at
the service transaction's current time. Event times are normalized to UTC.
Lists use `(starts_at ASC, id ASC)`, opaque cursors, a default limit of 20, and
a maximum of 50.

`PUT /api/groups/{id}/events/{eventID}/response` accepts exactly one
`response`, either `going` or `not_going`, and returns the complete
authoritative event with current counts and the viewer's response. The UPSERT,
membership check, event/group check, count read, and commit form one SQL
transaction. RSVP rows survive leave, but counts include only current owners
and members; rejoining therefore restores the previous response. Events also
survive their creator leaving. Creator summaries remain viewer-aware, so group
membership alone never exposes a private custom avatar.

The frontend uses the group endpoints for creation, discovery, invitations,
join requests, owner management, leaving, and the real Group Posts and Events tabs. Its
group directory has a global request generation, group
detail/members/requests/sent invitations use a per-group generation, group
posts and events have additional request generations, RSVP uses a separate
per-event generation, and the invitation inbox has a separate generation.
Successful mutations invalidate older reads and apply the
returned group as the authoritative state. A leave or realtime `chat:remove`
revokes the complete local group access, hides member-only actions and chat,
purges posts/comments/events/drafts, and makes pending detail/member/content responses
stale. It also invalidates pending chat-list requests, filters revoked group
chats from later list responses, and blocks opening a stale group-chat card.
Only an authoritative owner/member response from rejoin can clear that revoke
state and trigger a fresh content load. Event creation now persists one
notification for every current owner/member except the creator; RSVP itself
does not create a notification.
Group chat uses the realtime implementation described below.

## Notifications

Notifications are persisted in SQLite for follow starts and requests, group
invitations and join requests, and group events. Each user has a monotonic
notification revision and an unread count derived from persisted `read_at`
values. `GET /api/notifications` orders rows by `(created_at DESC, id DESC)`,
uses an opaque cursor, defaults to 20 rows, and allows at most 50. Mark-one and
mark-all operations are idempotent and increase the revision only when stored
state actually changes.

`PUT /api/notifications/{id}/action` accepts strict JSON containing one
`action`, either `accept` or `decline`, for follow requests, group invitations,
and group join requests. Source transition, lifecycle validation,
notification resolution/read state, one revision increment, authoritative
source read, and unread count are committed in one transaction. The endpoint
uses the physical follow or membership lifecycle ID, so an old notification
cannot act on a newly created request. Repeating the same resolved action is a
`200` no-op; the opposite action, a stale lifecycle, or a non-actionable type
returns `409`.

Notification actions and the original follow/group endpoints share the same
transaction-bound transition helpers. Pending follow requests promoted after
a profile becomes public resolve the original request without creating a
second follow-started notification. Removing a follow clears the public
`follow_id`; that externally visible historical update increments the
recipient revision once. Membership lifecycle IDs remain backend-only.

After commit, `notification:upsert` and `notifications:read-all` are published
best-effort to every active socket of the recipient. SQLite remains the source
of truth and the frontend refreshes the first notification page after actions
and reconnects. Notification revision and authoritative relationship/group
source are race-gated independently, so an unrelated newer notification does
not suppress a valid source transition and a stale action cannot restore
access after leave, revoke, or a newer lifecycle.

## Chats and realtime

`GET /api/chats` returns existing direct conversations and groups where the
viewer is currently an owner or member. It does not return possible DM peers.
Rows are ordered by activity descending, then direct chats before groups, then
the internal entity ID descending. Groups without messages use the current membership's
`updated_at` as activity, while a group with history uses its last message.

`GET /api/chats/direct/{userID}/messages` and
`GET /api/groups/{id}/chat/messages` return history in chronological display
order. Direct history remains available after unfollow when a conversation
already exists. Without an existing conversation, an eligible DM returns an
empty page and an ineligible DM returns `403`. Group history is available only
to current owners and members; rejoining restores the complete history. Chat
lists and histories use opaque cursors, default to 20 rows, and allow at most
50.

Chat unread state is persisted in SQLite. The chat list returns an
`unread_count` for every listed chat plus the authoritative user-level
`unread_count` and `revision`. Direct state belongs to the user/conversation
pair and survives unfollow. Group state belongs to the physical membership
lifecycle: leaving deletes it, while rejoining creates a fresh zero-unread
state through the latest existing group message without hiding older history.
`PUT /api/chats/direct/{userID}/read` and
`PUT /api/groups/{id}/chat/read` accept strict JSON with one positive
`through_message_id`. Markers advance by `(created_at, id)`; equal or older
markers are idempotent and do not bump the revision.

Live messages use the authenticated same-origin `GET /ws` WebSocket. Client
events are `chat:send`, `typing:start`, `typing:heartbeat`, and `typing:stop`;
server events are `presence:init`, `presence:update`, `presence:remove`,
`typing:update`, `chat:message`, `chat:unread`, `chat:remove`, `chat:error`,
`notification:upsert`, and `notifications:read-all`. `chat:remove`
immediately purges a group chat from every active tab when membership access is
revoked. Frames are limited to 16 KiB and message text is trimmed and limited
to 1–2000 Unicode code points. A user
may have up to eight active sockets. Presence and direct typing are visible
only while an accepted follow exists in either direction. Group sends, typing,
and history require current owner/member access.

WebSocket presence bootstrap is versioned around the peer query. Every
relationship mutation advances the Hub generation for both users, so
a snapshot read before that mutation cannot overwrite newer realtime access
state when the socket registers.

Every send carries a UUID `client_message_id`. SQLite enforces uniqueness per
sender, so retrying the same canonical target and body returns the existing
message without rebroadcasting it; reusing the UUID for different content
returns a conflict. The frontend keeps an optimistic bubble for 15 seconds and
retries with the same UUID. HTTP history, reconnect responses, and WebSocket
responses deduplicate by server/message IDs.

For a newly persisted message, every eligible recipient socket receives
`chat:message` followed by its personalized `chat:unread` state in the same
Hub completion. A revoked sender session suppresses both events. The frontend
marks a chat read only while Messages is the active screen, that conversation
is active, its authoritative history is displayed, and the browser tab is
visible. Bootstrap, reconnect, background preload, and older-history
pagination never mark messages read.

The Hub stores only `SHA-256(raw session token)` as its session identity. The
raw token stays in the connection read loop and is checked against `sessions`
inside each send/typing SQL transaction. Logout deletes the SQLite session and
synchronously revokes that one Hub session before returning `204`; other
browser sessions remain connected. Lease completion and revocation are ordered
inside the Hub event loop, so a completion processed after revocation cannot
enqueue an ack or broadcast. Current unfollow and group-leave hooks also remove
presence/typing and suppress stale already-authorized realtime delivery.

On shutdown, HTTP mutation/upgrade admission closes first, the Hub enters drain
mode and rejects new operations, the HTTP server shuts down, and SQLite closes
only after HTTP completion and `Hub.Done()`. Existing leases are canceled and
must complete; a deadline forces final socket shutdown without treating an
ordinary tab disconnect as session revocation.

Supported environment variables:

- `SOCIAL_NETWORK_HTTP_ADDR`
- `SOCIAL_NETWORK_DB_PATH`
- `SOCIAL_NETWORK_UPLOAD_DIR`
- `SOCIAL_NETWORK_FRONTEND_DIR` (local static frontend path; default `../frontend`)
- `SOCIAL_NETWORK_COOKIE_SECURE`

Implemented endpoints:

- `GET /api/health`
- `POST /api/auth/register` (`multipart/form-data`)
- `POST /api/auth/login` (JSON)
- `POST /api/auth/logout` (idempotent `204`; storage failures return `500`)
- `GET /api/auth/me` (authenticated)
- `PATCH /api/profile` (authenticated partial JSON update)
- `PUT /api/profile/avatar` (authenticated multipart upload, field name `avatar`)
- `DELETE /api/profile/avatar` (authenticated, idempotent)
- `GET /api/users` (authenticated cursor-paginated directory)
- `GET /api/users/{id}` (authenticated privacy-aware profile card)
- `GET|PUT|DELETE /api/users/{id}/follow` (relationship, follow, unfollow)
- `GET /api/users/{id}/followers`
- `GET /api/users/{id}/following`
- `GET /api/users/{id}/avatar` (authenticated and privacy-controlled)
- `GET /api/users/{id}/posts` (authenticated, privacy-filtered cursor pagination)
- `GET /api/follow-requests`
- `POST /api/follow-requests/{id}/accept`
- `DELETE /api/follow-requests/{id}`
- `GET /ws` (authenticated WebSocket)
- `POST /api/media` (authenticated multipart upload, field name `file`)
- `GET /uploads/{id}` (authenticated, owner-only)
- `POST /api/posts` (authenticated multipart post creation)
- `GET /api/posts/feed` (authenticated cursor feed)
- `GET /api/posts/{id}/media` (authenticated and privacy-controlled)
- `GET /api/posts/{id}/comments` (authenticated privacy-controlled pagination)
- `POST /api/posts/{id}/comments` (authenticated strict JSON creation)
- `GET|POST /api/groups` (authenticated cursor catalog and strict JSON creation)
- `GET /api/groups/{id}`
- `GET /api/groups/{id}/members`
- `POST|DELETE /api/groups/{id}/join-request`
- `GET /api/groups/{id}/join-requests` (owner-only)
- `POST /api/groups/{id}/join-requests/{userID}/accept` (owner-only)
- `DELETE /api/groups/{id}/join-requests/{userID}` (owner-only)
- `GET|POST /api/groups/{id}/invitations` (owner-only)
- `POST /api/groups/{id}/invitation/accept`
- `DELETE /api/groups/{id}/invitation`
- `DELETE /api/groups/{id}/membership`
- `GET|POST /api/groups/{id}/events` (owner/member event list and creation)
- `PUT /api/groups/{id}/events/{eventID}/response` (owner/member RSVP UPSERT)
- `GET /api/group-invitations`
- `GET /api/notifications` (authenticated cursor pagination, unread count and revision)
- `PUT /api/notifications/{id}/read`
- `PUT /api/notifications/read-all`
- `PUT /api/notifications/{id}/action` (strict actionable source transition)
- `GET /api/chats` (authenticated cursor-paginated chat list)
- `GET /api/chats/direct/{userID}/messages` (authenticated direct history)
- `PUT /api/chats/direct/{userID}/read` (strict persisted direct read marker)
- `GET /api/groups/{id}/chat/messages` (authenticated owner/member history)
- `PUT /api/groups/{id}/chat/read` (strict persisted group read marker)
- `GET /static/avatars/{male,female,neutral}.svg`
- `GET /` and frontend assets (local development and browser smoke only)

All other reserved API groups currently return JSON `501 Not Implemented`.

The frontend feed, profiles, suggestions, follow controls, persisted notifications, follower
lists, following lists, profile posts, post comments, direct chats, and group
chats use backend IDs and live API state. Chat unread badges and read markers
are persisted and revision-gated. Chat history paginates upward,
reconnect reloads list/current history, optimistic sends are retryable, and
stale responses are generation-gated across logout and access changes.
Group-dependent mock posts, comments, and right-rail events have been removed;
the Group Events tab uses persisted backend events and RSVP, and the
Notifications screen uses persisted history, unread state, actions, pagination,
and realtime refresh hints.

The local frontend file server does not replace the planned Docker topology.
The final setup keeps the backend private and serves the frontend through a
separate frontend/reverse-proxy container; the backend static handler is only a
development convenience and does not need to be used there.
