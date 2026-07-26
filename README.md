# Social Network

A full-stack Facebook-like social network built with Go, SQLite, WebSocket, a custom declarative JavaScript framework, and a two-container Docker deployment.

The application supports public and private profiles, followers, audience-controlled posts, media attachments, groups, events, persisted notifications, direct and group chats, and unread state.

· [Русская версия](README_RU.md) 
· [Backend documentation](backend/README.md) 
· [Frontend documentation](frontend/README.md)

## 📋 TOC

- [🚀 Quick start](#-quick-start)
- [📝 About](#-about)
- [✨ Features](#-features)
- [🏗️ Architecture](#️-architecture)
- [🧰 Technology stack](#-technology-stack)
- [🔐 Security and access control](#-security-and-access-control)
- [🐳 Docker deployment](#-docker-deployment)
- [💾 Persistence and migrations](#-persistence-and-migrations)
- [⚙️ Configuration](#️-configuration)
- [🧪 Local development and tests](#-local-development-and-tests)
- [📁 Project structure](#-project-structure)
- [📚 API documentation](#-api-documentation)
- [🧹 Cleanup](#-cleanup)
- [⚠️ Notes](#️-notes)
- [🧑‍💻 Author](#-author)

## 🚀 Quick start

### Prerequisites

- Docker Engine
- Docker Compose plugin
- port `8080` available, or another port selected through `SOCIAL_NETWORK_PORT`

### Run with Docker Compose

```bash
git clone https://01.tomorrow-school.ai/git/nyestaye/social-network
cd social-network

docker compose config
docker compose up --build -d
docker compose ps
```

Open:

```text
http://127.0.0.1:8080
```

To use another host port:

```bash
SOCIAL_NETWORK_PORT=8081 docker compose up --build -d
```

Stop the stack while preserving the database and uploaded files:

```bash
docker compose down --remove-orphans
```

## 📝 About

Social Network is a complete educational web application that implements the core behavior of a modern social platform.

The backend owns authentication, privacy, access control, persistence, migrations, media metadata, notification state, unread counters, and WebSocket authorization. The frontend is a static single-page application served by Caddy and communicates with the backend through same-origin HTTP and WebSocket routes.

The project deliberately keeps the backend private inside the Docker network. Only the frontend container publishes a host port.

## ✨ Features

### Authentication and profiles

- registration with email, password, first name, last name, and date of birth;
- optional GitHub sign-in and registration through a verified GitHub email;
- optional avatar, gender, nickname, and about text;
- HttpOnly session cookie with multiple independent sessions per user;
- login, persistent session restore, and logout available from every application page;
- editable profile data and avatar replacement/removal;
- public and private profile modes;
- controlled profile visibility for private accounts.

### Followers

- immediate follow for public profiles;
- follow requests for private profiles;
- accept and decline actions;
- unfollow and pending-request cancellation;
- followers and following lists with viewer-aware relationship state.

### Posts and comments

- personal posts with three privacy modes:
  - `public`;
  - `followers`;
  - `selected`;
- selected-audience posts restricted to chosen accepted followers;
- JPEG, PNG, GIF, and WebP attachments up to 20 MB;
- text-or-media creation for personal posts, group posts, and comments;
- paginated personal Home feed and viewer-aware profile activity;
- public posts remain visible in Home even when their author has a private profile;
- group posts appear only to active members and are labeled with their group in profiles;
- comments with optional JPEG, PNG, GIF, or WebP attachments;
- current access is rechecked whenever protected post or comment media is requested.

### Groups

- group creation with title and description;
- group directory and membership state;
- invitations from the owner or any active member;
- join requests accepted or declined by the group owner;
- invitation acceptance and rejection;
- member-only group posts and comments;
- group events with title, description, date/time, and RSVP:
  - Going;
  - Not going.

### Realtime chat

- direct chats when at least one accepted follow relation exists;
- member-only group chat;
- realtime message delivery through WebSocket;
- persisted direct and group history;
- emoji messages;
- typing and presence state;
- persisted per-chat unread counters;
- session revocation closes related realtime access.

### Notifications

Persisted notifications are created for the required social events:

- private-profile follow request;
- group invitation;
- group join request;
- group event creation.
- accepted follow (`follow_started`, implemented bonus).

Notifications support:

- realtime delivery;
- unread counts;
- mark one as read;
- mark all as read;
- accept/decline actions for actionable notifications;
- idempotent lifecycle handling.

### Media

- content-based JPEG, PNG, GIF, and WebP validation;
- 20 MB upload limit;
- transactional media metadata and relation creation;
- staged files removed when a transaction fails;
- authenticated avatar delivery and access-controlled post/comment media routes;
- no generic public upload directory.

## 🏗️ Architecture

```text
browser
  |
  v
social-network-frontend:8080
  Caddy, public container
  |
  +--> static SPA
  |
  +--> /api, /ws, /static/avatars
         |
         v
       social-network-backend:8080
         Go, private container
         |
         +--> SQLite
         +--> uploaded files
         +--> migrations
         +--> WebSocket hub
```

Only the frontend port is published.

Caddy routes exact and nested paths:

```text
/api
/api/*
/ws
/ws/*
/static/avatars
/static/avatars/*
```

Removed legacy paths return a direct `404` before the SPA fallback:

```text
/uploads
/uploads/*
```

Every other path is served from `/srv`, with `index.html` used for client-side routing.

## 🧰 Technology stack

| Layer                         | Technology                            |
|-------------------------------|---------------------------------------|
| Backend                       | Go `1.24.2`                           |
| Database                      | SQLite with `mattn/go-sqlite3`        |
| Migrations                    | `golang-migrate`                      |
| Authentication                | server-side sessions, cookies, bcrypt |
| Realtime                      | Gorilla WebSocket                     |
| IDs                           | Google UUID                           |
| Frontend                      | custom declarative `dc-runtime`       |
| Rendering layer               | React and ReactDOM                    |
| Static server / reverse proxy | Caddy                                 |
| Containers                    | Docker and Docker Compose             |

The browser application is not organized as a conventional React component tree. The `dc-runtime` application framework processes the build-time-composed `<x-dc>` template, while React and ReactDOM provide only the rendering layer. The root owns one shared state and application lifecycle; feature controllers contain actions and complete pure template view models, using narrow state/API/model/gate/helper/callback dependencies without access to the root component. Home, profile, and group posts share one pure presenter, and the root merges controller view models in a fixed order with duplicate-key detection. Production source lives under `frontend/src` and is generated into ignored `frontend/dist`.

## 🔐 Security and access control

- passwords are stored as bcrypt hashes;
- GitHub OAuth state and registration flows are random, server-side, one-time, time-limited, and bound to one browser through a separate HttpOnly nonce cookie; only its SHA-256 hash is persisted;
- GitHub email is accepted only from `/user/emails` when verified; provider access tokens are never persisted;
- OAuth registration completion requires exact same-origin `Origin`/Fetch Metadata and preserves only a validated relative `next` route;
- session tokens are transported through an HttpOnly, SameSite=Lax cookie;
- the backend is not published to the host in Docker;
- private-profile fields and content are returned only to the owner and accepted followers; custom avatars remain visible to authenticated users;
- post, comment, group, event, and chat access is checked on the backend;
- post and comment media access is revalidated against the current parent object policy;
- media MIME type is detected from file content, not trusted from the filename;
- successful protected media responses use `X-Content-Type-Options: nosniff`;
- Docker runtime processes use numeric non-root users;
- documented containers use read-only root filesystems and `no-new-privileges`;
- graceful shutdown stops new realtime work, drains WebSocket operations, and closes the HTTP server.

The local HTTP configuration uses:

```text
SOCIAL_NETWORK_COOKIE_SECURE=false
```

Set it to `true` when deploying behind HTTPS.

## 🐳 Docker deployment

The project produces two images:

```text
social-network-backend:local
social-network-frontend:local
```

Both images use pinned official base-image digests.

### Build images manually

Run from the repository root:

```bash
./scripts/build-images.sh
```

PowerShell:

```powershell
.\scripts\build-images.ps1
```

### Standalone launch

Create the private network and persistent named volumes once:

```bash
docker network create social-network
docker volume create social-network-db
docker volume create social-network-uploads
```

Start the private backend:

```bash
docker container run -d \
  --name social-network-backend \
  --network social-network \
  --read-only \
  --tmpfs /tmp \
  --security-opt no-new-privileges \
  --mount type=volume,source=social-network-db,target=/data/db \
  --mount type=volume,source=social-network-uploads,target=/data/uploads \
  social-network-backend:local
```

Check backend health:

```bash
docker inspect --format='{{json .State.Health}}' social-network-backend
```

Start the public frontend:

```bash
docker container run -d \
  --name social-network-frontend \
  --network social-network \
  --read-only \
  --tmpfs /tmp \
  --security-opt no-new-privileges \
  -e BACKEND_UPSTREAM=social-network-backend:8080 \
  -p "${SOCIAL_NETWORK_PORT:-8080}:8080" \
  social-network-frontend:local
```

Inspect containers, ports, health, and logs:

```bash
docker ps -a
docker port social-network-backend
docker port social-network-frontend
docker inspect --format='{{json .State.Health}}' social-network-frontend
docker logs social-network-backend
docker logs social-network-frontend
```

Stop and remove standalone containers without deleting data:

```bash
docker stop -t 15 social-network-frontend social-network-backend
docker rm social-network-frontend social-network-backend
docker network rm social-network
```

## 💾 Persistence and migrations

The application currently uses `16` versioned SQLite schema migrations.

Pending migrations are applied automatically before the backend starts listening. The current version and dirty state are stored in `schema_migrations`.

Schema SQL is embedded in the backend binary with `go:embed`; changing SQL requires rebuilding the backend image. Never edit an already applied migration to release a schema change—add a new version.

Compose and standalone launches use the same explicitly named volumes:

```text
social-network-db
social-network-uploads
```

A restart with the same volumes preserves:

- users;
- sessions;
- profiles and follows;
- posts and comments;
- groups and events;
- direct and group messages;
- notification and chat unread state;
- uploaded media.

The backend healthcheck calls:

```text
GET /api/health
```

It reports healthy only when the HTTP server and SQLite respond correctly.

The frontend healthcheck verifies both static delivery and the proxied backend health endpoint.

To inspect the migration state during local development:

```bash
cd backend
sqlite3 var/social-network.db \
  "SELECT version, dirty FROM schema_migrations;"
```

Expected current version:

```text
16 | 0
```

The opt-in demo dataset uses a separate embedded migration set and a separate `seed_migrations` table. It is never applied during normal startup:

```bash
docker compose exec backend /app/seed
```

The command is repeatable. Before the first seed version, a reserved demo email collision aborts without changing the database; after the seed has been applied, another run is an idempotent no-op. It creates three demo accounts with password `LoopDemo123!`:

```text
alice.demo@example.com
bob.demo@example.com
carol.demo@example.com
```

The backend runtime image includes the SQLite CLI:

```bash
docker compose exec backend sqlite3 /data/db/social-network.db
```

Example audit commands:

```sql
.tables
SELECT version, dirty FROM schema_migrations;
SELECT id, email, first_name, last_name, is_private FROM users ORDER BY id;
SELECT user_id, created_at, expires_at FROM sessions ORDER BY created_at DESC;
SELECT id, author_user_id, group_id, privacy,
       media_id, length(text) AS text_length
FROM posts ORDER BY id;
SELECT post_id, user_id FROM post_selected_users ORDER BY post_id, user_id;
```

Raw session tokens are deliberately excluded from documented audit queries.

## ⚙️ Configuration

Backend environment variables:

| Variable                              | Default                 | Purpose                                      |
|---------------------------------------|-------------------------|----------------------------------------------|
| `SOCIAL_NETWORK_HTTP_ADDR`            | `127.0.0.1:8080`        | backend listen address                       |
| `SOCIAL_NETWORK_DB_PATH`              | `var/social-network.db` | SQLite database path                         |
| `SOCIAL_NETWORK_UPLOAD_DIR`           | `var/uploads`           | uploaded file directory                      |
| `SOCIAL_NETWORK_FRONTEND_DIR`         | `../frontend/dist`      | generated local static frontend directory    |
| `SOCIAL_NETWORK_COOKIE_SECURE`        | `false`                 | Secure attribute for the session cookie      |
| `SOCIAL_NETWORK_PUBLIC_ORIGIN`        | empty                   | canonical public origin for OAuth checks      |
| `SOCIAL_NETWORK_TRUST_PROXY`          | `false`                 | trust a validated full `X-Forwarded-For` chain |
| `SOCIAL_NETWORK_GITHUB_CLIENT_ID`     | empty                   | GitHub OAuth App client ID                    |
| `SOCIAL_NETWORK_GITHUB_CLIENT_SECRET` | empty                   | GitHub OAuth App client secret                |
| `SOCIAL_NETWORK_GITHUB_REDIRECT_URL`  | empty                   | absolute GitHub OAuth callback URL            |

Compose-specific host setting:

| Variable              | Default | Purpose                      |
|-----------------------|---------|------------------------------|
| `SOCIAL_NETWORK_PORT` | `8080`  | published frontend host port |

Docker overrides backend paths to persistent `/data` mounts and disables backend static frontend serving.

GitHub OAuth is disabled when all three GitHub variables are empty and enabled only
when all three are set. A partial configuration stops startup with an error. For
Compose, copy `.env.example` to `.env`, fill all three values, and configure the
GitHub OAuth App callback to match `SOCIAL_NETWORK_GITHUB_REDIRECT_URL`. The Go
application does not read `.env`; Compose performs the substitution. The real
`.env` is ignored by Git.

The redirect URL must point exactly to `/api/auth/oauth/github/callback`. Its
normalized scheme, hostname, and effective port must match
`SOCIAL_NETWORK_PUBLIC_ORIGIN`; when the latter is empty it is derived from the
validated redirect URL. Compose enables `SOCIAL_NETWORK_TRUST_PROXY` because the
backend is reachable only through Caddy on the private network. Do not enable it
when clients can connect to the backend directly. If `SOCIAL_NETWORK_PORT` changes,
update the public origin and GitHub redirect URL accordingly.

## 🧪 Local development and tests

### Backend

The SQLite driver requires CGO and a working C compiler.

```bash
cd frontend
npm run build
cd ../backend
go run ./cmd/server
```

The local Go server serves `../frontend/dist` by default, keeping browser requests and the session cookie same-origin.

Run backend checks:

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

### Frontend

Frontend tests use Node's built-in test runner and do not require a production dependency install:

```bash
cd frontend
npm test
npm run build
```

Final local verification:

```text
Go tests:              passed
Go race tests:         passed
go vet:                passed
Frontend tests:        passed
Frontend build:        passed
docker compose config: passed
```

## 📁 Project structure

```text
social-network/
├── backend/
│   ├── cmd/
│   │   ├── healthcheck/
│   │   ├── seed/
│   │   └── server/
│   ├── internal/
│   │   ├── app/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── http/
│   │   ├── oauth/
│   │   ├── platform/
│   │   ├── realtime/
│   │   ├── repo/
│   │   │   └── sqlite/
│   │   │       ├── migrations/
│   │   │       └── seedmigrations/
│   │   └── service/
│   ├── Dockerfile
│   └── README.md
├── frontend/
│   ├── scripts/
│   ├── src/
│   │   ├── assets/
│   │   ├── css/
│   │   ├── js/
│   │   └── templates/
│   ├── test/
│   ├── Caddyfile
│   ├── Dockerfile
│   └── package.json
├── scripts/
│   ├── build-images.ps1
│   └── build-images.sh
├── compose.yaml
├── README.md
└── README_RU.md
```

## 📚 API documentation

The detailed backend documentation covers:

- migration contracts;
- authentication and profile rules;
- controlled media delivery;
- follower state;
- post privacy;
- comments and attachments;
- groups and membership transitions;
- events and RSVP;
- persisted notifications;
- direct and group chats;
- WebSocket message contracts;
- HTTP status behavior.

See:

```text
backend/README.md
```

## 🧹 Cleanup

Stop Compose containers while preserving data:

```bash
docker compose down --remove-orphans
```

Permanently remove the project database and uploaded files:

```bash
docker compose down -v --remove-orphans
```

For standalone cleanup:

```bash
docker volume rm social-network-db social-network-uploads
```

Do not use a global `docker system prune` for project cleanup.

## ⚠️ Notes

- Do not run standalone and Compose backends at the same time. Both use the same SQLite and upload volumes.
- Stop and remove the previous backend before switching launch modes.
- Concurrent access to the same SQLite file by two backend processes is not supported.
- The current interface is optimized primarily for desktop layouts.
- The complete application and persistence contracts live in `backend/README.md`.

## 🧑‍💻 Authors
Nazar Yestayev (@nyestaye)
Nurgul Ilyassova (@nilyasso)
Sultan Yersultan (@syersult)
Teniz Serikbayev (@tteniz)
Aiman Zhumabayeva (@azhumaba)
Aiymgul Gabdullina (@agabdullin)
