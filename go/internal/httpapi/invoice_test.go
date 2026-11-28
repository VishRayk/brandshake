package httpapi_test

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VishRayk/brandshake/internal/httpapi"
	"github.com/VishRayk/brandshake/internal/invoice"
)

// validGSTIN appends the correct check character to a 14-character prefix.
func validGSTIN(prefix14 string) string {
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	sum := 0
	for i := range 14 {
		v := strings.IndexByte(chars, prefix14[i])
		f := 1
		if i%2 == 1 {
			f = 2
		}
		p := v * f
		sum += p/36 + p%36
	}
	return prefix14 + string(chars[(36-sum%36)%36])
}

func newServerDB(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	var db *pgxpool.Pool
	srv := newServerWith(t, func(d *httpapi.Deps) { db = d.DB })
	return srv, db
}

var nonGSTBilling = map[string]any{
	"legal_name": "Maya Rao", "address": "12 MG Road, Bengaluru", "state": "Karnataka", "pan": "abcde1234f",
	"gst_registered": false, "upi_id": "maya.rao@okhdfcbank", "invoice_prefix": "inv", "payment_terms_days": 15,
}

func mustCreateInvoice(t *testing.T, srv *httptest.Server, tok string, body map[string]any) map[string]any {
	t.Helper()
	resp, out := do(t, srv, "POST", "/v1/invoices", tok, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create invoice: %d %v", resp.StatusCode, out)
	}
	return out["invoice"].(map[string]any)
}

