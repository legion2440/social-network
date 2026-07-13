# Social Network Backend

Minimal Go backend foundation adapted from the tested `forum` project.

## Run

From this `backend` directory:

```bash
go test ./...
go run ./cmd/server
```

The SQLite driver is `github.com/mattn/go-sqlite3`, so CGO and a working C
compiler are required. On Windows, run from Git Bash and verify that
`go env CGO_ENABLED` prints `1` before building.

The default address is `http://127.0.0.1:8080`. Runtime paths such as
`var/social-network.db` and `var/uploads` are relative to the backend working
directory.

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

To inspect the current version with the SQLite CLI:

```bash
sqlite3 var/social-network.db "SELECT version, dirty FROM schema_migrations;"
```

The previous `schema.sql` bootstrap database is disposable and is not
automatically converted. If `var/social-network.db` was created by that old
bootstrap, delete it once before starting this version; the migrations will
create a fresh database. Runtime database and upload files remain ignored by
Git.

All persisted dates and timestamps use UTC Unix seconds in SQLite, including
`date_of_birth`, user audit timestamps, session timestamps, and media
timestamps. A date of birth is represented as UTC midnight for its calendar
date, keeping storage consistent without mixing timestamp formats.

The prepared user model contains email, password hash, first and last name,
date of birth, optional `male`/`female` gender, optional nickname/about text,
an optional custom avatar media reference, and created/updated timestamps.
Deleting a user cascades to owned media metadata and sessions; deleting avatar
media sets the user's optional avatar reference to `NULL`.

Supported environment variables:

- `SOCIAL_NETWORK_HTTP_ADDR`
- `SOCIAL_NETWORK_DB_PATH`
- `SOCIAL_NETWORK_UPLOAD_DIR`
- `SOCIAL_NETWORK_COOKIE_SECURE`

Implemented endpoints:

- `GET /api/health`
- `GET /ws` (authenticated WebSocket)
- `POST /api/media` (authenticated multipart upload, field name `file`)
- `GET /uploads/{id}` (authenticated, owner-only)

All other reserved API groups currently return JSON `501 Not Implemented`.

Registration, login, logout, current-user/profile endpoints, profile editing,
and avatar-specific upload/delivery are intentionally not implemented yet.
