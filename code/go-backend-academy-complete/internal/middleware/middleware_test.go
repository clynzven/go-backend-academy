package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clynzven/go-backend-academy/internal/requestid"
)

// newLogger returns a JSON logger that writes into buf, so tests can read it.
func newLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, nil))
}

func TestRecovererTurnsAPanicIntoA500(t *testing.T) {
	var logs bytes.Buffer
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("something went badly wrong") })
	h := Chain(boom, RequestID, Recoverer(newLogger(&logs)))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/debug/panic", nil))

	if rec.Code != 500 || !strings.Contains(rec.Body.String(), `"error": "internal server error"`) {
		t.Fatalf("status = %d, body %s; want a JSON 500", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "badly wrong") {
		t.Fatal("the panic message leaked to the client")
	}
	for _, want := range []string{"panic recovered", "something went badly wrong", "goroutine", rec.Header().Get("X-Request-ID")} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log is missing %q:\n%s", want, logs.String())
		}
	}
}

func TestLoggerRecordsStatusBytesAndRequestID(t *testing.T) {
	var logs bytes.Buffer
	teapot := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("short and stout"))
	})
	h := Chain(teapot, RequestID, Logger(newLogger(&logs)))

	req := httptest.NewRequest("GET", "/tea", nil)
	req.Header.Set("X-Request-ID", "abc-123")
	h.ServeHTTP(httptest.NewRecorder(), req)

	var line map[string]any
	if err := json.Unmarshal(logs.Bytes(), &line); err != nil {
		t.Fatalf("log line is not JSON: %v\n%s", err, logs.String())
	}
	if line["status"] != 418.0 || line["bytes"] != 15.0 || line["request_id"] != "abc-123" || line["level"] != "WARN" {
		t.Fatalf("log line = %v", line)
	}
}

func TestRequestIDReplacesBadIncomingIDs(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestid.From(r.Context())
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "bad id with spaces\n<script>")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if len(seen) != 16 || seen != rec.Header().Get("X-Request-ID") {
		t.Fatalf("request id = %q, header %q; want a fresh 16-character id in both", seen, rec.Header().Get("X-Request-ID"))
	}
}

func TestChainRunsTheFirstMiddlewareOutermost(t *testing.T) {
	var order []string
	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+" in")
				next.ServeHTTP(w, r)
				order = append(order, name+" out")
			})
		}
	}
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { order = append(order, "handler") }),
		mark("A"), mark("B"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	if got := strings.Join(order, ", "); got != "A in, B in, handler, B out, A out" {
		t.Fatalf("order = %s", got)
	}
}

// Order matters: Logger must sit OUTSIDE Recoverer. Inside, the panic flies
// past Logger before it can write its line, so the crash is never logged as a request.
func TestLoggerMustWrapRecoverer(t *testing.T) {
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	count := func(h http.Handler, logs *bytes.Buffer) int {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
		return strings.Count(logs.String(), `"msg":"request"`)
	}

	var wrong bytes.Buffer
	if n := count(Chain(boom, Recoverer(newLogger(&wrong)), Logger(newLogger(&wrong))), &wrong); n != 0 {
		t.Fatalf("Recoverer outside Logger: %d request lines, expected 0 (the bug this test documents)", n)
	}
	var right bytes.Buffer
	if n := count(Chain(boom, Logger(newLogger(&right)), Recoverer(newLogger(&right))), &right); n != 1 {
		t.Fatalf("Logger outside Recoverer: %d request lines, want 1", n)
	}
}
