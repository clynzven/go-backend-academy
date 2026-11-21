# Lesson 1: Go Setup, Mental Models & Your First Raw HTTP Server

Branch: `lesson-01-complete` (start from `lesson-01-starter`).

## What this lesson adds
- `go.mod`: the module `github.com/clynzven/go-backend-academy` (Go 1.24+).
- `main.go`: a `net/http` server on `127.0.0.1:8080` with three handlers:
  - `GET /health` -> `ok`
  - `GET /hello?name=Clynz` -> `Hello, Clynz!`
  - every other path -> echoes the method and path (we fix this catch-all in Lesson 3).
- `main_test.go`: table-driven tests with `net/http/httptest` (no real network needed).

## Run it
```
go run .
curl localhost:8080/health
curl "localhost:8080/hello?name=Clynz"
curl -v localhost:8080/health
go test ./...
```

## Your turn
1. Add `GET /time` that returns the server time in RFC 3339 (`time.Now().Format(time.RFC3339)`), with a test.
2. Make `/hello` refuse names longer than 50 characters with status 400 (`http.StatusBadRequest`). Add a test row.
3. Start the server twice. Read the error. Why does the second one fail?
