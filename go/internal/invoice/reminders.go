package invoice

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/VishRayk/brandshake/internal/email"
)

// ReminderDays are how many days past due each reminder goes out (max 3).
var ReminderDays = []int{1, 7, 14}

// SendOverdueReminders queues reminder emails to brands for overdue
// invoices. Safe to run repeatedly and from several instances: each
// reminder is claimed with a compare-and-set on reminders_sent and has a
// dedupe key.
func (r *Repo) SendOverdueReminders(ctx context.Context) (int, error) {
	today := r.Today()
	rows, err := r.db.Query(ctx, `
		SELECT id::text, owner_id::text, reminders_sent, due_date
		FROM invoices
		WHERE status = 'issued' AND buyer_email <> '' AND payer_claimed_paid_at IS NULL
		  AND due_date < $1 AND reminders_sent < $2
		ORDER BY due_date LIMIT 500`, today, len(ReminderDays))
	if err != nil {
		return 0, err
	}
	type cand struct {
		id, owner string
		sent      int16
		due       time.Time
	}
	var cands []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.owner, &c.sent, &c.due); err != nil {
			rows.Close()
			return 0, err
		}
		cands = append(cands, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	sent := 0
	for _, c := range cands {
		daysOver := int(today.Sub(time.Date(c.due.Year(), c.due.Month(), c.due.Day(), 0, 0, 0, 0, IST)).Hours() / 24)
		n := int(c.sent)
		if daysOver < ReminderDays[n] {
			continue
		}
		inv, err := r.Get(ctx, c.owner, c.id)
		if err != nil {
			return sent, err
		}
		err = pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `
				UPDATE invoices SET reminders_sent = reminders_sent + 1, updated_at = now()
				WHERE id = $1 AND reminders_sent = $2 AND status = 'issued'`, inv.ID, n)
			if err != nil || tag.RowsAffected() == 0 {
				return err // another instance got it first
			}
			if err := email.Enqueue(ctx, tx, invoiceReminderEmail(inv, daysOver),
				fmt.Sprintf("invoice:%s:overdue:%d", inv.ID, n+1)); err != nil {
				return err
			}
			sent++
			return addEvent(ctx, tx, inv.ID, "reminder_sent", fmt.Sprintf("%d days overdue", daysOver))
		})
		if err != nil {
			return sent, err
		}
	}
	return sent, nil
}
