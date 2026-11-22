package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/clynzven/go-backend-academy/internal/auth"
	"github.com/clynzven/go-backend-academy/internal/db"
	"github.com/clynzven/go-backend-academy/migrations"
)

// Integration tests run the real app (router, middleware, stores) against a
// real PostgreSQL. They skip unless TEST_DATABASE_URL is set.
func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; run: make test-integration")
	}
	return url
}

// newTestServer starts the whole app on a random port with empty tables.
// It uses its own Postgres schema, so the other packages' tests (which run at
// the same time) never see or truncate its tables.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()
	pool, err := db.Open(ctx, withSchema(t, testDatabaseURL(t), "integration"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	if _, err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.ExecContext(ctx, `TRUNCATE tasks, invoices, invoice_lines RESTART IDENTITY`); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(newHandler(pool, slog.New(slog.NewJSONHandler(io.Discard, nil)), false, testSecret))
	t.Cleanup(srv.Close)
	return srv
}

// withSchema creates schema if needed and returns url with search_path set to it.
func withSchema(t *testing.T, url, schema string) string {
	t.Helper()
	pool, err := db.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(`CREATE SCHEMA IF NOT EXISTS ` + schema); err != nil {
		t.Fatal(err)
	}
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	return url + sep + "search_path=" + schema
}

// testSecret signs the tokens the integration tests send.
var testSecret = []byte("integration-test-secret-0123456789abcdef")

// bearer returns an Authorization header value for a user of tenant.
func bearer(t *testing.T, tenant string) string {
	t.Helper()
	token, err := auth.Sign(auth.Claims{UserID: 1, Tenant: tenant, Expiry: time.Now().Add(time.Hour).Unix()}, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + token
}

// call sends one request as tenant and decodes a JSON answer into out (if not nil).
func call(t *testing.T, srv *httptest.Server, method, path, tenant, body string, out any) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", bearer(t, tenant))
	req.Header.Set("X-Tenant-ID", tenant) // Part 1's header: ignored now
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("%s %s: %v in %s", method, path, err, data)
		}
	}
	return res
}

func TestTaskLifecycle(t *testing.T) {
	srv := newTestServer(t)

	var created struct{ ID int64 }
	if res := call(t, srv, "POST", "/tasks", "acme", `{"title":"Ship lesson 8"}`, &created); res.StatusCode != 201 {
		t.Fatalf("create: %d", res.StatusCode)
	}
	if res := call(t, srv, "PATCH", "/tasks/1", "acme", `{"done":true}`, nil); res.StatusCode != 200 {
		t.Fatalf("update: %d", res.StatusCode)
	}
	var list []struct{ Done bool }
	call(t, srv, "GET", "/tasks", "acme", "", &list)
	if len(list) != 1 || !list[0].Done {
		t.Fatalf("list = %+v, want one done task", list)
	}
	if res := call(t, srv, "GET", "/tasks/1", "globex", "", nil); res.StatusCode != 404 {
		t.Fatalf("other tenant: %d, want 404", res.StatusCode)
	}
	if res := call(t, srv, "DELETE", "/tasks/1", "acme", "", nil); res.StatusCode != 204 {
		t.Fatalf("delete: %d", res.StatusCode)
	}
}

func TestInvoiceLifecycle(t *testing.T) {
	srv := newTestServer(t)

	body := `{"customer":"Clynz Studio","lines":[{"description":"Logo design","quantity":2,"unit_cents":150000}]}`
	if res := call(t, srv, "POST", "/invoices", "acme", body, nil); res.StatusCode != 201 {
		t.Fatalf("create: %d", res.StatusCode)
	}
	var inv struct {
		TotalCents int64 `json:"total_cents"`
		Lines      []struct{}
	}
	call(t, srv, "GET", "/invoices/1", "acme", "", &inv)
	if inv.TotalCents != 300000 || len(inv.Lines) != 1 {
		t.Fatalf("get = %+v", inv)
	}
	var found []struct{}
	call(t, srv, "GET", "/invoices/search?customer=clynz", "acme", "", &found)
	if len(found) != 1 {
		t.Fatalf("search found %d, want 1", len(found))
	}
}

func TestEveryResponseHasARequestID(t *testing.T) {
	srv := newTestServer(t)
	res := call(t, srv, "GET", "/no-such-page", "acme", "", nil)
	if res.StatusCode != 404 || len(res.Header.Get("X-Request-ID")) != 16 {
		t.Fatalf("status %d, X-Request-ID %q", res.StatusCode, res.Header.Get("X-Request-ID"))
	}
}

// Ctrl+C in a test: start the real program, begin a slow request, cancel the
// context, and check that the request still finishes before run returns.
func TestGracefulShutdownFinishesRequestsInFlight(t *testing.T) {
	url := testDatabaseURL(t)
	env := map[string]string{"DATABASE_URL": withSchema(t, url, "integration"), "ADDR": "127.0.0.1:18080", "DEBUG_ROUTES": "1"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var logs bytes.Buffer
	runErr := make(chan error, 1)
	go func() { runErr <- run(ctx, func(k string) string { return env[k] }, &logs) }()
	waitUntilUp(t, "http://127.0.0.1:18080/health")

	slow := make(chan string, 1)
	go func() {
		res, err := http.Get("http://127.0.0.1:18080/debug/slow?seconds=2")
		if err != nil {
			slow <- "error: " + err.Error()
			return
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		slow <- string(body)
	}()
	time.Sleep(300 * time.Millisecond) // let the slow request start
	cancel()                           // the same as pressing Ctrl+C

	if got := <-slow; got != "finished after 2s\n" {
		t.Fatalf("slow request got %q, want it to finish", got)
	}
	if err := <-runErr; err != nil {
		t.Fatalf("run returned %v", err)
	}
	if !strings.Contains(logs.String(), "database closed") {
		t.Fatalf("missing shutdown logs:\n%s", logs.String())
	}
}

func waitUntilUp(t *testing.T, url string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if res, err := http.Get(url); err == nil {
			res.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server at %s never came up", url)
}
