package webhook

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// TestPreviewCannotReachInternalAddresses is the SSRF attack as a test: an
// "internal admin" server on loopback, and every spelling of its address.
func TestPreviewCannotReachInternalAddresses(t *testing.T) {
	var hits atomic.Int64
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte(`{"db_password":"s3cret"}`))
	}))
	defer internal.Close()
	port := internal.URL[strings.LastIndex(internal.URL, ":")+1:]

	mux := http.NewServeMux()
	NewHandler(nil).Register(mux)
	for _, host := range []string{"127.0.0.1", "localhost", "0.0.0.0", "[::]", "[::ffff:127.0.0.1]"} {
		target := "http://" + host + ":" + port + "/admin/secrets"
		req := httptest.NewRequest("POST", "/webhooks/test", strings.NewReader(`{"url":"`+target+`"}`))
		req = req.WithContext(httpx.WithIdentity(req.Context(), httpx.Identity{UserID: 1, Tenant: "acme"}))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		// 422 when the dialer blocks it; [::] can also fail earlier (400) on a machine without IPv6.
		if rec.Code != http.StatusUnprocessableEntity && !(host == "[::]" && rec.Code == http.StatusBadRequest) {
			t.Errorf("%s: status %d, want 422 (%s)", target, rec.Code, strings.TrimSpace(rec.Body.String()))
		}
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the internal server was reached %d times", n)
	}
}
