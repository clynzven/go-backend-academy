// Package webhook lets a tenant test a webhook URL: "send a sample event to
// this URL and show me what came back". Our server makes the request, so the
// URL must never lead inside our own network (SSRF).
package webhook

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/clynzven/go-backend-academy/internal/apperr"
	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// Handler serves POST /webhooks/test.
type Handler struct {
	client *http.Client
}

// NewHandler returns a Handler. A nil client means SafeClient().
func NewHandler(client *http.Client) *Handler {
	if client == nil {
		client = SafeClient()
	}
	return &Handler{client: client}
}

// Register adds the webhook routes to mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks/test", httpx.Handle(h.test))
}

type testRequest struct {
	URL string `json:"url"`
}

type testResult struct {
	Status  int    `json:"status"`
	Preview string `json:"preview"` // the first 200 bytes of the answer
}

const sampleEvent = `{"event":"invoice.paid","invoice_id":1,"test":true}`

func (h *Handler) test(w http.ResponseWriter, r *http.Request) error {
	if _, err := httpx.Tenant(r); err != nil { // logged-in users only
		return err
	}
	var req testRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	u, err := url.Parse(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return &apperr.ValidationError{Fields: map[string]string{"url": "must be an http or https URL"}}
	}

	out, err := http.NewRequestWithContext(r.Context(), http.MethodPost, u.String(), bytes.NewBufferString(sampleEvent))
	if err != nil {
		return &apperr.ValidationError{Fields: map[string]string{"url": "is not a valid URL"}}
	}
	out.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(out)
	if errors.Is(err, ErrBlocked) {
		return &apperr.ValidationError{Fields: map[string]string{"url": "must point to a public address"}}
	}
	if err != nil {
		return &apperr.BadRequestError{Msg: "could not deliver the test event to that URL"}
	}
	defer resp.Body.Close()
	preview, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
	httpx.WriteJSON(w, http.StatusOK, testResult{Status: resp.StatusCode, Preview: string(preview)})
	return nil
}
