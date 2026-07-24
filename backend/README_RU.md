# Social Network Backend

Backend социальной сети на Go и SQLite.

Этот файл описывает только backend: архитектуру, persistence, authorization, API-контракты, транзакционную работу с media, notifications и realtime. Обзор продукта, Docker quick start и full-stack структура находятся в [корневом README](../README_RU.md).

[English version](README.md) · [Документация frontend](../frontend/README_RU.md)

## 📋 Оглавление

- [🚀 Локальный запуск](#-локальный-запуск)
- [🏗️ Архитектура](#️-архитектура)
- [📦 Зависимости](#-зависимости)
- [💾 SQLite и migrations](#-sqlite-и-migrations)
- [🔑 Authentication и sessions](#-authentication-и-sessions)
- [👤 Profiles и follows](#-profiles-и-follows)
- [🔐 Authorization](#-authorization)
- [🖼️ Транзакционная работа с media](#️-транзакционная-работа-с-media)
- [📝 Posts и comments](#-posts-и-comments)
- [👥 Groups и events](#-groups-и-events)
- [🔔 Notifications](#-notifications)
- [💬 Chats и WebSocket](#-chats-и-websocket)
- [🌐 HTTP behavior](#-http-behavior)
- [🧪 Проверка](#-проверка)
- [📁 Структура](#-структура)

## 🚀 Локальный запуск

SQLite driver использует CGO, поэтому требуется рабочий C compiler.

```bash
go run ./cmd/server
```

Default address:

```text
http://127.0.0.1:8080
```

При local development Go server также обслуживает `../frontend`, поэтому HTTP, WebSocket, static assets и session cookie остаются same-origin.

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

Docker-развёртывание находится в [корневом README](../README_RU.md).

## 🏗️ Архитектура

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

| Package | Назначение |
|---|---|
| `internal/app` | dependency wiring, lifecycle server, graceful shutdown |
| `internal/config` | environment parsing и defaults |
| `internal/domain` | entities и enums |
| `internal/http` | routing, auth middleware, strict parsing, responses |
| `internal/service` | business rules, authorization, transactions |
| `internal/repo` | repository и transaction interfaces |
| `internal/repo/sqlite` | SQLite implementations и embedded migrations |
| `internal/realtime/ws` | authenticated sockets, presence, typing, delivery |
| `internal/platform` | clocks и ID generation |

Startup применяет pending migrations до начала прослушивания. Shutdown закрывает mutations и WebSocket upgrades, переводит Hub в drain, останавливает HTTP и последней закрывает SQLite.

## 📦 Зависимости

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

## 💾 SQLite и migrations

SQLite открывается с:

```text
PRAGMA foreign_keys = ON
PRAGMA busy_timeout = 5000
```

Используется одно database connection, чтобы connection-scoped PRAGMAs оставались последовательными и не возникала лишняя конкуренция SQLite writers.

Migrations embedded из:

```text
internal/repo/sqlite/migrations
```

Текущие версии:

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

Проверка migration state:

```bash
sqlite3 var/social-network.db \
  "SELECT version, dirty FROM schema_migrations;"
```

Ожидаемое состояние:

```text
15 | 0
```

Migration `000011` при перестройке `posts` сохраняет post/comment IDs, timestamps, selected audiences, media links и реальные high-water values в `sqlite_sequence`. Down migration запрещена, пока существуют group posts.

Migration `000015` добавляет optional comment media без перестройки `post_comments`. Down migration запрещена, пока есть comment attachments. Оба guard выполняются до schema changes, поэтому база не остаётся dirty.

## 🔑 Authentication и sessions

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

- `date_of_birth` использует строгий `DD-MM-YYYY`;
- password учитывает bcrypt limit 72 bytes;
- avatar optional, JPEG/PNG/GIF/WebP, максимум 20 MB;
- type определяется по содержимому файла.

User, initial notification/chat state, optional avatar relation и session коммитятся вместе.

Login:

```text
POST /api/auth/login
Content-Type: application/json
```

Authentication использует HttpOnly, SameSite=Lax cookie:

```text
social_network_session
```

Разрешены несколько sessions на одного user. Logout удаляет только текущую session и синхронно отзывает её Hub access. Expired sessions удаляются при чтении. WebSocket operations повторно проверяют raw session token в SQLite.

## 👤 Profiles и follows

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

Patch строгий: unknown fields и empty object возвращают `400`; required identity fields нельзя очистить; `gender` принимает `male`, `female` или `null`.

Avatar:

```text
PUT    /api/profile/avatar
DELETE /api/profile/avatar
```

Failed replacement сохраняет old avatar и удаляет staged new file.

Profile read:

```text
GET /api/users/{id}
```

Public profile, owner и accepted follower получают всю разрешённую информацию, включая email. Outsider приватного профиля получает только safe card. Directory rows и embedded summaries никогда не содержат email.

Follow state:

```text
GET|PUT|DELETE /api/users/{id}/follow
```

- public profile: immediate accepted follow;
- private profile: pending request;
- delete idempotent;
- repeated follow не понижает accepted state.

Request management:

```text
GET    /api/follow-requests
POST   /api/follow-requests/{id}/accept
DELETE /api/follow-requests/{id}
```

## 🔐 Authorization

| Resource | Access rule |
|---|---|
| public profile | любой authenticated user |
| private profile | owner или accepted follower |
| followers post | текущий accepted follower |
| selected post | accepted follower и selected audience row |
| group content | текущий `owner` или `member` |
| group management lists | только owner |
| group invitation creation | owner или member |
| event и RSVP | owner или member |
| direct chat send | accepted follow хотя бы в одном направлении |
| existing direct history | участники conversation |
| group chat/history | current owner/member |
| protected media | current parent-object policy |

Инварианты:

- leave немедленно закрывает group content/chat;
- selected audience rows переживают unfollow, но access не сохраняется;
- membership в группе не раскрывает private profile;
- protected media авторизуется при каждом read;
- stale frontend state не даёт backend access.

## 🖼️ Транзакционная работа с media

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

Любая ошибка удаляет staged/finalized file и откатывает rows.

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

Success response содержит detected MIME, actual length, `X-Content-Type-Options: nosniff` и `Cache-Control: private, no-store`.

Generic `POST /api/media` и backend `/uploads/{id}` отсутствуют.

## 📝 Posts и comments

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

Text trim, valid UTF-8, от 1 до 5000 Unicode code points. `selected` требует от 1 до 100 current accepted followers.

Reads:

```text
GET /api/posts/feed
GET /api/users/{id}/posts
GET /api/posts/{id}/media
```

Opaque cursor, default 20, maximum 50.

Group posts:

```text
GET|POST /api/groups/{id}/posts
```

Требуется current owner/member.

Comments:

```text
GET|POST /api/posts/{id}/comments
GET      /api/comments/{id}/media
```

Create принимает только strict multipart с одним required `text` и максимум одним optional `media`. Media-only возвращает `400`, wrong content type `415`, oversized input `413`.

Comment media проверяет parent access до attachment existence. Inaccessible parent возвращает `403`; missing attachment, metadata, ownership или file возвращают `404` только после successful access.

## 👥 Groups и events

Membership states:

```text
owner
member
invited
requested
none
```

Stale или repeated transition возвращает `409`. Owner не может leave, потому что ownership transfer и group deletion не реализованы.

Main routes:

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

Join-request и sent-invitation management owner-only. Invitation может отправить owner или member, follow relation не нужна.

Events:

```text
GET|POST /api/groups/{id}/events
PUT      /api/groups/{id}/events/{eventID}/response
```

Create требует title, description и future RFC3339 `starts_at`. RSVP принимает `going` или `not_going`. Membership check, UPSERT, counts и authoritative reread выполняются в одной transaction.

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

У user есть monotonic revision и unread count из persisted `read_at`.

Actions `accept`/`decline` поддерживаются для follow requests, group invitations и group join requests. Source transition, lifecycle validation, resolution, read state, revision и authoritative reread коммитятся вместе.

Old notification не может изменить новый request lifecycle. Повтор same action возвращает `200`, opposite или stale action `409`.

## 💬 Chats и WebSocket

HTTP:

```text
GET /api/chats
GET /api/chats/direct/{userID}/messages
GET /api/groups/{id}/chat/messages
PUT /api/chats/direct/{userID}/read
PUT /api/groups/{id}/chat/read
```

Chat unread persisted. Read markers только растут; equal/older marker idempotent.

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
message:                 1-2000 Unicode code points
sockets per user:        8
```

Каждый send имеет UUID `client_message_id`. Retry с теми же target/body возвращает persisted message без повторного broadcast. UUID с другим content возвращает `409`.

Hub хранит только `SHA-256(raw session token)`. Logout, unfollow и group leave обновляют realtime eligibility и блокируют stale delivery.

## 🌐 HTTP behavior

- `401`: invalid/missing session;
- `400`: malformed ID или invalid input;
- `403`: existing forbidden resource;
- `404`: absent или intentionally hidden resource;
- `405`: wrong method с `Allow`;
- `409`: stale/conflicting transition;
- `413`: oversized body/file;
- `415`: unsupported content type;
- `500`: storage/unexpected failure;
- unknown `/api/*`: JSON `404`.

## 🧪 Проверка

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

Покрываются migrations, sessions, profile privacy, follows, post access, media rollback, groups, events, notification lifecycle, chat unread, WebSocket delivery, session revoke, shutdown races, strict HTTP contracts и pagination.

Accepted result:

```text
go test -count=1 ./...        passed
go test -race -count=1 ./...  passed
go vet ./...                  passed
```

## 📁 Структура

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
