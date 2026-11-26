package httpapi

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/VishRayk/brandshake/internal/auth"
	"github.com/VishRayk/brandshake/internal/user"
)

type signupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sessionResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        user.User `json:"user"`
}

func (req signupRequest) validate() map[string]string {
	errs := map[string]string{}
	if addr, err := mail.ParseAddress(req.Email); err != nil || addr.Address != strings.TrimSpace(req.Email) {
		errs["email"] = "Enter a valid email address."
	}
	if n := utf8.RuneCountInString(req.Password); n < 8 || n > 128 {
		errs["password"] = "Password must be 8–128 characters."
	}
	if !user.Role(req.Role).Valid() {
		errs["role"] = "Role must be creator or brand."
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(req.DisplayName)); n < 1 || n > 80 {
		errs["display_name"] = "Name must be 1–80 characters."
	}
	return errs
}

func (a *api) signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if errs := req.validate(); len(errs) > 0 {
		writeValidation(w, errs)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	u, err := a.Users.Create(r.Context(), req.Email, hash, user.Role(req.Role), req.DisplayName)
	if errors.Is(err, user.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email_taken", "An account with this email already exists.")
		return
	}
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	a.writeSession(w, r, http.StatusCreated, u)
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	u, err := a.Users.ByEmail(r.Context(), req.Email)
	if err != nil && !errors.Is(err, user.ErrNotFound) {
		writeInternal(w, r, err)
		return
	}
	if !auth.CheckPasswordOrDummy(u.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect.")
		return
	}
	a.writeSession(w, r, http.StatusOK, u)
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	u, err := a.Users.ByID(r.Context(), p.UserID)
	if errors.Is(err, user.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
		return
	}
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]user.User{"user": u})
}

func (a *api) writeSession(w http.ResponseWriter, r *http.Request, status int, u user.User) {
	token, exp, err := a.Tokens.Issue(u.ID, string(u.Role))
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	writeJSON(w, status, sessionResponse{AccessToken: token, ExpiresAt: exp, User: u})
}
