# Part 2, Lesson 3: JWT attacks and token theft

Branch: `part2-lesson-03-complete` (start from `part2-lesson-03-starter`).

> Lab rule: the scripts in `exploit/` refuse any target that isn't localhost. The starter is vulnerable on purpose.

## The attacks
- `exploit/jwtnone`: an unsigned token, `{"alg":"none"}`, for tenant globex. The starter lets the token choose
  its own algorithm, so it's accepted.
- `exploit/jwtcrack`: one captured token, 50 common secrets, offline. The starter's default secret `changeme`
  falls in microseconds, and the attacker signs tokens for any tenant.
- `exploit/jwtpeek`: reads a token's claims (they're only base64, not encrypted) and shows how long a stolen
  copy keeps working: 30 days in the starter.

## The fixes
- `internal/auth/token.go`: `Verify` accepts `HS256` only (we pick the algorithm, never the token) and refuses
  secrets shorter than 32 bytes. Access tokens live 15 minutes.
- `cmd/api/main.go`: `JWT_SECRET` is required and must be at least 32 bytes (`openssl rand -base64 32`).
- `internal/auth/refresh.go` + `migrations/004_refresh_tokens.sql`: `POST /refresh` swaps a one-time refresh
  token for a new pair (rotation). Reusing a spent one revokes every token from that login (reuse detection).
  `POST /logout` revokes them too. Only SHA-256 hashes of refresh tokens are stored.
- Tests: `TestAlgNoneIsRejected`, `TestWeakSecretsAreRefused`, `TestAccessTokensAreShortLived`,
  `TestRunRefusesAWeakSecret`, `TestStolenRefreshTokenIsDetected`, `TestLogoutRevokesRefreshTokens`.

## Run it
```
export JWT_SECRET=$(openssl rand -base64 32)
go run ./exploit/jwtnone -tenant globex
go run ./exploit/jwtcrack -token "$TOKEN" -tenant globex
go run ./exploit/jwtpeek -token "$TOKEN"
```

## Honest limits
Logout can't recall an access token that's already out: it stays valid until it expires (15 minutes at most).
If you need instant revocation, check a denylist on every request, at the cost of a database lookup.
