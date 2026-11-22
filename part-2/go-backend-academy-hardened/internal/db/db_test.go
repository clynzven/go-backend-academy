package db

import (
	"context"
	"os"
	"testing"
	"testing/fstest"
)

// testURL returns TEST_DATABASE_URL, or skips the test when it isn't set,
// so `go test ./...` still passes on a machine without Postgres.
func testURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; start Postgres with docker compose up -d")
	}
	return url
}

func TestMigrateRunsEachFileOnce(t *testing.T) {
	ctx := context.Background()
	pool, err := Open(ctx, testURL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// A throwaway schema keeps this test away from the real tables.
	if _, err := pool.ExecContext(ctx, `DROP TABLE IF EXISTS migrate_demo; DELETE FROM schema_migrations WHERE version LIKE 'zz_%'`); err != nil {
		t.Log("first run: schema_migrations does not exist yet")
	}
	files := fstest.MapFS{"zz_001_demo.sql": {Data: []byte(`CREATE TABLE migrate_demo (id INT)`)}}
	defer pool.ExecContext(ctx, `DROP TABLE IF EXISTS migrate_demo; DELETE FROM schema_migrations WHERE version LIKE 'zz_%'`)

	first, err := Migrate(ctx, pool, files)
	if err != nil || len(first) != 1 {
		t.Fatalf("first Migrate = %v, %v; want 1 file applied", first, err)
	}
	second, err := Migrate(ctx, pool, files)
	if err != nil || len(second) != 0 {
		t.Fatalf("second Migrate = %v, %v; want nothing applied", second, err)
	}
}

func TestOpenFailsFastWithABadURL(t *testing.T) {
	_, err := Open(context.Background(), "postgres://nobody:wrong@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	if err == nil {
		t.Fatal("want an error for an unreachable database")
	}
}
