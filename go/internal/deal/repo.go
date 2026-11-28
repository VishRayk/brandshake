package deal

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound             = errors.New("deal not found")
	ErrCounterpartyNotFound = errors.New("no profile with that handle")
	ErrCounterpartySameRole = errors.New("counterparty has the same role")
	ErrCounterpartySelf     = errors.New("counterparty is yourself")
	ErrDeliverableNotFound  = errors.New("deliverable not found")
)

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

const dealSelect = `
	SELECT d.id::text, d.title, d.status, d.counterparty_name, d.counterparty_email,
	       cp.handle, cu.display_name, cu.role,
	       d.amount_minor, d.currency, d.revisions, d.usage_rights_days, d.usage_perpetual,
	       d.exclusivity_days, d.notes, d.source, d.created_at, d.updated_at, d.status_changed_at
	FROM deals d
	LEFT JOIN users cu ON cu.id = d.counterparty_user_id
	LEFT JOIN profiles cp ON cp.user_id = d.counterparty_user_id`

func scanDeal(row pgx.Row) (Deal, error) {
	var (
		d                          Deal
		cpHandle, cpName, cpRole   *string
		revisions, usage, exclusiv *int32
	)
	err := row.Scan(&d.ID, &d.Title, &d.Status, &d.Counterparty.Name, &d.Counterparty.Email,
		&cpHandle, &cpName, &cpRole,
		&d.AmountMinor, &d.Currency, &revisions, &usage, &d.UsagePerpetual,
		&exclusiv, &d.Notes, &d.Source, &d.CreatedAt, &d.UpdatedAt, &d.StatusChangedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Deal{}, ErrNotFound
	}
	if err != nil {
		return Deal{}, err
	}
	if cpHandle != nil {
		d.Counterparty.User = &UserLink{Handle: *cpHandle, DisplayName: deref(cpName), Role: deref(cpRole)}
	}
	d.Revisions, d.UsageRightsDays, d.ExclusivityDays = toInt(revisions), toInt(usage), toInt(exclusiv)
	d.Deliverables = []Deliverable{}
	return d, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toInt(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

// resolveCounterparty links a Brandshake handle to a user of the opposite role.
func resolveCounterparty(ctx context.Context, q pgx.Tx, ownerID, ownerRole, handle string) (userID *string, name string, err error) {
	if handle == "" {
		return nil, "", nil
	}
	var id, role, display string
	err = q.QueryRow(ctx, `
		SELECT u.id::text, u.role, u.display_name
		FROM profiles p JOIN users u ON u.id = p.user_id
		WHERE p.handle = $1`, handle).Scan(&id, &role, &display)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, "", ErrCounterpartyNotFound
	case err != nil:
		return nil, "", err
	case id == ownerID:
		return nil, "", ErrCounterpartySelf
	case role == ownerRole:
		return nil, "", ErrCounterpartySameRole
	}
	return &id, display, nil
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return nil
	}
	return &t
}

