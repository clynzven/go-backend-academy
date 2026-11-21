package main

// Invoice stores money as whole cents in an int64, never as a float64.
// 12.50 pesos is 1250 cents: exact, with no rounding surprises.
// Invoice endpoints arrive in lesson 6; today we only model the data.
type Invoice struct {
	ID          int64  `json:"id"`
	Customer    string `json:"customer"`
	AmountCents int64  `json:"amount_cents"`
}
