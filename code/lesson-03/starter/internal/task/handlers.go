package task

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// Handler serves the /tasks routes. It gets its store handed in (dependency
// injection) instead of reaching for a global variable.
type Handler struct {
	store *Store
}

// NewHandler returns a Handler that uses store.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register adds the task routes to mux. Each pattern is "METHOD /path";
// {id} is a wildcard that r.PathValue("id") reads.
func (h *Handler) Register(mux *http.ServeMux) {
	// TODO 2: register h.list, h.create, h.get and h.update with patterns like
	// "GET /tasks" and "GET /tasks/{id}".
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	tenant, ok := tenantFrom(w, r)
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.store.List(tenant))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	tenant, ok := tenantFrom(w, r)
	if !ok {
		return
	}
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	title, ok := validTitle(w, req.Title)
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, h.store.Add(tenant, title))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	tenant, ok := tenantFrom(w, r)
	if !ok {
		return
	}
	id, ok := idFrom(w, r)
	if !ok {
		return
	}
	t, found := h.store.Get(tenant, id)
	if !found {
		httpx.WriteError(w, http.StatusNotFound, "task not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	tenant, ok := tenantFrom(w, r)
	if !ok {
		return
	}
	id, ok := idFrom(w, r)
	if !ok {
		return
	}
	var req UpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Title != nil {
		title, ok := validTitle(w, *req.Title)
		if !ok {
			return
		}
		req.Title = &title
	}
	t, found := h.store.Update(tenant, id, req.Title, req.Done)
	if !found {
		httpx.WriteError(w, http.StatusNotFound, "task not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

// tenantFrom reads the X-Tenant-ID header. For now we trust it as sent;
// real authentication comes later in the academy.
func tenantFrom(w http.ResponseWriter, r *http.Request) (string, bool) {
	tenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenant == "" || len(tenant) > 64 {
		httpx.WriteError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return "", false
	}
	return tenant, true
}

// idFrom reads {id} from the path and checks it is a positive number.
func idFrom(w http.ResponseWriter, r *http.Request) (int64, bool) {
	// TODO 3: read the {id} wildcard with r.PathValue("id"), parse it with
	// strconv.ParseInt(s, 10, 64), and reply 400 "task id must be a positive
	// number" if it fails or is below 1.
	_ = strconv.ParseInt // you will need this in TODO 3; delete this line when you're done
	httpx.WriteError(w, http.StatusNotImplemented, "TODO 3")
	return 0, false
}

func validTitle(w http.ResponseWriter, raw string) (string, bool) {
	title := strings.TrimSpace(raw)
	if title == "" || len(title) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "title must be 1 to 200 characters")
		return "", false
	}
	return title, true
}
