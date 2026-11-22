package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestArgon2idRoundTrip(t *testing.T) {
	h1, h2 := HashPassword("correct horse battery"), HashPassword("correct horse battery")
	if h1 == h2 {
		t.Fatal("two hashes of the same password are equal: the salt is missing")
	}
	if !strings.HasPrefix(h1, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("hash = %s, want an argon2id PHC string", h1)
	}
	if ok, upgrade := checkPassword("correct horse battery", h1); !ok || upgrade {
		t.Fatalf("right password: ok=%v upgrade=%v", ok, upgrade)
	}
	if ok, _ := checkPassword("Correct horse battery", h1); ok {
		t.Fatal("wrong password accepted")
	}
}

// TestOldHashesAreUpgraded: a Lesson 2 SHA-256 hash still logs in once, and
// is replaced by argon2id on the spot.
func TestOldHashesAreUpgraded(t *testing.T) {
	s, mux := newService(t)
	sum := sha256.Sum256([]byte("old password"))
	if _, err := s.db.Exec(`INSERT INTO users (tenant_id, email, password_hash) VALUES ('acme', 'old@acme.test', $1)`,
		hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	if code, _ := post(t, mux, "/login", `{"email":"old@acme.test","password":"old password"}`); code != 200 {
		t.Fatalf("login with an old hash: %d", code)
	}
	var stored string
	if err := s.db.QueryRowContext(context.Background(),
		`SELECT password_hash FROM users WHERE email = 'old@acme.test'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, "$argon2id$") {
		t.Fatalf("hash after login = %s, want argon2id", stored)
	}
}

// BenchmarkPasswordHashes shows why the hash matters for a stolen database:
// every guess an attacker makes costs one of these.
func BenchmarkPasswordHashes(b *testing.B) {
	b.Run("sha256", func(b *testing.B) {
		for b.Loop() {
			sha256.Sum256([]byte("Globex-2026!"))
		}
	})
	b.Run("argon2id", func(b *testing.B) {
		for b.Loop() {
			HashPassword("Globex-2026!")
		}
	})
}