func simpleInvoice(extra map[string]any) map[string]any {
	b := map[string]any{
		"buyer_name": "FitBar Foods", "buyer_email": "accounts@fitbar.in",
		"items": []map[string]any{{"description": "Instagram Reel, protein launch", "quantity": 2, "unit_price_minor": 750_000}},
	}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func fy() string { return invoice.FinancialYear(time.Now().In(invoice.IST)) }

func TestBillingRules(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	brand := signup(t, srv, "brand@example.com", "brand")
	creator := signup(t, srv, "c@example.com", "creator")

	if resp, _ := do(t, srv, "PUT", "/v1/me/billing", brand, nonGSTBilling); resp.StatusCode != http.StatusForbidden {
		t.Errorf("brand billing: %d, want 403", resp.StatusCode)
	}
	if _, body := do(t, srv, "GET", "/v1/me/billing", creator, nil); body["billing"] != nil {
		t.Errorf("billing should start empty: %v", body)
	}

	resp, body := do(t, srv, "PUT", "/v1/me/billing", creator, map[string]any{
		"legal_name": "", "state": "Atlantis", "pan": "ABC", "gst_registered": true, "gstin": validGSTIN("27ABCDE1234F1Z"),
		"bank_account_number": "12", "bank_ifsc": "BAD", "invoice_prefix": "toolong",
	})
	f := fields(body)
	for _, k := range []string{"legal_name", "state", "pan", "bank_account_number", "bank_ifsc", "bank_account_name", "invoice_prefix"} {
		if f[k] == nil {
			t.Errorf("missing error for %s (%d)", k, resp.StatusCode)
		}
	}

	// GSTIN must match the chosen state, and PAN must match the GSTIN.
	gst := map[string]any{"legal_name": "Maya Rao", "state": "Karnataka", "gst_registered": true, "gst_rate": 18,
		"gstin": validGSTIN("27ABCDE1234F1Z"), "upi_id": "maya@okicici"}
	if _, body := do(t, srv, "PUT", "/v1/me/billing", creator, gst); fields(body)["gstin"] == nil {
		t.Error("GSTIN from another state accepted")
	}
	gst["gstin"], gst["pan"] = validGSTIN("29ABCDE1234F1Z"), "ZZZZZ9999Z"
	if _, body := do(t, srv, "PUT", "/v1/me/billing", creator, gst); fields(body)["pan"] == nil {
		t.Error("PAN that doesn't match the GSTIN accepted")
	}
	gst["pan"] = ""
	resp, body = do(t, srv, "PUT", "/v1/me/billing", creator, gst)
	if resp.StatusCode != 200 || body["billing"].(map[string]any)["pan"] != "ABCDE1234F" {
		t.Fatalf("valid GST billing: %d %v", resp.StatusCode, body)
	}

	if _, body := do(t, srv, "PUT", "/v1/me/billing", creator, map[string]any{"legal_name": "Maya", "state": "Goa"}); fields(body)["upi_id"] == nil {
		t.Error("billing without any payment method accepted")
	}
}

func TestInvoiceLifecycleWithoutGST(t *testing.T) {
	t.Parallel()
	srv, db := newServerDB(t)
	tok := signup(t, srv, "maya@example.com", "creator")

	if resp, body := do(t, srv, "POST", "/v1/invoices", tok, simpleInvoice(nil)); resp.StatusCode != 422 ||
		body["error"].(map[string]any)["code"] != "billing_required" {
		t.Fatalf("invoice without billing: %d %v", resp.StatusCode, body)
	}
	do(t, srv, "PUT", "/v1/me/billing", tok, nonGSTBilling)

	inv := mustCreateInvoice(t, srv, tok, simpleInvoice(nil))
	if inv["number"] != "INV/"+fy()[2:]+"/0001" || inv["status"] != "issued" || inv["gst_mode"] != "none" {
		t.Errorf("invoice: %v", inv)
	}
	if inv["subtotal_minor"] != float64(1_500_000) || inv["total_minor"] != float64(1_500_000) || inv["cgst_minor"] != float64(0) {
		t.Errorf("totals: %v", inv)
	}
	issue, _ := time.Parse("2006-01-02", inv["issue_date"].(string))
	if inv["due_date"] != issue.AddDate(0, 0, 15).Format("2006-01-02") {
		t.Errorf("due date should follow payment terms: %v", inv["due_date"])
	}
	if !strings.HasPrefix(inv["upi_uri"].(string), "upi://pay?am=15000.00&cu=INR&pa=maya.rao%40okhdfcbank") {
		t.Errorf("upi uri: %v", inv["upi_uri"])
	}
	if !strings.HasPrefix(inv["public_url"].(string), "https://brandshake.test/i/") {
		t.Errorf("public url: %v", inv["public_url"])
	}
	if seller := inv["seller"].(map[string]any); seller["legal_name"] != "Maya Rao" || seller["pan"] != "ABCDE1234F" {
		t.Errorf("seller snapshot: %v", seller)
	}
	second := mustCreateInvoice(t, srv, tok, simpleInvoice(map[string]any{"buyer_email": ""}))
	if second["number"] != "INV/"+fy()[2:]+"/0002" {
		t.Errorf("second number: %v", second["number"])
	}

	// Send by email: needs an address; can't resend immediately.
	id := inv["id"].(string)
	if resp, _ := do(t, srv, "POST", "/v1/invoices/"+second["id"].(string)+"/send", tok, nil); resp.StatusCode != 422 {
		t.Errorf("send without email: %d", resp.StatusCode)
	}
	resp, body := do(t, srv, "POST", "/v1/invoices/"+id+"/send", tok, nil)
	if resp.StatusCode != 200 || body["invoice"].(map[string]any)["sent_at"] == nil {
		t.Fatalf("send: %d %v", resp.StatusCode, body)
	}
	if resp, _ := do(t, srv, "POST", "/v1/invoices/"+id+"/send", tok, nil); resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("immediate resend: %d", resp.StatusCode)
	}
	var queued int
	_ = db.QueryRow(context.Background(), `SELECT count(*) FROM jobs WHERE kind = 'email.send' AND payload->>'to' = 'accounts@fitbar.in'`).Scan(&queued)
	if queued != 1 {
		t.Errorf("queued emails to buyer = %d, want 1", queued)
	}

	// Public page: no login, hides creator-only fields, logs one view.
	token := inv["public_url"].(string)[len("https://brandshake.test/i/"):]
	for range 2 {
		resp, body := do(t, srv, "GET", "/v1/public/invoices/"+token, "", nil)
		if resp.StatusCode != 200 || resp.Header.Get("Cache-Control") != "no-store" {
			t.Fatalf("public: %d %v", resp.StatusCode, body)
		}
		if _, leaked := body["invoice"].(map[string]any)["deal_id"]; leaked {
			t.Error("public invoice exposes deal_id")
		}
	}
	if resp, _ := do(t, srv, "GET", "/v1/public/invoices/not-a-real-token-at-all-xyz", "", nil); resp.StatusCode != 404 {
		t.Errorf("bad token: %d", resp.StatusCode)
	}

	// Brand reports payment; creator is notified and confirms.
	resp, body = do(t, srv, "POST", "/v1/public/invoices/"+token+"/claim-paid", "", map[string]string{"reference": "UPI 4321"})
	if resp.StatusCode != 200 || body["invoice"].(map[string]any)["payer_claimed_paid_at"] == nil || body["invoice"].(map[string]any)["status"] != "issued" {
		t.Fatalf("claim paid: %d %v", resp.StatusCode, body)
	}
	if resp, _ := do(t, srv, "POST", "/v1/public/invoices/"+token+"/claim-paid", "", map[string]string{}); resp.StatusCode != 409 {
		t.Errorf("second claim: %d", resp.StatusCode)
	}
	_ = db.QueryRow(context.Background(), `SELECT count(*) FROM jobs WHERE payload->>'to' = 'maya@example.com' AND payload->>'kind' = 'invoice_payer_claimed'`).Scan(&queued)
	if queued != 1 {
		t.Errorf("creator notification queued = %d, want 1", queued)
	}

	if resp, _ := do(t, srv, "POST", "/v1/invoices/"+id+"/paid", tok, map[string]string{"paid_on": "2999-01-01"}); resp.StatusCode != 422 {
		t.Errorf("future paid_on: %d", resp.StatusCode)
	}
	resp, body = do(t, srv, "POST", "/v1/invoices/"+id+"/paid", tok, map[string]string{"reference": "UPI 4321"})
	if resp.StatusCode != 200 || body["invoice"].(map[string]any)["status"] != "paid" || body["invoice"].(map[string]any)["upi_uri"] != nil {
		t.Fatalf("mark paid: %d %v", resp.StatusCode, body)
	}
	if resp, _ := do(t, srv, "POST", "/v1/invoices/"+id+"/void", tok, nil); resp.StatusCode != 409 {
		t.Errorf("void after paid: %d", resp.StatusCode)
	}

	_, body = do(t, srv, "GET", "/v1/invoices/"+id+"/events", tok, nil)
	var kinds []string
	for _, e := range body["events"].([]any) {
		kinds = append(kinds, e.(map[string]any)["kind"].(string))
	}
	if fmt.Sprint(kinds) != "[paid payer_claimed_paid viewed sent issued]" {
		t.Errorf("events = %v", kinds)
	}

	// Void the open one; it can't be paid afterwards.
	sid := second["id"].(string)
	if resp, _ := do(t, srv, "POST", "/v1/invoices/"+sid+"/void", tok, nil); resp.StatusCode != 200 {
		t.Errorf("void: %d", resp.StatusCode)
	}
	if resp, _ := do(t, srv, "POST", "/v1/invoices/"+sid+"/paid", tok, map[string]string{}); resp.StatusCode != 409 {
		t.Errorf("pay after void: %d", resp.StatusCode)
	}
}

