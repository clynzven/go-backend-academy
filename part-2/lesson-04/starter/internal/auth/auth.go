package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/clynzven/go-backend-academy/internal/apperr"
	"github.com/clynzven/go-backend-academy/internal/httpx"
)

// HashPassword turns a password into what we store in users.password_hash.
//
// TODO: one fast, unsalted SHA-256 is billions of guesses per second on a GPU.
// Use argon2id with a random salt (golang.org/x/crypto/argon2), and rate-limit
// POST /login per IP and per account.
func HashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func passwordMatches(password, stored string) bool {
	return subtle.ConstantTimeCompare([]byte(HashPassword(password)), []byte(stored)) == 1
}

// Service logs users in and checks tokens.
type Service struct {
	db     *sql.DB
	secret []byte
	now    func() time.Time
}

// NewService returns a Service that signs tokens with secret.
func NewService(db *sql.DB, secret []byte) *Service {
	return &Service{db: db, secret: secret, now: time.Now}
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
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	var (
		id     int64
		tenant string
		stored string
	)
	err := s.db.QueryRowContext(r.Context(),
		`SELECT id, tenant_id, password_hash FROM users WHERE email = $1`,
		strings.ToLower(req.Email)).Scan(&id, &tenant, &stored)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !passwordMatches(req.Password, stored)) {
		return apperr.ErrUnauthorized // same answer for "no such user" and "wrong password"
	}
	if err != nil {
		return err
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
