# Lesson 7: Custom middleware

Branch: `lesson-07-complete` (start from `lesson-07-starter`).

## What this lesson adds
- `internal/middleware`:
  - `Middleware` is `func(http.Handler) http.Handler`; `Chain(h, A, B)` runs A, then B, then h.
  - `RequestID`: keeps a valid incoming `X-Request-ID` or makes a new one; puts it in the context and the response.
  - `Logger`: one structured `log/slog` JSON line per request (method, path, status, bytes, duration_ms, request_id),
    at WARN for 4xx and ERROR for 5xx.
  - `Recoverer`: a panic becomes a clean JSON 500; the log gets the panic and the stack trace.
  - `statusRecorder`: a carbon copy of the status and size the handler wrote (with `Unwrap` for `http.ResponseController`).
- `internal/requestid`: the request ID in a `context.Context`, under a private key type.
- `httpx.WriteErr` logs 500s with `slog`, including the request ID, so the log line and the client's
  `X-Request-ID` match.
- `DEBUG_ROUTES=1` adds `GET /debug/panic` (a handler with a nil-map bug) for the demo. Never in production.

## Run it
```
docker compose up -d
export DATABASE_URL='postgres://app:app@127.0.0.1:5432/taskdb?sslmode=disable'
DEBUG_ROUTES=1 go run ./cmd/api
curl -i localhost:8080/debug/panic       # 500 JSON, server still up
curl -i localhost:8080/tasks -H 'X-Tenant-ID: acme'
go test -race ./...
```

## Your turn
1. Add a `Timing` middleware that sets a `Server-Timing: app;dur=<ms>` header (hint: before the first write).
2. Log the tenant (`X-Tenant-ID`) in the request line, but only when it's valid.
3. Make `Logger` skip `/health` so load-balancer checks don't flood the log, with a test.
