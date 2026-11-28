package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/VishRayk/brandshake/internal/auth"
	"github.com/VishRayk/brandshake/internal/deal"
	"github.com/VishRayk/brandshake/internal/httpapi"
	"github.com/VishRayk/brandshake/internal/invoice"
	"github.com/VishRayk/brandshake/internal/profile"
	"github.com/VishRayk/brandshake/internal/testdb"
	"github.com/VishRayk/brandshake/internal/user"
)

func newServer(t *testing.T) *httptest.Server {
	return newServerWith(t, nil)
}

// newServerWith lets a test adjust dependencies (e.g. plug in a fake AI service).
func newServerWith(t *testing.T, configure func(*httpapi.Deps)) *httptest.Server {
	t.Helper()
	db := testdb.New(t)
	deps := httpapi.Deps{
		DB:       db,
		Users:    user.NewRepo(db),
		Profiles: profile.NewRepo(db),
		Deals:    deal.NewRepo(db),
		Invoices: invoice.NewRepo(db, "https://brandshake.test"),
		Tokens:   auth.NewTokens([]byte("test-secret-test-secret-test-secret"), time.Hour),
	}
	if configure != nil {
		configure(&deps)
	}
	srv := httptest.NewServer(httpapi.NewHandler(deps))
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, srv *httptest.Server, method, path, token string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, srv.URL+path, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func signupBody(email string) map[string]string {
	return map[string]string{
		"email": email, "password": "correct-horse-battery", "role": "creator", "display_name": "Arjun Lifts",
	}
}

func TestSignupLoginMe(t *testing.T) {
	t.Parallel()
	srv := newServer(t)

	resp, body := do(t, srv, "POST", "/v1/auth/signup", "", signupBody("Arjun@Example.com"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("signup status = %d, body = %v", resp.StatusCode, body)
	}
	if got := body["user"].(map[string]any)["email"]; got != "arjun@example.com" {
		t.Errorf("email not normalized: %v", got)
	}
	if _, leaked := body["user"].(map[string]any)["password_hash"]; leaked {
		t.Error("password hash leaked in response")
	}

	resp, body = do(t, srv, "POST", "/v1/auth/login", "", map[string]string{
		"email": "arjun@example.com", "password": "correct-horse-battery",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, body = %v", resp.StatusCode, body)
	}
	token := body["access_token"].(string)

	resp, body = do(t, srv, "GET", "/v1/me", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d, body = %v", resp.StatusCode, body)
	}
	if got := body["user"].(map[string]any)["role"]; got != "creator" {
		t.Errorf("role = %v, want creator", got)
	}
}

func TestSignupDuplicateEmail(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	do(t, srv, "POST", "/v1/auth/signup", "", signupBody("dup@example.com"))
	resp, body := do(t, srv, "POST", "/v1/auth/signup", "", signupBody("DUP@example.com"))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409, body = %v", resp.StatusCode, body)
	}
}

func TestSignupValidation(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	resp, body := do(t, srv, "POST", "/v1/auth/signup", "", map[string]string{
		"email": "not-an-email", "password": "short", "role": "admin", "display_name": "",
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
	fields := body["error"].(map[string]any)["fields"].(map[string]any)
	for _, f := range []string{"email", "password", "role", "display_name"} {
		if _, ok := fields[f]; !ok {
			t.Errorf("missing validation error for %q", f)
		}
	}
}

func TestLoginWrongPassword(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	do(t, srv, "POST", "/v1/auth/signup", "", signupBody("a@example.com"))

	for _, email := range []string{"a@example.com", "nobody@example.com"} {
		resp, _ := do(t, srv, "POST", "/v1/auth/login", "", map[string]string{
			"email": email, "password": "wrong-password",
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", email, resp.StatusCode)
		}
	}
}

func TestMeRequiresToken(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	for _, tok := range []string{"", "garbage"} {
		resp, _ := do(t, srv, "GET", "/v1/me", tok, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("token %q: status = %d, want 401", tok, resp.StatusCode)
		}
	}
}

func TestRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	b := signupBody("x@example.com")
	b["is_admin"] = "true"
	resp, _ := do(t, srv, "POST", "/v1/auth/signup", "", b)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
