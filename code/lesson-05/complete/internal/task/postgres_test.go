package task

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/clynzven/go-backend-academy/internal/apperr"
	"github.com/clynzven/go-backend-academy/internal/db"
	"github.com/clynzven/go-backend-academy/migrations"
)

// newPostgresStore connects to TEST_DATABASE_URL with a fresh tasks table, or
// skips the test when it isn't set.
func newPostgresStore(t *testing.T) *PostgresStore {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; start Postgres with docker compose up -d")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	if _, err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.ExecContext(ctx, `TRUNCATE tasks RESTART IDENTITY`); err != nil {
		t.Fatal(err)
	}
	return NewPostgresStore(pool)
}

func TestPostgresStore(t *testing.T) {
	s := newPostgresStore(t)
	ctx := context.Background()

	added, err := s.Add(ctx, "acme", "Buy milk")
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "acme", added.ID)
	if err != nil || got.Title != "Buy milk" {
		t.Fatalf("Get = %+v, %v", got, err)
	}

	// The unique index refuses a second open "buy MILK" for the same tenant...
	if _, err := s.Add(ctx, "acme", "buy MILK"); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("duplicate Add: err = %v, want ErrConflict", err)
	}
	// ...but another tenant may use the same title.
	if _, err := s.Add(ctx, "globex", "Buy milk"); err != nil {
		t.Fatalf("other tenant Add: %v", err)
	}

	// Tenants can't see each other's tasks.
	if _, err := s.Get(ctx, "globex", added.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("cross-tenant Get: err = %v, want ErrNotFound", err)
	}

	// PATCH only what was sent: done changes, the title stays.
	done := true
	updated, err := s.Update(ctx, "acme", added.ID, nil, &done)
	if err != nil || !updated.Done || updated.Title != "Buy milk" {
		t.Fatalf("Update = %+v, %v", updated, err)
	}

	list, err := s.List(ctx, "acme")
	if err != nil || len(list) != 1 {
		t.Fatalf("List = %v, %v; want 1 task", list, err)
	}
}
