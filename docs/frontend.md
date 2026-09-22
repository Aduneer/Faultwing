# Frontend

The dashboard lives in `web/` and uses React, TypeScript, and Vite. The current
development and CI toolchain uses Node.js 26 and npm 12.

## Local development

Start the database, API, and worker from the repository root:

```bash
make db-up
make run
make run-worker
```

Run the API and worker in separate terminals. Install frontend dependencies and
start Vite in a third terminal:

```bash
make frontend-install
make frontend-dev
```

Open <http://127.0.0.1:5173>. Vite proxies `/api` requests and WebSocket
upgrades to the API at `127.0.0.1:8080`, preserving the browser's same-origin
session behavior.

Use **Explore the interactive demo** when you only want to inspect the UI.
Demo issues are browser-local fixtures and are always labelled as demo data.

## Session and realtime behavior

The dashboard stores its login response in `sessionStorage`. A tab can reconnect
without putting a token in the URL, but closing the tab ends that browser
session and sessions are not shared across tabs.

Realtime connections use the current page origin and send the session token as
their first WebSocket message. Notifications trigger a normal API refetch; the
WebSocket is not treated as an authoritative data store.

## Open in Editor

Issue details parse common Python, JavaScript, and Go-style stack-frame
locations. The first time **Open in Editor** is used, the dashboard asks for an
editor and the absolute path to that project's local checkout.

Built-in URL schemes are available for:

- Visual Studio Code and Visual Studio Code Insiders
- Cursor
- Zed
- IntelliJ IDEA, GoLand, PyCharm, WebStorm, PhpStorm, Rider, CLion, RubyMine,
  RustRover, and DataGrip through JetBrains Toolbox

Other editors can use a custom URL template with `{path}`, `{pathEncoded}`,
`{pathNoLeadingSlash}`, `{line}`, and `{column}` placeholders. Browsers may ask
for confirmation before opening a local application.

Editor choice is stored globally in `localStorage`. Source roots and JetBrains
project names are stored per Faultwing project. These settings never leave the
browser, and the parsed location can always be copied instead of opened.

## Build and tests

Build the production bundle:

```bash
make frontend-build
```

The focused Playwright suite covers user-visible behavior, responsive layouts,
and the main API contract without mirroring every DOM node. Install Chromium
once, then run it:

```bash
cd web
npx playwright install chromium
cd ..
make frontend-test
```

## Production boundary

The Vite development server is not a production server. A deployment must
serve the built `web/dist/` files and proxy both HTTP requests under `/api` and
WebSocket upgrades under `/api/v1/projects/:projectID/realtime` to the Go API
on the same public origin.

Faultwing does not yet provide that reverse-proxy/container configuration. See
[architecture.md](architecture.md) for the remaining operational limits.

The frontend design and image assets were produced with AI assistance.
