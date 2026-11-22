// Package middleware wraps handlers with work every request needs:
// a request ID, a log line, and a safety net for panics.
//
// A middleware is a function that takes a handler and returns a bigger one:
// a gift-wrapping machine. Wrap again for more layers.
package middleware

import (
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/clynzven/go-backend-academy/internal/httpx"
	"github.com/clynzven/go-backend-academy/internal/requestid"
)

// Middleware wraps one handler in another.
type Middleware func(http.Handler) http.Handler

// Chain wraps h so that the FIRST middleware listed is the outermost layer:
// Chain(h, A, B) handles a request as A -> B -> h, and the response goes back B -> A.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// validID accepts request IDs sent by a client or a proxy, if they look sane.
var validID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// RequestID gives every request an ID: the X-Request-ID it came with (if valid),
// or a new one. It goes into the context and back in the response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validID.MatchString(id) {
			id = requestid.New()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(requestid.With(r.Context(), id)))
	})
}

// Logger writes one structured log line per request, after it finishes:
// method, path, status, bytes, how long it took, and the request ID.
func Logger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)

			level := slog.LevelInfo
			switch {
			case rec.Status() >= 500:
				level = slog.LevelError
			case rec.Status() >= 400:
				level = slog.LevelWarn
			}
			log.LogAttrs(r.Context(), level, "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.Status()),
				slog.Int("bytes", rec.bytes),
				slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
				slog.String("request_id", requestid.From(r.Context())),
			)
		})
	}
}

// Recoverer is the safety net under the trapeze: if a handler panics, the
// client gets a clean JSON 500, the log gets the panic and its stack trace,
// and the server keeps serving everyone else.
func Recoverer(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w}
			defer func() {
				p := recover()
				if p == nil {
					return
				}
				if p == http.ErrAbortHandler { // a deliberate abort: let net/http handle it
					panic(p)
				}
				log.ErrorContext(r.Context(), "panic recovered",
					slog.Any("panic", p),
					slog.String("request_id", requestid.From(r.Context())),
					slog.String("stack", string(debug.Stack())),
				)
				if !rec.wroteHeader { // too late to change the status once it's sent
					httpx.WriteError(rec, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(rec, r)
		})
	}
}

// MaxBody caps every request body at limit bytes. http.MaxBytesReader stops
// reading at the limit and makes the next Read fail with *http.MaxBytesError,
// so a 300 MB upload costs us at most limit bytes of memory, not 300 MB.
func MaxBody(limit int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeaders sets the response headers every JSON API should send. They
// cost nothing and close doors that browsers would otherwise leave open.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")                                     // JSON stays JSON
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'") // nothing to run or frame
		h.Set("X-Frame-Options", "DENY")                                               // older browsers: no framing
		h.Set("Referrer-Policy", "no-referrer")                                        // URLs don't leak to other sites
		h.Set("Cache-Control", "no-store")                                             // private data stays out of caches
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")      // HTTPS only, 2 years (once on TLS)
		next.ServeHTTP(w, r)
	})
}

// statusRecorder is a carbon copy of what the handler wrote: it passes
// everything through, and remembers the status code and the body size.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (rec *statusRecorder) WriteHeader(status int) {
	if !rec.wroteHeader {
		rec.status, rec.wroteHeader = status, true
	}
	rec.ResponseWriter.WriteHeader(status)
}

func (rec *statusRecorder) Write(b []byte) (int, error) {
	if !rec.wroteHeader {
		rec.WriteHeader(http.StatusOK) // writing a body without a status means 200
	}
	n, err := rec.ResponseWriter.Write(b)
	rec.bytes += n
	return n, err
}

// Status is 200 when the handler wrote nothing at all, like net/http itself.
func (rec *statusRecorder) Status() int {
	if rec.status == 0 {
		return http.StatusOK
	}
	return rec.status
}

// Unwrap lets http.ResponseController reach the real writer (Flush, deadlines).
func (rec *statusRecorder) Unwrap() http.ResponseWriter {
	return rec.ResponseWriter
}
