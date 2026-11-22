package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// newRouter builds the ServeMux with method + path patterns (Go 1.22+).
// Unknown paths get 404 and a wrong method gets 405, automatically, and
// httpx.JSONErrors makes those two replies JSON like every other error.
// registrar is anything that adds its own routes: the task and invoice handlers.
type registrar interface {
	Register(mux *http.ServeMux)
}

func newRouter(db pinger, handlers ...registrar) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", homeHandler) // {$}: exactly "/", nothing more
	mux.HandleFunc("GET /health", healthHandler(db))
	mux.HandleFunc("GET /hello", helloHandler)
	for _, h := range handlers {
		h.Register(mux)
	}
	return httpx.JSONErrors(mux)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "Task & Invoice API v0.8")
}

// pinger is the one thing /health needs from the database. *sql.DB has it.
type pinger interface {
	PingContext(ctx context.Context) error
}

// healthHandler answers "ok" only when the database answers too, within one
// second. A load balancer then stops sending traffic to a broken instance.
func healthHandler(db pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "database unavailable")
			return
		}
		fmt.Fprintln(w, "ok")
	}
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
