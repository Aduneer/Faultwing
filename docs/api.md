# API

The API listens on `http://localhost:8080` by default and returns JSON unless a
successful endpoint has no response body.

Faultwing has two bearer-token types:

- `faultwing_session_...` authenticates a dashboard user and expires after seven
  days. Use it for projects and user-scoped issue routes.
- `faultwing_...` authenticates one project. Use it for event ingestion and the
  API-key-scoped issue routes.

API and session tokens are secrets. The examples below contain placeholders,
not working credentials.

## Service health

`GET /health` reports that the API process is running. `GET /ready` also checks
that PostgreSQL is reachable.

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## User authentication

Register a dashboard user:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"learner@example.com","password":"correct horse battery staple"}'
```

Registration creates the user but does not create a session. Log in separately
to receive a seven-day token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"learner@example.com","password":"correct horse battery staple"}'
```

```json
{
  "user": {
    "id": 1,
    "email": "learner@example.com",
    "created_at": "2026-09-22T12:00:00Z"
  },
  "token": "faultwing_session_your_session_token",
  "expires_at": "2026-09-29T12:00:00Z"
}
```

Log out by revoking the session token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H 'Authorization: Bearer faultwing_session_your_session_token'
```

A successful logout returns `204 No Content`.

## Projects

Create a project using a session token:

```bash
curl -X POST http://localhost:8080/api/v1/projects \
  -H 'Authorization: Bearer faultwing_session_your_session_token' \
  -H 'Content-Type: application/json' \
  -d '{"name":"My App"}'
```

The response contains the new project and its API key. The plaintext key is
only returned by this request, so copy it before navigating away.

```json
{
  "project": {
    "id": 1,
    "owner_id": 1,
    "name": "My App",
    "created_at": "2026-09-22T12:05:00Z"
  },
  "api_key": "faultwing_your_api_key"
}
```

List the signed-in user's projects:

```bash
curl http://localhost:8080/api/v1/projects \
  -H 'Authorization: Bearer faultwing_session_your_session_token'
```

API-key rotation and revocation endpoints are not implemented yet. Create a
new project if you need a different key during local development.

## Events

Start the worker with `make run-worker`, then submit an event using the
project's API key:

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H 'Authorization: Bearer faultwing_your_api_key' \
  -H 'Content-Type: application/json' \
  -d '{
    "exception_type":"DatabaseTimeoutError",
    "message":"database connection timed out",
    "stacktrace":"db/client.go:42",
    "environment":"production",
    "release":"1.3.2"
  }'
```

`exception_type`, `message`, and `environment` are required. `stacktrace` and
`release` may be empty. Accepted events return `202 Accepted` with the durable
job's receipt:

```json
{
  "job_id": 1,
  "queued_at": "2026-09-22T12:06:00Z"
}
```

Ingestion is limited per project to 60 events per minute with a burst of 10 by
default. A rejected request returns `429 Too Many Requests` with a
`Retry-After` header. Set `EVENT_RATE_LIMIT_PER_MINUTE` and
`EVENT_RATE_LIMIT_BURST` to positive integers to change these limits.

## Issues

The dashboard uses user-scoped routes and verifies that the signed-in user
owns the project:

```bash
curl 'http://localhost:8080/api/v1/projects/1/issues?status=open&limit=25' \
  -H 'Authorization: Bearer faultwing_session_your_session_token'

curl http://localhost:8080/api/v1/projects/1/issues/7 \
  -H 'Authorization: Bearer faultwing_session_your_session_token'

curl -X PATCH http://localhost:8080/api/v1/projects/1/issues/7 \
  -H 'Authorization: Bearer faultwing_session_your_session_token' \
  -H 'Content-Type: application/json' \
  -d '{"status":"resolved"}'
```

The project API key can access equivalent routes without a project ID:

```bash
curl 'http://localhost:8080/api/v1/issues?status=open&limit=25' \
  -H 'Authorization: Bearer faultwing_your_api_key'

curl http://localhost:8080/api/v1/issues/7 \
  -H 'Authorization: Bearer faultwing_your_api_key'
```

`status` may be `open`, `resolved`, or `ignored`. The page limit defaults to 50
and must be between 1 and 100. List responses have this shape:

```json
{
  "issues": [],
  "next_cursor": "opaque-value-for-the-next-page"
}
```

When `next_cursor` is present, pass it back unchanged:

```bash
curl 'http://localhost:8080/api/v1/issues?status=open&limit=25&cursor=NEXT_CURSOR' \
  -H 'Authorization: Bearer faultwing_your_api_key'
```

Issue detail responses also include distinct `environments` and `releases`
observed for that issue. Update an issue with `PATCH` and one of the three
supported states:

```bash
curl -X PATCH http://localhost:8080/api/v1/issues/7 \
  -H 'Authorization: Bearer faultwing_your_api_key' \
  -H 'Content-Type: application/json' \
  -d '{"status":"ignored"}'
```

## Realtime issue updates

Dashboard clients subscribe to an owned project at
`ws://localhost:8080/api/v1/projects/1/realtime`. Because browser WebSocket
connections cannot set an `Authorization` header, send the session token as
the first message within five seconds:

```json
{"token":"faultwing_session_your_session_token"}
```

After authenticating the session and checking project ownership, the server
confirms the subscription:

```json
{"type":"realtime.ready","project_id":1}
```

Processing an event then produces a project-scoped notification:

```json
{"type":"issue.updated","project_id":1,"issue_id":7}
```

Treat notifications as signals to refetch the issue or list. PostgreSQL
notifications are deliberately lightweight and are not a durable event
history, so clients should also refetch after reconnecting.

## Errors

HTTP errors use a small JSON envelope:

```json
{"error":"human-readable message"}
```

Authentication failures return `401`, attempts to access another user's
project are hidden behind `404`, validation failures return `400`, and
duplicate emails or project names return `409`.
