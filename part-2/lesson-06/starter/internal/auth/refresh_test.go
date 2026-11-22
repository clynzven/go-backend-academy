package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/clynzven/go-backend-academy/internal/db"
	"github.com/clynzven/go-backend-academy/migrations"
)

// newService connects to TEST_DATABASE_URL with empty user tables, or skips.
func newService(t *testing.T) (*Service, *http.ServeMux) {
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
	if _, err := pool.ExecContext(ctx, `TRUNCATE users, refresh_tokens RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	s := NewService(pool, secret)
	if _, err := s.AddUser(ctx, "acme", "alice@acme.test", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.Register(mux)
	return s, mux
}

func post(t *testing.T, mux *http.ServeMux, path, body string) (int, Tokens) {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", path, strings.NewReader(body)))
	var tok Tokens
	_ = json.Unmarshal(rec.Body.Bytes(), &tok)
	return rec.Code, tok
}

// TestStolenRefreshTokenIsDetected: the thief and the real user both hold the
// same refresh token. Whoever uses it second triggers reuse detection, and
// every token from that login stops working, the thief's new one included.
func TestStolenRefreshTokenIsDetected(t *testing.T) {
	_, mux := newService(t)
	code, first := post(t, mux, "/login", `{"email":"alice@acme.test","password":"correct horse battery"}`)
	if code != 200 || first.RefreshToken == "" {
		t.Fatalf("login: %d %+v", code, first)
	}

	code, thief := post(t, mux, "/refresh", `{"refresh_token":"`+first.RefreshToken+`"}`) // the thief, first
	if code != 200 {
		t.Fatalf("first refresh: %d", code)
	}
	if code, _ := post(t, mux, "/refresh", `{"refresh_token":"`+first.RefreshToken+`"}`); code != 401 { // Alice
		t.Fatalf("reusing a spent refresh token: %d, want 401", code)
	}
	if code, _ := post(t, mux, "/refresh", `{"refresh_token":"`+thief.RefreshToken+`"}`); code != 401 {
		t.Fatalf("the thief's new refresh token still works (%d): the family was not revoked", code)
	}
}

func TestLogoutRevokesRefreshTokens(t *testing.T) {
	_, mux := newService(t)
	_, tok := post(t, mux, "/login", `{"email":"alice@acme.test","password":"correct horse battery"}`)
	if code, _ := post(t, mux, "/logout", `{"refresh_token":"`+tok.RefreshToken+`"}`); code != 204 {
		t.Fatalf("logout: %d, want 204", code)
	}
	if code, _ := post(t, mux, "/refresh", `{"refresh_token":"`+tok.RefreshToken+`"}`); code != 401 {
		t.Fatalf("refresh after logout: %d, want 401", code)
	}
}
