package invoice

import (
	"net/http"
	"strings"

	"github.com/clynzven/go-backend-academy/internal/apperr"
	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// Handler serves the /invoices routes.
type Handler struct {
	store *Store
}

// NewHandler returns a Handler that uses store.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register adds the invoice routes. "/invoices/search" is more specific than
// "/invoices/{id}", so the mux sends searches to search, not to get.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /invoices", httpx.Handle(h.list))
	mux.HandleFunc("POST /invoices", httpx.Handle(h.create))
	mux.HandleFunc("GET /invoices/search", httpx.Handle(h.search))
	mux.HandleFunc("GET /invoices/{id}", httpx.Handle(h.get))
	mux.HandleFunc("DELETE /invoices/{id}", httpx.Handle(h.delete))
	mux.HandleFunc("POST /invoices/{id}/pay", httpx.Handle(h.pay))
	mux.HandleFunc("GET /invoices/{id}/payments", httpx.Handle(h.payments))
}

// pay charges an open invoice. Send an Idempotency-Key header to make retries
// safe: the same key returns the same payment (200) instead of paying again.
func (h *Handler) pay(w http.ResponseWriter, r *http.Request) error {
	tenant, err := httpx.Tenant(r)
	if err != nil {
		return err
	}
	id, err := httpx.PathID(r)
	if err != nil {
		return err
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) > 100 {
		return &apperr.BadRequestError{Msg: "Idempotency-Key is longer than 100 characters"}
	}
	p, replayed, err := h.store.Pay(r.Context(), tenant, id, key)
	if err != nil {
		return err
	}
	if replayed {
		w.Header().Set("Idempotent-Replayed", "true")
		httpx.WriteJSON(w, http.StatusOK, p)
		return nil
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
	return nil
}

func (h *Handler) payments(w http.ResponseWriter, r *http.Request) error {
	tenant, err := httpx.Tenant(r)
	if err != nil {
		return err
	}
	id, err := httpx.PathID(r)
	if err != nil {
		return err
	}
	list, err := h.store.Payments(r.Context(), tenant, id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, list)
	return nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	tenant, err := httpx.Tenant(r)
	if err != nil {
		return err
	}
	invoices, err := h.store.List(r.Context(), tenant)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, invoices)
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	tenant, err := httpx.Tenant(r)
	if err != nil {
		return err
	}
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	if err := req.Validate(); err != nil {
		return err
	}
	inv, err := h.store.Create(r.Context(), tenant, req)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, inv)
	return nil
}

// search: GET /invoices/search?customer=acme
func (h *Handler) search(w http.ResponseWriter, r *http.Request) error {
	tenant, err := httpx.Tenant(r)
	if err != nil {
		return err
	}
	q := strings.TrimSpace(r.URL.Query().Get("customer"))
	if q == "" || len(q) > 100 {
		return &apperr.ValidationError{Fields: map[string]string{"customer": "must be 1 to 100 characters"}}
	}
	invoices, err := h.store.Search(r.Context(), tenant, q)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, invoices)
	return nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	tenant, err := httpx.Tenant(r)
	if err != nil {
		return err
	}
	id, err := httpx.PathID(r)
	if err != nil {
		return err
	}
	inv, err := h.store.Get(r.Context(), tenant, id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, inv)
	return nil
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) error {
	tenant, err := httpx.Tenant(r)
	if err != nil {
		return err
	}
	id, err := httpx.PathID(r)
	if err != nil {
		return err
	}
	if err := h.store.Delete(r.Context(), tenant, id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