func TestInvoiceGST(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "gst@example.com", "creator")
	do(t, srv, "PUT", "/v1/me/billing", tok, map[string]any{
		"legal_name": "Maya Rao Media", "state": "Karnataka", "gst_registered": true, "gst_rate": 18,
		"gstin": validGSTIN("29ABCDE1234F1Z"), "upi_id": "maya@okicici",
	})

	resp, body := do(t, srv, "POST", "/v1/invoices", tok, simpleInvoice(nil))
	if resp.StatusCode != 422 || fields(body)["buyer_state"] == nil {
		t.Fatalf("GST invoice without buyer state: %d %v", resp.StatusCode, body)
	}

	intra := mustCreateInvoice(t, srv, tok, simpleInvoice(map[string]any{"buyer_state": "Karnataka"}))
	if intra["gst_mode"] != "intra" || intra["cgst_minor"] != float64(135_000) || intra["sgst_minor"] != float64(135_000) || intra["total_minor"] != float64(1_770_000) {
		t.Errorf("intra-state: %v", intra)
	}
	// Buyer state taken from their GSTIN.
	inter := mustCreateInvoice(t, srv, tok, simpleInvoice(map[string]any{"buyer_gstin": validGSTIN("27AAPFU0939F1Z")}))
	if inter["gst_mode"] != "inter" || inter["buyer_state"] != "Maharashtra" || inter["igst_minor"] != float64(270_000) {
		t.Errorf("inter-state: %v", inter)
	}
	resp, body = do(t, srv, "POST", "/v1/invoices", tok, simpleInvoice(map[string]any{"buyer_state": "Goa", "buyer_gstin": validGSTIN("27AAPFU0939F1Z")}))
	if fields(body)["buyer_gstin"] == nil {
		t.Errorf("GSTIN/state mismatch accepted: %d", resp.StatusCode)
	}
}

