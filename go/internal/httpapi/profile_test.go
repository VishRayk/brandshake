package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// signup creates an account and returns its access token.
func signup(t *testing.T, srv *httptest.Server, email, role string) string {
	t.Helper()
	resp, body := do(t, srv, "POST", "/v1/auth/signup", "", map[string]string{
		"email": email, "password": "correct-horse-battery", "role": role, "display_name": "Test " + role,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("signup %s: %d %v", email, resp.StatusCode, body)
	}
	return body["access_token"].(string)
}

func fields(body map[string]any) map[string]any {
	e, _ := body["error"].(map[string]any)
	f, _ := e["fields"].(map[string]any)
	return f
}

func TestCreatorProfileLifecycle(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "maya@example.com", "creator")

	// No profile yet.
	resp, body := do(t, srv, "GET", "/v1/me/profile", tok, nil)
	if resp.StatusCode != 200 || body["profile"] != nil {
		t.Fatalf("empty profile: %d %v", resp.StatusCode, body)
	}

	resp, body = do(t, srv, "PUT", "/v1/me/profile", tok, map[string]any{
		"handle": "@Maya.Rao", "headline": "Fitness & food creator", "city": "Bengaluru",
		"state": "Karnataka", "niches": []string{"fitness", "food", "fitness"}, "languages": []string{"en", "kn"},
	})
	if resp.StatusCode != 200 {
		t.Fatalf("put profile: %d %v", resp.StatusCode, body)
	}
	p := body["profile"].(map[string]any)
	if p["handle"] != "maya.rao" {
		t.Errorf("handle not normalised: %v", p["handle"])
	}
	if n := p["niches"].([]any); len(n) != 2 {
		t.Errorf("niches not deduplicated: %v", n)
	}
	if p["open_to_collabs"] != true {
		t.Errorf("open_to_collabs should default to true")
	}

	resp, body = do(t, srv, "PUT", "/v1/me/socials", tok, map[string]any{"socials": []map[string]any{
		{"platform": "instagram", "handle": "@maya.rao", "url": "https://www.instagram.com/maya.rao", "followers": 48200, "verified": true},
		{"platform": "youtube", "handle": "MayaRao", "url": "https://youtube.com/@MayaRao", "followers": 12000},
	}})
	if resp.StatusCode != 200 {
		t.Fatalf("put socials: %d %v", resp.StatusCode, body)
	}

	resp, body = do(t, srv, "PUT", "/v1/me/rates", tok, map[string]any{"rates": []map[string]any{
		{"deliverable": "Instagram Reel", "price_minor": 1_500_000, "note": "Includes 1 revision"},
		{"deliverable": "Story (3 frames)", "price_minor": 500_000},
	}})
	if resp.StatusCode != 200 {
		t.Fatalf("put rates: %d %v", resp.StatusCode, body)
	}

	resp, body = do(t, srv, "PUT", "/v1/me/samples", tok, map[string]any{"samples": []map[string]any{
		{"title": "Protein bar launch Reel", "url": "https://www.instagram.com/reel/abc123", "brand_name": "FitBar"},
	}})
	if resp.StatusCode != 200 {
		t.Fatalf("put samples: %d %v", resp.StatusCode, body)
	}
	if got := body["samples"].([]any)[0].(map[string]any)["platform"]; got != "instagram" {
		t.Errorf("platform not inferred from URL: %v", got)
	}

	// Public profile: no auth, no email, client can't self-verify.
	resp, body = do(t, srv, "GET", "/v1/profiles/MAYA.RAO", "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("public profile: %d %v", resp.StatusCode, body)
	}
	pub := body["profile"].(map[string]any)
	if _, leaked := pub["email"]; leaked {
		t.Error("public profile leaks email")
	}
	if pub["role"] != "creator" || pub["display_name"] != "Test creator" {
		t.Errorf("unexpected public profile: %v", pub)
	}
	socials := pub["socials"].([]any)
	if len(socials) != 2 || socials[0].(map[string]any)["verified"] != false {
		t.Errorf("socials wrong or self-verified: %v", socials)
	}
	if rates := pub["rates"].([]any); len(rates) != 2 || rates[0].(map[string]any)["price_minor"] != float64(1_500_000) {
		t.Errorf("rates wrong: %v", rates)
	}
	if resp.Header.Get("Cache-Control") == "" {
		t.Error("public profile should be cacheable")
	}

	// Replacing a list replaces it entirely.
	do(t, srv, "PUT", "/v1/me/socials", tok, map[string]any{"socials": []map[string]any{}})
	_, body = do(t, srv, "GET", "/v1/me/profile", tok, nil)
	if s := body["socials"].([]any); len(s) != 0 {
		t.Errorf("socials not cleared: %v", s)
	}
}

func TestHandleRules(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	a := signup(t, srv, "a@example.com", "creator")
	b := signup(t, srv, "b@example.com", "brand")

	if resp, body := do(t, srv, "PUT", "/v1/me/profile", a, map[string]any{"handle": "taken_one"}); resp.StatusCode != 200 {
		t.Fatalf("first claim: %d %v", resp.StatusCode, body)
	}

	resp, body := do(t, srv, "PUT", "/v1/me/profile", b, map[string]any{"handle": "taken_one"})
	if resp.StatusCode != http.StatusUnprocessableEntity || fields(body)["handle"] == nil {
		t.Fatalf("duplicate handle: %d %v", resp.StatusCode, body)
	}

	for _, h := range []string{"admin", "ab", "has space", "-dash", "double..dot", "end."} {
		resp, body := do(t, srv, "PUT", "/v1/me/profile", b, map[string]any{"handle": h})
		if resp.StatusCode != http.StatusUnprocessableEntity || fields(body)["handle"] == nil {
			t.Errorf("handle %q accepted: %d %v", h, resp.StatusCode, body)
		}
	}

	checks := map[string]bool{"taken_one": false, "admin": false, "fresh.name": true}
	for h, want := range checks {
		_, body := do(t, srv, "GET", "/v1/handles/"+h, b, nil)
		if body["available"] != want {
			t.Errorf("availability of %q = %v, want %v", h, body["available"], want)
		}
	}
	// Your own handle counts as available to you.
	if _, body := do(t, srv, "GET", "/v1/handles/taken_one", a, nil); body["available"] != true {
		t.Errorf("own handle should be available to its owner")
	}
}

