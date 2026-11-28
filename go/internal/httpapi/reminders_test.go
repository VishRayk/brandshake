package httpapi_test

import (
	"context"
	"testing"
	"time"

	"github.com/VishRayk/brandshake/internal/invoice"
	"github.com/VishRayk/brandshake/internal/reminders"
)

func TestOverdueInvoiceReminders(t *testing.T) {
	t.Parallel()
	srv, db := newServerDB(t)
	ctx := context.Background()
	tok := signup(t, srv, "r@example.com", "creator")
	do(t, srv, "PUT", "/v1/me/billing", tok, nonGSTBilling)

	today := time.Now().In(invoice.IST)
	day := func(off int) string { return today.AddDate(0, 0, off).Format("2006-01-02") }
	// Due 3 days ago: first reminder (1 day) is due, second (7 days) is not.
	inv := mustCreateInvoice(t, srv, tok, simpleInvoice(map[string]any{"issue_date": day(-20), "due_date": day(-3)}))
	// Not overdue, no email, or already reported paid: never reminded.
	mustCreateInvoice(t, srv, tok, simpleInvoice(map[string]any{"due_date": day(5)}))
	mustCreateInvoice(t, srv, tok, simpleInvoice(map[string]any{"issue_date": day(-20), "due_date": day(-10), "buyer_email": ""}))
	claimed := mustCreateInvoice(t, srv, tok, simpleInvoice(map[string]any{"issue_date": day(-20), "due_date": day(-10)}))
	do(t, srv, "POST", "/v1/public/invoices/"+claimed["public_url"].(string)[len("https://brandshake.test/i/"):]+"/claim-paid", "", map[string]string{})

	s := &reminders.Scheduler{DB: db, Invoices: invoice.NewRepo(db, "https://brandshake.test"), PublicURL: "https://brandshake.test"}
	count := func() int {
		var n int
		_ = db.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE payload->>'kind' = 'invoice_overdue'`).Scan(&n)
		return n
	}
	for range 3 { // re-running never duplicates
		if err := s.RunOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got := count(); got != 1 {
		t.Fatalf("overdue reminders queued = %d, want 1", got)
	}
	_, body := do(t, srv, "GET", "/v1/invoices/"+inv["id"].(string), tok, nil)
	if body["invoice"].(map[string]any)["reminders_sent"] != 1.0 {
		t.Errorf("reminders_sent = %v", body["invoice"].(map[string]any)["reminders_sent"])
	}

	// Eight days overdue: the 7-day reminder goes out too, once.
	_, _ = db.Exec(ctx, `UPDATE invoices SET due_date = $2 WHERE id::text = $1`, inv["id"], day(-8))
	_ = s.RunOnce(ctx)
	_ = s.RunOnce(ctx)
	if got := count(); got != 2 {
		t.Fatalf("after 8 days: %d reminders, want 2", got)
	}
	// Paid invoices are never reminded again.
	do(t, srv, "POST", "/v1/invoices/"+inv["id"].(string)+"/paid", tok, map[string]string{})
	_, _ = db.Exec(ctx, `UPDATE invoices SET due_date = $2 WHERE id::text = $1`, inv["id"], day(-30))
	_ = s.RunOnce(ctx)
	if got := count(); got != 2 {
		t.Fatalf("paid invoice reminded: %d", got)
	}
}

func TestDeliverableReminders(t *testing.T) {
	t.Parallel()
	srv, db := newServerDB(t)
	ctx := context.Background()
	tok := signup(t, srv, "owner@example.com", "creator")
	today := time.Now().In(invoice.IST)
	day := func(off int) string { return today.AddDate(0, 0, off).Format("2006-01-02") }

	newDeal(t, srv, tok, map[string]any{"title": "FitBar", "counterparty_name": "FitBar", "deliverables": []map[string]any{
		{"title": "Reel", "platform": "instagram", "due_date": day(1)},
		{"title": "Story", "platform": "instagram", "due_date": day(-1)},
		{"title": "Done reel", "platform": "instagram", "due_date": day(1), "done": true},
		{"title": "Later", "platform": "instagram", "due_date": day(5)},
	}})
	lost := newDeal(t, srv, tok, map[string]any{"title": "Gone", "counterparty_name": "X", "deliverables": []map[string]any{
		{"title": "Never", "platform": "x", "due_date": day(1)},
	}})
	do(t, srv, "PUT", "/v1/deals/"+lost["id"].(string)+"/status", tok, map[string]string{"status": "lost"})

	s := &reminders.Scheduler{DB: db, Invoices: invoice.NewRepo(db, "https://brandshake.test"), PublicURL: "https://brandshake.test"}
	_ = s.RunOnce(ctx)
	_ = s.RunOnce(ctx)
	rows, _ := db.Query(ctx, `SELECT payload->>'kind', payload->>'subject' FROM jobs WHERE payload->>'to' = 'owner@example.com' ORDER BY 2`)
	defer rows.Close()
	var got []string
	for rows.Next() {
		var k, subj string
		_ = rows.Scan(&k, &subj)
		got = append(got, k+": "+subj)
	}
	want := []string{"deliverable_due: Due tomorrow: Reel", "deliverable_overdue: Overdue: Story"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("deliverable reminders = %v, want %v", got, want)
	}
}
