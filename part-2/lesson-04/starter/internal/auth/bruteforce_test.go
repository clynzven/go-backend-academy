package auth

import (
	"net/http"
	"testing"
)

// TestBruteForceIsStopped: 5 wrong passwords lock the account, and the lock
// holds even for the right password; per IP, an 11th attempt is refused.
func TestBruteForceIsStopped(t *testing.T) {
	_, mux := newService(t)
	login := func(email, pw string) int {
		code, _ := post(t, mux, "/login", `{"email":"`+email+`","password":"`+pw+`"}`)
		return code
	}
	for i := 1; i <= 5; i++ {
		if code := login("alice@acme.test", "guess"); code != http.StatusUnauthorized {
			t.Fatalf("wrong password %d: %d, want 401", i, code)
		}
	}
	if code := login("alice@acme.test", "correct horse battery"); code != http.StatusTooManyRequests {
		t.Fatalf("right password on a locked account: %d, want 429", code)
	}
	for i := 7; i <= 10; i++ {
		login("someone@else.test", "guess")
	}
	if code := login("another@else.test", "guess"); code != http.StatusTooManyRequests {
		t.Fatalf("11th attempt from one IP: %d, want 429", code)
	}
}
