# Lesson 2: Structs, Pointers & Strict JSON

Branch: `lesson-02-complete` (start from `lesson-02-starter`).

## What this lesson adds
- `task.go`: the `Task` struct with JSON tags, a separate `createTaskRequest` (clients can't set `id` or `done`),
  an in-memory `taskStore` (pointer receivers + a `sync.Mutex`), and `/tasks` (`GET` lists, `POST` creates).
- `json.go`: `decodeJSON` rejects unknown fields, wrong types, empty bodies and extra data;
  `writeJSON` / `writeError` send JSON with the right status code.
- `invoice.go`: money as whole cents (`int64`), never `float64`.
- Tests: strict decoding (`json_test.go`), create/list, validation and 100 concurrent requests (`task_test.go`).

## Run it
```
go run .
curl -i -X POST localhost:8080/tasks -d '{"title":"Buy milk"}'
curl localhost:8080/tasks
curl -X POST localhost:8080/tasks -d '{"title":"Hack","is_admin":true}'
go test -race ./...
```

Not yet: a size limit on request bodies (Part 2 of the academy), and friendly error messages (lesson 4).

## Your turn
1. Add a `GET /tasks/count` route that returns `{"count": N}`, with a test.
2. Add an optional `"notes"` field (max 500 characters) to tasks. Which struct(s) need it, and why?
3. Change `add` to a value receiver `(s taskStore)`. Run `go vet ./...` and the tests. What do you see, and why?
