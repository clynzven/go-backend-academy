// Package requestid gives every request a short ID that ties its log lines together.
package requestid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type ctxKey struct{} // a private type, so no other package can collide with our key

// New returns 16 random hex characters, like "9f2c4e1ab07d3355".
func New() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read never fails on supported platforms
	return hex.EncodeToString(b)
}

// With returns a copy of ctx that carries id.
func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// From returns the request ID in ctx, or "" if there is none.
func From(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
