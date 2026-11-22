package httpx

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

func TestDecodeJSONIsStrictAndPolite(t *testing.T) {
	type input struct {
		Title string `json:"title"`
	}
	tests := []struct {
		name, body string
		wantErr    string // "" means the body must be accepted
	}{
		{"valid", `{"title":"Buy milk"}`, ""},
		{"unknown field", `{"title":"x","is_admin":true}`, `unknown field "is_admin"`},
		{"wrong type", `{"title":42}`, `field "title" must be a string`},
		{"empty body", ``, "must not be empty"},
		{"two objects", `{"title":"a"}{"title":"b"}`, "single JSON object"},
		{"cut short", `{"title":`, "ends too early"},
		{"not JSON", `title=Buy milk`, "malformed JSON at position"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			var in input
			err := DecodeJSON(r, &in)

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want no error, got %v", err)
				}
				return
			}
			var bad *apperr.BadRequestError
			if !errors.As(err, &bad) {
				t.Fatalf("want a *apperr.BadRequestError, got %T: %v", err, err)
			}
			if !strings.Contains(bad.Msg, tc.wantErr) {
				t.Fatalf("message = %q, want it to contain %q", bad.Msg, tc.wantErr)
			}
			if strings.Contains(bad.Msg, "input.") || strings.Contains(bad.Msg, "Go struct") {
				t.Fatalf("message leaks Go internals: %q", bad.Msg)
			}
		})
	}
}
