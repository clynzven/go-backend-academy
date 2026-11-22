package main

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/clynzven/go-backend-academy/internal/auth"
	"github.com/clynzven/go-backend-academy/internal/invoice"
	"github.com/clynzven/go-backend-academy/internal/middleware"
	"github.com/clynzven/go-backend-academy/internal/task"
)

// newHandler builds the whole HTTP side of the app: stores, handlers, router
// and middleware. main and the integration tests both use it.
func newHandler(pool *sql.DB, logger *slog.Logger, debug bool, secret []byte) http.Handler {
	authn := auth.NewService(pool, secret)
	routes := []registrar{
		authn,
		task.NewHandler(task.NewPostgresStore(pool)),
		invoice.NewHandler(invoice.NewStore(pool)),
	}
	if debug {
		routes = append(routes, debugRoutes{})
	}

	// Every request passes the layers in this order: RequestID, Logger,
	// Recoverer, MaxBody, Auth, then the router. Responses travel back the other way.
	return middleware.Chain(newRouter(pool, routes...),
		middleware.RequestID,
		middleware.Logger(logger),
		middleware.Recoverer(logger),
		middleware.MaxBody(maxBodyBytes),
		authn.Middleware,
	)
}
