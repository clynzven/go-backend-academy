# Part 2, Lesson 5: Race condition, an invoice paid twice

Branch: `part2-lesson-05-starter` (the solution is `part2-lesson-05-complete`).

This starter is vulnerable on purpose: run the exploit first, then fix the `TODO`s until `go test ./...` passes.

> Lab rule: the scripts in `exploit/` refuse any target that isn't localhost. The starter is vulnerable on purpose.

## The attack
`exploit/race` creates an invoice and fires 50 `POST /invoices/{id}/pay` requests at the same instant. The
starter checks "is it still open?", charges the card, then marks it paid. All 50 requests pass the check before
any of them writes "paid", so the customer is charged 50 times.

## The fixes
- `internal/invoice/payments.go`: one atomic, conditional `UPDATE invoices SET status = 'paid' ... WHERE
  status = 'open' RETURNING ...` claims the invoice inside a transaction. Postgres locks the row, so only one
  request gets a row back; the rest get `409 Conflict`. If the charge fails, the transaction rolls back.
- `Idempotency-Key` header: the same key returns the same payment (`200`, `Idempotent-Replayed: true`), so a
  client retrying after a timeout can't pay twice. A unique `(tenant_id, idempotency_key)` backs it up.
- `migrations/005_payments.sql`: `invoices.status` and the `payments` table. `FakeProvider` stands in for a
  payment provider (20 ms per charge).
- Tests: `TestConcurrentPaymentsChargeOnce` (50 goroutines, exactly one wins), `TestIdempotencyKeyReplays`.

## A real bug from building this lesson
The first version hung under 50 parallel requests: inside the transaction it asked the pool for a *second*
connection (`s.db.QueryRow`). With every connection already holding a transaction, nobody could get one.
Inside a transaction, use the transaction for every query.
