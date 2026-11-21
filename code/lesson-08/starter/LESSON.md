# Lesson 8: Graceful shutdown and integration testing

Branch: `lesson-08-starter`. Follow the TODOs in `cmd/api/main.go` and `cmd/api/integration_test.go` until the tests pass. The finished version is `lesson-08-complete`. This is the finished Part 1 API.

## What this lesson adds
- `cmd/api/main.go`: `main` only calls `run(ctx, getenv, stdout)`, so tests can start and stop the real program.
  - `signal.NotifyContext` cancels the context on Ctrl+C (SIGINT) or SIGTERM (`docker stop`, Kubernetes).
  - `http.Server` + `Shutdown`: stop accepting new connections, let requests in flight finish
    (up to 10 seconds), then close the database pool. A second Ctrl+C exits immediately.
  - `ADDR` (default `127.0.0.1:8080`) is configurable.
- `cmd/api/app.go`: `newHandler` builds stores, handlers, router and middleware, shared by `run` and the tests.
- `cmd/api/integration_test.go`: the whole API over real HTTP (`httptest.NewServer`) against real Postgres,
  in its own schema so it can run alongside the other packages' tests. Plus a test that cancels `run`'s
  context mid-request and checks the request still finishes.
- `DEBUG_ROUTES=1` adds `GET /debug/slow?seconds=5` for the shutdown demo.
- `Makefile` (`make test`, `make test-integration`) and `.github/workflows/ci.yml` (Postgres service + `go test -race`).

## Run it
```
make db-up
DEBUG_ROUTES=1 DATABASE_URL='postgres://app:app@127.0.0.1:5432/taskdb?sslmode=disable' go run ./cmd/api
curl 'localhost:8080/debug/slow?seconds=5' &   # then press Ctrl+C in the server's terminal
make test-integration
```

## What Part 2 attacks
Part 1 is correct but naive on purpose. Part 2 (Attack & Patch Lab) exploits and fixes, among others:
- the server has no timeouts (Slowloris) and no request body limit (memory exhaustion);
- `X-Tenant-ID` is trusted as sent, so anyone can read another tenant's data (BOLA/IDOR).

## Your turn
1. Add `ReadHeaderTimeout: 5 * time.Second` to the `http.Server` and explain what it stops (preview of Part 2).
2. Make the shutdown timeout configurable with `SHUTDOWN_TIMEOUT=15s` (`time.ParseDuration`), with a test.
3. Add an integration test for the 422 response of `POST /invoices`, checking every field name.
