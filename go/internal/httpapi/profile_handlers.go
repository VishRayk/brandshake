package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/VishRayk/brandshake/internal/auth"
	"github.com/VishRayk/brandshake/internal/profile"
	"github.com/VishRayk/brandshake/internal/store"
	"github.com/VishRayk/brandshake/internal/user"
)

type profileRequest struct {
	Handle        string   `json:"handle"`
	Headline      string   `json:"headline"`
	Bio           string   `json:"bio"`
	City          string   `json:"city"`
	State         string   `json:"state"`
	Website       string   `json:"website"`
	Niches        []string `json:"niches"`
	Languages     []string `json:"languages"`
	OpenToCollabs *bool    `json:"open_to_collabs"`
	Industry      string   `json:"industry"`
}

type myProfileResponse struct {
	Profile *profile.Profile        `json:"profile"`
	Socials []profile.SocialAccount `json:"socials"`
	Rates   []profile.Rate          `json:"rates"`
	Samples []profile.Sample        `json:"samples"`
}

func principal(r *http.Request) auth.Principal {
	p, _ := auth.PrincipalFrom(r.Context())
	return p
}

// writeStoreError maps common database errors to responses.
func writeStoreError(w http.ResponseWriter, r *http.Request, err error) {
	if store.IsForeignKeyViolation(err) {
		// The account was deleted while its token was still valid.
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
		return
	}
	writeInternal(w, r, err)
}

func (a *api) getMyProfile(w http.ResponseWriter, r *http.Request) {
	uid := principal(r).UserID
	resp := myProfileResponse{}
	p, err := a.Profiles.Get(r.Context(), uid)
	switch {
	case errors.Is(err, profile.ErrNotFound):
	case err != nil:
		writeInternal(w, r, err)
		return
	default:
		resp.Profile = &p
	}
	if resp.Socials, resp.Rates, resp.Samples, err = a.Profiles.Lists(r.Context(), uid); err != nil {
		writeInternal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *api) putMyProfile(w http.ResponseWriter, r *http.Request) {
	pr := principal(r)
	var req profileRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	p := profile.Profile{
		Handle: req.Handle, Headline: req.Headline, Bio: req.Bio, City: req.City, State: req.State,
		Website: req.Website, Niches: req.Niches, Languages: req.Languages, Industry: req.Industry,
		OpenToCollabs: req.OpenToCollabs == nil || *req.OpenToCollabs,
	}
	p.Normalize()
	if errs := p.Validate(pr.Role); len(errs) > 0 {
		writeValidation(w, errs)
		return
	}

	saved, err := a.Profiles.Upsert(r.Context(), pr.UserID, p)
	if errors.Is(err, profile.ErrHandleTaken) {
		writeValidation(w, map[string]string{"handle": "This handle is already taken."})
		return
	}
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]profile.Profile{"profile": saved})
}

func (a *api) checkHandle(w http.ResponseWriter, r *http.Request) {
	h := profile.NormalizeHandle(r.PathValue("handle"))
	resp := map[string]any{"handle": h, "available": false}
	if msg := profile.CheckHandle(h); msg != "" {
		resp["reason"] = msg
		writeJSON(w, http.StatusOK, resp)
		return
	}
	ok, err := a.Profiles.HandleAvailable(r.Context(), h, principal(r).UserID)
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	resp["available"] = ok
	if !ok {
		resp["reason"] = "This handle is already taken."
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *api) putSocials(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Socials []profile.SocialAccount `json:"socials"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if errs := profile.ValidateSocials(req.Socials); len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	if err := a.Profiles.ReplaceSocials(r.Context(), principal(r).UserID, req.Socials); err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"socials": nonNilSlice(req.Socials)})
}

func (a *api) putRates(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", "Only creators have a rate card.") {
		return
	}
	var req struct {
		Rates []profile.Rate `json:"rates"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if errs := profile.ValidateRates(req.Rates); len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	if err := a.Profiles.ReplaceRates(r.Context(), principal(r).UserID, req.Rates); err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rates": nonNilSlice(req.Rates)})
}

func (a *api) putSamples(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", "Only creators add work samples.") {
		return
	}
	var req struct {
		Samples []profile.Sample `json:"samples"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if errs := profile.ValidateSamples(req.Samples); len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	if err := a.Profiles.ReplaceSamples(r.Context(), principal(r).UserID, req.Samples); err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"samples": nonNilSlice(req.Samples)})
}

func (a *api) publicProfile(w http.ResponseWriter, r *http.Request) {
	pub, err := a.Profiles.PublicByHandle(r.Context(), profile.NormalizeHandle(r.PathValue("handle")))
	if errors.Is(err, profile.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "This profile doesn't exist.")
		return
	}
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, map[string]profile.Public{"profile": pub})
}

// exportMyData returns everything stored about the caller (DPDP Act right of access).
func (a *api) exportMyData(w http.ResponseWriter, r *http.Request) {
	uid := principal(r).UserID
	u, err := a.Users.ByID(r.Context(), uid)
	if errors.Is(err, user.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
		return
	}
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	export := map[string]any{"exported_at": time.Now().UTC(), "account": u}
	if p, err := a.Profiles.Get(r.Context(), uid); err == nil {
		export["profile"] = p
	} else if !errors.Is(err, profile.ErrNotFound) {
		writeInternal(w, r, err)
		return
	}
	socials, rates, samples, err := a.Profiles.Lists(r.Context(), uid)
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	export["socials"], export["rates"], export["samples"] = socials, rates, samples

	w.Header().Set("Content-Disposition", `attachment; filename="brandshake-data.json"`)
	writeJSON(w, http.StatusOK, export)
}

// deleteAccount permanently deletes the caller's account and all their data.
func (a *api) deleteAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	uid := principal(r).UserID
	u, err := a.Users.ByID(r.Context(), uid)
	if errors.Is(err, user.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
		return
	}
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeValidation(w, map[string]string{"password": "Password is incorrect."})
		return
	}
	if err := a.Users.Delete(r.Context(), uid); err != nil {
		writeInternal(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func requireRole(w http.ResponseWriter, r *http.Request, role, msg string) bool {
	if principal(r).Role != role {
		writeError(w, http.StatusForbidden, "forbidden", msg)
		return false
	}
	return true
}

func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
