// Command api is the Multi-Tenant Task & Invoice API.
//
// Built with Go's standard library plus one database driver: tasks and
// invoices live in PostgreSQL, every request is logged as JSON, and Ctrl+C
// shuts the server down gracefully.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clynzven/go-backend-academy/internal/auth"
	"github.com/clynzven/go-backend-academy/internal/db"
	"github.com/clynzven/go-backend-academy/internal/httpx"
	"github.com/clynzven/go-backend-academy/migrations"
)

// shutdownTimeout is how long requests in flight may take to finish after Ctrl+C.
const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(context.Background(), os.Getenv, os.Stdout); err != nil {
		slog.Error("api stopped with an error", slog.String("err", err.Error()))
		os.Exit(1)
	}
}

// run is main without the os.Exit, so a test can start and stop the whole program.
func run(ctx context.Context, getenv func(string) string, stdout io.Writer) error {
	// One JSON line per event: easy to search, filter and ship to a log service.
	logger := slog.New(slog.NewJSONHandler(stdout, nil))
	slog.SetDefault(logger)

	// The doorbell: ctx is cancelled on Ctrl+C (SIGINT) or SIGTERM (docker stop, Kubernetes).
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Configuration comes from the environment, never from the source code.
	url := getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL is not set (see .env.example)")
	}
	addr := getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080" // this computer only
	}

	// The key that signs login tokens: required, and at least 32 random bytes.
	// Make one with: openssl rand -base64 32
	secret := getenv("JWT_SECRET")
	if len(secret) < auth.MinSecretBytes {
		return fmt.Errorf("JWT_SECRET must be at least %d bytes (try: openssl rand -base64 32)", auth.MinSecretBytes)
	}

	// The secret our payment provider signs its webhooks with.
	webhookSecret := getenv("WEBHOOK_SECRET")
	if len(webhookSecret) < 32 {
		return errors.New("WEBHOOK_SECRET must be at least 32 bytes (your payment provider gives you one)")
	}

	// Our own reverse proxies, allowed to tell us the client's address in
	// X-Forwarded-For (comma-separated IPs or CIDRs). Empty: trust nobody.
	trusted, err := httpx.ParseTrusted(getenv("TRUSTED_PROXIES"))
	if err != nil {
		return fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}

	pool, err := db.Open(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()

	applied, err := db.Migrate(ctx, pool, migrations.FS)
	if err != nil {
		return err
	}
	logger.Info("database ready", slog.Int("migrations_applied", len(applied)))

	debug := getenv("DEBUG_ROUTES") == "1"
	if debug {
		logger.Warn("debug routes enabled")
	}

	// newServer sets the timeouts and header limit that Slowloris needs us to forget.
	srv := newServer(addr, newHandler(pool, logger, debug, []byte(secret), []byte(webhookSecret), trusted), defaultLimits)

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("Task & Invoice API listening", slog.String("addr", "http://"+addr))
		serveErr <- srv.ListenAndServe() // blocks until Shutdown or a failure
	}()

	select {
	case err := <-serveErr: // the server never started (for example, the port is taken)
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done(): // Ctrl+C
	}
	stop() // a second Ctrl+C now kills the program immediately

	// Close the front door, serve the customers already inside, then turn off the lights.
	logger.Info("shutting down: finishing requests in flight", slog.String("timeout", shutdownTimeout.String()))
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	logger.Info("server stopped")

	if err := pool.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	logger.Info("database closed, bye")
	return nil
}
