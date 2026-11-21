package invoice

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clynzven/go-backend-academy/internal/apperr"
	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// Money must arrive as whole cents: 12.5 is not a valid int64.
func TestInvoiceRejectsFractionalCents(t *testing.T) {
	body := `{"customer":"Clynz","lines":[{"description":"Coffee","quantity":1,"unit_cents":12.5}]}`
	r := httptest.NewRequest("POST", "/invoices", strings.NewReader(body))
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req); err == nil {
		t.Fatal("want an error for unit_cents 12.5, got none")
	}
}

func TestValidateReportsEveryBrokenField(t *testing.T) {
	req := CreateRequest{Customer: " ", Lines: []Line{{Description: "Logo", Quantity: 0, UnitCents: -1}}}
	var verr *apperr.ValidationError
	if err := req.Validate(); !errors.As(err, &verr) {
		t.Fatalf("Validate() = %v, want a *ValidationError", err)
	}
	for _, field := range []string{"customer", "lines[0].quantity", "lines[0].unit_cents"} {
		if _, ok := verr.Fields[field]; !ok {
			t.Errorf("missing field %q in %v", field, verr.Fields)
		}
	}
}

func TestTotalIsExactInCents(t *testing.T) {
	lines := []Line{{Quantity: 3, UnitCents: 10}, {Quantity: 1, UnitCents: 20}} // 0.10 x3 + 0.20
	if got := Total(lines); got != 50 {
		t.Fatalf("Total = %d, want 50 cents exactly", got)
	}
}
