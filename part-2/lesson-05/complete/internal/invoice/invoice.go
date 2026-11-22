// Package invoice is everything about invoices: the data, the Postgres store
// and the HTTP handlers.
package invoice

import (
	"fmt"
	"strings"
	"time"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// Invoice stores money as whole cents in an int64, never as a float64.
// 12.50 pesos is 1250 cents: exact, with no rounding surprises.
type Invoice struct {
	ID         int64     `json:"id"`
	Customer   string    `json:"customer"`
	Lines      []Line    `json:"lines,omitempty"`
	TotalCents int64     `json:"total_cents"`
	Status     string    `json:"status"` // "open" or "paid"
	CreatedAt  time.Time `json:"created_at"`
}

// Line is one row on an invoice: 3 × "Logo design" at 150000 cents each.
type Line struct {
	Description string `json:"description"`
	Quantity    int64  `json:"quantity"`
	UnitCents   int64  `json:"unit_cents"`
}

// CreateRequest is what a client may send to create an invoice.
type CreateRequest struct {
	Customer string `json:"customer"`
	Lines    []Line `json:"lines"`
}

// Limits keep totals far away from int64 overflow and requests small.
const (
	maxLines     = 100
	maxQuantity  = 10_000
	maxUnitCents = 100_000_000 // one million pesos per unit
)

// Validate checks the business rules and reports every broken field at once.
func (req *CreateRequest) Validate() error {
	fields := map[string]string{}
	req.Customer = strings.TrimSpace(req.Customer)
	if req.Customer == "" || len(req.Customer) > 200 {
		fields["customer"] = "must be 1 to 200 characters"
	}
	if len(req.Lines) == 0 || len(req.Lines) > maxLines {
		fields["lines"] = fmt.Sprintf("must have 1 to %d lines", maxLines)
	}
	for i := range req.Lines {
		l := &req.Lines[i]
		l.Description = strings.TrimSpace(l.Description)
		if l.Description == "" || len(l.Description) > 200 {
			fields[fmt.Sprintf("lines[%d].description", i)] = "must be 1 to 200 characters"
		}
		if l.Quantity < 1 || l.Quantity > maxQuantity {
			fields[fmt.Sprintf("lines[%d].quantity", i)] = fmt.Sprintf("must be 1 to %d", maxQuantity)
		}
		if l.UnitCents < 0 || l.UnitCents > maxUnitCents {
			fields[fmt.Sprintf("lines[%d].unit_cents", i)] = fmt.Sprintf("must be 0 to %d", maxUnitCents)
		}
	}
	if len(fields) > 0 {
		return &apperr.ValidationError{Fields: fields}
	}
	return nil
}

// Total adds up the lines, in cents.
func Total(lines []Line) int64 {
	var total int64
	for _, l := range lines {
		total += l.Quantity * l.UnitCents
	}
	return total
}