func addEvent(ctx context.Context, tx pgx.Tx, dealID, actorID, kind string, from, to *Status, detail string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO deal_events (deal_id, actor_id, kind, from_status, to_status, detail)
		VALUES ($1, $2, $3, $4, $5, $6)`, dealID, actorID, kind, from, to, detail)
	return err
}

// writeDeliverables replaces a deal's deliverables, keeping done_at for
// items that were already done.
func writeDeliverables(ctx context.Context, tx pgx.Tx, dealID string, items []DeliverableInput) error {
	prevDone := map[string]time.Time{}
	rows, err := tx.Query(ctx, `SELECT title, done_at FROM deal_deliverables WHERE deal_id = $1 AND done`, dealID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var title string
		var at *time.Time
		if err := rows.Scan(&title, &at); err != nil {
			rows.Close()
			return err
		}
		if at != nil {
			prevDone[title] = *at
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM deal_deliverables WHERE deal_id = $1`, dealID); err != nil {
		return err
	}
	for i, d := range items {
		var doneAt *time.Time
		if d.Done {
			t, ok := prevDone[d.Title]
			if !ok {
				t = time.Now()
			}
			doneAt = &t
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO deal_deliverables (deal_id, position, title, platform, due_date, done, done_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			dealID, i, d.Title, d.Platform, parseDate(d.DueDate), d.Done, doneAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) Create(ctx context.Context, ownerID, ownerRole string, in Input) (Deal, error) {
	var id string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		cpID, cpName, err := resolveCounterparty(ctx, tx, ownerID, ownerRole, in.CounterpartyHandle)
		if err != nil {
			return err
		}
		name := in.CounterpartyName
		if name == "" {
			name = cpName
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO deals (owner_id, counterparty_user_id, counterparty_name, counterparty_email, title, status,
				amount_minor, revisions, usage_rights_days, usage_perpetual, exclusivity_days, notes, source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING id::text`,
			ownerID, cpID, name, in.CounterpartyEmail, in.Title, in.Status,
			in.AmountMinor, in.Revisions, in.UsageRightsDays, in.UsagePerpetual, in.ExclusivityDays, in.Notes, in.Source,
		).Scan(&id); err != nil {
			return err
		}
		if err := writeDeliverables(ctx, tx, id, in.Deliverables); err != nil {
			return err
		}
		return addEvent(ctx, tx, id, ownerID, "created", nil, &in.Status, in.Source)
	})
	if err != nil {
		return Deal{}, err
	}
	return r.Get(ctx, ownerID, id)
}

// Update replaces a deal's details and deliverables. Status is changed
// only through SetStatus, so every transition is recorded.
func (r *Repo) Update(ctx context.Context, ownerID, ownerRole, id string, in Input) (Deal, error) {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var dealID string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM deals WHERE id::text = $1 AND owner_id = $2 FOR UPDATE`,
			id, ownerID).Scan(&dealID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		cpID, cpName, err := resolveCounterparty(ctx, tx, ownerID, ownerRole, in.CounterpartyHandle)
		if err != nil {
			return err
		}
		name := in.CounterpartyName
		if name == "" {
			name = cpName
		}
		if _, err := tx.Exec(ctx, `
			UPDATE deals SET counterparty_user_id = $3, counterparty_name = $4, counterparty_email = $5, title = $6,
				amount_minor = $7, revisions = $8, usage_rights_days = $9, usage_perpetual = $10,
				exclusivity_days = $11, notes = $12, updated_at = now()
			WHERE id = $1 AND owner_id = $2`,
			dealID, ownerID, cpID, name, in.CounterpartyEmail, in.Title,
			in.AmountMinor, in.Revisions, in.UsageRightsDays, in.UsagePerpetual, in.ExclusivityDays, in.Notes); err != nil {
			return err
		}
		if err := writeDeliverables(ctx, tx, dealID, in.Deliverables); err != nil {
			return err
		}
		return addEvent(ctx, tx, dealID, ownerID, "updated", nil, nil, "")
	})
	if err != nil {
		return Deal{}, err
	}
	return r.Get(ctx, ownerID, id)
}

func (r *Repo) SetStatus(ctx context.Context, ownerID, id string, to Status) (Deal, error) {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var dealID string
		var from Status
		if err := tx.QueryRow(ctx, `SELECT id::text, status FROM deals WHERE id::text = $1 AND owner_id = $2 FOR UPDATE`,
			id, ownerID).Scan(&dealID, &from); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if from == to {
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE deals SET status = $2, status_changed_at = now(), updated_at = now() WHERE id = $1`,
			dealID, to); err != nil {
			return err
		}
		return addEvent(ctx, tx, dealID, ownerID, "status_changed", &from, &to, "")
	})
	if err != nil {
		return Deal{}, err
	}
	return r.Get(ctx, ownerID, id)
}

func (r *Repo) SetDeliverableDone(ctx context.Context, ownerID, id string, position int, done bool) (Deal, error) {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var dealID, title string
		var was bool
		err := tx.QueryRow(ctx, `
			SELECT d.id::text, dd.title, dd.done FROM deals d
			JOIN deal_deliverables dd ON dd.deal_id = d.id AND dd.position = $3
			WHERE d.id::text = $1 AND d.owner_id = $2
			FOR UPDATE OF dd`, id, ownerID, position).Scan(&dealID, &title, &was)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDeliverableNotFound
		}
		if err != nil || was == done {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE deal_deliverables SET done = $3, done_at = CASE WHEN $3 THEN now() END
			WHERE deal_id = $1 AND position = $2`, dealID, position, done); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE deals SET updated_at = now() WHERE id = $1`, dealID); err != nil {
			return err
		}
		kind := "deliverable_done"
		if !done {
			kind = "deliverable_reopened"
		}
		return addEvent(ctx, tx, dealID, ownerID, kind, nil, nil, title)
	})
	if err != nil {
		return Deal{}, err
	}
	return r.Get(ctx, ownerID, id)
}

func (r *Repo) Get(ctx context.Context, ownerID, id string) (Deal, error) {
	d, err := scanDeal(r.db.QueryRow(ctx, dealSelect+` WHERE d.id::text = $1 AND d.owner_id = $2`, id, ownerID))
	if err != nil {
		return Deal{}, err
	}
	deals := []Deal{d}
	if err := r.loadDeliverables(ctx, deals); err != nil {
		return Deal{}, err
	}
	return deals[0], nil
}

type ListFilter struct {
	Status Status
	Query  string
}

func (r *Repo) List(ctx context.Context, ownerID string, f ListFilter) ([]Deal, error) {
	sql := dealSelect + ` WHERE d.owner_id = $1`
	args := []any{ownerID}
	if f.Status != "" {
		args = append(args, f.Status)
		sql += ` AND d.status = $2`
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		args = append(args, "%"+escapeLike(q)+"%")
		n := len(args)
		sql += ` AND (d.title ILIKE $` + strconv.Itoa(n) + ` OR d.counterparty_name ILIKE $` + strconv.Itoa(n) + `)`
	}
	sql += ` ORDER BY d.updated_at DESC LIMIT 500`

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deals := []Deal{}
	for rows.Next() {
		d, err := scanDeal(rows)
		if err != nil {
			return nil, err
		}
		deals = append(deals, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return deals, r.loadDeliverables(ctx, deals)
}

func (r *Repo) loadDeliverables(ctx context.Context, deals []Deal) error {
	if len(deals) == 0 {
		return nil
	}
	ids := make([]string, len(deals))
	index := map[string]int{}
	for i, d := range deals {
		ids[i] = d.ID
		index[d.ID] = i
	}
	rows, err := r.db.Query(ctx, `
		SELECT deal_id::text, title, platform, due_date, done, done_at
		FROM deal_deliverables WHERE deal_id::text = ANY($1) ORDER BY deal_id, position`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			dealID string
			d      Deliverable
			due    *time.Time
		)
		if err := rows.Scan(&dealID, &d.Title, &d.Platform, &due, &d.Done, &d.DoneAt); err != nil {
			return err
		}
		if due != nil {
			s := due.Format(dateLayout)
			d.DueDate = &s
		}
		i := index[dealID]
		deals[i].Deliverables = append(deals[i].Deliverables, d)
	}
	return rows.Err()
}

func (r *Repo) Delete(ctx context.Context, ownerID, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM deals WHERE id::text = $1 AND owner_id = $2`, id, ownerID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (r *Repo) Events(ctx context.Context, ownerID, id string) ([]Event, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM deals WHERE id::text = $1 AND owner_id = $2)`,
		id, ownerID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	rows, _ := r.db.Query(ctx, `
		SELECT e.id, e.kind, e.from_status, e.to_status, e.detail, e.created_at
		FROM deal_events e WHERE e.deal_id::text = $1 ORDER BY e.id DESC LIMIT 200`, id)
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Event])
}

// Summary aggregates a user's deals for the dashboard. today is in the
// user's timezone (IST for now).
func (r *Repo) Summary(ctx context.Context, ownerID string, today time.Time) (Summary, error) {
	s := Summary{Counts: map[Status]int{}, DueSoon: []DueItem{}}
	for _, st := range Statuses {
		s.Counts[st] = 0
	}
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())

	rows, err := r.db.Query(ctx, `
		SELECT status, count(*), coalesce(sum(amount_minor), 0),
		       coalesce(sum(amount_minor) FILTER (WHERE status = 'paid' AND status_changed_at >= $2), 0)
		FROM deals WHERE owner_id = $1 GROUP BY status`, ownerID, monthStart)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var (
			st             Status
			n              int
			sum, paidMonth int64
		)
		if err := rows.Scan(&st, &n, &sum, &paidMonth); err != nil {
			rows.Close()
			return s, err
		}
		s.Counts[st] = n
		if st.Active() {
			s.ActiveCount += n
			s.ActiveValueMinor += sum
		}
		if st == StatusPosted {
			s.AwaitingPayment = sum
		}
		s.PaidThisMonth += paidMonth
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return s, err
	}

	day := today.Format(dateLayout)
	if err := r.db.QueryRow(ctx, `
		SELECT count(*) FROM deal_deliverables dd JOIN deals d ON d.id = dd.deal_id
		WHERE d.owner_id = $1 AND NOT dd.done AND dd.due_date < $2::date
		  AND d.status NOT IN ('paid', 'lost')`, ownerID, day).Scan(&s.Overdue); err != nil {
		return s, err
	}

	rows, err = r.db.Query(ctx, `
		SELECT d.id::text, d.title, d.counterparty_name, dd.title, dd.platform, dd.due_date
		FROM deal_deliverables dd JOIN deals d ON d.id = dd.deal_id
		WHERE d.owner_id = $1 AND NOT dd.done AND dd.due_date IS NOT NULL
		  AND dd.due_date <= $2::date + 14 AND d.status NOT IN ('paid', 'lost')
		ORDER BY dd.due_date, d.title LIMIT 10`, ownerID, day)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var it DueItem
		var due time.Time
		if err := rows.Scan(&it.DealID, &it.DealTitle, &it.Counterpart, &it.Title, &it.Platform, &due); err != nil {
			return s, err
		}
		it.DueDate = due.Format(dateLayout)
		it.Overdue = it.DueDate < day
		s.DueSoon = append(s.DueSoon, it)
	}
	return s, rows.Err()
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
