package invoice

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// Money must arrive as whole cents: 12.5 is not a valid int64.
func TestInvoiceRejectsFractionalCents(t *testing.T) {
	r := httptest.NewRequest("POST", "/invoices", strings.NewReader(`{"customer":"Clynz","amount_cents":12.5}`))
	var inv Invoice
	if err := httpx.DecodeJSON(r, &inv); err == nil {
		t.Fatal("want an error for amount_cents 12.5, got none")
	}
}
