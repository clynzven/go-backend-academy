package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clynzven/go-backend-academy/internal/task"
)

var secret = []byte("unit-test-secret-0123456789abcdef0123")

func token(t *testing.T, tenant string) string {
	t.Helper()
	tok, err := Sign(Claims{UserID: 1, Tenant: tenant, Expiry: time.Now().Add(time.Hour).Unix()}, secret)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestTokenRoundTripAndTampering(t *testing.T) {
	tok := token(t, "acme")
	c, err := Verify(tok, secret, time.Now())
	if err != nil || c.Tenant != "acme" {
		t.Fatalf("Verify = %+v, %v; want tenant acme", c, err)
	}

	// Swap the claims for globex's, keep the old signature.
	parts := strings.Split(tok, ".")
	forged := parts[0] + "." + b64.EncodeToString([]byte(`{"sub":1,"tenant":"globex","exp":9999999999}`)) + "." + parts[2]
	if _, err := Verify(forged, secret, time.Now()); err == nil {
		t.Fatal("a token with edited claims was accepted")
	}
	if _, err := Verify(tok, secret, time.Now().Add(31*24*time.Hour)); err == nil {
		t.Fatal("an expired token was accepted")
	}
}

// TestHeaderCannotChooseTheTenant is the BOLA attack as a test: a user of
// acme asks for globex's data with X-Tenant-ID. The tenant must come from
// the token only.
func TestHeaderCannotChooseTheTenant(t *testing.T) {
	store := task.NewMemoryStore()
	mux := http.NewServeMux()
	task.NewHandler(store).Register(mux)
	h := NewService(nil, secret).Middleware(mux)

	add := func(tenant, title string) {
		req := httptest.NewRequest("POST", "/tasks", strings.NewReader(`{"title":"`+title+`"}`))
		req.Header.Set("Authorization", "Bearer "+token(t, tenant))
		req.Header.Set("X-Tenant-ID", tenant) // an honest client: header and token agree
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 201 {
			t.Fatalf("create %s: %d %s", title, rec.Code, rec.Body)
		}
	}
	add("acme", "Acme plan")
	add("globex", "Globex secret plan")

	req := httptest.NewRequest("GET", "/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token(t, "acme"))
	req.Header.Set("X-Tenant-ID", "globex") // the attack
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var got []task.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if len(got) != 1 || got[0].Title != "Acme plan" {
		t.Fatalf("acme's token + X-Tenant-ID: globex returned %+v; want only acme's task", got)
	}

	// No token at all: the header alone must not be enough.
	req = httptest.NewRequest("GET", "/tasks", nil)
	req.Header.Set("X-Tenant-ID", "globex")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("X-Tenant-ID without a token: status %d, want 401", rec.Code)
	}
}