func TestInvoiceValidationAndPrivacy(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "v@example.com", "creator")
	other := signup(t, srv, "o@example.com", "creator")
	do(t, srv, "PUT", "/v1/me/billing", tok, nonGSTBilling)
	do(t, srv, "PUT", "/v1/me/billing", other, nonGSTBilling)

	resp, body := do(t, srv, "POST", "/v1/invoices", tok, map[string]any{
		"buyer_name": "", "buyer_email": "nope", "issue_date": "2999-01-01", "due_date": "2000-01-01",
		"items": []map[string]any{{"description": "", "quantity": 5000, "unit_price_minor": -1, "sac_code": "12"}},
	})
	f := fields(body)
	for _, k := range []string{"buyer_name", "buyer_email", "issue_date", "items[0].description", "items[0].quantity", "items[0].unit_price_minor", "items[0].sac_code"} {
		if f[k] == nil {
			t.Errorf("missing error for %s (%d)", k, resp.StatusCode)
		}
	}
	if _, body := do(t, srv, "POST", "/v1/invoices", tok, map[string]any{"buyer_name": "X", "items": []any{}}); fields(body)["items"] == nil {
		t.Error("invoice without items accepted")
	}

	inv := mustCreateInvoice(t, srv, tok, simpleInvoice(nil))
	for _, path := range []string{"/v1/invoices/" + inv["id"].(string), "/v1/invoices/" + inv["id"].(string) + "/events"} {
		if resp, _ := do(t, srv, "GET", path, other, nil); resp.StatusCode != 404 {
			t.Errorf("%s by another creator: %d", path, resp.StatusCode)
		}
	}
	if resp, _ := do(t, srv, "POST", "/v1/invoices/"+inv["id"].(string)+"/paid", other, map[string]string{}); resp.StatusCode != 404 {
		t.Errorf("mark paid by another creator: %d", resp.StatusCode)
	}
	// A deal owned by someone else can't be linked.
	otherDeal := newDeal(t, srv, other, map[string]any{"title": "Theirs", "counterparty_name": "X"})
	if _, body := do(t, srv, "POST", "/v1/invoices", tok, simpleInvoice(map[string]any{"deal_id": otherDeal["id"]})); fields(body)["deal_id"] == nil {
		t.Error("linked another user's deal")
	}
}

func TestPayingInvoiceClosesItsDeal(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "d@example.com", "creator")
	do(t, srv, "PUT", "/v1/me/billing", tok, nonGSTBilling)
	d := newDeal(t, srv, tok, map[string]any{"title": "FitBar launch", "counterparty_name": "FitBar", "amount_minor": 1_500_000})
	do(t, srv, "PUT", "/v1/deals/"+d["id"].(string)+"/status", tok, map[string]string{"status": "posted"})

	inv := mustCreateInvoice(t, srv, tok, simpleInvoice(map[string]any{"deal_id": d["id"]}))
	if inv["deal_id"] != d["id"] {
		t.Fatalf("deal not linked: %v", inv["deal_id"])
	}
	do(t, srv, "POST", "/v1/invoices/"+inv["id"].(string)+"/paid", tok, map[string]string{})
	_, body := do(t, srv, "GET", "/v1/deals/"+d["id"].(string), tok, nil)
	if body["deal"].(map[string]any)["status"] != "paid" {
		t.Errorf("deal status = %v, want paid", body["deal"].(map[string]any)["status"])
	}
	_, body = do(t, srv, "GET", "/v1/deals/"+d["id"].(string)+"/events", tok, nil)
	if e := body["events"].([]any)[0].(map[string]any); e["kind"] != "status_changed" || e["detail"] != "invoice paid" {
		t.Errorf("latest deal event: %v", e)
	}
}

