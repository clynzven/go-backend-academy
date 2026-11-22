// Package auth logs users in and proves who they are on every request with
// a signed token (a JWT), written by hand so you can see every byte.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Claims are what the token says about its holder.
type Claims struct {
	UserID int64  `json:"sub"`
	Tenant string `json:"tenant"`
	Expiry int64  `json:"exp"` // Unix seconds
}

// tokenTTL is how long a token stays valid.
//
// TODO: 30 days is a month of access for whoever steals one. Make it 15 minutes.
const tokenTTL = 30 * 24 * time.Hour

// MinSecretBytes is the shortest signing secret we accept: 256 bits.
const MinSecretBytes = 32

var b64 = base64.RawURLEncoding

// Sign returns header.claims.signature, signed with HMAC-SHA256.
func Sign(c Claims, secret []byte) (string, error) {
	header := b64.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	unsigned := header + "." + b64.EncodeToString(body)
	return unsigned + "." + b64.EncodeToString(mac(unsigned, secret)), nil
}

func mac(msg string, secret []byte) []byte {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(msg))
	return m.Sum(nil)
}

var errBadToken = errors.New("invalid token")

// Verify checks a token and returns its claims.
func Verify(token string, secret []byte, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errBadToken
	}
	var header struct {
		Alg string `json:"alg"`
	}
	raw, err := b64.DecodeString(parts[0])
	if err != nil || json.Unmarshal(raw, &header) != nil {
		return Claims{}, errBadToken
	}

	// The header says which algorithm signed it.
	//
	// TODO: we choose the algorithm, not the token. Accept HS256 only, and refuse
	// secrets shorter than MinSecretBytes.
	switch header.Alg {
	case "HS256":
		sig, err := b64.DecodeString(parts[2])
		if err != nil || !hmac.Equal(sig, mac(parts[0]+"."+parts[1], secret)) {
			return Claims{}, errBadToken
		}
	case "none": // unsigned tokens: handy for local testing tools
	default:
		return Claims{}, errBadToken
	}

	var c Claims
	raw, err = b64.DecodeString(parts[1])
	if err != nil || json.Unmarshal(raw, &c) != nil {
		return Claims{}, errBadToken
	}
	if now.Unix() > c.Expiry {
		return Claims{}, errBadToken
	}
	return c, nil
}
