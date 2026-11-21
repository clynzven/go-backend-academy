# Lesson 3: Project Layout & Modern Routing

Branch: `lesson-03-complete` (start from `lesson-03-starter`).

## What this lesson adds
- A real layout: `cmd/api/` starts the program; `internal/` holds packages other projects can't import.
  - `internal/httpx`: `DecodeJSON`, `WriteJSON`, `WriteError` (moved from `json.go`).
  - `internal/task`: `Task`, `Store` (per tenant), `Handler` with `Register(mux)`.
  - `internal/invoice`: the `Invoice` struct (endpoints in lesson 6).
- Go 1.22+ routing: `"GET /tasks"`, `"POST /tasks"`, `"GET /tasks/{id}"`, `"PATCH /tasks/{id}"`, `"GET /{$}"`.
  Unknown paths get 404, wrong methods get 405 with an `Allow` header, automatically.
- Tenants: every task call needs an `X-Tenant-ID` header; tenants never see each other's tasks.
- `PATCH` uses pointer fields (`*string`, `*bool`) to tell "not sent" from "false".

## Run it
```
go run ./cmd/api
curl -X POST localhost:8080/tasks -H 'X-Tenant-ID: acme' -d '{"title":"Buy milk"}'
curl localhost:8080/tasks/1 -H 'X-Tenant-ID: acme'
curl -X PATCH localhost:8080/tasks/1 -H 'X-Tenant-ID: acme' -d '{"done":true}'
go test -race ./...
```

Not yet: we trust `X-Tenant-ID` as sent (real authentication comes later in the academy), and the router's own
404/405 replies are plain text (lesson 4 makes every error JSON).

## Your turn
1. Add `DELETE /tasks/{id}` that returns 204 No Content, with a test (including another tenant trying it).
2. Add `GET /tasks?done=true` filtering with `r.URL.Query()`. Does it need a new route pattern? Why not?
3. Change `"GET /{$}"` to `"GET /"`, run the tests, and explain the failure.
