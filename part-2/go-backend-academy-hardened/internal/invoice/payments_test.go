package invoice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// TestConcurrentPaymentsChargeOnce is the race as a test: 50 requests pay the
// same invoice at the same moment. Exactly one may win.
func TestConcurrentPaymentsChargeOnce(t *testing.T) {
	s := newStore(t)
	s.Charger = FakeProvider{Latency: 20 * time.Millisecond}
	inv := create(t, s, "acme", "Acme Corp")

	var mu sync.Mutex
	won, refused := 0, 0
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := s.Pay(context.Background(), "acme", inv.ID, "")
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				won++
			case errors.Is(err, apperr.ErrAlreadyPaid):
				refused++
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	payments, err := s.Payments(context.Background(), "acme", inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if won != 1 || refused != 49 || len(payments) != 1 {
		t.Fatalf("50 concurrent payments: %d paid, %d refused, %d payment rows; want 1, 49, 1", won, refused, len(payments))
	}
}

// TestIdempotencyKeyReplays: retries with the same key get the same payment.
func TestIdempotencyKeyReplays(t *testing.T) {
	s := newStore(t)
	inv := create(t, s, "acme", "Acme Corp")

	ids := make(chan int64, 10)
	replays := make(chan bool, 10)
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, replayed, err := s.Pay(context.Background(), "acme", inv.ID, "order-7f3a")
			if err != nil {
				t.Error(err)
				return
			}
			ids <- p.ID
			replays <- replayed
		}()
	}
	wg.Wait()
	close(ids)
	close(replays)
	first, fresh := int64(0), 0
	for id := range ids {
		if first == 0 {
			first = id
		}
		if id != first {
			t.Fatalf("same key, different payments: %d and %d", first, id)
		}
	}
	for r := range replays {
		if !r {
			fresh++
		}
	}
	if fresh != 1 {
		t.Fatalf("%d requests created a payment, want exactly 1 (the rest replayed)", fresh)
	}
}
