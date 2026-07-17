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

`date_of_birth` is stored as SQLite `TEXT` and exposed in JSON strictly as
`DD-MM-YYYY`. It must be a real calendar date, so values such as `31-02-1992`
are rejected. User audit timestamps, session timestamps, and media timestamps
continue to use UTC Unix seconds in SQLite.

The user model contains email, password hash, first and last name, date of
birth, optional `male`/`female` gender, optional nickname/about text, an
optional custom avatar media reference, and created/updated timestamps. Gender
accepts only `NULL`, `male`, or `female`; every other value is an error.
Deleting a user cascades to owned media metadata and sessions; deleting avatar
media sets the user's optional avatar reference to `NULL`.

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
on gender. Auth responses expose the computed `avatar_url`. Custom avatars use
the current owner-only `/uploads/{id}` route; this is not the future public
profile-avatar contract.

Login accepts JSON with `email` and `password`. Authentication currently uses
an HttpOnly, SameSite=Lax session cookie. Request token extraction is separate
from cookie transport so a Bearer source can be added later. Multiple sessions
per user are allowed, and logout removes only the current session. Missing
tokens and already-absent sessions are successful no-ops; session storage
failures return `500`.

Supported environment variables:

- `SOCIAL_NETWORK_HTTP_ADDR`
- `SOCIAL_NETWORK_DB_PATH`
- `SOCIAL_NETWORK_UPLOAD_DIR`
- `SOCIAL_NETWORK_COOKIE_SECURE`

Implemented endpoints:

- `GET /api/health`
- `POST /api/auth/register` (`multipart/form-data`)
- `POST /api/auth/login` (JSON)
- `POST /api/auth/logout` (idempotent `204`; storage failures return `500`)
- `GET /api/auth/me` (authenticated)
- `GET /ws` (authenticated WebSocket)
- `POST /api/media` (authenticated multipart upload, field name `file`)
- `GET /uploads/{id}` (authenticated, owner-only)
- `GET /static/avatars/{male,female,neutral}.svg`

All other reserved API groups currently return JSON `501 Not Implemented`.

Profile editing, approval workflow, and privacy-aware delivery of avatars for
other users are intentionally not implemented yet.
