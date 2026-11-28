// Package reminders periodically queues reminder emails: overdue invoices
// to brands, and due or overdue deliverables to deal owners.
package reminders

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VishRayk/brandshake/internal/deal"
	"github.com/VishRayk/brandshake/internal/invoice"
)

// lockID keeps one scan running at a time across API instances.
const lockID int64 = 7_270_113_002

type Scheduler struct {
	DB        *pgxpool.Pool
	Invoices  *invoice.Repo
	PublicURL string
	Every     time.Duration
}

// Run scans immediately and then every s.Every until ctx is done.
func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(s.Every)
	defer t.Stop()
	for {
		if err := s.RunOnce(ctx); err != nil && ctx.Err() == nil {
			slog.Error("reminders: scan failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// RunOnce performs one scan if no other instance is scanning.
func (s *Scheduler) RunOnce(ctx context.Context) error {
	conn, err := s.DB.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	var got bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockID).Scan(&got); err != nil || !got {
		return err
	}
	defer conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, lockID) //nolint:errcheck

	inv, err := s.Invoices.SendOverdueReminders(ctx)
	if err != nil {
		return err
	}
	del, err := deal.SendDeliverableReminders(ctx, s.DB, s.Invoices.Today(), s.PublicURL)
	if err != nil {
		return err
	}
	if inv+del > 0 {
		slog.Info("reminders queued", "invoices", inv, "deliverables", del)
	}
	return nil
}
