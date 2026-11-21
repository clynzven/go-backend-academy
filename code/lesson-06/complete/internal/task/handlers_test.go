package task

import (
	"context"
	"encoding/json"
	"fmt"
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
	NewHandler(NewMemoryStore()).Register(mux)
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

func TestDeleteAnswers204ThenNotFound(t *testing.T) {
	mux := newMux()
	do(t, mux, "POST", "/tasks", "acme", `{"title":"Short-lived"}`)

	if rec := do(t, mux, "DELETE", "/tasks/1", "acme", ""); rec.Code != 204 || rec.Body.Len() != 0 {
		t.Fatalf("DELETE: status = %d, body %q; want 204 and no body", rec.Code, rec.Body)
	}
	if rec := do(t, mux, "DELETE", "/tasks/1", "acme", ""); rec.Code != 404 {
		t.Fatalf("second DELETE: status = %d, want 404", rec.Code)
	}
}

func TestDuplicateOpenTaskIsAConflict(t *testing.T) {
	mux := newMux()
	do(t, mux, "POST", "/tasks", "acme", `{"title":"Buy milk"}`)

	rec := do(t, mux, "POST", "/tasks", "acme", `{"title":"buy MILK"}`)
	if rec.Code != 409 || !strings.Contains(rec.Body.String(), "already exists") {
		t.Fatalf("duplicate: status = %d, body %s; want 409 already exists", rec.Code, rec.Body)
	}
	// Another tenant may use the same title.
	if rec := do(t, mux, "POST", "/tasks", "globex", `{"title":"Buy milk"}`); rec.Code != 201 {
		t.Fatalf("other tenant: status = %d, want 201", rec.Code)
	}
}

func TestErrorsAreJSONWithTheRightStatus(t *testing.T) {
	mux := newMux()
	tests := []struct {
		name, method, target, body string
		want                       int
		wantBody                   string
	}{
		{"unknown task", "GET", "/tasks/99", "", 404, `"error": "task 99 not found"`},
		{"wrong type", "POST", "/tasks", `{"title":42}`, 400, `field \"title\" must be a string`},
		{"validation", "POST", "/tasks", `{"title":""}`, 422, `"title": "must be 1 to 200 characters"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, mux, tc.method, tc.target, "acme", tc.body)
			if rec.Code != tc.want || !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("status = %d, body = %s; want %d containing %s", rec.Code, rec.Body, tc.want, tc.wantBody)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("Content-Type = %q, want JSON", ct)
			}
		})
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
		{"blank title", "POST", "/tasks", "acme", `{"title":"  "}`, 422},
		{"cannot set id", "POST", "/tasks", "acme", `{"title":"x","id":9}`, 400},
		{"wrong method", "PUT", "/tasks/1", "acme", "", 405},
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
	store := NewMemoryStore()
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			do(t, mux, "POST", "/tasks", "acme", fmt.Sprintf(`{"title":"task %d"}`, i))
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	if list, _ := store.List(context.Background(), "acme"); len(list) != 100 {
		n := len(list)
		t.Fatalf("got %d tasks, want 100", n)
	}
}
