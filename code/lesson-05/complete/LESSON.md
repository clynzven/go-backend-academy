# Lesson 5: PostgreSQL with database/sql and Docker Compose

Branch: `lesson-05-complete` (start from `lesson-05-starter`).

## What this lesson adds
- `compose.yaml`: PostgreSQL 16 in Docker, reachable only from this computer, with a healthcheck.
- `migrations/`: numbered `.sql` files, embedded into the binary with `//go:embed`.
  `001_tasks.sql` creates the `tasks` table; a unique index enforces "no two open tasks with the same title".
- `internal/db`: `Open` builds the connection pool (`database/sql` + the `pgx` driver) and pings it;
  `Migrate` applies new migrations once each, in a transaction.
- `internal/task`: a `Store` interface with two implementations: `PostgresStore` (production)
  and `MemoryStore` (fast unit tests). Every query is parameterized (`$1`, `$2`) and filters by tenant.
- Every store method takes a `context.Context`, so a cancelled request stops its query.
- `/health` now pings the database (1-second timeout) and answers 503 when it's down.
- Configuration comes from the environment: `DATABASE_URL` (see `.env.example`).

`database/sql` is the standard library's universal socket; `github.com/jackc/pgx/v5` is the one
non-stdlib dependency in Part 1: the Postgres plug for that socket.

## Run it
```
docker compose up -d
export DATABASE_URL='postgres://app:app@127.0.0.1:5432/taskdb?sslmode=disable'
go run ./cmd/api
curl -X POST localhost:8080/tasks -H 'X-Tenant-ID: acme' -d '{"title":"Survive a restart"}'
# stop the server (Ctrl+C), start it again, and the task is still there:
curl localhost:8080/tasks -H 'X-Tenant-ID: acme'
```

Tests: `go test ./...` runs everything; the Postgres tests skip unless `TEST_DATABASE_URL` is set:
```
psql 'postgres://app:app@127.0.0.1:5432/taskdb' -c 'CREATE DATABASE taskdb_test'
TEST_DATABASE_URL='postgres://app:app@127.0.0.1:5432/taskdb_test?sslmode=disable' go test -race ./...
```

Not yet: we still trust `X-Tenant-ID` as sent, and the server has no timeouts (both fixed later in the academy).

## Your turn
1. Add `GET /tasks?done=true|false` that filters in SQL with a `$2` parameter (never string building).
2. Log `pool.Stats()` (open, in use, idle) once at startup, and again after 50 parallel requests.
3. Stop Postgres (`docker compose stop`) and call `/health` and `GET /tasks`: explain each status you get.
