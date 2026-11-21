// Command api is the Multi-Tenant Task & Invoice API.
//
// Built with nothing but Go's standard library: net/http, encoding/json and a
// router (ServeMux). Lesson 2 adds JSON and an in-memory task store.
package main

import (
	"fmt"
	"log"
	"net/http"
)

// addr is where the server listens. 127.0.0.1 means "this computer only",
// so nobody else on your Wi-Fi can reach the API while we build it.
const addr = "127.0.0.1:8080"

func main() {
	mux := newRouter(newTaskStore())

	log.Printf("Task & Invoice API listening on http://%s", addr)

	// ListenAndServe blocks forever, serving each request on its own goroutine.
	// It only returns if something goes wrong (for example, the port is taken).
	// Heads-up: this simple version has no timeouts. We harden it later in the course.
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// newRouter builds the ServeMux: the receptionist that reads the path on
// each request and sends it to the right handler. main and the tests share it.
func newRouter(tasks *taskStore) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/hello", helloHandler)
	mux.HandleFunc("/tasks", tasks.tasksHandler)
	return mux
}

// homeHandler answers every path no other handler claims, and echoes the
// method and path back, so you can see what the server received.
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Task & Invoice API v0.1\nYou asked for: %s %s\n", r.Method, r.URL.Path)
}

// healthHandler is what a load balancer or Docker calls to ask "are you alive?".
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK) // 200; optional, but explicit is clearer
	fmt.Fprintln(w, "ok")
}

// helloHandler reads a value from the query string: /hello?name=Clynz
func helloHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Gopher"
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Hello, %s!\n", name)
}
