-- 002: invoices and their lines. Money is whole cents in a BIGINT, never a float.
CREATE TABLE invoices (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  TEXT        NOT NULL,
    customer   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX invoices_tenant_idx ON invoices (tenant_id, id);

CREATE TABLE invoice_lines (
    id          BIGSERIAL PRIMARY KEY,
    invoice_id  BIGINT NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    description TEXT   NOT NULL,
    quantity    INT    NOT NULL CHECK (quantity > 0),
    unit_cents  BIGINT NOT NULL CHECK (unit_cents >= 0)
);
CREATE INDEX invoice_lines_invoice_idx ON invoice_lines (invoice_id);
