package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONIsStrict(t *testing.T) {
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
			r := httptest.NewRequest("POST", "/tasks", strings.NewReader(tc.body))
			var req createTaskRequest
			err := decodeJSON(r, &req)

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

// Money must arrive as whole cents: 12.5 is not a valid int64.
func TestInvoiceRejectsFractionalCents(t *testing.T) {
	r := httptest.NewRequest("POST", "/invoices", strings.NewReader(`{"customer":"Clynz","amount_cents":12.5}`))
	var inv Invoice
	if err := decodeJSON(r, &inv); err == nil {
		t.Fatal("want an error for amount_cents 12.5, got none")
	}
}
