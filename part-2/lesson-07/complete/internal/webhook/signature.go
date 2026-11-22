package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Tolerance is how old (or how far in the future) a signed event may be.
// A captured request is useless to an attacker after 5 minutes.
const Tolerance = 5 * time.Minute

var (
	errNoSignature = errors.New("missing or malformed Webhook-Signature")
	errTooOld      = errors.New("signature timestamp outside the 5-minute window")
	errBadSig      = errors.New("signature does not match")
)

// Sign returns a Webhook-Signature header value: t=<unix time>,v1=<hex HMAC>.
// The HMAC covers the timestamp AND the exact body bytes, so neither can be
// changed without the secret.
func Sign(secret, body []byte, t time.Time) string {
	return fmt.Sprintf("t=%d,v1=%s", t.Unix(), hex.EncodeToString(mac(secret, t.Unix(), body)))
}

func mac(secret []byte, unix int64, body []byte) []byte {
	m := hmac.New(sha256.New, secret)
	fmt.Fprintf(m, "%d.", unix)
	m.Write(body)
	return m.Sum(nil)
}

// Verify checks a Webhook-Signature header against the raw body.
func Verify(header string, secret, body []byte, now time.Time) error {
	var unix int64
	var sig []byte
	for _, part := range strings.Split(header, ",") {
		k, v, _ := strings.Cut(strings.TrimSpace(part), "=")
		switch k {
		case "t":
			unix, _ = strconv.ParseInt(v, 10, 64)
		case "v1":
			sig, _ = hex.DecodeString(v)
		}
	}
	if unix == 0 || len(sig) == 0 {
		return errNoSignature
	}
	if d := now.Sub(time.Unix(unix, 0)); d > Tolerance || d < -Tolerance {
		return errTooOld
	}
	if !hmac.Equal(sig, mac(secret, unix, body)) { // constant time: no byte-by-byte timing leak
		return errBadSig
	}
	return nil
}
