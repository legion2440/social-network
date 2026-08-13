# Cross-platform desktop client

Electron desktop client for the `social-network` project. The implementation lives in `desktop/` and reuses the existing frontend, HTTP API, sessions, GitHub authentication flow, and WebSocket endpoint.

· [Русская версия](README_RU.md)

Packaged binaries are not stored in the repository. Build output is written to `desktop/release/`, which is ignored by Git.

## Features

- persistent login until the backend session expires or the user logs out;
- email/password authentication through the existing backend;
- GitHub sign-in through the existing server-side OAuth flow when configured;
- realtime direct/group chat and presence;
- native notifications for incoming messages;
- emoji support from the existing chat UI;
- registration opens the website directly in registration mode;
- offline warning, readable cached chat history, and blocked sending while disconnected;
- automatic recovery when the configured server becomes reachable again;
- interactive message search that loads older history while searching;
- include, exclude, fuzzy, equality, and numeric comparison search operators;
- Windows installer, Linux AppImage, and macOS DMG targets.

## Prerequisites

- Node.js 22 or newer;
- npm;
- a running `social-network` backend/frontend reachable from the computer that runs Loop.

Start the web application from the repository root when it is running on the same machine:

```bash
docker compose up --build
```

## Server connection

On the first launch, when no server has been saved and `SOCIAL_NETWORK_URL` is not set, Loop shows a separate **Connect to your server** setup window. The field is prefilled with:

```text
http://127.0.0.1:8080
```

Enter an address that is reachable from the current computer or virtual machine and press **Connect**. Loop validates `/api/health` before saving the address in the Electron user-data directory.

The server address is not shown in the normal application UI. To change it later, use **Settings → Server…**. On Windows and Linux the native menu may be hidden until `Alt` is pressed. On macOS use **Loop → Server Settings…**. Changing the server restarts Loop.

If a previously configured server is temporarily unavailable, Loop starts in offline mode instead of forcing the setup screen. It checks the server periodically and refreshes the application automatically after connectivity returns, clearing stale connection errors without a manual reload.

`SOCIAL_NETWORK_URL` remains available as an explicit override for development or managed launches. When it is set, it has priority over the saved setting and the Server Settings field is read-only. `SOCIAL_NETWORK_WEB_URL` can separately override the website opened for registration; otherwise it follows the configured server.

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

## Run without packaging

From the repository root:

```bash
cd desktop
npm ci
npm run dev
```

`npm run dev` builds the current frontend first and then starts Electron.

## Build an installer or package

Install the locked dependencies:

```bash
cd desktop
npm ci
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

Build the package on the corresponding operating system. The packages are unsigned, so the operating system can show its normal warning for locally built applications.

## GitHub sign-in

The desktop client uses the GitHub flow already implemented by the web application. It opens the flow in a dedicated Electron window using the persistent desktop session. Navigation in that window is restricted to `github.com` and the configured social-network origin. After the backend creates an authenticated session, Loop copies the resulting session cookies to the local application origin and reloads the main window.

No GitHub client ID or client secret is embedded in the desktop application or committed to the repository. GitHub sign-in is available when the backend is configured with:

```text
SOCIAL_NETWORK_GITHUB_CLIENT_ID
SOCIAL_NETWORK_GITHUB_CLIENT_SECRET
SOCIAL_NETWORK_GITHUB_REDIRECT_URL
```

The callback URL must point to:

```text
/api/auth/oauth/github/callback
```

When those variables are empty, the backend does not advertise the GitHub provider and the GitHub button is hidden.

## Offline behavior

When the configured server cannot be reached:

- Loop displays the offline state;
- previously cached chat data remains readable;
- chat sending is intercepted and shows the same no-connection message instead of silently doing nothing;
- realtime sockets are closed until connectivity returns.

The local HTTP cache stores response bodies needed for offline chat access but strips authentication-related response headers. Session cookies stay in Electron's cookie store and are not written into `offline-cache/http-cache.json`. Cached `/api/auth/me` data expires after the backend's 24-hour session lifetime; chat cache entries have a bounded lifetime as well. Older cache formats are sanitized when loaded.

## Message search

Search is evaluated immediately after every input change. When the active conversation still has older pages, Loop loads those pages automatically while the query is active and updates the match count as history arrives.

Plain words are inclusive substring filters. These operators are also supported:

```text
+word               include
include:word        include
-word               exclude
exclude:word        exclude
~word               fuzzy match
fuzzy:word          fuzzy match
```

Quoted phrases can contain spaces:

```text
"release candidate"
exclude:"not ready"
```

Numeric values contained in message text can be compared with:

```text
=10
!=10
>10
<10
>=10
<=10
```

Operators can be combined, for example:

```text
+invoice -rejected >100 <200
```

## Tests

From `desktop/`:

```bash
npm test
```

The desktop tests cover the local proxy, offline cache migration and expiry, authentication-header stripping, WebSocket proxy headers, server settings, GitHub session handoff and navigation restrictions, search operators, and frontend integration hooks.

Frontend checks can be run separately:

```bash
cd ../frontend
npm test
npm run build
```

## Manual verification

A useful end-to-end check uses one account in the normal browser application and another in Loop. Verify session persistence after restart, realtime messages in both directions, presence changes, native notifications, offline history and blocked sending, automatic recovery after the server returns, message search across older history, GitHub sign-in when configured, and logout.

## Implementation notes

Electron serves the built frontend through a loopback-only HTTP server and proxies `/api`, `/static/avatars`, and `/ws` to the configured social-network origin. This preserves the frontend's existing same-origin behavior without maintaining a second application implementation.

The main window uses the persistent `persist:loop` partition. Desktop-specific functions are exposed through sandboxed preload bridges with `contextIsolation` enabled and Node integration disabled. Stable `data-loop-*` hooks connect the desktop adapter to the chat template instead of relying on generated visual class numbers.
