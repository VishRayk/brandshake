// Package email sends transactional email through a pluggable provider and
// records every message in the emails table.
package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VishRayk/brandshake/internal/jobs"
)

type Message struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	Kind    string `json:"kind"` // e.g. "invoice_sent", "invoice_overdue"
}

func (m Message) Validate() error {
	a, err := mail.ParseAddress(m.To)
	switch {
	case err != nil || a.Address != m.To:
		return fmt.Errorf("invalid recipient %q", m.To)
	case strings.TrimSpace(m.Subject) == "" || strings.ContainsAny(m.Subject, "\r\n"):
		return fmt.Errorf("invalid subject")
	case m.Kind == "":
		return fmt.Errorf("missing kind")
	}
	return nil
}

// Sender delivers a message and returns the provider's message ID.
type Sender interface {
	Name() string
	Send(ctx context.Context, m Message) (string, error)
}

// LogSender writes emails to the log instead of sending them. Used in
// development and tests; messages are still stored in the emails table.
type LogSender struct{}

func (LogSender) Name() string { return "log" }

func (LogSender) Send(ctx context.Context, m Message) (string, error) {
	slog.InfoContext(ctx, "email (not sent: log provider)", "to", m.To, "subject", m.Subject, "kind", m.Kind)
	return "", nil
}

// JobKind is the background job that sends one email.
const JobKind = "email.send"

// Enqueue schedules an email for sending. Use db = a transaction so the
// email exists only if the surrounding change commits.
func Enqueue(ctx context.Context, db jobs.Execer, m Message, dedupeKey string) error {
	if err := m.Validate(); err != nil {
		return err
	}
	_, err := jobs.Enqueue(ctx, db, JobKind, m, jobs.EnqueueOptions{DedupeKey: dedupeKey, MaxAttempts: 6})
	return err
}

// Handler returns the job handler that sends and records emails.
func Handler(db *pgxpool.Pool, sender Sender) jobs.Handler {
	return func(ctx context.Context, job jobs.Job) error {
		var m Message
		if err := json.Unmarshal(job.Payload, &m); err != nil {
			return jobs.Permanent(fmt.Errorf("decode email: %w", err))
		}
		if err := m.Validate(); err != nil {
			return jobs.Permanent(err)
		}
		id, err := sender.Send(ctx, m)
		if err != nil {
			return err
		}
		_, err = db.Exec(ctx, `
			INSERT INTO emails (to_address, subject, body_text, kind, provider, provider_id, job_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, m.To, m.Subject, m.Text, m.Kind, sender.Name(), id, job.ID)
		return err
	}
}

// Recent returns the latest emails sent to an address (development outbox).
func Recent(ctx context.Context, db *pgxpool.Pool, to string, limit int) ([]Stored, error) {
	rows, err := db.Query(ctx, `
		SELECT id, to_address, subject, body_text, kind, provider, created_at::text
		FROM emails WHERE to_address = $1 ORDER BY id DESC LIMIT $2`, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Stored{}
	for rows.Next() {
		var s Stored
		if err := rows.Scan(&s.ID, &s.To, &s.Subject, &s.Text, &s.Kind, &s.Provider, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type Stored struct {
	ID        int64  `json:"id"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Text      string `json:"text"`
	Kind      string `json:"kind"`
	Provider  string `json:"provider"`
	CreatedAt string `json:"created_at"`
}
