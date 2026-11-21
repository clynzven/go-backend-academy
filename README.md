# Go Backend Academy, Part 1: Zero to Production API

A Multi-Tenant Task & Invoice API in pure Go, built lesson by lesson with the standard library
(`net/http`, `encoding/json`, `database/sql`, `context`, `log/slog`) and one dependency: the `pgx`
PostgreSQL driver. No web framework.

Companion code for the **ClynzVen Code Academy** YouTube course (8 videos, or one full-length video with chapters).

| # | Lesson | Folder |
|---|---|---|
| 1 | Your First Raw HTTP Server | [`code/lesson-01`](code/lesson-01) |
| 2 | Structs, Pointers & Strict JSON | [`code/lesson-02`](code/lesson-02) |
| 3 | Project Layout & Modern Routing | [`code/lesson-03`](code/lesson-03) |
| 4 | Errors Mapped to HTTP Status | [`code/lesson-04`](code/lesson-04) |
| 5 | PostgreSQL with database/sql | [`code/lesson-05`](code/lesson-05) |
| 6 | CRUD & Stopping SQL Injection | [`code/lesson-06`](code/lesson-06) |
| 7 | Middleware: Logs, Timing, Recovery | [`code/lesson-07`](code/lesson-07) |
| 8 | Graceful Shutdown & Integration Tests | [`code/lesson-08`](code/lesson-08) |

The finished project is in [`code/go-backend-academy-complete`](code/go-backend-academy-complete).

## How each lesson is organized

- `code/lesson-NN/starter`: where the video starts. It compiles, has `TODO` comments, and has failing tests.
- `code/lesson-NN/complete`: the result. `go vet` is clean and every test passes.
- Each folder has a `LESSON.md` with a summary and the exercises.
- The same code is also on branches `lesson-NN-starter` and `lesson-NN-complete` (project at the root).
- See what a lesson changed: `diff -ru code/lesson-04/complete code/lesson-05/complete`.

## Requirements

- Go 1.24 or newer.
- From lesson 5: Docker (PostgreSQL 16 via `docker compose`), plus `curl`; `jq` and `psql` are handy.

```
cd code/go-backend-academy-complete
make db-up
DATABASE_URL='postgres://app:app@127.0.0.1:5432/taskdb?sslmode=disable' go run ./cmd/api
make test-integration
```

## On purpose: what Part 1 leaves for Part 2

Part 1 is correct but naive in a few places, because Part 2 (the Attack & Patch Lab) exploits and fixes them:
no server timeouts, no request body size limit, and `X-Tenant-ID` trusted as sent. Don't deploy it to the
internet as-is.

## License

Code: MIT (see `LICENSE`).
