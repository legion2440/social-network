# Social Network

The application runs as two Linux containers:

```text
browser
  -> social-network-frontend:8080 (Caddy, public)
       -> static frontend
       -> /api, /ws, /uploads, /static/avatars
            -> social-network-backend:8080 (private)
                 -> SQLite
                 -> uploaded files
```

Only the frontend port is published. Caddy keeps HTTP, the session cookie, and
WebSocket traffic same-origin. The backend image contains neither the frontend
nor Go source code.

## Prerequisites

- Docker Engine with the Compose plugin
- port `8080` available, or `SOCIAL_NETWORK_PORT` set to another host port

Both images use pinned official base-image digests. The backend is built with
CGO because `mattn/go-sqlite3` requires it. Both runtime processes use numeric
non-root users, and the documented launches use read-only root filesystems.

## Build the images

Run from the repository root:

```bash
docker image build -f backend/Dockerfile -t social-network-backend:local .
docker image build -f frontend/Dockerfile -t social-network-frontend:local .
```

## Standalone Docker launch

Create the private network and persistent named volumes once:

```bash
docker network create social-network
docker volume create social-network-db
docker volume create social-network-uploads
```

Start the private backend first:

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

Wait until its image healthcheck reports `healthy`:

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

Open `http://127.0.0.1:${SOCIAL_NETWORK_PORT:-8080}`. Inspect the two
containers and verify that only the frontend publishes a host port:

```bash
docker ps -a
docker port social-network-backend
docker port social-network-frontend
docker inspect --format='{{json .State.Health}}' social-network-frontend
docker logs social-network-backend
docker logs social-network-frontend
```

Stop and remove the containers without deleting their data:

```bash
docker stop -t 15 social-network-frontend social-network-backend
docker rm social-network-frontend social-network-backend
docker network rm social-network
```

## Docker Compose

Compose builds the same images and uses the same two named volumes:

```bash
docker compose config
docker compose up --build -d
docker compose ps
docker compose logs -f
```

Open `http://127.0.0.1:8080`, or the port selected through
`SOCIAL_NETWORK_PORT`.

Stop the stack while preserving the database and uploads:

```bash
docker compose down --remove-orphans
```

The volumes are deliberately named `social-network-db` and
`social-network-uploads`, without a Compose project prefix. This lets
standalone and Compose launches verify persistence in both directions.

**Do not run standalone and Compose backends at the same time.** They would
mount the same SQLite database and uploads. Before switching launch modes,
stop and remove the previous frontend and, especially, the previous backend.
Concurrent access by two backend processes is not supported.

## Health and persistence

The backend image checks `GET /api/health` with a small dependency-free Go
binary. It is healthy only when the HTTP server and SQLite respond correctly.
The frontend image checks both `/` and proxied `/api/health`, proving static
delivery and backend connectivity.

Pending migrations run automatically before the backend starts listening. A
restart with the same volumes preserves users, sessions, messages, unread
state, and uploaded media. SQLite database, WAL/SHM files, and uploads are not
part of either image.

To inspect runtime properties:

```bash
docker inspect --format='backend user={{.Config.User}} readonly={{.HostConfig.ReadonlyRootfs}} ports={{json .NetworkSettings.Ports}}' social-network-backend
docker inspect --format='frontend user={{.Config.User}} readonly={{.HostConfig.ReadonlyRootfs}} ports={{json .NetworkSettings.Ports}}' social-network-frontend
docker history --no-trunc social-network-backend:local
docker history --no-trunc social-network-frontend:local
```

The local HTTP setup uses `SOCIAL_NETWORK_COOKIE_SECURE=false`. Set it to
`true` when deploying behind HTTPS.

## Destructive data reset

This permanently removes the project database and uploaded files:

```bash
docker compose down -v --remove-orphans
```

For a standalone cleanup, first remove both containers and then delete only
the project-owned volumes:

```bash
docker volume rm social-network-db social-network-uploads
```

Do not use a global `docker system prune` for project cleanup.

## Local development and tests

The Go backend still serves `frontend/` directly for local development. Docker
production uses Caddy and leaves backend static serving disabled.

```bash
cd backend
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...

cd ../frontend
npm test
```
