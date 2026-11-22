package webhook

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/clynzven/go-backend-academy/internal/apperr"
	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// Inbound receives events from our payment provider: "invoice 7 of tenant
// acme was paid". The provider can't log in, so every event must carry a
// signature made with the secret we share with it.
type Inbound struct {
	db     *sql.DB
	secret []byte
	now    func() time.Time
}

// NewInbound returns the receiver for POST /webhooks/payments.
func NewInbound(db *sql.DB, secret []byte) *Inbound {
	return &Inbound{db: db, secret: secret, now: time.Now}
}

// Register adds POST /webhooks/payments.
func (in *Inbound) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks/payments", httpx.Handle(in.payment))
}

// Event is what the provider sends.
type Event struct {
	ID          string `json:"event_id"`
	Type        string `json:"type"`
	Tenant      string `json:"tenant"`
	InvoiceID   int64  `json:"invoice_id"`
	AmountCents int64  `json:"amount_cents"`
}

func (in *Inbound) payment(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(r.Body) // the exact bytes: the signature covers them
	if err != nil {
		return &apperr.BadRequestError{Msg: "could not read the body"}
	}
	if err := Verify(r.Header.Get("Webhook-Signature"), in.secret, body, in.now()); err != nil {
		slog.WarnContext(r.Context(), "webhook rejected", slog.String("err", err.Error()))
		return apperr.Unauthorized("invalid or missing webhook signature")
	}
	var ev Event
	if err := json.Unmarshal(body, &ev); err != nil || ev.ID == "" || ev.Type != "payment.succeeded" ||
		ev.Tenant == "" || ev.InvoiceID < 1 || ev.AmountCents < 0 {
		return &apperr.BadRequestError{Msg: "not a payment.succeeded event"}
	}

	tx, err := in.db.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Seen this event id before? Then it's a replay (or the provider retrying):
	// answer 200 so it stops, and change nothing.
	res, err := tx.ExecContext(r.Context(),
		`INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING`, ev.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "duplicate event, ignored"})
		return nil
	}
	if err := applyPayment(r, tx, ev); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "processed"})
	return nil
}

// applyPayment marks the invoice paid and records the payment, inside tx.
func applyPayment(r *http.Request, tx *sql.Tx, ev Event) error {
	if _, err := tx.ExecContext(r.Context(),
		`UPDATE invoices SET status = 'paid' WHERE tenant_id = $1 AND id = $2`, ev.Tenant, ev.InvoiceID); err != nil {
		return fmt.Errorf("apply payment: %w", err)
	}
	if _, err := tx.ExecContext(r.Context(),
		`INSERT INTO payments (invoice_id, tenant_id, amount_cents) VALUES ($1, $2, $3)`,
		ev.InvoiceID, ev.Tenant, ev.AmountCents); err != nil {
		return fmt.Errorf("apply payment: %w", err)
	}
	return nil
}
