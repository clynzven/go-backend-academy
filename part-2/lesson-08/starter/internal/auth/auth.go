package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/clynzven/go-backend-academy/internal/apperr"
	"github.com/clynzven/go-backend-academy/internal/httpx"
	"github.com/clynzven/go-backend-academy/internal/ratelimit"
)

// Service logs users in and checks tokens.
type Service struct {
	db     *sql.DB
	secret []byte
	now    func() time.Time

	perIP      ratelimit.Limiter // login attempts per client IP
	perAccount ratelimit.Limiter // failed logins per email: the lockout

	// TrustedProxies may tell us the client's address in X-Forwarded-For.
	TrustedProxies []netip.Prefix
}

// NewService returns a Service that signs tokens with secret.
//
// TODO: three copies of the API mean three separate in-memory limits. Keep the
// buckets in Postgres (ratelimit.NewPostgres) so every copy shares one.
func NewService(db *sql.DB, secret []byte) *Service {
	return &Service{db: db, secret: secret, now: time.Now,
		perIP:      ratelimit.New(6*time.Second, 10), // 10 at once, then 10 a minute
		perAccount: ratelimit.New(3*time.Minute, 5),  // 5 wrong passwords, then 1 every 3 minutes
	}
}

// clientIP: behind the proxy every connection comes from the proxy's address,
// so we read the client's address from X-Forwarded-For.
//
// TODO: the first entry is whatever the client typed. Trust the header only
// from s.TrustedProxies, and take the LAST entry (httpx.ClientIP).
func (s *Service) clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

// AddUser creates a user in tenant (used by cmd/adduser and the tests).
func (s *Service) AddUser(ctx context.Context, tenant, email, password string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (tenant_id, email, password_hash) VALUES ($1, $2, $3) RETURNING id`,
		tenant, strings.ToLower(email), HashPassword(password)).Scan(&id)
	return id, err
}

// Register adds POST /login, POST /refresh and POST /logout.
func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /login", httpx.Handle(s.login))
	mux.HandleFunc("POST /refresh", httpx.Handle(s.refresh))
	mux.HandleFunc("POST /logout", httpx.Handle(s.logout))
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Service) login(w http.ResponseWriter, r *http.Request) error {
	// Rate limit before anything expensive: argon2id costs 19 MiB per guess.
	if ok, wait := s.perIP.Allow(r.Context(), "ip:"+s.clientIP(r)); !ok {
		return &apperr.TooManyRequestsError{RetryAfter: wait}
	}
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	email := strings.ToLower(req.Email)
	if ok, wait := s.perAccount.Peek(r.Context(), email); !ok { // locked: don't even check the password
		return &apperr.TooManyRequestsError{RetryAfter: wait}
	}

	var (
		id     int64
		tenant string
		stored string
	)
	err := s.db.QueryRowContext(r.Context(),
		`SELECT id, tenant_id, password_hash FROM users WHERE email = $1`, email).Scan(&id, &tenant, &stored)
	if errors.Is(err, sql.ErrNoRows) {
		checkPassword(req.Password, dummyHash) // same time as a real check
		return apperr.ErrUnauthorized
	}
	if err != nil {
		return err
	}
	ok, upgrade := checkPassword(req.Password, stored)
	if !ok {
		s.perAccount.Allow(r.Context(), email) // a failure spends one of the account's 5 tokens
		return apperr.ErrUnauthorized          // same answer for "no such user" and "wrong password"
	}
	s.perAccount.Reset(r.Context(), email)
	if upgrade { // an old SHA-256 hash: we know the password now, so store argon2id
		if _, err := s.db.ExecContext(r.Context(),
			`UPDATE users SET password_hash = $1 WHERE id = $2`, HashPassword(req.Password), id); err != nil {
			return err
		}
	}
	tokens, err := s.issue(r.Context(), s.db, id, tenant, randomText(16)) // a new family per login
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, tokens)
	return nil
}

// Middleware verifies "Authorization: Bearer <token>" and puts the identity
// it proves into the request context. No token: the request goes on without
// an identity, and any route that needs a tenant answers 401.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			next.ServeHTTP(w, r)
			return
		}
		token, ok := strings.CutPrefix(header, "Bearer ")
		claims, err := Verify(token, s.secret, s.now())
		if !ok || err != nil {
			httpx.WriteErr(w, r, apperr.ErrUnauthorized)
			return
		}
		id := httpx.Identity{UserID: claims.UserID, Tenant: claims.Tenant}
		next.ServeHTTP(w, r.WithContext(httpx.WithIdentity(r.Context(), id)))
	})
}
