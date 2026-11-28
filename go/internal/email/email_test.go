package email_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VishRayk/brandshake/internal/email"
	"github.com/VishRayk/brandshake/internal/jobs"
	"github.com/VishRayk/brandshake/internal/testdb"
)

type failingSender struct{}

func (failingSender) Name() string { return "failing" }
func (failingSender) Send(context.Context, email.Message) (string, error) {
	return "", errors.New("provider down")
}

func TestEmailJobSendsAndRecords(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := context.Background()

	msg := email.Message{To: "brand@example.com", Subject: "Invoice INV/26-27/0001", Text: "Hello", Kind: "invoice_sent"}
	if err := email.Enqueue(ctx, db, msg, "invoice:1:sent"); err != nil {
		t.Fatal(err)
	}
	if err := email.Enqueue(ctx, db, msg, "invoice:1:sent"); err != nil {
		t.Fatal(err) // deduplicated, not an error
	}

	r := jobs.NewRunner(db)
	r.Handle(email.JobKind, email.Handler(db, email.LogSender{}))
	for {
		if worked, err := r.RunOnce(ctx); err != nil {
			t.Fatal(err)
		} else if !worked {
			break
		}
	}
	got, err := email.Recent(ctx, db, "brand@example.com", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Subject != msg.Subject || got[0].Provider != "log" {
		t.Fatalf("recorded emails: %+v", got)
	}
}

func TestEmailFailuresRetry(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := context.Background()
	_ = email.Enqueue(ctx, db, email.Message{To: "a@example.com", Subject: "Hi", Text: "x", Kind: "test"}, "")
	r := jobs.NewRunner(db)
	r.Handle(email.JobKind, email.Handler(db, failingSender{}))
	r.RunOnce(ctx)
	var st string
	_ = db.QueryRow(ctx, `SELECT status FROM jobs`).Scan(&st)
	if st != "pending" {
		t.Fatalf("status = %s, want pending (retry)", st)
	}
	if got, _ := email.Recent(ctx, db, "a@example.com", 10); len(got) != 0 {
		t.Fatal("failed email was recorded as sent")
	}
}

func TestMessageValidation(t *testing.T) {
	bad := []email.Message{
		{To: "not-an-email", Subject: "Hi", Kind: "x"},
		{To: "a@example.com", Subject: "Hi\r\nBcc: evil@example.com", Kind: "x"},
		{To: "a@example.com", Subject: "", Kind: "x"},
		{To: "a@example.com", Subject: "Hi"},
	}
	for _, m := range bad {
		if m.Validate() == nil {
			t.Errorf("accepted %+v", m)
		}
	}
}