func TestInvoiceNumbersUnderConcurrency(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "busy@example.com", "creator")
	do(t, srv, "PUT", "/v1/me/billing", tok, nonGSTBilling)

	const n = 10
	numbers := make(chan string, n)
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			resp, body := do(t, srv, "POST", "/v1/invoices", tok, simpleInvoice(nil))
			if resp.StatusCode != http.StatusCreated {
				t.Errorf("create: %d %v", resp.StatusCode, body)
				return
			}
			numbers <- body["invoice"].(map[string]any)["number"].(string)
		})
	}
	wg.Wait()
	close(numbers)
	seen := map[string]bool{}
	for num := range numbers {
		if seen[num] {
			t.Errorf("duplicate invoice number %s", num)
		}
		seen[num] = true
	}
	for i := 1; i <= n; i++ {
		if want := fmt.Sprintf("INV/%s/%04d", fy()[2:], i); !seen[want] {
			t.Errorf("missing %s (numbers must be gapless)", want)
		}
	}
}

func TestInvoiceSummaryAndCSV(t *testing.T) {
	t.Parallel()
	srv := newServer(t)
	tok := signup(t, srv, "csv@example.com", "creator")
	do(t, srv, "PUT", "/v1/me/billing", tok, nonGSTBilling)

	today := time.Now().In(invoice.IST)
	old := today.AddDate(0, 0, -40).Format("2006-01-02")
	if invoice.FinancialYear(today.AddDate(0, 0, -40)) != fy() {
		old = today.Format("2006-01-02") // keep both invoices in this FY near 1 April
	}
	overdue := mustCreateInvoice(t, srv, tok, simpleInvoice(map[string]any{"issue_date": old, "due_date": old, "buyer_name": "=HYPERLINK(\"http://evil\")"}))
	paid := mustCreateInvoice(t, srv, tok, simpleInvoice(nil))
	do(t, srv, "POST", "/v1/invoices/"+paid["id"].(string)+"/paid", tok, map[string]string{"reference": "NEFT 99"})

	_, body := do(t, srv, "GET", "/v1/invoices/summary", tok, nil)
	s := body["summary"].(map[string]any)
	wantOverdue := 1.0
	if old == today.Format("2006-01-02") {
		wantOverdue = 0
	}
	if s["outstanding_count"] != 1.0 || s["outstanding_minor"] != 1_500_000.0 || s["overdue_count"] != wantOverdue ||
		s["paid_this_month_minor"] != 1_500_000.0 || s["paid_this_fy_minor"] != 1_500_000.0 {
		t.Errorf("summary: %v", s)
	}
	if wantOverdue == 1 {
		_, body = do(t, srv, "GET", "/v1/invoices?status=overdue", tok, nil)
		if l := body["invoices"].([]any); len(l) != 1 || l[0].(map[string]any)["id"] != overdue["id"] || l[0].(map[string]any)["overdue"] != true {
			t.Errorf("overdue filter: %v", l)
		}
	}

	req, _ := http.NewRequest("GET", srv.URL+"/v1/invoices/export.csv?fy="+fy(), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	rows, err := csv.NewReader(strings.NewReader(string(raw))).ReadAll()
	if err != nil || len(rows) != 3 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/csv") {
		t.Fatalf("csv: %v rows=%d err=%v\n%s", resp.Header, len(rows), err, raw)
	}
	if rows[1][4] != `'=HYPERLINK("http://evil")` {
		t.Errorf("formula not neutralised: %q", rows[1][4])
	}
	if rows[2][3] != "paid" || rows[2][11] != "15000.00" || rows[2][13] != "NEFT 99" {
		t.Errorf("paid row: %v", rows[2])
	}
}
