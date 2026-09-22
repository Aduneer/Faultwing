# Architecture

Faultwing runs as three application processes around one PostgreSQL database:

```text
                             ┌─────────────────┐
Application ── HTTP ───────► │ Go API          │ ◄──── React dashboard
                             │ auth + queries  │       HTTP + WebSocket
                             └────────┬────────┘
                                      │
                                      ▼
                             ┌─────────────────┐
                             │ PostgreSQL      │
                             │ jobs + domain   │
                             │ data + NOTIFY   │
                             └────────┬────────┘
                                      │
                                      ▼
                             ┌─────────────────┐
                             │ Go worker       │
                             │ fingerprinting  │
                             │ aggregation     │
                             └─────────────────┘
```

## API process

`cmd/api` owns the public HTTP routes. Its middleware authenticates either a
project API key (ingestion and API-key issue routes) or a user session
(dashboard routes). Project ownership is checked before a dashboard user can
read or change issues.

The API also:

- applies embedded SQL migrations while opening the database;
- limits ingestion independently for each authenticated project;
- exposes liveness and database-aware readiness endpoints;
- listens for PostgreSQL issue notifications and forwards them to WebSocket
  clients connected to the matching project; and
- performs graceful HTTP shutdown on `SIGINT` or `SIGTERM`.

## Event ingestion and worker

`POST /api/v1/events` validates the payload and inserts a row into
`event_jobs`. Returning `202 Accepted` means that the job is durable in
PostgreSQL, not that the event has already been grouped.

`cmd/worker` polls for available jobs. A worker claims one row with
`FOR UPDATE SKIP LOCKED`, allowing multiple worker processes to operate
without processing the same available job concurrently. It then:

1. fingerprints the exception within its project;
2. creates or updates the matching issue;
3. records the individual event;
4. publishes a PostgreSQL `NOTIFY` message; and
5. deletes the completed job in the same transaction.

A failed job remains in `event_jobs`. Its attempt count is increased and it is
rescheduled with exponential backoff capped at five minutes. There is not yet
a dead-letter policy or maximum retry count.

## Persistence model

```text
User
 ├── Session
 └── Project
      ├── API key
      ├── Event job
      └── Issue
           └── Event
```

API keys and session tokens are only stored as SHA-256 hashes. Passwords are
stored as bcrypt hashes. An issue fingerprint is unique inside a project, so
two projects can report the same exception without sharing an issue.

The ordered SQL files under `migrations/` are the schema source of truth. They
are embedded in the Go binary and recorded in `schema_migrations` after they
are applied.

## Realtime updates

Browser WebSockets cannot attach the normal `Authorization` header, so the
dashboard sends its session token as the first message after connecting. The
server authenticates it, verifies project ownership, and then subscribes the
connection to that project's in-process hub.

PostgreSQL notifications contain only a project ID and issue ID. They are a
refetch signal, not a durable event stream. The dashboard therefore reloads
data after a notification and after reconnecting. If an API instance misses a
notification, PostgreSQL remains authoritative.

Each API process maintains its own WebSocket connections, while every API
process may listen to the shared PostgreSQL channel. This is sufficient for
the current single-database design without adding a separate message broker.

## Frontend boundary

The React application uses the same origin for HTTP and WebSocket requests.
Vite proxies `/api` to the Go API during development; a production deployment
will need an equivalent reverse proxy.

The dashboard keeps its session in `sessionStorage`. Editor selection and
per-project local checkout paths are browser-only settings in `localStorage`.
Those local paths never enter an API request.

## Current operational limits

- Production container and reverse-proxy configuration are not yet provided.
- TLS, secret management, backups, and retention policies are deployment
  responsibilities that still need documentation and testing.
- Rate-limit state lives in one API process and is not coordinated across
  multiple API replicas.
- Registration and login do not yet have dedicated abuse throttles, and event
  ingestion does not yet enforce a request-body size limit.
- Project API-key rotation and revocation are not implemented.
- Event jobs retry indefinitely; there is no dead-letter inspection workflow.
- Realtime notifications are intentionally non-durable.
