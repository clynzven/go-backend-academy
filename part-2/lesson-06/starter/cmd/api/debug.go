package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// debugRoutes are only registered when DEBUG_ROUTES=1, for demos and local
// testing. Never turn them on in production.
type debugRoutes struct{}

func (debugRoutes) Register(mux *http.ServeMux) {
	// A handler with a bug: it panics. Recoverer turns that into a clean 500.
	mux.HandleFunc("GET /debug/panic", func(w http.ResponseWriter, r *http.Request) {
		var counts map[string]int
		counts["oops"]++ // writing to a nil map panics
	})

	// A slow request, to watch graceful shutdown wait for it: /debug/slow?seconds=5
	mux.HandleFunc("GET /debug/slow", func(w http.ResponseWriter, r *http.Request) {
		seconds, err := strconv.Atoi(r.URL.Query().Get("seconds"))
		if err != nil || seconds < 1 || seconds > 30 {
			seconds = 5
		}
		select {
		case <-time.After(time.Duration(seconds) * time.Second):
			fmt.Fprintf(w, "finished after %ds\n", seconds)
		case <-r.Context().Done(): // the client gave up
		}
	})
}
