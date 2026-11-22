# Part 2, Lesson 2: BOLA, reading another tenant's invoices

Branch: `part2-lesson-02-starter` (the solution is `part2-lesson-02-complete`).

This starter is vulnerable on purpose: run the exploit first, then fix the `TODO`s until `go test ./...` passes.

> Lab rule: the scripts in `exploit/` refuse any target that isn't localhost. The starter is vulnerable on purpose.

## The attack
`exploit/bola` asks `GET /invoices` once per tenant name, only changing the `X-Tenant-ID` header. The starter
has a working login, but the data routes still believe the header, so every tenant's invoices come back.
That's Broken Object Level Authorization (BOLA, also called IDOR), number one on the OWASP API Top 10.

## The fix
- `internal/auth`: `POST /login` (email + password) returns a token signed with HMAC-SHA256 (a JWT, by hand).
  `Middleware` verifies `Authorization: Bearer <token>` and puts the user's identity in the request context.
- `internal/httpx.Tenant` now reads the tenant from that verified identity. The header is ignored completely;
  no identity means `401 Unauthorized`.
- `cmd/adduser` creates users; `migrations/003_users.sql` adds the table.
- Test: `TestHeaderCannotChooseTheTenant` (acme's token + `X-Tenant-ID: globex` sees only acme's data).

## Run it
```
DATABASE_URL=... go run ./cmd/adduser -tenant acme -email alice@acme.test -password 'correct horse'
TOKEN=$(curl -s localhost:8080/login -d '{"email":"alice@acme.test","password":"correct horse"}' | jq -r .token)
go run ./exploit/bola -token "$TOKEN"
```

## Honest limits
The token code is deliberately minimal, and it has holes: it trusts the token's own `alg` header, the secret
has a weak default, and tokens live for 30 days. Lesson 3 attacks exactly those. Passwords are stored as plain
SHA-256 for now; Lesson 4 replaces that.
