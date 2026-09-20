# API

## Projects

Start PostgreSQL with `make db-up`, then start the API with `make run`.

Create a project:

```bash
curl -X POST http://localhost:8080/api/v1/projects \
  -H 'Content-Type: application/json' \
  -d '{"name":"My App"}'
```

The response contains an API key. It is only returned once, so copy it before
submitting events.

## Events

Create an event using the project's API key:

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

## Issues

List issues for the API key's project, optionally filtered by status:

```bash
curl 'http://localhost:8080/api/v1/issues?status=open' \
  -H 'Authorization: Bearer fly_your_api_key'
```

Get an issue with its affected environments and releases:

```bash
curl http://localhost:8080/api/v1/issues/1 \
  -H 'Authorization: Bearer fly_your_api_key'
```
