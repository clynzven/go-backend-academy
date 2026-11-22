package invoice

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// Payment is money received for one invoice.
type Payment struct {
	ID          int64     `json:"id"`
	InvoiceID   int64     `json:"invoice_id"`
	AmountCents int64     `json:"amount_cents"`
	CreatedAt   time.Time `json:"created_at"`
}

// Charger takes money from the customer. In production this is a payment
// provider's API; in this course it's FakeProvider.
type Charger interface {
	Charge(ctx context.Context, tenant string, invoiceID, cents int64) error
}

// FakeProvider pretends to be a payment provider: every charge succeeds and
// takes Latency, like a real network call.
type FakeProvider struct{ Latency time.Duration }

func (f FakeProvider) Charge(ctx context.Context, _ string, _, _ int64) error {
	select {
	case <-time.After(f.Latency):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Pay charges the customer for an open invoice, exactly once.
//
// The claim is one atomic UPDATE: only the request that flips status from
// 'open' to 'paid' gets a row back. Postgres locks the row while it updates,
// so 50 requests at once queue up, and every one after the first sees 'paid'
// and changes nothing. If the charge fails, the transaction rolls back and
// the invoice is open again.
//
// key is the client's Idempotency-Key ("" for none). The same key always gets
// the same payment back, so a retry after a timeout can't pay twice.
func (s *Store) Pay(ctx context.Context, tenant string, id int64, key string) (p Payment, replayed bool, err error) {
	// TODO: two requests can both read "open" before either writes "paid".
	// Claim the invoice with ONE conditional UPDATE ... WHERE status = 'open'
	// inside a transaction, and honour the Idempotency-Key (key).
	inv, err := s.Get(ctx, tenant, id)
	if err != nil {
		return Payment{}, false, err
	}
	if inv.Status == "paid" {
		return Payment{}, false, fmt.Errorf("invoice %d %w", id, apperr.ErrAlreadyPaid)
	}
	if err := s.Charger.Charge(ctx, tenant, id, inv.TotalCents); err != nil {
		return Payment{}, false, fmt.Errorf("charge invoice %d: %w", id, err)
	}
	p = Payment{InvoiceID: id, AmountCents: inv.TotalCents}
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO payments (invoice_id, tenant_id, amount_cents) VALUES ($1, $2, $3) RETURNING id, created_at`,
		id, tenant, inv.TotalCents).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return Payment{}, false, fmt.Errorf("record payment: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE invoices SET status = 'paid' WHERE tenant_id = $1 AND id = $2`, tenant, id); err != nil {
		return Payment{}, false, fmt.Errorf("pay invoice: %w", err)
	}
	p.CreatedAt = p.CreatedAt.UTC()
	return p, false, nil
}

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// paymentByKey finds the payment made with an idempotency key. Inside Pay it
// runs on the open transaction: asking the pool for a second connection while
// holding one can deadlock when every connection is busy.
func paymentByKey(ctx context.Context, q queryer, tenant, key string) (Payment, error) {
	var p Payment
	err := q.QueryRowContext(ctx,
		`SELECT id, invoice_id, amount_cents, created_at FROM payments WHERE tenant_id = $1 AND idempotency_key = $2`,
		tenant, key).Scan(&p.ID, &p.InvoiceID, &p.AmountCents, &p.CreatedAt)
	p.CreatedAt = p.CreatedAt.UTC()
	return p, err
}

// Payments lists the payments of one invoice.
func (s *Store) Payments(ctx context.Context, tenant string, id int64) ([]Payment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, invoice_id, amount_cents, created_at FROM payments
		  WHERE tenant_id = $1 AND invoice_id = $2 ORDER BY id`, tenant, id)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()
	list := []Payment{}
	for rows.Next() {
		var p Payment
		if err := rows.Scan(&p.ID, &p.InvoiceID, &p.AmountCents, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("list payments: %w", err)
		}
		p.CreatedAt = p.CreatedAt.UTC()
		list = append(list, p)
	}
	return list, rows.Err()
}
