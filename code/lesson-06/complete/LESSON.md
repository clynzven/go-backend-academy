# Lesson 6: Full CRUD and SQL parameterization

Branch: `lesson-06-complete` (start from `lesson-06-starter`).

## What this lesson adds
- `DELETE /tasks/{id}`: 204 No Content, or 404 when `RowsAffected()` is 0.
- `migrations/002_invoices.sql`: `invoices` and `invoice_lines` (money in BIGINT cents, CHECK constraints,
  `ON DELETE CASCADE`).
- `internal/invoice`: create (one transaction for the invoice and all its lines), list with totals,
  get with lines, delete, and `GET /invoices/search?customer=...`.
- `internal/httpx/request.go`: `Tenant` and `PathID`, shared by the task and invoice handlers.
- The SQL injection lesson: the starter's search glues the user's text into the SQL with `fmt.Sprintf`.
  `' OR 1=1 --` then returns every tenant's invoices. The fix passes it as a parameter (`$2`),
  and `TestSearchIsNotInjectable` keeps it fixed.

| Method | Path | Answer |
|---|---|---|
| GET | /invoices | 200, this tenant's invoices with `total_cents` |
| POST | /invoices | 201, or 422 listing every invalid field |
| GET | /invoices/search?customer=... | 200 |
| GET | /invoices/{id} | 200 with lines, or 404 |
| DELETE | /invoices/{id} | 204, or 404 |

## Run it
```
docker compose up -d
export DATABASE_URL='postgres://app:app@127.0.0.1:5432/taskdb?sslmode=disable'
go run ./cmd/api
curl -X POST localhost:8080/invoices -H 'X-Tenant-ID: acme' \
  -d '{"customer":"Clynz Studio","lines":[{"description":"Logo design","quantity":2,"unit_cents":150000}]}'
curl -G localhost:8080/invoices/search -H 'X-Tenant-ID: acme' --data-urlencode "customer=' OR 1=1 --"
TEST_DATABASE_URL='postgres://app:app@127.0.0.1:5432/taskdb_test?sslmode=disable' go test -race ./...
```

Not yet: `X-Tenant-ID` is still trusted as sent. Part 2 shows why that header alone is not security.

## Your turn
1. Add `PATCH /invoices/{id}` to rename the customer, with a test (404 for another tenant).
2. `%` and `_` in the search text are LIKE wildcards. Escape them so `customer=50%` means a literal "50%".
3. Add `GET /invoices?limit=20&after_id=...` pagination with parameters only, and a test.
