package deal

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VishRayk/brandshake/internal/email"
)

// SendDeliverableReminders emails deal owners about deliverables due
// tomorrow and deliverables that became overdue yesterday. Dedupe keys make
// it safe to run as often as needed.
func SendDeliverableReminders(ctx context.Context, db *pgxpool.Pool, today time.Time, publicURL string) (int, error) {
	rows, err := db.Query(ctx, `
		SELECT d.id::text, d.title, d.counterparty_name, dd.position, dd.title, dd.due_date, u.email
		FROM deal_deliverables dd
		JOIN deals d ON d.id = dd.deal_id
		JOIN users u ON u.id = d.owner_id
		WHERE NOT dd.done AND d.status NOT IN ('paid', 'lost')
		  AND dd.due_date IN ($1::date + 1, $1::date - 1)
		LIMIT 1000`, today)
	if err != nil {
		return 0, err
	}
	type item struct {
		dealID, dealTitle, counterparty, title, to string
		position                                   int16
		due                                        time.Time
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.dealID, &it.dealTitle, &it.counterparty, &it.position, &it.title, &it.due, &it.to); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	queued := 0
	for _, it := range items {
		due := it.due.Format(dateLayout)
		overdue := due < today.Format(dateLayout)
		kind, subject, line := "deliverable_due", fmt.Sprintf("Due tomorrow: %s", it.title), "is due tomorrow"
		if overdue {
			kind, subject, line = "deliverable_overdue", fmt.Sprintf("Overdue: %s", it.title), "was due yesterday"
		}
		msg := email.Message{
			To: it.to, Kind: kind, Subject: subject,
			Text: fmt.Sprintf("%q for %s (%s) %s (%s).\n\nOpen the deal: %s/deals/%s\n\nTick it off when it's done so Brandshake stops reminding you.\n",
				it.title, it.dealTitle, it.counterparty, line, due, publicURL, it.dealID),
		}
		key := fmt.Sprintf("deliverable:%s:%d:%s:%s", it.dealID, it.position, due, kind)
		if err := email.Enqueue(ctx, db, msg, key); err != nil {
			return queued, err
		}
		queued++
	}
	return queued, nil
}
