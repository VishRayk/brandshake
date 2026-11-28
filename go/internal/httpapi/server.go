// Package httpapi exposes the Brandshake REST API.
package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VishRayk/brandshake/internal/aiclient"
	"github.com/VishRayk/brandshake/internal/auth"
	"github.com/VishRayk/brandshake/internal/deal"
	"github.com/VishRayk/brandshake/internal/email"
	"github.com/VishRayk/brandshake/internal/profile"
	"github.com/VishRayk/brandshake/internal/user"
)

type Deps struct {
	DB             *pgxpool.Pool
	Users          *user.Repo
	Profiles       *profile.Repo
	Deals          *deal.Repo
	AI             *aiclient.Client // nil disables AI features
	DevMode        bool             // enables development-only endpoints
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
	authed := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, requireAuth(h)) }
	authed("GET /v1/me", a.me)
	authed("DELETE /v1/me", a.deleteAccount)
	authed("GET /v1/me/export", a.exportMyData)
	authed("GET /v1/me/profile", a.getMyProfile)
	authed("PUT /v1/me/profile", a.putMyProfile)
	authed("PUT /v1/me/socials", a.putSocials)
	authed("PUT /v1/me/rates", a.putRates)
	authed("PUT /v1/me/samples", a.putSamples)
	authed("GET /v1/handles/{handle}", a.checkHandle)

	authed("GET /v1/deals", a.listDeals)
	authed("POST /v1/deals", a.createDeal)
	authed("GET /v1/deals/summary", a.dealSummary)
	authed("POST /v1/deals/draft", a.draftDeal)
	authed("GET /v1/deals/{id}", a.getDeal)
	authed("PUT /v1/deals/{id}", a.updateDeal)
	authed("DELETE /v1/deals/{id}", a.deleteDeal)
	authed("PUT /v1/deals/{id}/status", a.setDealStatus)
	authed("PUT /v1/deals/{id}/deliverables/{position}", a.setDeliverableDone)
	authed("GET /v1/deals/{id}/events", a.dealEvents)

	mux.HandleFunc("GET /v1/profiles/{handle}", a.publicProfile)

	if d.DevMode {
		// Emails the log provider "sent" to you; lets you click links in development.
		authed("GET /v1/dev/outbox", a.devOutbox)
	}

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
	if err := a.DB.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "Database unavailable.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *api) devOutbox(w http.ResponseWriter, r *http.Request) {
	u, err := a.Users.ByID(r.Context(), principal(r).UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
		return
	}
	emails, err := email.Recent(r.Context(), a.DB, u.Email, 50)
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"emails": emails})
}
