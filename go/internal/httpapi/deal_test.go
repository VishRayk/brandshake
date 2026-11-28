package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/VishRayk/brandshake/internal/aiclient"
	"github.com/VishRayk/brandshake/internal/httpapi"
)

func newDeal(t *testing.T, srv *httptest.Server, tok string, body map[string]any) map[string]any {
	t.Helper()
	resp, out := do(t, srv, "POST", "/v1/deals", tok, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create deal: %d %v", resp.StatusCode, out)
	}
	return out["deal"].(map[string]any)
}

func TestDealLifecycle(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "maya@example.com", "creator")

	d := newDeal(t, srv, tok, map[string]any{
		"title": "  FitBar protein launch ", "counterparty_name": "FitBar", "counterparty_email": "Priya@FitBar.in",
		"amount_minor": 1_500_000, "revisions": 1, "usage_rights_days": 90,
		"deliverables": []map[string]any{
			{"title": "Instagram reel 1 of 2", "platform": "instagram", "due_date": "2026-12-10"},
			{"title": "Instagram reel 2 of 2", "platform": "instagram"},
		},
	})
	id := d["id"].(string)
	if d["title"] != "FitBar protein launch" || d["status"] != "lead" || d["source"] != "manual" {
		t.Errorf("unexpected deal: %v", d)
	}
	if d["counterparty"].(map[string]any)["email"] != "priya@fitbar.in" {
		t.Errorf("email not normalised: %v", d["counterparty"])
	}
	if dl := d["deliverables"].([]any); len(dl) != 2 || dl[0].(map[string]any)["due_date"] != "2026-12-10" {
		t.Errorf("deliverables: %v", dl)
	}

	// Status changes are recorded; setting the same status twice is a no-op.
	for _, st := range []string{"confirmed", "confirmed", "in_progress"} {
		if resp, body := do(t, srv, "PUT", "/v1/deals/"+id+"/status", tok, map[string]string{"status": st}); resp.StatusCode != 200 {
			t.Fatalf("status %s: %d %v", st, resp.StatusCode, body)
		}
	}
	resp, body := do(t, srv, "PUT", "/v1/deals/"+id+"/deliverables/0", tok, map[string]bool{"done": true})
	if resp.StatusCode != 200 {
		t.Fatalf("toggle: %d %v", resp.StatusCode, body)
	}
	first := body["deal"].(map[string]any)["deliverables"].([]any)[0].(map[string]any)
	if first["done"] != true || first["done_at"] == nil {
		t.Errorf("deliverable not done: %v", first)
	}
	if resp, _ := do(t, srv, "PUT", "/v1/deals/"+id+"/deliverables/7", tok, map[string]bool{"done": true}); resp.StatusCode != 404 {
		t.Errorf("missing deliverable: %d", resp.StatusCode)
	}

	// A full update keeps the status and the done_at of finished deliverables.
	resp, body = do(t, srv, "PUT", "/v1/deals/"+id, tok, map[string]any{
		"title": "FitBar launch", "status": "lead", "counterparty_name": "FitBar", "amount_minor": 1_800_000, "usage_perpetual": true,
		"usage_rights_days": 30,
		"deliverables": []map[string]any{
			{"title": "Instagram reel 1 of 2", "platform": "instagram", "done": true},
			{"title": "Story", "platform": "instagram"},
		},
	})
	if resp.StatusCode != 200 {
		t.Fatalf("update: %d %v", resp.StatusCode, body)
	}
	u := body["deal"].(map[string]any)
	if u["status"] != "in_progress" || u["amount_minor"] != float64(1_800_000) {
		t.Errorf("update changed status or lost amount: %v", u)
	}
	if u["usage_perpetual"] != true || u["usage_rights_days"] != nil {
		t.Errorf("perpetual usage should clear days: %v", u)
	}
	if got := u["deliverables"].([]any)[0].(map[string]any)["done_at"]; got != first["done_at"] {
		t.Errorf("done_at changed on update: %v -> %v", first["done_at"], got)
	}

	_, body = do(t, srv, "GET", "/v1/deals/"+id+"/events", tok, nil)
	var kinds []string
	for _, e := range body["events"].([]any) {
		kinds = append(kinds, e.(map[string]any)["kind"].(string))
	}
	want := []string{"updated", "deliverable_done", "status_changed", "status_changed", "created"}
	if fmt.Sprint(kinds) != fmt.Sprint(want) {
		t.Errorf("events = %v, want %v", kinds, want)
	}

	if resp, _ := do(t, srv, "DELETE", "/v1/deals/"+id, tok, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	if resp, _ := do(t, srv, "GET", "/v1/deals/"+id, tok, nil); resp.StatusCode != 404 {
		t.Errorf("deleted deal still readable: %d", resp.StatusCode)
	}
}