func TestBrandProfileRules(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "hello@fitbar.example", "brand")

	resp, body := do(t, srv, "PUT", "/v1/me/profile", tok, map[string]any{
		"handle": "fitbar", "industry": "food_beverage", "website": "https://fitbar.example",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("brand profile: %d %v", resp.StatusCode, body)
	}
	if body["profile"].(map[string]any)["open_to_collabs"] != false {
		t.Error("brands should not be marked open to collabs")
	}

	resp, body = do(t, srv, "PUT", "/v1/me/profile", tok, map[string]any{
		"handle": "fitbar", "industry": "crypto", "niches": []string{"fitness"}, "website": "fitbar.example",
	})
	f := fields(body)
	if resp.StatusCode != http.StatusUnprocessableEntity || f["industry"] == nil || f["niches"] == nil || f["website"] == nil {
		t.Fatalf("invalid brand profile accepted: %d %v", resp.StatusCode, body)
	}

	for _, path := range []string{"/v1/me/rates", "/v1/me/samples"} {
		if resp, _ := do(t, srv, "PUT", path, tok, map[string]any{}); resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s: brand got %d, want 403", path, resp.StatusCode)
		}
	}
}

func TestListValidation(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "v@example.com", "creator")

	resp, body := do(t, srv, "PUT", "/v1/me/socials", tok, map[string]any{"socials": []map[string]any{
		{"platform": "instagram", "handle": "me", "url": "https://evil.example/instagram.com/me", "followers": 10},
		{"platform": "instagram", "handle": "me2", "url": "https://instagram.com/me2", "followers": 10},
		{"platform": "myspace", "handle": "x", "url": "https://myspace.com/x"},
	}})
	f := fields(body)
	if resp.StatusCode != http.StatusUnprocessableEntity ||
		f["socials[0].url"] == nil || f["socials[1].platform"] == nil || f["socials[2].platform"] == nil {
		t.Fatalf("bad socials accepted: %d %v", resp.StatusCode, body)
	}

	resp, body = do(t, srv, "PUT", "/v1/me/rates", tok, map[string]any{"rates": []map[string]any{
		{"deliverable": "", "price_minor": -1},
	}})
	f = fields(body)
	if resp.StatusCode != http.StatusUnprocessableEntity || f["rates[0].deliverable"] == nil || f["rates[0].price_minor"] == nil {
		t.Fatalf("bad rates accepted: %d %v", resp.StatusCode, body)
	}
}

func TestPublicProfileNotFound(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	if resp, _ := do(t, srv, "GET", "/v1/profiles/nobody_here", "", nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestExportAndDeleteAccount(t *testing.T) {
	t.Parallel()
	srv, db := newServerDB(t)
	tok := signup(t, srv, "leaving@example.com", "creator")
	do(t, srv, "PUT", "/v1/me/profile", tok, map[string]any{"handle": "leaving_soon"})
	// An email addressed to them, already sent, and one still queued.
	_, _ = db.Exec(context.Background(), `INSERT INTO emails (to_address, subject, body_text, kind, provider) VALUES ('leaving@example.com', 's', 'b', 'test', 'log')`)
	_, _ = db.Exec(context.Background(), `INSERT INTO jobs (kind, payload) VALUES ('email.send', '{"to": "leaving@example.com"}')`)

	resp, body := do(t, srv, "GET", "/v1/me/export", tok, nil)
	if resp.StatusCode != 200 || body["account"].(map[string]any)["email"] != "leaving@example.com" || body["profile"] == nil {
		t.Fatalf("export: %d %v", resp.StatusCode, body)
	}
	if _, leaked := body["account"].(map[string]any)["password_hash"]; leaked {
		t.Fatal("export leaks the password hash")
	}

	if resp, _ := do(t, srv, "DELETE", "/v1/me", tok, map[string]string{"password": "wrong"}); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("delete with wrong password: %d", resp.StatusCode)
	}
	if resp, _ := do(t, srv, "DELETE", "/v1/me", tok, map[string]string{"password": "correct-horse-battery"}); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d", resp.StatusCode)
	}

	// Everything is gone; the old token no longer works for anything.
	if resp, _ := do(t, srv, "GET", "/v1/profiles/leaving_soon", "", nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("profile still public after delete: %d", resp.StatusCode)
	}
	if resp, _ := do(t, srv, "GET", "/v1/me", tok, nil); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("/v1/me after delete: %d", resp.StatusCode)
	}
	if resp, _ := do(t, srv, "PUT", "/v1/me/profile", tok, map[string]any{"handle": "ghost_user"}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("profile write after delete: %d", resp.StatusCode)
	}
	if resp, _ := do(t, srv, "POST", "/v1/auth/login", "", map[string]string{"email": "leaving@example.com", "password": "correct-horse-battery"}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("login after delete: %d", resp.StatusCode)
	}
	var leftover int
	_ = db.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM emails WHERE to_address = 'leaving@example.com') + (SELECT count(*) FROM jobs WHERE payload->>'to' = 'leaving@example.com')`).Scan(&leftover)
	if leftover != 0 {
		t.Errorf("%d emails/jobs for the deleted address remain", leftover)
	}
}
