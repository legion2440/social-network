# Loop Desktop

Electron desktop client for the `social-network` project. It reuses the existing web frontend and backend instead of maintaining a second chat implementation.

## What the desktop layer adds

- persistent Electron session and cookies between restarts;
- real-time chat and presence through the existing `/ws` endpoint;
- native desktop notifications for incoming messages;
- registration opens the social-network website in the default browser;
- offline warning and blocked message sending while disconnected;
- persistent local cache for the current session, chat list, and message history;
- background cache warming for chat history while online;
- interactive message search with results updated while typing;
- installers for Windows, Linux, and macOS.

The renderer is served from a loopback-only HTTP server. `/api`, `/static/avatars`, and `/ws` are proxied to the configured social-network origin so the existing same-origin API, session cookie, and WebSocket code can be reused unchanged.

## Development

Start the existing social network first from the repository root:

```bash
docker compose up --build
```

Then install the desktop dependencies once:

```bash
cd desktop
npm install
```

Run the desktop application:

```bash
npm run dev
```

By default the desktop client connects to `http://127.0.0.1:8080`.

To connect to another running social-network instance, set `SOCIAL_NETWORK_URL` before starting the app. `SOCIAL_NETWORK_WEB_URL` can be set separately when the registration website should use a different public URL.

PowerShell example:

```powershell
$env:SOCIAL_NETWORK_URL = "http://192.168.1.20:8080"
$env:SOCIAL_NETWORK_WEB_URL = "http://192.168.1.20:8080"
npm run dev
```

Bash example:

```bash
SOCIAL_NETWORK_URL=http://192.168.1.20:8080 \
SOCIAL_NETWORK_WEB_URL=http://192.168.1.20:8080 \
npm run dev
```

## Tests

```bash
npm test
```

The desktop tests cover cache selection, offline cache fallback, login/session cache seeding, cookie rewriting, static-path safety, and WebSocket same-origin proxy headers.

## Packaging

Build for the current platform or explicitly select a target:

```bash
npm run dist
npm run dist:win
npm run dist:linux
npm run dist:mac
```

Artifacts are written to `desktop/release/`:

- Windows: `Loop-<version>-Setup.exe`
- Linux: `Loop-<version>.AppImage`
- macOS: `Loop-<version>.dmg`

The repository workflow `.github/workflows/desktop-build.yml` builds the three platform artifacts on native GitHub-hosted runners. Local cross-compilation is therefore not required for the normal project workflow.

## Offline behavior

When online, the desktop adapter caches the authenticated user, conversation pages, and message-history pages. If connectivity is lost, cached conversations remain readable, a visible offline warning is shown, WebSocket traffic is stopped, and sending a message is blocked with the same offline warning.

Logging out clears the desktop HTTP cache. Session expiry is still controlled by the existing backend.

## Signing

The default build is unsigned. Code-signing identities can be added later without changing the application architecture.
