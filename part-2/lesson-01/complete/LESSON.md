# Part 2, Lesson 1: Slowloris and the memory bomb

Branch: `part2-lesson-01-complete` (start from `part2-lesson-01-starter`).
Part 2 is the Attack & Patch Lab: every lesson attacks the Part 1 API on your own machine, then fixes it.

> Lab rule: the scripts in `exploit/` refuse any target that isn't localhost. Only attack systems you own.
> The starter branch is vulnerable on purpose. Never deploy it.

## The attacks
- `exploit/slowloris`: opens 1,100 connections and sends headers one line every 10 seconds, never finishing.
  Part 1's server has no timeouts, so it holds every one of them. With Linux's usual limit of 1,024 open
  files, it can't accept anyone else: `accept4: too many open files`, and `GET /health` times out.
- `exploit/bigbody`: posts a 300 MB JSON title. Part 1 reads all of it before validating, so the server
  grows from about 12 MB to over 1 GB of memory, only to answer 422.

## The fixes
- `cmd/api/server.go`: `newServer` sets `ReadHeaderTimeout` (5 s), `ReadTimeout`, `WriteTimeout`,
  `IdleTimeout` and `MaxHeaderBytes`. Their zero values mean "no limit".
- `internal/middleware`: `MaxBody(1 MB)` wraps every body in `http.MaxBytesReader`.
- `internal/httpx`: `DecodeJSON` turns `*http.MaxBytesError` into `apperr.TooLargeError`, and `WriteErr`
  answers `413 Content Too Large` with the limit in the message.
- Tests: `cmd/api/server_test.go` (a never-ending header is cut; every limit is set) and
  `TestMaxBodyStopsAHugeUpload` (an endless body gets 413 after about 1 KB).

## Run it
```
make db-up
(ulimit -n 1024; DATABASE_URL='postgres://app:app@127.0.0.1:5432/taskdb?sslmode=disable' go run ./cmd/api)
ulimit -n 4096; go run ./exploit/slowloris -conns 1100 -for 20s
go run ./exploit/bigbody -mb 300
go test ./...
```
On macOS the default open-file limit is 256, so the attack works with fewer connections (try `-conns 400`).

## Honest limits
Timeouts make Slowloris expensive, not impossible: an attacker who reconnects every few seconds can still
fill the server's connection slots. Raise the open-file limit in production (systemd `LimitNOFILE`) and
limit connections per IP at the edge; Lesson 8 adds the proxy layer.
