package auth

import (
	"testing"
	"time"
)

// TestAlgNoneIsRejected: an unsigned token that names its own algorithm
// ("none") must never be accepted, whatever its claims say.
func TestAlgNoneIsRejected(t *testing.T) {
	header := b64.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	claims := b64.EncodeToString([]byte(`{"sub":999,"tenant":"globex","exp":9999999999}`))
	if _, err := Verify(header+"."+claims+".", secret, time.Now()); err == nil {
		t.Fatal(`a token with "alg":"none" was accepted`)
	}
}

// TestWeakSecretsAreRefused: a short secret can be guessed offline from one
// captured token, so Verify refuses to work with one.
func TestWeakSecretsAreRefused(t *testing.T) {
	weak := []byte("changeme")
	tok, err := Sign(Claims{UserID: 1, Tenant: "acme", Expiry: time.Now().Add(time.Minute).Unix()}, weak)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(tok, weak, time.Now()); err == nil {
		t.Fatal("a token signed with an 8-byte secret was accepted")
	}
}

// TestAccessTokensAreShortLived: a stolen access token is worth minutes.
func TestAccessTokensAreShortLived(t *testing.T) {
	if tokenTTL > 15*time.Minute {
		t.Fatalf("tokenTTL = %s, want 15 minutes or less", tokenTTL)
	}
}
