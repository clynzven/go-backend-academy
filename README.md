# Go Backend Academy

> **Vulnerable on purpose.** Part 1's code and every Part 2 `starter` folder contain security holes that the
> course exploits on camera. Use them to learn on your own machine; never deploy them. The `exploit/` scripts
> refuse any target that isn't localhost.

- **Part 1: Zero to Production API** builds the API (below).
- **Part 2: Attack & Patch Lab** attacks it and hardens it: see [Part 2](#part-2-attack--patch-lab).

## Part 1: Zero to Production API

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
- From Part 1 lesson 5: Docker (PostgreSQL 16 via `docker compose`), plus `curl`; `jq` and `psql` are handy.
- Part 2 adds one package from lesson 4: `golang.org/x/crypto` (argon2id).

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

## Part 2: Attack & Patch Lab

Every lesson starts from the previous one's hardened code, runs a real exploit against it on localhost, then
patches it and turns the exploit into a test.

| # | Lesson | Attack | Fix | Folder |
|---|---|---|---|---|
| 1 | Slowloris & the Memory Bomb | 1,100 slow connections; one 300 MB request | server timeouts, `http.MaxBytesReader` | [`part-2/lesson-01`](part-2/lesson-01) |
| 2 | BOLA: Another Tenant's Invoices | change `X-Tenant-ID`, read another company's data | login + signed token, tenant from the token | [`part-2/lesson-02`](part-2/lesson-02) |
| 3 | JWT Attacks & Token Theft | `alg: none`, cracked secret, 30-day stolen token | HS256 only, 32-byte secret, 15-min tokens, refresh rotation | [`part-2/lesson-03`](part-2/lesson-03) |
| 4 | Brute Force & Credential Stuffing | 500 guesses in 180 ms | argon2id, per-IP and per-account limits | [`part-2/lesson-04`](part-2/lesson-04) |
| 5 | Race Condition: Paid Twice | 50 simultaneous payments, 50 charges | atomic conditional `UPDATE`, idempotency keys | [`part-2/lesson-05`](part-2/lesson-05) |
| 6 | SSRF: The Webhook Inside Job | webhook tester fetches an internal admin page | IP check in `net.Dialer.Control` | [`part-2/lesson-06`](part-2/lesson-06) |
| 7 | Forged & Replayed Webhooks | unsigned "paid" event; one event replayed 5 times | HMAC + timestamp, event ids applied once | [`part-2/lesson-07`](part-2/lesson-07) |
| 8 | Distributed Defense: Spoofed IPs | spoofed `X-Forwarded-For`, per-copy limits | trusted proxies, limits in Postgres, security headers | [`part-2/lesson-08`](part-2/lesson-08) |

The finished, hardened project is in [`part-2/go-backend-academy-hardened`](part-2/go-backend-academy-hardened).
From lesson 3 the API needs `JWT_SECRET` (32+ bytes; `openssl rand -base64 32`), and from lesson 7 `WEBHOOK_SECRET`.

- `part-2/lesson-NN/starter`: vulnerable, with the exploit scripts in `exploit/` and tests that fail until you patch it.
- `part-2/lesson-NN/complete`: patched; every test passes and the exploits fail.
- Branches: `part2-lesson-NN-starter` and `part2-lesson-NN-complete`.

```
cd part-2/lesson-01/starter
(ulimit -n 1024; DATABASE_URL='postgres://app:app@127.0.0.1:5432/taskdb?sslmode=disable' go run ./cmd/api) &
ulimit -n 4096; go run ./exploit/slowloris -conns 1100 -for 20s
```

## Troubleshooting

- **Tests fail with "cannot truncate a table referenced in a foreign key constraint"**: you ran a later lesson's
  tests first, and the test database still has its tables. Each folder expects a fresh test database:
  `psql -h 127.0.0.1 -U app -d taskdb -c 'DROP DATABASE taskdb_test' -c 'CREATE DATABASE taskdb_test'`
  (CI starts every folder from an empty one).
- **The API refuses to start**: from Part 2 lesson 3 it needs `JWT_SECRET`, and from lesson 7 `WEBHOOK_SECRET`,
  each at least 32 bytes. See `.env.example`.

## License

Code: MIT (see `LICENSE`).
