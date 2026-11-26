// Package httpapi exposes the Brandshake REST API.
package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/VishRayk/brandshake/internal/auth"
	"github.com/VishRayk/brandshake/internal/user"
)

type Deps struct {
	DB             *sql.DB
	Users          *user.Repo
	Tokens         *auth.Tokens
	AllowedOrigins []string
}

type api struct{ Deps }

func NewHandler(d Deps) http.Handler {
	a := &api{Deps: d}
	requireAuth := d.Tokens.Middleware(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /readyz", a.ready)

	mux.HandleFunc("POST /v1/auth/signup", a.signup)
	mux.HandleFunc("POST /v1/auth/login", a.login)
	mux.Handle("GET /v1/me", requireAuth(http.HandlerFunc(a.me)))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "Route not found.")
	})

	var h http.Handler = mux
	for _, mw := range []func(http.Handler) http.Handler{
		securityHeaders, cors(d.AllowedOrigins), logRequests, recoverPanics, requestID,
	} {
		h = mw(h)
	}
	return h
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *api) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.DB.PingContext(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "Database unavailable.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
