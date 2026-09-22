# Frontend

The dashboard lives in `web/` and uses React, TypeScript, and Vite. Node 26
and npm 12 are the supported local toolchain in this workspace.

```sh
make frontend-install
make frontend-dev
```

The Vite development server proxies `/api` and WebSocket upgrades to the API
at `127.0.0.1:8080`. The browser should construct realtime URLs from its own
origin (for example, `/api/v1/projects/1/realtime`); this keeps the same-origin
session during development while Vite forwards the upgrade.

In production, put the dashboard and API behind the same origin (or configure
an equivalent reverse proxy) and forward both `/api` HTTP requests and
`/api/v1/projects/:projectID/realtime` WebSocket upgrades. Dashboard sessions
are kept in `sessionStorage` so a tab can reconnect without putting a token in
the URL; this intentionally logs the user out when the tab/session ends and
does not provide cross-tab persistence.

Build the production bundle with `make frontend-build`. End-to-end tests use
Playwright via `make frontend-test`; install a browser once with
`cd web && npx playwright install chromium` when a local browser is absent.

The UI also supports a clearly labelled demo mode when no API is available;
demo data is local-only and is never presented as live monitoring data. Any
AI-generated visual work used in the dashboard should be disclosed alongside
the relevant published screenshot or showcase material.

Issue details can open a parsed stack-frame location in a local editor. Editor
choice is stored globally in the browser, while source checkout roots are kept
per project so relative runtime paths can be mapped to local files. Presets are
included for VS Code, VS Code Insiders, Cursor, Zed, and JetBrains IDEs
launched through Toolbox. JetBrains settings also record the IDE product and
its local project name because those are part of the Toolbox navigation URL.
Other editors can use a custom URL template with `{path}`, `{pathEncoded}`,
`{pathNoLeadingSlash}`, `{line}`, and `{column}`. These settings never leave
the browser, and the file location can always be copied when no editor
protocol is configured.
