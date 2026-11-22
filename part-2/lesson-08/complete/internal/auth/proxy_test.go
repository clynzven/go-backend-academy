package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

// attempt is one login through our proxy (127.0.0.1): the client sent its own
// X-Forwarded-For, and the proxy appended the real address after it.
func attempt(mux http.Handler, spoofed string) int {
	req := httptest.NewRequest("POST", "/login", strings.NewReader(`{"email":"nobody@else.test","password":"guess"}`))
	req.RemoteAddr = "127.0.0.1:41000"
	if spoofed != "" {
		req.Header.Set("X-Forwarded-For", spoofed+", 198.51.100.7")
	} else {
		req.Header.Set("X-Forwarded-For", "198.51.100.7")
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec.Code
}

// TestSpoofedForwardedForIsIgnored: a new fake address on every request must
// not give the attacker a fresh rate-limit bucket.
func TestSpoofedForwardedForIsIgnored(t *testing.T) {
	s, mux := newService(t)
	s.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	limited := 0
	for i := 0; i < 15; i++ {
		if attempt(mux, fmt.Sprintf("10.9.0.%d", i)) == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited != 5 {
		t.Fatalf("15 attempts with spoofed X-Forwarded-For: %d got 429, want 5 (10 allowed)", limited)
	}
}

// TestLimitIsSharedAcrossInstances: two copies of the API, one database. The
// attacker alternates between them; the limit is still 10 in total.
func TestLimitIsSharedAcrossInstances(t *testing.T) {
	s1, mux1 := newService(t)
	s2 := NewService(s1.db, secret)
	mux2 := http.NewServeMux()
	s2.Register(mux2)
	s1.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	s2.TrustedProxies = s1.TrustedProxies

	limited := 0
	for i := 0; i < 16; i++ {
		mux := mux1
		if i%2 == 1 {
			mux = mux2
		}
		if attempt(mux, "") == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited != 6 {
		t.Fatalf("16 attempts over 2 instances: %d got 429, want 6 (10 allowed in total)", limited)
	}
}
