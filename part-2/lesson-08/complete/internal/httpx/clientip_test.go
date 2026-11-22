package httpx

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	trusted, err := ParseTrusted("10.0.0.5, 192.168.0.0/16")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, remote, xff, want string
	}{
		{"direct client, no header", "203.0.113.7:5000", "", "203.0.113.7"},
		{"direct client lies in the header", "203.0.113.7:5000", "1.2.3.4", "203.0.113.7"},
		{"through our proxy", "10.0.0.5:4000", "198.51.100.9", "198.51.100.9"},
		{"through our proxy, client lies first", "10.0.0.5:4000", "1.2.3.4, 198.51.100.9", "198.51.100.9"},
		{"trusted range", "192.168.3.3:4000", "198.51.100.9", "198.51.100.9"},
		{"proxy without the header", "10.0.0.5:4000", "", "10.0.0.5"},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = c.remote
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := ClientIP(r, trusted); got != c.want {
			t.Errorf("%s: ClientIP = %s, want %s", c.name, got, c.want)
		}
	}
}
