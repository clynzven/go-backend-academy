package webhook

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/clynzven/go-backend-academy/internal/db"
	"github.com/clynzven/go-backend-academy/migrations"
)

var whSecret = []byte("whsec_unit-test-0123456789abcdef0123")

// newInbound connects to TEST_DATABASE_URL in its own schema (so other
// packages' tests can't touch its invoices), with one open invoice: id 1.
func newInbound(t *testing.T) (*sql.DB, *http.ServeMux) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; start Postgres with docker compose up -d")
	}
	ctx := context.Background()
	admin, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS webhook_test`); err != nil {
		t.Fatal(err)
	}
	admin.Close()
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	pool, err := db.Open(ctx, dsn+sep+"search_path=webhook_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	if _, err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.ExecContext(ctx,
		`TRUNCATE invoices, invoice_lines, payments, processed_events RESTART IDENTITY`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.ExecContext(ctx, `INSERT INTO invoices (tenant_id, customer) VALUES ('acme', 'Acme Corp')`); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewInbound(pool, whSecret).Register(mux)
	return pool, mux
}

func send(mux *http.ServeMux, body, signature string) int {
	req := httptest.NewRequest("POST", "/webhooks/payments", strings.NewReader(body))
	if signature != "" {
		req.Header.Set("Webhook-Signature", signature)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec.Code
}

const paid = `{"event_id":"evt_42","type":"payment.succeeded","tenant":"acme","invoice_id":1,"amount_cents":12900}`

func counts(t *testing.T, pool *sql.DB) (status string, payments int) {
	t.Helper()
	if err := pool.QueryRow(`SELECT status, (SELECT count(*) FROM payments) FROM invoices WHERE id = 1`).
		Scan(&status, &payments); err != nil {
		t.Fatal(err)
	}
	return status, payments
}

// TestForgedEventIsRejected: anyone can POST to the webhook URL. Without the
// provider's signature, nothing may change.
func TestForgedEventIsRejected(t *testing.T) {
	pool, mux := newInbound(t)
	if code := send(mux, paid, ""); code != http.StatusUnauthorized {
		t.Fatalf("unsigned event: %d, want 401", code)
	}
	if code := send(mux, paid, Sign([]byte("a-guessed-secret-0123456789abcdef"), []byte(paid), time.Now())); code != 401 {
		t.Fatalf("event signed with the wrong secret: %d, want 401", code)
	}
	if status, n := counts(t, pool); status != "open" || n != 0 {
		t.Fatalf("after forged events: invoice %s with %d payments, want open with 0", status, n)
	}
}

// TestReplayedEventIsIgnored: a real, signed event sent again (an attacker
// replaying it, or the provider retrying) is applied once.
func TestReplayedEventIsIgnored(t *testing.T) {
	pool, mux := newInbound(t)
	sig := Sign(whSecret, []byte(paid), time.Now())
	for i := 0; i < 3; i++ {
		if code := send(mux, paid, sig); code != http.StatusOK {
			t.Fatalf("send %d: %d, want 200", i+1, code)
		}
	}
	if status, n := counts(t, pool); status != "paid" || n != 1 {
		t.Fatalf("after 3 copies: invoice %s with %d payments, want paid with 1", status, n)
	}
	if code := send(mux, paid, Sign(whSecret, []byte(paid), time.Now().Add(-10*time.Minute))); code != 401 {
		t.Fatalf("a copy signed 10 minutes ago: %d, want 401", code)
	}
}
