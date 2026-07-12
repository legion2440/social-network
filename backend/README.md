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
