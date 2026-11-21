package task

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// do sends one request to a mux with the task routes, as tenant (if not "").
func do(t *testing.T, mux *http.ServeMux, method, target, tenant, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if tenant != "" {
		req.Header.Set("X-Tenant-ID", tenant)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(NewStore()).Register(mux)
	return mux
}

func TestCreateGetAndUpdate(t *testing.T) {
	mux := newMux()

	if rec := do(t, mux, "POST", "/tasks", "acme", `{"title":"Buy milk"}`); rec.Code != 201 {
		t.Fatalf("POST: status = %d, want 201 (%s)", rec.Code, rec.Body)
	}
	if rec := do(t, mux, "GET", "/tasks/1", "acme", ""); rec.Code != 200 {
		t.Fatalf("GET /tasks/1: status = %d, want 200", rec.Code)
	}

	rec := do(t, mux, "PATCH", "/tasks/1", "acme", `{"done":true}`)
	var got Task
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != 200 {
		t.Fatalf("PATCH: status %d, err %v (%s)", rec.Code, err, rec.Body)
	}
	if !got.Done || got.Title != "Buy milk" {
		t.Fatalf("PATCH done only: got %+v, want done with the same title", got)
	}
}

func TestTenantsAreSeparate(t *testing.T) {
	mux := newMux()
	do(t, mux, "POST", "/tasks", "acme", `{"title":"Acme secret"}`)

	if rec := do(t, mux, "GET", "/tasks/1", "globex", ""); rec.Code != 404 {
		t.Fatalf("other tenant GET /tasks/1: status = %d, want 404", rec.Code)
	}
	var list []Task
	_ = json.Unmarshal(do(t, mux, "GET", "/tasks", "globex", "").Body.Bytes(), &list)
	if len(list) != 0 {
		t.Fatalf("other tenant sees %d tasks, want 0", len(list))
	}
}

func TestBadRequests(t *testing.T) {
	tests := []struct {
		name, method, target, tenant, body string
		want                               int
	}{
		{"no tenant header", "GET", "/tasks", "", "", 400},
		{"id is not a number", "GET", "/tasks/abc", "acme", "", 400},
		{"unknown task", "GET", "/tasks/42", "acme", "", 404},
		{"blank title", "POST", "/tasks", "acme", `{"title":"  "}`, 400},
		{"cannot set id", "POST", "/tasks", "acme", `{"title":"x","id":9}`, 400},
		{"wrong method", "DELETE", "/tasks/1", "acme", "", 405},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, newMux(), tc.method, tc.target, tc.tenant, tc.body)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tc.want, rec.Body)
			}
		})
	}
}

// 100 requests at once, each on its own goroutine, like a real server.
// Run with -race: without the mutex this test reports a data race.
func TestConcurrentCreates(t *testing.T) {
	store := NewStore()
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			do(t, mux, "POST", "/tasks", "acme", `{"title":"task"}`)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	if n := len(store.List("acme")); n != 100 {
		t.Fatalf("got %d tasks, want 100", n)
	}
}
