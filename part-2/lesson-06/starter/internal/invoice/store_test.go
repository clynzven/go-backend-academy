package invoice

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/clynzven/go-backend-academy/internal/apperr"
	"github.com/clynzven/go-backend-academy/internal/db"
	"github.com/clynzven/go-backend-academy/internal/httpx"
	"github.com/clynzven/go-backend-academy/migrations"
)

// newStore connects to TEST_DATABASE_URL with empty invoice tables, or skips.
func newStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; start Postgres with docker compose up -d")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	if _, err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.ExecContext(ctx, `TRUNCATE invoices, invoice_lines, payments RESTART IDENTITY`); err != nil {
		t.Fatal(err)
	}
	return NewStore(pool)
}

func create(t *testing.T, s *Store, tenant, customer string) Invoice {
	t.Helper()
	inv, err := s.Create(context.Background(), tenant, CreateRequest{
		Customer: customer,
		Lines:    []Line{{Description: "Logo design", Quantity: 2, UnitCents: 150000}, {Description: "Hosting", Quantity: 1, UnitCents: 99900}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func TestCreateGetDelete(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	inv := create(t, s, "acme", "Clynz Studio")

	got, err := s.Get(ctx, "acme", inv.ID)
	if err != nil || got.TotalCents != 399900 || len(got.Lines) != 2 {
		t.Fatalf("Get = %+v, %v; want total 399900 and 2 lines", got, err)
	}
	if _, err := s.Get(ctx, "globex", inv.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("cross-tenant Get: err = %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx, "acme", inv.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "acme", inv.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("second Delete: err = %v, want ErrNotFound", err)
	}
}

// The regression test for lesson 6's bug: a quote in the search text must be
// data, never SQL. With string-built SQL this payload returned every
// tenant's invoices.
func TestSearchIsNotInjectable(t *testing.T) {
	s := newStore(t)
	create(t, s, "acme", "Acme Corp")
	create(t, s, "globex", "Globex Secret Client")

	mux := http.NewServeMux()
	NewHandler(s).Register(mux)
	search := func(q string) []Invoice {
		req := httptest.NewRequest("GET", "/invoices/search?customer="+url.QueryEscape(q), nil)
		req = req.WithContext(httpx.WithIdentity(req.Context(), httpx.Identity{UserID: 1, Tenant: "acme"}))
		req.Header.Set("X-Tenant-ID", "acme")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("search %q: status %d, body %s", q, rec.Code, rec.Body)
		}
		var found []Invoice
		if err := json.Unmarshal(rec.Body.Bytes(), &found); err != nil {
			t.Fatal(err)
		}
		return found
	}

	if found := search("acme"); len(found) != 1 {
		t.Fatalf("normal search found %d invoices, want 1", len(found))
	}
	if found := search("' OR 1=1 --"); len(found) != 0 {
		t.Fatalf("injection payload found %d invoices (%+v), want 0", len(found), found)
	}
}

// A failed line insert must roll back the whole invoice.
func TestCreateIsAllOrNothing(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	// quantity 0 passes Go (we skip Validate here) but breaks the CHECK in Postgres.
	_, err := s.Create(ctx, "acme", CreateRequest{Customer: "Half done", Lines: []Line{
		{Description: "ok", Quantity: 1, UnitCents: 100},
		{Description: "bad", Quantity: 0, UnitCents: 100},
	}})
	if err == nil {
		t.Fatal("want an error from the CHECK constraint")
	}
	if list, _ := s.List(ctx, "acme"); len(list) != 0 {
		t.Fatalf("found %d invoices after a failed create, want 0", len(list))
	}
}
