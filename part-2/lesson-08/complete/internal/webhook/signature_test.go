package webhook

import (
	"errors"
	"testing"
	"time"
)

func TestSignature(t *testing.T) {
	secret := []byte("whsec_unit-test-0123456789abcdef0123")
	body := []byte(`{"event_id":"evt_1","type":"payment.succeeded"}`)
	now := time.Unix(1_800_000_000, 0)
	header := Sign(secret, body, now)

	if err := Verify(header, secret, body, now.Add(time.Minute)); err != nil {
		t.Fatalf("a fresh, correct signature: %v", err)
	}
	if err := Verify(header, secret, []byte(`{"event_id":"evt_1","type":"payment.succeeded","x":1}`), now); !errors.Is(err, errBadSig) {
		t.Errorf("edited body: err = %v, want errBadSig", err)
	}
	if err := Verify(header, secret, body, now.Add(10*time.Minute)); !errors.Is(err, errTooOld) {
		t.Errorf("10 minutes later: err = %v, want errTooOld", err)
	}
	if err := Verify("", secret, body, now); !errors.Is(err, errNoSignature) {
		t.Errorf("no header: err = %v, want errNoSignature", err)
	}
}
