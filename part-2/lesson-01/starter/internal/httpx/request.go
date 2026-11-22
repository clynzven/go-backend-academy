package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// Tenant reads the X-Tenant-ID header. For now we trust it as sent;
// real authentication comes later in the academy.
func Tenant(r *http.Request) (string, error) {
	tenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenant == "" || len(tenant) > 64 {
		return "", &apperr.BadRequestError{Msg: "missing or invalid X-Tenant-ID header"}
	}
	return tenant, nil
}

// PathID reads the {id} wildcard and checks it is a positive number.
func PathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, &apperr.BadRequestError{Msg: "id must be a positive number"}
	}
	return id, nil
}
