# Lesson 4: Errors Mapped to HTTP Status

Branch: `lesson-04-complete` (start from `lesson-04-starter`).

## What this lesson adds
- `internal/apperr`: the kinds of errors our app returns.
  - Sentinels `ErrNotFound` (404) and `ErrConflict` (409), wrapped with `%w` to add details: `task 42 not found`.
  - Types `*BadRequestError` (400) and `*ValidationError` (422, with a message per field).
- `internal/httpx/errors.go`: handlers now return `error`; `httpx.Handle` and `WriteErr` turn every error into
  the right status and a JSON body, in one place. Unknown errors become a plain 500 and the real cause goes to the log.
- `httpx.JSONErrors` makes the router's own 404 and 405 replies JSON too (the 405 keeps its `Allow` header).
- `DecodeJSON` gives friendly messages (`field "title" must be a string`) instead of Go type names.
- The store returns errors: unknown IDs are `ErrNotFound`, a second open task with the same title is `ErrConflict`.
- Use `errors.Is` / `errors.As`, never `==`: `==` only compares the outer error, so wrapped errors are missed.

| Error | Status |
|---|---|
| `*apperr.BadRequestError` | 400 Bad Request |
| `*apperr.ValidationError` | 422 Unprocessable Entity |
| wraps `apperr.ErrNotFound` | 404 Not Found |
| wraps `apperr.ErrConflict` | 409 Conflict |
| anything else | 500, logged, never shown |

## Run it
```
go run ./cmd/api
curl -i -X POST localhost:8080/tasks -H 'X-Tenant-ID: acme' -d '{"title":42}'
curl -i -X POST localhost:8080/tasks -H 'X-Tenant-ID: acme' -d '{"title":""}'
curl -i localhost:8080/tasks/42 -H 'X-Tenant-ID: acme'
go test -race ./...
```

Not yet: we still trust `X-Tenant-ID` as sent (real authentication comes later in the academy). Tasks live in
memory until lesson 5 moves them into PostgreSQL.

## Your turn
1. Give `PATCH /tasks/{id}` the same duplicate-title check, returning 409, with a test.
2. Add a `"done"` rule: reopening a task whose title is now taken by another open task must be a 409 too.
3. Make a handler return `errors.New("disk full")` and check that the client sees only "internal server error"
   while the log shows the real cause.
