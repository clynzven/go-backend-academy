package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/clynzven/go-backend-academy/internal/apperr"
	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// refreshTTL is how long a refresh token can renew the session.
const refreshTTL = 30 * 24 * time.Hour

// Tokens is what login and refresh return.
type Tokens struct {
	Token        string `json:"token"`         // short-lived access token (15 minutes)
	RefreshToken string `json:"refresh_token"` // one use only: swap it for a new pair
	ExpiresIn    int    `json:"expires_in"`    // seconds until Token expires
}

// randomText returns n random bytes, base64url encoded.
func randomText(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// hashToken is what we store: a database leak must not hand out live tokens.
func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// issue signs an access token and stores a new refresh token in family.
func (s *Service) issue(ctx context.Context, db execer, userID int64, tenant, family string) (Tokens, error) {
	access, err := Sign(Claims{UserID: userID, Tenant: tenant, Expiry: s.now().Add(tokenTTL).Unix()}, s.secret)
	if err != nil {
		return Tokens{}, err
	}
	refresh := randomText(32)
	if _, err := db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, family, token_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, family, hashToken(refresh), s.now().Add(refreshTTL)); err != nil {
		return Tokens{}, err
	}
	return Tokens{Token: access, RefreshToken: refresh, ExpiresIn: int(tokenTTL.Seconds())}, nil
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// refresh swaps a refresh token for a new pair. Each refresh token works once:
// if a spent one comes back, someone else has a copy, so the whole family
// (every token from that login) is revoked, the thief's copy included.
func (s *Service) refresh(w http.ResponseWriter, r *http.Request) error {
	var req refreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	ctx := r.Context()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // a no-op after Commit

	var (
		id, userID     int64
		family, tenant string
		expires        time.Time
		used, revoked  sql.NullTime
	)
	err = tx.QueryRowContext(ctx,
		`SELECT r.id, r.user_id, r.family, r.expires_at, r.used_at, r.revoked_at, u.tenant_id
		   FROM refresh_tokens r JOIN users u ON u.id = r.user_id
		  WHERE r.token_hash = $1
		    FOR UPDATE OF r`, hashToken(req.RefreshToken)).
		Scan(&id, &userID, &family, &expires, &used, &revoked, &tenant)
	if errors.Is(err, sql.ErrNoRows) {
		return apperr.ErrUnauthorized
	}
	if err != nil {
		return err
	}
	if revoked.Valid || s.now().After(expires) {
		return apperr.ErrUnauthorized
	}
	if used.Valid { // reuse: revoke the whole family, then refuse
		if _, err := tx.ExecContext(ctx,
			`UPDATE refresh_tokens SET revoked_at = now() WHERE family = $1 AND revoked_at IS NULL`, family); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		slog.WarnContext(ctx, "refresh token reuse: family revoked", slog.Int64("user_id", userID))
		return apperr.ErrUnauthorized
	}

	if _, err := tx.ExecContext(ctx, `UPDATE refresh_tokens SET used_at = now() WHERE id = $1`, id); err != nil {
		return err
	}
	tokens, err := s.issue(ctx, tx, userID, tenant, family)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, tokens)
	return nil
}

// logout revokes every token from the same login. The access token still
// works until it expires (at most 15 minutes): that's the price of not
// checking the database on every request.
func (s *Service) logout(w http.ResponseWriter, r *http.Request) error {
	var req refreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(r.Context(),
		`UPDATE refresh_tokens SET revoked_at = now()
		  WHERE family = (SELECT family FROM refresh_tokens WHERE token_hash = $1) AND revoked_at IS NULL`,
		hashToken(req.RefreshToken)); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