func TestDealsArePrivateToTheirOwner(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	owner := signup(t, srv, "owner@example.com", "creator")
	other := signup(t, srv, "other@example.com", "creator")
	id := newDeal(t, srv, owner, map[string]any{"title": "Private", "counterparty_name": "Acme"})["id"].(string)

	for _, c := range []struct{ method, path string }{
		{"GET", "/v1/deals/" + id}, {"PUT", "/v1/deals/" + id + "/status"}, {"DELETE", "/v1/deals/" + id},
		{"GET", "/v1/deals/" + id + "/events"}, {"GET", "/v1/deals/not-a-uuid"},
	} {
		if resp, _ := do(t, srv, c.method, c.path, other, map[string]string{"status": "paid"}); resp.StatusCode != 404 {
			t.Errorf("%s %s by another user: %d, want 404", c.method, c.path, resp.StatusCode)
		}
	}
	if _, body := do(t, srv, "GET", "/v1/deals", other, nil); len(body["deals"].([]any)) != 0 {
		t.Error("another user's deal leaked into the list")
	}
}

func TestDealValidation(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "v@example.com", "creator")
	resp, body := do(t, srv, "POST", "/v1/deals", tok, map[string]any{
		"title": "", "status": "won", "counterparty_email": "nope", "amount_minor": -5, "revisions": 99,
		"deliverables": []map[string]any{{"title": "", "platform": "myspace", "due_date": "10/12/2026"}},
	})
	f := fields(body)
	for _, k := range []string{"title", "status", "counterparty_name", "counterparty_email", "amount_minor", "revisions",
		"deliverables[0].title", "deliverables[0].platform", "deliverables[0].due_date"} {
		if f[k] == nil {
			t.Errorf("missing validation error for %s (status %d)", k, resp.StatusCode)
		}
	}
}

func TestDealCounterpartyLinking(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	creator := signup(t, srv, "c@example.com", "creator")
	brand := signup(t, srv, "b@example.com", "brand")
	otherCreator := signup(t, srv, "c2@example.com", "creator")
	do(t, srv, "PUT", "/v1/me/profile", brand, map[string]any{"handle": "fitbar", "industry": "food_beverage"})
	do(t, srv, "PUT", "/v1/me/profile", otherCreator, map[string]any{"handle": "rival"})
	do(t, srv, "PUT", "/v1/me/profile", creator, map[string]any{"handle": "maya"})

	d := newDeal(t, srv, creator, map[string]any{"title": "Linked", "counterparty_handle": "@FitBar"})
	cp := d["counterparty"].(map[string]any)
	if cp["name"] != "Test brand" || cp["user"].(map[string]any)["handle"] != "fitbar" {
		t.Errorf("counterparty not linked: %v", cp)
	}

	for handle, want := range map[string]string{"rival": "counterparty_handle", "maya": "counterparty_handle", "ghost": "counterparty_handle"} {
		resp, body := do(t, srv, "POST", "/v1/deals", creator, map[string]any{"title": "x", "counterparty_handle": handle})
		if resp.StatusCode != http.StatusUnprocessableEntity || fields(body)[want] == nil {
			t.Errorf("handle %q: %d %v", handle, resp.StatusCode, body)
		}
	}
}

