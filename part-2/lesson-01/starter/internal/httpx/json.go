// Package httpx holds small HTTP helpers shared by every handler:
// strict JSON in, JSON out, and one place that turns errors into responses.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// DecodeJSON reads exactly one JSON object from the request body into dst.
// It is strict on purpose, and every problem comes back as a
// *apperr.BadRequestError with a message that is safe to show the client.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // {"title":"x","is_admin":true} -> error

	if err := dec.Decode(dst); err != nil {
		// TODO: if err is an *http.MaxBytesError (errors.As), return
		// &apperr.TooLargeError{Limit: ...} so the client gets 413, not 400.
		return &apperr.BadRequestError{Msg: jsonProblem(err)}
	}

	// A second Decode must hit the end of the body. Anything else means the
	// client sent more than one JSON value, like {"title":"a"}{"title":"b"}.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &apperr.BadRequestError{Msg: "request body must contain a single JSON object"}
	}
	return nil
}

// jsonProblem turns encoding/json's errors into short, public messages.
// The raw ones can leak Go type names like "CreateRequest.title".
func jsonProblem(err error) string {
	var syntax *json.SyntaxError
	var wrongType *json.UnmarshalTypeError

	switch {
	case errors.Is(err, io.EOF):
		return "request body must not be empty"
	case errors.Is(err, io.ErrUnexpectedEOF):
		return "malformed JSON: the body ends too early"
	case errors.As(err, &syntax):
		return fmt.Sprintf("malformed JSON at position %d", syntax.Offset)
	case errors.As(err, &wrongType) && wrongType.Field != "":
		return fmt.Sprintf("field %q must be a %s", wrongType.Field, wrongType.Type)
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		return "unknown field " + strings.TrimPrefix(err.Error(), "json: unknown field ")
	default:
		return "malformed JSON"
	}
}

// WriteJSON sends v as indented JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		// The status line is already sent, so all we can do is log it.
		slog.Error("write JSON response", slog.String("err", err.Error()))
	}
}

// WriteError sends {"error": "..."} with the given status code.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}
