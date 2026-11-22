-- 005: invoices can be paid, once. Each payment is a row; an idempotency key
-- (one per tenant) lets a client retry a payment without paying twice.
ALTER TABLE invoices ADD COLUMN status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'paid'));

CREATE TABLE payments (
    id              BIGSERIAL PRIMARY KEY,
    invoice_id      BIGINT      NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    tenant_id       TEXT        NOT NULL,
    amount_cents    BIGINT      NOT NULL CHECK (amount_cents >= 0),
    idempotency_key TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);
CREATE INDEX payments_invoice_idx ON payments (invoice_id);
