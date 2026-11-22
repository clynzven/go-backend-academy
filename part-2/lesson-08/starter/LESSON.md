# Part 2, Lesson 8: Distributed defense, spoofed IPs and shared limits

Branch: `part2-lesson-08-starter` (the solution is `part2-lesson-08-complete`).

This starter is vulnerable on purpose: run the exploit first, then fix the `TODO`s until `go test ./...` passes. This is the finished, hardened API.

> Lab rule: the scripts in `exploit/` refuse any target that isn't localhost. The starter is vulnerable on purpose.

## The setup
Three copies of the API (`:8081`, `:8082`, `:8083`) behind `cmd/proxy` (`:8000`): round-robin, and
`X-Forwarded-For` like nginx (keep what the client sent, append the real address).

## The attacks
`exploit/flood` sends login attempts through the proxy.
- With `-spoof`, each request claims a new IP in `X-Forwarded-For`. The starter reads the *first* entry, which
  the client typed, so every attempt gets a fresh per-IP bucket: never rate limited, and every attempt costs us
  a full argon2id check.
- With `-email`, one account: the starter's lockout lives in each process's memory, so three copies allow
  three times as many wrong passwords.

## The fixes
- `internal/httpx/clientip.go`: `ClientIP` trusts `X-Forwarded-For` only when the connection comes from one of
  `TRUSTED_PROXIES`, and then takes the *last* entry, the one our proxy added.
- `internal/ratelimit/postgres.go` + `migrations/007_rate_limits.sql`: the token buckets live in Postgres, one
  atomic `INSERT ... ON CONFLICT DO UPDATE` per attempt, shared by every copy and surviving restarts.
- `internal/middleware.SecureHeaders`: `nosniff`, a `default-src 'none'` CSP, `DENY` framing, `no-referrer`,
  `no-store`, HSTS.
- `cmd/proxy`: also caps requests in flight per client IP (the edge limit from lesson 1).
- `Dockerfile` + `compose.yaml --profile cluster`: the same setup in containers (not tested in the course's
  recording environment, which can't pull images).
- Tests: `TestClientIP`, `TestSpoofedForwardedForIsIgnored`, `TestLimitIsSharedAcrossInstances`, `TestSecurityHeaders`.
