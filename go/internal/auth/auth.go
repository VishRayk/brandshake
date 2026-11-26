// Package auth handles password hashing, access tokens and the
// authentication middleware.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(b), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// dummyHash is compared against when a user does not exist, so login
// takes the same time whether or not the email is registered.
var dummyHash, _ = HashPassword("timing-equalizer-password")

func CheckPasswordOrDummy(hash string, password string) bool {
	if hash == "" {
		_ = CheckPassword(dummyHash, password)
		return false
	}
	return CheckPassword(hash, password)
}

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type Tokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokens(secret []byte, ttl time.Duration) *Tokens {
	return &Tokens{secret: secret, ttl: ttl, now: time.Now}
}

func (t *Tokens) Issue(userID, role string) (token string, expiresAt time.Time, err error) {
	now := t.now()
	expiresAt = now.Add(t.ttl)
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    "brandshake",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	return token, expiresAt, err
}

var ErrInvalidToken = errors.New("invalid token")

func (t *Tokens) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(tok *jwt.Token) (any, error) {
		return t.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer("brandshake"),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	return claims, nil
}

type ctxKey struct{}

// Principal is the authenticated caller.
type Principal struct {
	UserID string
	Role   string
}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}

// Middleware rejects requests without a valid bearer token and stores the
// caller's identity in the request context.
func (t *Tokens) Middleware(onUnauthorized http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || raw == "" {
				onUnauthorized(w, r)
				return
			}
			claims, err := t.Parse(raw)
			if err != nil {
				onUnauthorized(w, r)
				return
			}
			ctx := WithPrincipal(r.Context(), Principal{UserID: claims.Subject, Role: claims.Role})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
