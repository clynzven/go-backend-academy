# Part 2, Lesson 6: SSRF, the webhook inside job

Branch: `part2-lesson-06-starter` (the solution is `part2-lesson-06-complete`).

This starter is vulnerable on purpose: run the exploit first, then fix the `TODO`s until `go test ./...` passes.

> Lab rule: the scripts in `exploit/` refuse any target that isn't localhost. The starter is vulnerable on purpose.

## The feature
`POST /webhooks/test {"url": "..."}` sends a sample event to a tenant's webhook URL and shows the first 200 bytes
of the answer: the "Send test event" button every SaaS has.

## The attack
Our server makes that request, from inside our network. `exploit/internaladmin` plays an internal service that
trusts local traffic (fake secrets on `127.0.0.1:9900`). `exploit/ssrf` asks the webhook tester for it, spelled
four ways. The starter blocks `localhost` and `127.*` by string, but `0.0.0.0` and `[::ffff:127.0.0.1]` walk
straight past and return the secrets. In the cloud the classic target is `169.254.169.254`, the metadata
service that hands out credentials.

## The fix
- `internal/webhook/safehttp.go`: `SafeClient()` checks the real IP in `net.Dialer.Control`, after DNS and right
  before connect. Loopback, private, link-local, unspecified, multicast and CGNAT ranges are refused. Every
  spelling, DNS name, redirect hop and DNS rebinding trick ends up there as a plain IP. No proxy, 3 redirects
  at most, 10-second timeout.
- `internal/webhook/handler.go`: blocked URLs answer `422`.
- Tests: `TestIsPublic`, `TestSafeClientRefusesLoopback`, `TestPreviewCannotReachInternalAddresses`.

## Run it
```
go run ./exploit/internaladmin &
go run ./exploit/ssrf -token "$TOKEN"
```
