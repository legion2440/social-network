# Social Network Backend

Go and SQLite backend for the Social Network application.

This file documents backend-specific architecture, persistence, authorization, API contracts, media transactions, notifications, and realtime behavior. For the product overview, Docker quick start, and full-stack structure, see the [root README](../README.md).

[Русская версия](README_RU.md) · [Frontend documentation](../frontend/README.md)

## 📋 TOC

- [🚀 Local run](#-local-run)
- [🏗️ Architecture](#️-architecture)
- [📦 Dependencies](#-dependencies)
- [💾 SQLite and migrations](#-sqlite-and-migrations)
- [🔑 Authentication and sessions](#-authentication-and-sessions)
- [👤 Profiles and followers](#-profiles-and-followers)
- [🔐 Authorization](#-authorization)
- [🖼️ Transactional media](#️-transactional-media)
- [📝 Posts and comments](#-posts-and-comments)
- [👥 Groups and events](#-groups-and-events)
- [🔔 Notifications](#-notifications)
- [💬 Chats and WebSocket](#-chats-and-websocket)
- [🌐 HTTP behavior](#-http-behavior)
- [🧪 Verification](#-verification)
- [📁 Structure](#-structure)

## 🚀 Local run

The SQLite driver uses CGO and requires a working C compiler.

```bash
go run ./cmd/server
```

Default address:

```text
http://127.0.0.1:8080
```

During local development the Go server also serves `../frontend`, keeping HTTP, WebSocket, static assets, and the session cookie same-origin.

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

Docker deployment is documented in the [root README](../README.md).

## 🏗️ Architecture

```text
cmd/server
   |
   v
internal/app
   |
   +--> config
   +--> SQLite + migrations
   +--> repositories
   +--> services
   +--> HTTP router
   +--> WebSocket Hub
```

| Package | Responsibility |
|---|---|
| `internal/app` | dependency wiring, server lifecycle, graceful shutdown |
| `internal/config` | environment parsing and defaults |
| `internal/domain` | entities and enums |
| `internal/http` | routing, auth middleware, strict parsing, responses |
| `internal/service` | business rules, authorization, transactions |
| `internal/repo` | repository and transaction interfaces |
| `internal/repo/sqlite` | SQLite implementations and embedded migrations |
| `internal/realtime/ws` | authenticated sockets, presence, typing, delivery |
| `internal/platform` | clocks and ID generation |

Startup applies migrations before listening. Shutdown closes mutation and WebSocket-upgrade admission, drains the Hub, stops HTTP, then closes SQLite.

## 📦 Dependencies

```text
github.com/golang-migrate/migrate/v4
github.com/google/uuid
github.com/gorilla/websocket
github.com/mattn/go-sqlite3
golang.org/x/crypto
```

Module:

```text
module social-network/backend
go 1.24.2
```

## 💾 SQLite and migrations

SQLite opens with:

```text
PRAGMA foreign_keys = ON
PRAGMA busy_timeout = 5000
```

One database connection keeps connection-scoped PRAGMAs consistent and reduces avoidable SQLite writer contention.

Migrations are embedded from:

```text
internal/repo/sqlite/migrations
```

Current versions:

```text
000001_create_users
000002_create_media
000003_add_user_avatar_media
000004_create_sessions
000005_add_user_privacy
000006_create_follows
000007_create_posts
000008_create_post_comments
000009_create_groups
000010_create_chats
000011_add_group_posts
000012_create_group_events
000013_create_notifications
000014_create_chat_read_states
000015_add_comment_media
```

Inspect the local migration state:

```bash
sqlite3 var/social-network.db \
  "SELECT version, dirty FROM schema_migrations;"
```

Expected:

```text
15 | 0
```

Migration `000011` preserves post/comment IDs, timestamps, selected audiences, media links, and real `sqlite_sequence` high-water values while rebuilding `posts`. Its down migration is rejected while group posts exist.

Migration `000015` adds optional comment media without rebuilding `post_comments`. Its down migration is rejected while comment attachments exist. Both guards run before schema changes so the database is not left dirty.

## 🔑 Authentication and sessions

Registration:

```text
POST /api/auth/register
Content-Type: multipart/form-data
```

Required:

```text
email
password
first_name
last_name
date_of_birth
```

Optional:

```text
gender
nickname
about_me
avatar
```

Rules:

- `date_of_birth` is strict `DD-MM-YYYY`;
- password length respects bcrypt's 72-byte limit;
- avatar is optional JPEG, PNG, GIF, or WebP up to 20 MB;
- media type is detected from content.

User creation, initial notification/chat state, optional avatar relation, and session creation are committed together.

Login:

```text
POST /api/auth/login
Content-Type: application/json
```

Authentication uses the HttpOnly, SameSite=Lax cookie:

```text
social_network_session
```

Multiple sessions per user are allowed. Logout removes only the current session and synchronously revokes its Hub access. Expired sessions are deleted when read. WebSocket operations recheck the raw session token against SQLite.

## 👤 Profiles and followers

Own profile update:

```text
PATCH /api/profile
```

Supported fields:

```text
first_name
last_name
date_of_birth
gender
nickname
about_me
is_private
```

The patch is strict: unknown fields and empty objects return `400`; required identity fields cannot be cleared; `gender` accepts `male`, `female`, or `null`.

Avatar:

```text
PUT    /api/profile/avatar
DELETE /api/profile/avatar
```

A failed replacement preserves the old avatar and removes the staged new file.

Profile read:

```text
GET /api/users/{id}
```

Public profiles, owners, and accepted followers receive full permitted profile data, including email. Private-profile outsiders receive only a safe card. Directory rows and embedded user summaries never contain email.

Follow state:

```text
GET|PUT|DELETE /api/users/{id}/follow
```

- public profile: immediate accepted follow;
- private profile: pending request;
- delete is idempotent;
- repeated follow never demotes accepted state.

Request management:

```text
GET    /api/follow-requests
POST   /api/follow-requests/{id}/accept
DELETE /api/follow-requests/{id}
```

## 🔐 Authorization

| Resource | Access rule |
|---|---|
| public profile | every authenticated user |
| private profile | owner or accepted follower |
| followers post | current accepted follower |
| selected post | accepted follower plus selected audience row |
| group content | current `owner` or `member` |
| group management lists | owner only |
| group invitation creation | owner or member |
| event and RSVP | owner or member |
| direct chat send | accepted follow in at least one direction |
| existing direct history | conversation participants |
| group chat/history | current owner or member |
| protected media | current parent-object policy |

Important invariants:

- leaving a group immediately revokes its content and chat;
- selected audience rows survive unfollow, but access does not;
- group membership never reveals an otherwise private profile;
- protected media is authorized on every read;
- stale frontend state cannot grant backend access.

## 🖼️ Transactional media

Upload lifecycle:

```text
strict parse
-> stage file
-> transaction
-> current access check
-> media row
-> ownership check
-> parent relation
-> finalize file
-> commit
-> keep
```

Any failure removes the staged or finalized file and rolls back rows.

Accepted media:

```text
JPEG
PNG
GIF
WebP
maximum 20 MB
```

Controlled routes:

```text
GET /api/users/{id}/avatar
GET /api/posts/{id}/media
GET /api/comments/{id}/media
```

Successful responses include the detected MIME, actual length, `X-Content-Type-Options: nosniff`, and `Cache-Control: private, no-store`.

There is no generic `POST /api/media` or backend `/uploads/{id}` route.

## 📝 Posts and comments

Personal post create:

```text
POST /api/posts
Content-Type: multipart/form-data
```

Required:

```text
text
privacy
```

Privacy:

```text
public
followers
selected
```

Text is trimmed, valid UTF-8, and 1 to 5000 Unicode code points. `selected` requires 1 to 100 current accepted followers.

Reads:

```text
GET /api/posts/feed
GET /api/users/{id}/posts
GET /api/posts/{id}/media
```

Pagination uses opaque cursors, default 20, maximum 50.

Group posts:

```text
GET|POST /api/groups/{id}/posts
```

Current owner/member access is required.

Comments:

```text
GET|POST /api/posts/{id}/comments
GET      /api/comments/{id}/media
```

Create accepts only strict multipart with one required `text` and at most one optional `media`. Media-only comments return `400`, wrong content type returns `415`, and oversized input returns `413`.

Comment media checks parent access before checking whether an attachment exists. An inaccessible parent returns `403`; missing attachment, metadata, ownership, or file returns `404` only after access succeeds.

## 👥 Groups and events

Membership states:

```text
owner
member
invited
requested
none
```

Stale or repeated transitions return `409`. Owners cannot leave because ownership transfer and group deletion are not implemented.

Main group routes:

```text
GET|POST /api/groups
GET      /api/groups/{id}
GET      /api/groups/{id}/members

POST|DELETE /api/groups/{id}/join-request
GET         /api/groups/{id}/join-requests
POST        /api/groups/{id}/join-requests/{userID}/accept
DELETE      /api/groups/{id}/join-requests/{userID}

GET|POST /api/groups/{id}/invitations
POST     /api/groups/{id}/invitation/accept
DELETE   /api/groups/{id}/invitation
GET      /api/group-invitations

DELETE /api/groups/{id}/membership
```

Join-request and sent-invitation management is owner-only. Invitation creation is available to owners and members and does not require a follow relation.

Events:

```text
GET|POST /api/groups/{id}/events
PUT      /api/groups/{id}/events/{eventID}/response
```

Create requires title, description, and future RFC3339 `starts_at`. RSVP accepts `going` or `not_going`. Membership check, UPSERT, counts, and authoritative reread occur in one transaction.

## 🔔 Notifications

Persisted types:

```text
follow_started
follow_request
group_invitation
group_join_request
group_event
```

Endpoints:

```text
GET /api/notifications
PUT /api/notifications/{id}/read
PUT /api/notifications/read-all
PUT /api/notifications/{id}/action
```

Each user has a monotonic revision and unread count derived from persisted `read_at`.

Actions support `accept` and `decline` for follow requests, group invitations, and group join requests. Source transition, lifecycle validation, resolution, read state, revision, and authoritative source reread are committed together.

Old notifications cannot mutate a newer request lifecycle. Repeating the same completed action is a `200` no-op; opposite or stale actions return `409`.

## 💬 Chats and WebSocket

HTTP:

```text
GET /api/chats
GET /api/chats/direct/{userID}/messages
GET /api/groups/{id}/chat/messages
PUT /api/chats/direct/{userID}/read
PUT /api/groups/{id}/chat/read
```

Chat unread state is persisted. Read markers only advance; equal or older markers are idempotent.

WebSocket:

```text
GET /ws
```

Client events:

```text
chat:send
typing:start
typing:heartbeat
typing:stop
```

Server events:

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

Limits:

```text
frame:                   16 KiB
message:                 1 to 2000 Unicode code points
sockets per user:        8
```

Every send has a UUID `client_message_id`. Retrying the same target and body returns the persisted message without rebroadcasting. Reusing the UUID for different content returns `409`.

The Hub stores only `SHA-256(raw session token)`. Logout, unfollow, and group leave update realtime eligibility and suppress stale delivery.

## 🌐 HTTP behavior

- `401`: missing or invalid session;
- `400`: malformed ID or invalid input;
- `403`: existing but forbidden resource;
- `404`: absent or intentionally hidden resource;
- `405`: wrong method with `Allow`;
- `409`: stale or conflicting transition;
- `413`: oversized body or file;
- `415`: unsupported content type;
- `500`: storage or unexpected failure;
- unknown `/api/*`: JSON `404`.

## 🧪 Verification

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

Coverage includes migrations, sessions, profile privacy, follows, post access, media rollback, groups, events, notification lifecycle, chat unread, WebSocket delivery, session revoke, shutdown races, strict HTTP contracts, and pagination.

Accepted result:

```text
go test -count=1 ./...        passed
go test -race -count=1 ./...  passed
go vet ./...                  passed
```

## 📁 Structure

```text
backend/
├── cmd/
│   ├── healthcheck/
│   └── server/
├── internal/
│   ├── app/
│   ├── config/
│   ├── domain/
│   ├── http/
│   ├── platform/
│   ├── realtime/ws/
│   ├── repo/sqlite/migrations/
│   └── service/
├── Dockerfile
├── go.mod
├── go.sum
├── README.md
└── README_RU.md
```
