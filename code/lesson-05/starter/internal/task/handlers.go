package task

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/clynzven/go-backend-academy/internal/apperr"
	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// Handler serves the /tasks routes. It gets its store handed in (dependency
// injection) instead of reaching for a global variable.
type Handler struct {
	store Store // an interface: Postgres in production, memory in unit tests
}

// NewHandler returns a Handler that uses store.
func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

// Register adds the task routes to mux. Every handler returns an error, and
// httpx.Handle turns that error into the right HTTP response.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /tasks", httpx.Handle(h.list))
	mux.HandleFunc("POST /tasks", httpx.Handle(h.create))
	mux.HandleFunc("GET /tasks/{id}", httpx.Handle(h.get))
	mux.HandleFunc("PATCH /tasks/{id}", httpx.Handle(h.update))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	tenant, err := tenantFrom(r)
	if err != nil {
		return err
	}
	tasks, err := h.store.List(r.Context(), tenant)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, tasks)
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	tenant, err := tenantFrom(r)
	if err != nil {
		return err
	}
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	title, err := validTitle(req.Title)
	if err != nil {
		return err
	}
	t, err := h.store.Add(r.Context(), tenant, title)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, t)
	return nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	tenant, err := tenantFrom(r)
	if err != nil {
		return err
	}
	id, err := idFrom(r)
	if err != nil {
		return err
	}
	t, err := h.store.Get(r.Context(), tenant, id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, t)
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	tenant, err := tenantFrom(r)
	if err != nil {
		return err
	}
	id, err := idFrom(r)
	if err != nil {
		return err
	}
	var req UpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	if req.Title != nil {
		title, err := validTitle(*req.Title)
		if err != nil {
			return err
		}
		req.Title = &title
	}
	t, err := h.store.Update(r.Context(), tenant, id, req.Title, req.Done)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, t)
	return nil
}

// tenantFrom reads the X-Tenant-ID header. For now we trust it as sent;
// real authentication comes later in the academy.
func tenantFrom(r *http.Request) (string, error) {
	tenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenant == "" || len(tenant) > 64 {
		return "", &apperr.BadRequestError{Msg: "missing or invalid X-Tenant-ID header"}
	}
	return tenant, nil
}

// idFrom reads {id} from the path and checks it is a positive number.
func idFrom(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, &apperr.BadRequestError{Msg: "task id must be a positive number"}
	}
	return id, nil
}

// validTitle checks the business rule for titles: 1 to 200 characters.
// Breaking it is a validation error (422), not a broken request (400).
func validTitle(raw string) (string, error) {
	title := strings.TrimSpace(raw)
	if title == "" || len(title) > 200 {
		return "", &apperr.ValidationError{
			Fields: map[string]string{"title": "must be 1 to 200 characters"},
		}
	}
	return title, nil
}
