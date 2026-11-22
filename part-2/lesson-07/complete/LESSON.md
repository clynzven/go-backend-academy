# Part 2, Lesson 7: Forged and replayed webhooks

Branch: `part2-lesson-07-complete` (start from `part2-lesson-07-starter`).

> Lab rule: the scripts in `exploit/` refuse any target that isn't localhost. The starter is vulnerable on purpose.

## The feature
`POST /webhooks/payments`: our payment provider tells us "invoice 4 of tenant acme was paid". It can't log in,
so it signs every event with a secret we share (`WEBHOOK_SECRET`): `Webhook-Signature: t=<unix>,v1=<HMAC>`.

## The attacks
- `exploit/webhookforge`: anyone can POST to that URL. The starter never checks the signature, so an unsigned
  event marks an invoice paid without any money moving.
- `exploit/provider -save` sends a real, signed event and keeps a copy (what an attacker might capture from a
  log or a proxy). `exploit/replay` sends that copy again: the starter records a payment every time.

## The fixes
- `internal/webhook/signature.go`: `Verify` recomputes the HMAC-SHA256 over the timestamp *and* the raw body,
  compares with `hmac.Equal` (constant time), and refuses timestamps more than 5 minutes off.
- `internal/webhook/inbound.go`: unsigned or badly signed events get `401`. Every applied event id goes into
  `processed_events` (`migrations/006_processed_events.sql`) in the same transaction; a repeated id answers
  `200 duplicate event, ignored` and changes nothing (the provider's own retries are safe too).
- `WEBHOOK_SECRET` is required at startup (32 bytes or more).
- Tests: `TestSignature`, `TestForgedEventIsRejected`, `TestReplayedEventIsIgnored`.
