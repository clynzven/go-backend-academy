package ratelimit

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// Postgres is a Limiter whose buckets live in the rate_limits table, so
// three copies of the API behind a proxy share ONE limit, and a restart
// doesn't reset it. Each call is one atomic statement.
type Postgres struct {
	db    *sql.DB
	name  string // keeps limiters apart: "login-ip", "login-account"
	every time.Duration
	burst int
}

// NewPostgres returns a shared Limiter: burst at once, then one per every.
func NewPostgres(db *sql.DB, name string, every time.Duration, burst int) *Postgres {
	return &Postgres{db: db, name: name, every: every, burst: burst}
}

// allowSQL refills the bucket by the time that passed and takes one token if
// there is one, in a single INSERT ... ON CONFLICT DO UPDATE. Postgres locks
// the row, so concurrent requests on any copy of the API see each other.
// (Every expression in SET reads the row as it was before the update.)
const allowSQL = `
INSERT INTO rate_limits AS r (key, tokens, allowed, updated_at)
VALUES ($1, $2 - 1, true, now())
ON CONFLICT (key) DO UPDATE SET
    allowed    = LEAST($2, r.tokens + EXTRACT(EPOCH FROM now() - r.updated_at) / $3) >= 1,
    tokens     = LEAST($2, r.tokens + EXTRACT(EPOCH FROM now() - r.updated_at) / $3)
                 - CASE WHEN LEAST($2, r.tokens + EXTRACT(EPOCH FROM now() - r.updated_at) / $3) >= 1
                        THEN 1 ELSE 0 END,
    updated_at = now()
RETURNING allowed, tokens`

func (p *Postgres) Allow(ctx context.Context, key string) (bool, time.Duration) {
	var allowed bool
	var tokens float64
	err := p.db.QueryRowContext(ctx, allowSQL, p.name+":"+key, p.burst, p.every.Seconds()).Scan(&allowed, &tokens)
	if err != nil {
		slog.ErrorContext(ctx, "rate limit", slog.String("err", err.Error()))
		return false, time.Second // fail closed: no database, no unlimited guessing
	}
	if !allowed {
		return false, time.Duration((1 - tokens) * float64(p.every))
	}
	return true, 0
}

func (p *Postgres) Peek(ctx context.Context, key string) (bool, time.Duration) {
	var tokens float64
	err := p.db.QueryRowContext(ctx,
		`SELECT LEAST($2, tokens + EXTRACT(EPOCH FROM now() - updated_at) / $3) FROM rate_limits WHERE key = $1`,
		p.name+":"+key, p.burst, p.every.Seconds()).Scan(&tokens)
	if err == sql.ErrNoRows {
		return true, 0 // never seen: a full bucket
	}
	if err != nil {
		slog.ErrorContext(ctx, "rate limit", slog.String("err", err.Error()))
		return false, time.Second
	}
	if tokens < 1 {
		return false, time.Duration((1 - tokens) * float64(p.every))
	}
	return true, 0
}

func (p *Postgres) Reset(ctx context.Context, key string) {
	if _, err := p.db.ExecContext(ctx, `DELETE FROM rate_limits WHERE key = $1`, p.name+":"+key); err != nil {
		slog.ErrorContext(ctx, "rate limit", slog.String("err", err.Error()))
	}
}
