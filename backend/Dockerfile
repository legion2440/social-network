# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e

FROM golang:1.24.2-bookworm@sha256:79390b5e5af9ee6e7b1173ee3eac7fadf6751a545297672916b59bfa0ecf6f71 AS builder

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=1 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/server \
    ./cmd/server

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/healthcheck \
    ./cmd/healthcheck

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818

ARG APP_UID=10001
ARG APP_GID=10001

RUN groupadd --gid "${APP_GID}" app \
    && useradd --uid "${APP_UID}" --gid "${APP_GID}" \
        --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin app \
    && mkdir -p /app /data/db /data/uploads \
    && chown -R "${APP_UID}:${APP_GID}" /app /data

COPY --from=builder --chown=10001:10001 /out/server /app/server
COPY --from=builder --chown=10001:10001 /out/healthcheck /app/healthcheck

ENV SOCIAL_NETWORK_HTTP_ADDR=0.0.0.0:8080 \
    SOCIAL_NETWORK_DB_PATH=/data/db/social-network.db \
    SOCIAL_NETWORK_UPLOAD_DIR=/data/uploads \
    SOCIAL_NETWORK_FRONTEND_DIR=/app/frontend-disabled \
    SOCIAL_NETWORK_COOKIE_SECURE=false

USER 10001:10001
WORKDIR /app

EXPOSE 8080
STOPSIGNAL SIGTERM

HEALTHCHECK --interval=5s --timeout=2s --start-period=10s --retries=5 \
    CMD ["/app/healthcheck"]

ENTRYPOINT ["/app/server"]
