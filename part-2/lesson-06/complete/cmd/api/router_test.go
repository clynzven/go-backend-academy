package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clynzven/go-backend-academy/internal/task"
)

func TestRoutes(t *testing.T) {
	tests := []struct {
		name, method, target string
		wantStatus           int
		wantBody             string
	}{
		{"home is exact", "GET", "/", 200, "Task & Invoice API"},
		{"health says ok", "GET", "/health", 200, "ok\n"},
		{"hello with name", "GET", "/hello?name=Clynz", 200, "Hello, Clynz!\n"},
		{"unknown path is 404 JSON", "GET", "/this-does-not-exist", 404, `"error": "not found"`},
		{"wrong method is 405 JSON", "POST", "/health", 405, `"error": "method not allowed"`},
	}
	mux := newRouter(fakeDB{}, task.NewHandler(task.NewMemoryStore()))
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.target, nil))
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", rec.Body.String(), tc.wantBody)
			}
		})
	}
}

// Router errors are JSON now, but a 405 must still say which methods ARE allowed.
func TestMethodNotAllowedListsAllowedMethods(t *testing.T) {
	mux := newRouter(fakeDB{}, task.NewHandler(task.NewMemoryStore()))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("DELETE", "/tasks", nil))
	if got := rec.Header().Get("Allow"); !strings.Contains(got, "GET") || !strings.Contains(got, "POST") {
		t.Fatalf("Allow = %q, want GET and POST", got)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON", ct)
	}
}

// fakeDB stands in for the database in router tests.
type fakeDB struct{ err error }

func (f fakeDB) PingContext(context.Context) error { return f.err }

func TestHealthReportsADownDatabase(t *testing.T) {
	mux := newRouter(fakeDB{err: errors.New("connection refused")}, task.NewHandler(task.NewMemoryStore()))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), "database unavailable") {
		t.Fatalf("status = %d, body %q; want 503 database unavailable", rec.Code, rec.Body)
	}
}