func TestDealSummary(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "s@example.com", "creator")
	today := time.Now().In(time.FixedZone("IST", 19800))
	day := func(offset int) string { return today.AddDate(0, 0, offset).Format("2006-01-02") }

	mk := func(title string, amount int, status string, due string) {
		id := newDeal(t, srv, tok, map[string]any{
			"title": title, "counterparty_name": "Brand", "amount_minor": amount,
			"deliverables": []map[string]any{{"title": title + " post", "platform": "instagram", "due_date": due}},
		})["id"].(string)
		do(t, srv, "PUT", "/v1/deals/"+id+"/status", tok, map[string]string{"status": status})
	}
	mk("A", 1_000_000, "confirmed", day(3))
	mk("B", 500_000, "posted", day(-2))
	mk("C", 200_000, "paid", day(-10))
	mk("D", 900_000, "lead", day(30))

	resp, body := do(t, srv, "GET", "/v1/deals/summary", tok, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("summary: %d %v", resp.StatusCode, body)
	}
	var s struct {
		Counts           map[string]int `json:"counts"`
		ActiveCount      int            `json:"active_count"`
		ActiveValueMinor int64          `json:"active_value_minor"`
		AwaitingPayment  int64          `json:"awaiting_payment_minor"`
		PaidThisMonth    int64          `json:"paid_this_month_minor"`
		Overdue          int            `json:"overdue_deliverables"`
		DueSoon          []struct {
			DealTitle string `json:"deal_title"`
			Overdue   bool   `json:"overdue"`
		} `json:"due_soon"`
	}
	raw, _ := json.Marshal(body["summary"])
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if s.ActiveCount != 2 || s.ActiveValueMinor != 1_500_000 || s.AwaitingPayment != 500_000 || s.PaidThisMonth != 200_000 {
		t.Errorf("money/counts wrong: %+v", s)
	}
	if s.Counts["lead"] != 1 || s.Counts["lost"] != 0 {
		t.Errorf("counts: %v", s.Counts)
	}
	if s.Overdue != 1 || len(s.DueSoon) != 2 || s.DueSoon[0].DealTitle != "B" || !s.DueSoon[0].Overdue {
		t.Errorf("due soon: overdue=%d %+v", s.Overdue, s.DueSoon)
	}
}

func fakeAI(t *testing.T, status int, response any) *aiclient.Client {
	t.Helper()
	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/deals/draft" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(ai.Close)
	return aiclient.New(ai.URL, 5*time.Second)
}

func TestDraftDealFromText(t *testing.T) {
	t.Parallel()
	ai := fakeAI(t, 200, map[string]any{
		"draft": map[string]any{
			"counterparty_name": "Priya at FitBar", "counterparty_email": "priya@fitbar.in", "title": "FitBar launch",
			"deliverables": []map[string]any{
				{"platform": "instagram", "format": "reel", "quantity": 2, "due_date": "2026-12-10"},
				{"platform": "tiktok", "format": "video", "quantity": 1, "due_date": "soon"},
			},
			"price_amount": 15000.0, "currency": "INR", "revisions": 1,
			"usage_rights":     map[string]any{"granted": true, "channels": []string{"paid ads"}, "duration_days": 90, "perpetual": false},
			"exclusivity_days": nil, "open_questions": []string{"Who ships the product?"}, "summary": "FitBar proposes 2 reels.",
		},
		"risk_flags": []map[string]any{}, "provider": "fake",
	})
	srv := newServerWith(t, func(d *httpapi.Deps) { d.AI = ai })
	tok := signup(t, srv, "d@example.com", "creator")

	resp, body := do(t, srv, "POST", "/v1/deals/draft", tok, map[string]string{"text": "Hi Maya, ..."})
	if resp.StatusCode != 200 {
		t.Fatalf("draft: %d %v", resp.StatusCode, body)
	}
	s := body["suggestion"].(map[string]any)
	if s["amount_minor"] != float64(1_500_000) || s["status"] != "negotiating" || s["source"] != "ai_draft" || s["usage_rights_days"] != float64(90) {
		t.Errorf("suggestion: %v", s)
	}
	dl := s["deliverables"].([]any)
	if len(dl) != 3 || dl[0].(map[string]any)["title"] != "Instagram reel 1 of 2" || dl[2].(map[string]any)["platform"] != "other" || dl[2].(map[string]any)["due_date"] != "" {
		t.Errorf("deliverables: %v", dl)
	}

	// The suggestion is accepted as-is by the create endpoint.
	s["counterparty_handle"] = ""
	newDeal(t, srv, tok, s)

	if resp, _ := do(t, srv, "POST", "/v1/deals/draft", tok, map[string]string{"text": "  "}); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("empty text: %d", resp.StatusCode)
	}
}

func TestDraftDealWhenAIIsDown(t *testing.T) {
	t.Parallel()
	for name, ai := range map[string]*aiclient.Client{
		"error":    fakeAI(t, 503, map[string]string{"detail": "quota"}),
		"no-ai":    nil,
		"no-route": aiclient.New("http://127.0.0.1:1", time.Second),
	} {
		srv := newServerWith(t, func(d *httpapi.Deps) { d.AI = ai })
		tok := signup(t, srv, name+"@example.com", "creator")
		resp, body := do(t, srv, "POST", "/v1/deals/draft", tok, map[string]string{"text": "hello"})
		if resp.StatusCode != http.StatusServiceUnavailable || body["error"].(map[string]any)["code"] != "ai_unavailable" {
			t.Errorf("%s: %d %v", name, resp.StatusCode, body)
		}
	}
}
