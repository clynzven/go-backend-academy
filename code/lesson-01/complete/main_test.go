package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	tests := []struct {
		name, method, target string
		wantStatus           int
		wantBody             string
	}{
		{"health says ok", "GET", "/health", 200, "ok\n"},
		{"hello default", "GET", "/hello", 200, "Hello, Gopher!\n"},
		{"hello with name", "GET", "/hello?name=Clynz", 200, "Hello, Clynz!\n"},
		{"home echoes request", "POST", "/tasks", 200, "You asked for: POST /tasks\n"},
	}
	mux := newRouter()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, nil)
			rec := httptest.NewRecorder() // a fake ResponseWriter that records everything
			mux.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", rec.Body.String(), tc.wantBody)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
				t.Fatalf("Content-Type = %q, want text/plain", ct)
			}
		})
	}
}
