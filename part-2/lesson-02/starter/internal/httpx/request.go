package httpx

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// Identity is who sent the request, as proven by their token: never by a
// header the client can type.
type Identity struct {
	UserID int64
	Tenant string
}

type identityKey struct{}

// WithIdentity returns a copy of ctx carrying id. Only the auth middleware
// calls this, after it has verified a token.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// IdentityFrom returns the verified identity in ctx, if there is one.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok
}

// Tenant reads the X-Tenant-ID header, exactly like Part 1: we trust it as sent.
//
// TODO: take the tenant from the verified identity instead
// (IdentityFrom(r.Context())), and return apperr.ErrUnauthorized when there is none.
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
