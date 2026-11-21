package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateAndListTasks(t *testing.T) {
	mux := newRouter(newTaskStore())

	// Create two tasks.
	for _, title := range []string{"Buy milk", "Send invoice"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("POST", "/tasks", strings.NewReader(`{"title":"`+title+`"}`)))
		if rec.Code != 201 {
			t.Fatalf("POST %q: status = %d, want 201 (body %s)", title, rec.Code, rec.Body)
		}
	}

	// List them: same store, so both must be there, in order.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/tasks", nil))
	var got []Task
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("GET /tasks: %v", err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[1].Title != "Send invoice" {
		t.Fatalf("GET /tasks = %+v, want 2 tasks in order", got)
	}
}

func TestTaskInputIsValidated(t *testing.T) {
	tests := []struct {
		name, method, body string
		want               int
	}{
		{"blank title", "POST", `{"title":"   "}`, 400},
		{"cannot set id", "POST", `{"title":"x","id":99}`, 400},
		{"wrong method", "DELETE", ``, 405},
	}
	mux := newRouter(newTaskStore())
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(tc.method, "/tasks", strings.NewReader(tc.body)))
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.want, rec.Body)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

// 100 requests at once, each on its own goroutine, like a real server.
// Run with -race: without the mutex this test reports a data race.
func TestConcurrentCreates(t *testing.T) {
	store := newTaskStore()
	mux := newRouter(store)
	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest("POST", "/tasks", strings.NewReader(`{"title":"task"}`)))
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	if n := len(store.list()); n != 100 {
		t.Fatalf("got %d tasks, want 100", n)
	}
}
