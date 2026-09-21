# API

## Service health

`GET /health` reports that the API process is running. `GET /ready` also checks
that PostgreSQL is reachable.

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## User authentication

Register a dashboard user, then log in to receive a seven-day session token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"learner@example.com","password":"correct horse battery staple"}'

curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"learner@example.com","password":"correct horse battery staple"}'
```

Log out by revoking the returned token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H 'Authorization: Bearer fly_session_your_session_token'
```

## Projects

Start PostgreSQL with `make db-up`, then start the API with `make run`.

Create a project using the session token returned by login:

```bash
curl -X POST http://localhost:8080/api/v1/projects \
  -H 'Authorization: Bearer fly_session_your_session_token' \
  -H 'Content-Type: application/json' \
  -d '{"name":"My App"}'
```

The response contains an API key. It is only returned once, so copy it before
submitting events.

List the signed-in user's projects:

```bash
curl http://localhost:8080/api/v1/projects \
  -H 'Authorization: Bearer fly_session_your_session_token'
```

## Events

Start the background worker in another terminal with `make run-worker`, then
create an event using the project's API key:

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H 'Authorization: Bearer fly_your_api_key' \
  -H 'Content-Type: application/json' \
  -d '{
    "exception_type":"DatabaseTimeoutError",
    "message":"database connection timed out",
    "stacktrace":"db/client.go:42",
    "environment":"production",
    "release":"1.3.2"
  }'
```

Accepted events return `202 Accepted` with a queue receipt:

```json
{
  "job_id": 1,
  "queued_at": "2026-09-20T12:00:00Z"
}
```

Event ingestion is limited per project to 60 events per minute with a burst of
10 by default. A rejected request returns `429 Too Many Requests` with a
`Retry-After` header. Set `EVENT_RATE_LIMIT_PER_MINUTE` and
`EVENT_RATE_LIMIT_BURST` to positive integers to change these limits.

## Issues

Dashboard users access issues through an owned project. Listing supports the
same status filter and cursor pagination; the limit defaults to 50 and cannot
exceed 100:

```bash
curl 'http://localhost:8080/api/v1/projects/1/issues?status=open&limit=25' \
  -H 'Authorization: Bearer fly_session_your_session_token'

curl http://localhost:8080/api/v1/projects/1/issues/7 \
  -H 'Authorization: Bearer fly_session_your_session_token'

curl -X PATCH http://localhost:8080/api/v1/projects/1/issues/7 \
  -H 'Authorization: Bearer fly_session_your_session_token' \
  -H 'Content-Type: application/json' \
  -d '{"status":"resolved"}'
```

The original API-key-scoped routes remain available. List issues for the API
key's project:

```bash
curl 'http://localhost:8080/api/v1/issues?status=open&limit=25' \
  -H 'Authorization: Bearer fly_your_api_key'
```

When `next_cursor` is present, pass it unchanged to retrieve the next API-key
page:

```bash
curl 'http://localhost:8080/api/v1/issues?status=open&limit=25&cursor=NEXT_CURSOR' \
  -H 'Authorization: Bearer fly_your_api_key'
```

Get an issue with its affected environments and releases:

```bash
curl http://localhost:8080/api/v1/issues/1 \
  -H 'Authorization: Bearer fly_your_api_key'
```

Resolve, reopen, or ignore an issue:

```bash
curl -X PATCH http://localhost:8080/api/v1/issues/1 \
  -H 'Authorization: Bearer fly_your_api_key' \
  -H 'Content-Type: application/json' \
  -d '{"status":"resolved"}'
```
