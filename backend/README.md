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
Lists for a public profile are available to every authenticated user. Lists for
a private profile are available only to its owner and accepted followers;
pending followers and outsiders receive `403`. Pending requests are available
to their target through `GET /api/follow-requests`; the owner accepts one with
`POST /api/follow-requests/{id}/accept` or rejects it with
`DELETE /api/follow-requests/{id}`. Pending relations never count as followers.

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
- `GET|PUT|DELETE /api/users/{id}/follow` (relationship, follow, unfollow)
- `GET /api/users/{id}/followers`
- `GET /api/users/{id}/following`
- `GET /api/users/{id}/avatar` (authenticated and privacy-controlled)
- `GET /api/follow-requests`
- `POST /api/follow-requests/{id}/accept`
- `DELETE /api/follow-requests/{id}`
- `GET /ws` (authenticated WebSocket)
- `POST /api/media` (authenticated multipart upload, field name `file`)
- `GET /uploads/{id}` (authenticated, owner-only)
- `GET /static/avatars/{male,female,neutral}.svg`
- `GET /` and frontend assets (local development and browser smoke only)

All other reserved API groups currently return JSON `501 Not Implemented`.

Posts and frontend follower controls are intentionally not implemented yet.

The local frontend file server does not replace the planned Docker topology.
The final setup keeps the backend private and serves the frontend through a
separate frontend/reverse-proxy container; the backend static handler is only a
development convenience and does not need to be used there.
