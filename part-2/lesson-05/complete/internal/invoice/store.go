package invoice

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// Store keeps invoices in PostgreSQL. Every query filters by tenant, and every
// value travels as a parameter ($1, $2), never inside the SQL text.
type Store struct {
	db      *sql.DB
	Charger Charger // takes the money; a fake payment provider unless replaced
}

// NewStore returns a store that uses the connection pool db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db, Charger: FakeProvider{Latency: 20 * time.Millisecond}}
}

// summarySQL lists invoices with their totals, computed by Postgres.
const summarySQL = `
SELECT i.id, i.customer, i.created_at,
       COALESCE((SELECT SUM(l.quantity * l.unit_cents) FROM invoice_lines l WHERE l.invoice_id = i.id), 0),
       i.status
FROM invoices i`

// Create saves an invoice and its lines in ONE transaction: all of it, or
// nothing. A crash halfway can never leave an invoice without its lines.
func (s *Store) Create(ctx context.Context, tenant string, req CreateRequest) (Invoice, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Invoice{}, fmt.Errorf("create invoice: %w", err)
	}
	defer tx.Rollback() // undoes everything unless Commit succeeded

	inv := Invoice{Customer: req.Customer, Lines: req.Lines, TotalCents: Total(req.Lines), Status: "open"}
	err = tx.QueryRowContext(ctx,
		`INSERT INTO invoices (tenant_id, customer) VALUES ($1, $2) RETURNING id, created_at`,
		tenant, req.Customer).Scan(&inv.ID, &inv.CreatedAt)
	if err != nil {
		return Invoice{}, fmt.Errorf("create invoice: %w", err)
	}
	for _, l := range req.Lines {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO invoice_lines (invoice_id, description, quantity, unit_cents) VALUES ($1, $2, $3, $4)`,
			inv.ID, l.Description, l.Quantity, l.UnitCents)
		if err != nil {
			return Invoice{}, fmt.Errorf("create invoice line: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Invoice{}, fmt.Errorf("create invoice: %w", err)
	}
	inv.CreatedAt = inv.CreatedAt.UTC()
	return inv, nil
}

// List returns one tenant's invoices with totals, newest first.
func (s *Store) List(ctx context.Context, tenant string) ([]Invoice, error) {
	return s.query(ctx, summarySQL+` WHERE i.tenant_id = $1 ORDER BY i.id DESC`, tenant)
}

// Search finds this tenant's invoices whose customer contains q (any case).
// The text the user typed is a PARAMETER: Postgres treats it as data only, so
// a quote in it can never end the string and change the query.
func (s *Store) Search(ctx context.Context, tenant, q string) ([]Invoice, error) {
	return s.query(ctx,
		summarySQL+` WHERE i.tenant_id = $1 AND i.customer ILIKE '%' || $2 || '%' ORDER BY i.id DESC`,
		tenant, q)
}

// Get returns one invoice with its lines, if it belongs to this tenant.
func (s *Store) Get(ctx context.Context, tenant string, id int64) (Invoice, error) {
	found, err := s.query(ctx, summarySQL+` WHERE i.tenant_id = $1 AND i.id = $2`, tenant, id)
	if err != nil {
		return Invoice{}, err
	}
	if len(found) == 0 {
		return Invoice{}, fmt.Errorf("invoice %d %w", id, apperr.ErrNotFound)
	}
	inv := found[0]

	rows, err := s.db.QueryContext(ctx,
		`SELECT description, quantity, unit_cents FROM invoice_lines WHERE invoice_id = $1 ORDER BY id`, id)
	if err != nil {
		return Invoice{}, fmt.Errorf("get invoice lines: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var l Line
		if err := rows.Scan(&l.Description, &l.Quantity, &l.UnitCents); err != nil {
			return Invoice{}, fmt.Errorf("get invoice lines: %w", err)
		}
		inv.Lines = append(inv.Lines, l)
	}
	return inv, rows.Err()
}

// Delete removes an invoice; its lines go with it (ON DELETE CASCADE).
func (s *Store) Delete(ctx context.Context, tenant string, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM invoices WHERE tenant_id = $1 AND id = $2`, tenant, id)
	if err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	} else if n == 0 {
		return fmt.Errorf("invoice %d %w", id, apperr.ErrNotFound)
	}
	return nil
}

// query runs a summarySQL query and scans the rows.
func (s *Store) query(ctx context.Context, query string, args ...any) ([]Invoice, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query invoices: %w", err)
	}
	defer rows.Close()

	invoices := []Invoice{}
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(&inv.ID, &inv.Customer, &inv.CreatedAt, &inv.TotalCents, &inv.Status); err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		inv.CreatedAt = inv.CreatedAt.UTC()
		invoices = append(invoices, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query invoices: %w", err)
	}
	return invoices, nil
}
