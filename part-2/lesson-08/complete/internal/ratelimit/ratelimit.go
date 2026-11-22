// Package ratelimit is a token bucket per key (an IP address, an email...).
// Each key gets Burst tokens; one comes back every Every.
//
// Memory keeps the buckets in this process. Postgres keeps them in a table,
// so every copy of the API shares one limit.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Limiter hands out tokens per key.
type Limiter interface {
	// Allow takes a token for key. If there is none, it returns false and how
	// long until the next one.
	Allow(ctx context.Context, key string) (bool, time.Duration)
	// Peek reports whether key has a token, without spending it.
	Peek(ctx context.Context, key string) (bool, time.Duration)
	// Reset forgets key, for example after a successful login.
	Reset(ctx context.Context, key string)
}

// Memory is a Limiter in this process's memory: fine for one copy of the API.
type Memory struct {
	Every time.Duration // one new token per Every
	Burst int           // the most tokens a key can save up

	mu      sync.Mutex
	buckets map[string]*bucket
	now     func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// New returns an in-memory Limiter: burst requests at once, then one per every.
func New(every time.Duration, burst int) *Memory {
	return &Memory{Every: every, Burst: burst, buckets: map[string]*bucket{}, now: time.Now}
}

func (l *Memory) Allow(_ context.Context, key string) (bool, time.Duration) { return l.take(key, true) }
func (l *Memory) Peek(_ context.Context, key string) (bool, time.Duration)  { return l.take(key, false) }

func (l *Memory) take(key string, spend bool) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(l.Burst), last: now}
		l.buckets[key] = b
	}
	b.tokens = min(float64(l.Burst), b.tokens+float64(now.Sub(b.last))/float64(l.Every))
	b.last = now
	if b.tokens < 1 {
		return false, time.Duration((1 - b.tokens) * float64(l.Every))
	}
	if spend {
		b.tokens--
	}
	return true, 0
}

func (l *Memory) Reset(_ context.Context, key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}
