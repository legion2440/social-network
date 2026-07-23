# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e

FROM caddy:2.10.2-alpine@sha256:4c6e91c6ed0e2fa03efd5b44747b625fec79bc9cd06ac5235a779726618e530d

ARG APP_UID=10001
ARG APP_GID=10001

USER root

RUN addgroup -g "${APP_GID}" app \
    && adduser -D -H -u "${APP_UID}" -G app app \
    && mkdir -p /srv \
    && chown -R "${APP_UID}:${APP_GID}" /srv

COPY --chown=10001:10001 frontend/index.html /srv/index.html
COPY --chown=10001:10001 frontend/css/ /srv/css/
COPY --chown=10001:10001 frontend/js/ /srv/js/
COPY --chown=10001:10001 frontend/assets/ /srv/assets/
COPY --chown=10001:10001 frontend/Caddyfile /etc/caddy/Caddyfile

ENV XDG_CONFIG_HOME=/tmp/caddy/config \
    XDG_DATA_HOME=/tmp/caddy/data

USER 10001:10001
WORKDIR /srv

EXPOSE 8080
STOPSIGNAL SIGTERM

HEALTHCHECK --interval=5s --timeout=2s --start-period=10s --retries=5 \
    CMD ["sh", "-c", "wget -q -O /dev/null http://127.0.0.1:8080/ && wget -q -O /dev/null http://127.0.0.1:8080/api/health"]

ENTRYPOINT ["caddy"]
CMD ["run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
