# Part 2, Lesson 4: Brute force and credential stuffing

Branch: `part2-lesson-04-starter` (the solution is `part2-lesson-04-complete`).

This starter is vulnerable on purpose: run the exploit first, then fix the `TODO`s until `go test ./...` passes.

> Lab rule: the scripts in `exploit/` refuse any target that isn't localhost. The starter is vulnerable on purpose.

## The attack
`exploit/stuffing` guesses one account's password over HTTP, 20 at a time, from about 500 passwords real
attackers try first (common words, the company name, a year, a symbol). The starter has no limits and stores
passwords as plain SHA-256, so it finds `Globex-2026!` in about a second.

## The fixes
- `internal/auth/password.go`: argon2id (`golang.org/x/crypto/argon2`, OWASP settings: 19 MiB, 2 passes),
  a random salt per password, stored as a PHC string. Old SHA-256 hashes still log in once and are upgraded.
  Unknown emails check a dummy hash, so timing doesn't reveal who has an account.
- `internal/ratelimit`: a token bucket per key, in memory.
- `POST /login`: 10 attempts per IP, then one every 6 seconds; 5 wrong passwords lock the account (one more
  try every 3 minutes), even for the right password. Both answer `429 Too Many Requests` with `Retry-After`.
  The limit runs before the expensive hash.
- Tests: `TestArgon2idRoundTrip`, `TestBruteForceIsStopped`, `TestOldHashesAreUpgraded`, `TestBurstThenRefill`.

## Honest limits
The limits live in one process's memory: restart the API and they reset, and three copies of the API mean three
separate limits. A lockout can also be abused to lock someone out on purpose. Lesson 8 moves the limits into
Postgres, shared by every copy.
