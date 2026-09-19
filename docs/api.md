# API

## Events

Start PostgreSQL with `make db-up`, then start the API with `make run`.

Create an event:

```bash
curl -X POST http://localhost:8080/events \
  -H 'Content-Type: application/json' \
  -d '{"message":"database connection timed out","stacktrace":"db/client.go:42"}'
```

List events:

```bash
curl http://localhost:8080/events
```
