package webhook

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestIsPublic(t *testing.T) {
	for addr, want := range map[string]bool{
		"8.8.8.8": true, "1.1.1.1": true, "2606:4700:4700::1111": true,
		"127.0.0.1": false, "::1": false, "0.0.0.0": false, "::": false, "::ffff:127.0.0.1": false,
		"10.0.0.5": false, "172.16.3.4": false, "192.168.1.1": false, "169.254.169.254": false,
		"100.64.0.1": false, "fd00::1": false, "fe80::1": false, "224.0.0.1": false,
	} {
		if got := isPublic(netip.MustParseAddr(addr)); got != want {
			t.Errorf("isPublic(%s) = %v, want %v", addr, got, want)
		}
	}
}

func TestSafeClientRefusesLoopback(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler()) // listens on 127.0.0.1
	defer srv.Close()
	_, err := SafeClient().Get(srv.URL)
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("GET %s: err = %v, want ErrBlocked", srv.URL, err)
	}
}
