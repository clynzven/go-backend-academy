// Package ratelimit is a token bucket per key (an IP address, an email...),
// in memory. Each key gets Burst tokens; one comes back every Every.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter hands out tokens per key.
type Limiter struct {
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

// New returns a Limiter: burst requests at once, then one per every.
func New(every time.Duration, burst int) *Limiter {
	return &Limiter{Every: every, Burst: burst, buckets: map[string]*bucket{}, now: time.Now}
}

// Allow takes a token for key. If there is none, it returns false and how
// long until the next one.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	return l.take(key, true)
}

// Peek reports whether key has a token, without spending it. The login uses
// it for the lockout: only failed attempts spend tokens.
func (l *Limiter) Peek(key string) (bool, time.Duration) {
	return l.take(key, false)
}

func (l *Limiter) take(key string, spend bool) (bool, time.Duration) {
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

// Reset forgets key, for example after a successful login.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}
