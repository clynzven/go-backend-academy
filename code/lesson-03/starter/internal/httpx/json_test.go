package httpx

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONIsStrict(t *testing.T) {
	type input struct {
		Title string `json:"title"`
	}
	tests := []struct {
		name, body string
		wantErr    string // "" means the body must be accepted
	}{
		{"valid", `{"title":"Buy milk"}`, ""},
		{"unknown field", `{"title":"x","is_admin":true}`, `unknown field "is_admin"`},
		{"wrong type", `{"title":42}`, "cannot unmarshal number"},
		{"empty body", ``, "must not be empty"},
		{"two objects", `{"title":"a"}{"title":"b"}`, "single JSON object"},
		{"broken JSON", `{"title":`, "unexpected EOF"},
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
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}
