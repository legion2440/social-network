# Cross-platform desktop client

Electron desktop client for the `social-network` project. The implementation lives in `desktop/` and reuses the existing frontend, HTTP API, sessions, and WebSocket endpoint.

No packaged binaries are stored in the repository. Build output is written to `desktop/release/`, which is ignored by Git.

## Features

- persistent login between application restarts until the backend session expires or the user logs out;
- realtime direct/group chat and presence through the existing `/ws` endpoint;
- native desktop notifications for incoming messages;
- emoji support from the existing chat UI;
- registration opens the regular social-network website in the default browser;
- offline warning, readable cached chat history, and blocked sending while disconnected;
- interactive message search that updates while typing, without a search button;
- Windows, Linux AppImage, and macOS package targets.

## Prerequisites

- the social-network backend/frontend must be running and reachable;
- Node.js 22 or newer and npm are required to run or build the desktop client.

Start the existing application from the repository root:

```bash
docker compose up --build
```

By default the desktop client connects to:

```text
http://127.0.0.1:8080
```

## Run without packaging

From the repository root:

```bash
cd desktop
npm install
npm run dev
```

`npm run dev` builds the current frontend first and then starts Electron.

## Build an installer/package

Install dependencies once:

```bash
cd desktop
npm install
```

Build for the current platform:

```bash
npm run dist
```

Or select a target explicitly:

```bash
npm run dist:win
npm run dist:linux
npm run dist:mac
```

Build output is placed in:

```text
desktop/release/
```

Expected package names:

- Windows: `Loop-<version>-Setup.exe`
- Linux: `Loop-<version>.AppImage`
- macOS: `Loop-<version>.dmg`

For normal audit use, build the package on the corresponding operating system. Packaged files are intentionally not committed to Git, so repository hosting limits do not affect them.

## Connect to another server

Set `SOCIAL_NETWORK_URL` before starting or building/running the client. `SOCIAL_NETWORK_WEB_URL` controls the website opened for registration and can be configured separately.

PowerShell:

```powershell
$env:SOCIAL_NETWORK_URL = "http://192.168.1.20:8080"
$env:SOCIAL_NETWORK_WEB_URL = "http://192.168.1.20:8080"
npm run dev
```

Bash:

```bash
SOCIAL_NETWORK_URL=http://192.168.1.20:8080 \
SOCIAL_NETWORK_WEB_URL=http://192.168.1.20:8080 \
npm run dev
```

If no variables are set, both values default to `http://127.0.0.1:8080`.

## Tests

From `desktop/`:

```bash
npm test
```

The tests cover the local proxy, offline cache fallback, session cache seeding, cookie rewriting, static-path safety, and WebSocket proxy headers.

Frontend checks can be run separately:

```bash
cd ../frontend
npm test
npm run build
```

## Audit smoke test

Use two users, for example one in the normal browser application and one in the desktop client.

1. Log in to the desktop client.
2. Close and reopen it; the session should remain active.
3. Send a message browser → desktop and desktop → browser; both directions should update in realtime.
4. Verify online/offline presence changes without refreshing.
5. Receive a message while the desktop window is not focused and verify the native notification.
6. Disconnect the network: the offline warning must appear, cached messages must remain readable, and sending must be blocked.
7. Reconnect and verify realtime operation resumes.
8. Open Messages and type in the message search field; matching results must update immediately while typing.
9. Log out; logout must remain available from the application UI.

## Implementation notes

Electron serves the built frontend from a loopback-only local HTTP server and proxies `/api`, `/static/avatars`, and `/ws` to the configured social-network origin. This keeps the existing same-origin frontend/API behavior instead of duplicating the application in a second client.

The Electron session uses a persistent partition. Desktop-specific behavior is exposed through a sandboxed preload bridge with `contextIsolation` enabled and Node integration disabled in the renderer.

The default packages are unsigned. Operating systems can therefore show their normal warning for locally built unsigned applications.
