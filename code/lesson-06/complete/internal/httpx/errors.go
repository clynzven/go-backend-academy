package httpx

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// HandlerFunc is a handler that returns an error instead of writing it.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handle turns a HandlerFunc into a normal http.HandlerFunc: if the handler
// returns an error, WriteErr sends the right response. One place, every route.
func Handle(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			WriteErr(w, r, err)
		}
	}
}

// WriteErr maps an error to an HTTP status and a JSON body.
// Known kinds get a clear public message. Anything else is a bug or an
// outage: the client gets a plain 500, and the real cause goes to our log.
func WriteErr(w http.ResponseWriter, r *http.Request, err error) {
	var bad *apperr.BadRequestError
	var invalid *apperr.ValidationError

	switch {
	case errors.As(err, &bad):
		WriteError(w, http.StatusBadRequest, bad.Msg)
	case errors.As(err, &invalid):
		WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":  "validation failed",
			"fields": invalid.Fields,
		})
	case errors.Is(err, apperr.ErrNotFound):
		WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, apperr.ErrConflict):
		WriteError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("ERROR %s %s: %v", r.Method, r.URL.Path, err) // private: for us
		WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

// JSONErrors wraps a ServeMux so that its own replies (404 for an unknown
// path, 405 for a wrong method) are JSON like every other error. Requests
// that match a route pass straight through.
func JSONErrors(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern == "" {
			w = &routerErrorWriter{ResponseWriter: w}
		}
		mux.ServeHTTP(w, r)
	})
}

// routerErrorWriter swaps the mux's plain-text 404/405 for a JSON body.
// Headers the mux already set, like Allow on a 405, are kept.
type routerErrorWriter struct {
	http.ResponseWriter
	replaced bool
}

func (rw *routerErrorWriter) WriteHeader(status int) {
	if status != http.StatusNotFound && status != http.StatusMethodNotAllowed {
		rw.ResponseWriter.WriteHeader(status)
		return
	}
	rw.replaced = true
	rw.Header().Del("X-Content-Type-Options")
	WriteError(rw.ResponseWriter, status, strings.ToLower(http.StatusText(status)))
}

func (rw *routerErrorWriter) Write(b []byte) (int, error) {
	if rw.replaced {
		return len(b), nil // drop "404 page not found": we already sent JSON
	}
	return rw.ResponseWriter.Write(b)
}
