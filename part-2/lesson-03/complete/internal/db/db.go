// Package db opens the PostgreSQL connection pool and applies migrations.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver with database/sql
)

// Open creates a connection pool and checks that the database answers.
// sql.Open alone doesn't connect: it only prepares the pool.
func Open(ctx context.Context, url string) (*sql.DB, error) {
	pool, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// The pool is a taxi rank: a few connections wait (idle), some are busy,
	// and there is a cap on the whole fleet.
	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxIdleTime(5 * time.Minute)
	pool.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.PingContext(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return pool, nil
}

// Migrate applies every .sql file in migrations that hasn't run yet, in name
// order. It records what ran in schema_migrations, so running it again is
// safe. Everything happens in one transaction holding an advisory lock, so two
// copies of the API starting at the same moment can't both apply a file.
func Migrate(ctx context.Context, pool *sql.DB, migrations fs.FS) ([]string, error) {
	names, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)

	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // undoes everything unless Commit succeeded

	// Wait here while another process migrates. The lock ends with the transaction.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(5432001)`); err != nil {
		return nil, fmt.Errorf("lock migrations: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}

	var applied []string
	for _, name := range names {
		var done bool
		err := tx.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, name).Scan(&done)
		if err != nil {
			return nil, fmt.Errorf("check migration %s: %w", name, err)
		}
		if done {
			continue
		}
		body, err := fs.ReadFile(migrations, name)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			return nil, fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			return nil, fmt.Errorf("record migration %s: %w", name, err)
		}
		applied = append(applied, name)
	}
	return applied, tx.Commit()
}
