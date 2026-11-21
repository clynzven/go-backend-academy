// Command api is the Multi-Tenant Task & Invoice API.
//
// Lesson 1 STARTER: follow the TODOs, then run `go test ./...` until it passes.
package main

import (
	"fmt"
	"log"
	"net/http"
)

// addr is where the server listens. 127.0.0.1 means "this computer only".
const addr = "127.0.0.1:8080"

func main() {
	mux := newRouter()

	log.Printf("Task & Invoice API listening on http://%s", addr)

	// TODO 4: start the server with http.ListenAndServe(addr, mux).
	// If it returns an error, stop the program with log.Fatal(err).
	_ = mux
}

// newRouter builds the ServeMux: the receptionist that reads the path on
// each request and sends it to the right handler. main and the tests share it.
func newRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	// TODO 1: register healthHandler on "/health" and helloHandler on "/hello".
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
	// TODO 2: set Content-Type to "text/plain; charset=utf-8",
	// write status 200 (http.StatusOK), then write the body "ok\n".
}

// helloHandler reads a value from the query string: /hello?name=Clynz
func helloHandler(w http.ResponseWriter, r *http.Request) {
	// TODO 3: read the "name" query parameter with r.URL.Query().Get("name").
	// Default to "Gopher" when it is empty, set the Content-Type header,
	// and write "Hello, <name>!\n".
}
