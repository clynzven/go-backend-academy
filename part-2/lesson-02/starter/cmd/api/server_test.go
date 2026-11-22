package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// startServer runs newServer on a free local port with the given limits.
func startServer(t *testing.T, l limits) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	srv := newServer(ln.Addr().String(), ok, l)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return ln.Addr().String()
}

// TestSlowHeadersAreCut is Slowloris in one connection: headers that never
// finish. The server must hang up once ReadHeader has passed.
func TestSlowHeadersAreCut(t *testing.T) {
	l := defaultLimits
	l.ReadHeader = 200 * time.Millisecond // short, so the test is fast
	addr := startServer(t, l)

	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	fmt.Fprint(c, "GET / HTTP/1.1\r\nHost: localhost\r\n") // no blank line: never finished

	start := time.Now()
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, err = io.ReadAll(c) // returns when the server closes the connection
	if err != nil {
		t.Fatalf("the server kept the slow connection open: %v", err)
	}
	if took := time.Since(start); took > time.Second {
		t.Fatalf("slow connection cut after %s, want about %s", took, l.ReadHeader)
	}
}

// TestNormalRequestsStillWork makes sure the limits don't hurt real clients.
func TestNormalRequestsStillWork(t *testing.T) {
	addr := startServer(t, defaultLimits)
	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

// TestEveryLimitIsSet fails if someone deletes a timeout: zero means "forever".
func TestEveryLimitIsSet(t *testing.T) {
	srv := newServer("127.0.0.1:0", http.NotFoundHandler(), defaultLimits)
	checks := map[string]bool{
		"ReadHeaderTimeout": srv.ReadHeaderTimeout > 0,
		"ReadTimeout":       srv.ReadTimeout > 0,
		"WriteTimeout":      srv.WriteTimeout > 0,
		"IdleTimeout":       srv.IdleTimeout > 0,
		"MaxHeaderBytes":    srv.MaxHeaderBytes > 0 && srv.MaxHeaderBytes <= 64<<10,
	}
	for name, ok := range checks {
		if !ok {
			t.Errorf("%s is not set (or too large)", name)
		}
	}
}
