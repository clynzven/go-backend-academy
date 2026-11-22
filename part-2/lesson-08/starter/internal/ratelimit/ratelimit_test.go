package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestBurstThenRefill(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(0, 0)
	l := New(time.Minute, 3)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow(ctx, "1.2.3.4"); !ok {
			t.Fatalf("request %d refused inside the burst", i+1)
		}
	}
	ok, wait := l.Allow(ctx, "1.2.3.4")
	if ok || wait != time.Minute {
		t.Fatalf("4th request: ok=%v wait=%s; want refused, retry in 1m", ok, wait)
	}
	if ok, _ := l.Allow(ctx, "5.6.7.8"); !ok {
		t.Fatal("another key was refused: buckets must be per key")
	}
	now = now.Add(time.Minute)
	if ok, _ := l.Allow(ctx, "1.2.3.4"); !ok {
		t.Fatal("no token after one minute")
	}
}
