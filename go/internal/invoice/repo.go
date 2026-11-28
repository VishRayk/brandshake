package invoice

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VishRayk/brandshake/internal/deal"
	"github.com/VishRayk/brandshake/internal/email"
)

var (
	ErrNotFound      = errors.New("invoice not found")
	ErrNoBilling     = errors.New("billing details missing")
	ErrDealNotFound  = errors.New("deal not found")
	ErrNotIssued     = errors.New("invoice is not open")
	ErrNoBuyerEmail  = errors.New("invoice has no buyer email")
	ErrSentRecently  = errors.New("invoice was sent moments ago")
	ErrAlreadyClaims = errors.New("payment already reported")
)

// IST is the business timezone for dates (invoice dates, overdue checks).
var IST = time.FixedZone("IST", 5*60*60+30*60)

type Repo struct {
	db        *pgxpool.Pool
	publicURL string
	now       func() time.Time
}

func NewRepo(db *pgxpool.Pool, publicURL string) *Repo {
	return &Repo{db: db, publicURL: strings.TrimRight(publicURL, "/"), now: time.Now}
}

// Today is the current calendar date in IST.
func (r *Repo) Today() time.Time {
	n := r.now().In(IST)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, IST)
}

/* ---------- billing ---------- */

const billingCols = `legal_name, address, state, pan, gst_registered, gstin, gst_rate, upi_id,
	bank_account_name, bank_account_number, bank_ifsc, bank_name, invoice_prefix, payment_terms_days, updated_at`

func scanBilling(row pgx.Row) (Billing, error) {
	var b Billing
	var rate, terms int16
	err := row.Scan(&b.LegalName, &b.Address, &b.State, &b.PAN, &b.GSTRegistered, &b.GSTIN, &rate, &b.UPIID,
		&b.BankAccountName, &b.BankAccountNumber, &b.BankIFSC, &b.BankName, &b.InvoicePrefix, &terms, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Billing{}, ErrNoBilling
	}
	b.GSTRate, b.PaymentTermsDays = int(rate), int(terms)
	return b, err
}

func (r *Repo) Billing(ctx context.Context, userID string) (Billing, error) {
	return scanBilling(r.db.QueryRow(ctx, `SELECT `+billingCols+` FROM billing_profiles WHERE user_id = $1`, userID))
}

func (r *Repo) SaveBilling(ctx context.Context, userID string, b Billing) (Billing, error) {
	return scanBilling(r.db.QueryRow(ctx, `
		INSERT INTO billing_profiles (user_id, legal_name, address, state, pan, gst_registered, gstin, gst_rate, upi_id,
			bank_account_name, bank_account_number, bank_ifsc, bank_name, invoice_prefix, payment_terms_days)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (user_id) DO UPDATE SET
			legal_name = EXCLUDED.legal_name, address = EXCLUDED.address, state = EXCLUDED.state, pan = EXCLUDED.pan,
			gst_registered = EXCLUDED.gst_registered, gstin = EXCLUDED.gstin, gst_rate = EXCLUDED.gst_rate,
			upi_id = EXCLUDED.upi_id, bank_account_name = EXCLUDED.bank_account_name,
			bank_account_number = EXCLUDED.bank_account_number, bank_ifsc = EXCLUDED.bank_ifsc,
			bank_name = EXCLUDED.bank_name, invoice_prefix = EXCLUDED.invoice_prefix,
			payment_terms_days = EXCLUDED.payment_terms_days, updated_at = now()
		RETURNING `+billingCols,
		userID, b.LegalName, b.Address, b.State, b.PAN, b.GSTRegistered, b.GSTIN, b.GSTRate, b.UPIID,
		b.BankAccountName, b.BankAccountNumber, b.BankIFSC, b.BankName, b.InvoicePrefix, b.PaymentTermsDays))
}

/* ---------- invoices ---------- */

const invoiceCols = `i.id::text, i.number, i.fy, i.status, i.issue_date, i.due_date, i.seller,
	i.buyer_name, i.buyer_email, i.buyer_address, i.buyer_state, i.buyer_gstin, i.gst_mode, i.gst_rate,
	i.subtotal_minor, i.cgst_minor, i.sgst_minor, i.igst_minor, i.total_minor, i.notes, i.deal_id::text,
	i.public_token, i.sent_at, i.payer_claimed_paid_at, i.payer_reference, i.paid_at, i.paid_reference,
	i.voided_at, i.reminders_sent, i.created_at`

func (r *Repo) scanInvoice(row pgx.Row) (Invoice, error) {
	var (
		inv          Invoice
		issue, due   time.Time
		seller       []byte
		rate, remind int16
	)
	err := row.Scan(&inv.ID, &inv.Number, &inv.FY, &inv.Status, &issue, &due, &seller,
		&inv.BuyerName, &inv.BuyerEmail, &inv.BuyerAddress, &inv.BuyerState, &inv.BuyerGSTIN, &inv.GSTMode, &rate,
		&inv.Subtotal, &inv.CGST, &inv.SGST, &inv.IGST, &inv.Total, &inv.Notes, &inv.DealID,
		&inv.PublicToken, &inv.SentAt, &inv.PayerClaimedPaidAt, &inv.PayerReference, &inv.PaidAt, &inv.PaidReference,
		&inv.VoidedAt, &remind, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, err
	}
	if err := json.Unmarshal(seller, &inv.Seller); err != nil {
		return Invoice{}, err
	}
	inv.GSTRate, inv.RemindersSent = int(rate), int(remind)
	inv.IssueDate, inv.DueDate = issue.Format(dateLayout), due.Format(dateLayout)
	inv.Overdue = inv.Status == StatusIssued && inv.DueDate < r.Today().Format(dateLayout)
	inv.PublicURL = r.publicURL + "/i/" + inv.PublicToken
	if inv.Status == StatusIssued && inv.Seller.UPIID != "" && inv.Total > 0 {
		inv.UPIURI = UPIPayURI(inv.Seller.UPIID, inv.Seller.LegalName, inv.Total, inv.Number)
	}
	inv.Items = []Item{}
	return inv, nil
}

func (r *Repo) loadItems(ctx context.Context, inv *Invoice) error {
	rows, _ := r.db.Query(ctx, `
		SELECT description, sac_code, quantity, unit_price_minor, amount_minor
		FROM invoice_items WHERE invoice_id = $1 ORDER BY position`, inv.ID)
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Item])
	if err == nil {
		inv.Items = items
	}
	return err
}

func newToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func addEvent(ctx context.Context, tx pgx.Tx, id, kind, detail string) error {
	_, err := tx.Exec(ctx, `INSERT INTO invoice_events (invoice_id, kind, detail) VALUES ($1, $2, $3)`, id, kind, detail)
	return err
}

// Create issues an invoice: validates against the seller's billing details,
// takes the next number for the financial year and stores a snapshot of the
// seller, all in one transaction.
func (r *Repo) Create(ctx context.Context, ownerID string, in Input) (Invoice, Errors, error) {
	var id string
	var verrs Errors
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		b, err := scanBilling(tx.QueryRow(ctx, `SELECT `+billingCols+` FROM billing_profiles WHERE user_id = $1 FOR SHARE`, ownerID))
		if err != nil {
			return err
		}
		issue, due, errs := in.Validate(b, r.Today())
		if len(errs) > 0 {
			verrs = errs
			return nil
		}
		var dealID *string
		if in.DealID != "" {
			var d string
			if err := tx.QueryRow(ctx, `SELECT id::text FROM deals WHERE id::text = $1 AND owner_id = $2`, in.DealID, ownerID).Scan(&d); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrDealNotFound
				}
				return err
			}
			dealID = &d
		}

		fy := FinancialYear(issue)
		var seq int
		if err := tx.QueryRow(ctx, `
			INSERT INTO invoice_counters (user_id, fy, last_number) VALUES ($1, $2, 1)
			ON CONFLICT (user_id, fy) DO UPDATE SET last_number = invoice_counters.last_number + 1
			RETURNING last_number`, ownerID, fy).Scan(&seq); err != nil {
			return err
		}
		number := InvoiceNumber(b.InvoicePrefix, fy, seq)

		mode := ModeFor(b, in.BuyerState)
		rate := 0
		if mode != GSTNone {
			rate = b.GSTRate
		}
		var subtotal int64
		for _, it := range in.Items {
			subtotal += int64(it.Quantity) * it.UnitPriceMinor
		}
		t := Compute(subtotal, mode, rate)

		seller, _ := json.Marshal(Seller{
			LegalName: b.LegalName, Address: b.Address, State: b.State, PAN: b.PAN, GSTIN: b.GSTIN, UPIID: b.UPIID,
			BankAccountName: b.BankAccountName, BankAccountNumber: b.BankAccountNumber, BankIFSC: b.BankIFSC, BankName: b.BankName,
		})
		token, err := newToken()
		if err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO invoices (owner_id, deal_id, number, fy, issue_date, due_date, seller,
				buyer_name, buyer_email, buyer_address, buyer_state, buyer_gstin, gst_mode, gst_rate,
				subtotal_minor, cgst_minor, sgst_minor, igst_minor, total_minor, notes, public_token)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
			RETURNING id::text`,
			ownerID, dealID, number, fy, issue, due, seller,
			in.BuyerName, in.BuyerEmail, in.BuyerAddress, in.BuyerState, in.BuyerGSTIN, mode, rate,
			t.Subtotal, t.CGST, t.SGST, t.IGST, t.Total, in.Notes, token).Scan(&id); err != nil {
			return err
		}
		for i, it := range in.Items {
			if _, err := tx.Exec(ctx, `
				INSERT INTO invoice_items (invoice_id, position, description, sac_code, quantity, unit_price_minor, amount_minor)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				id, i, it.Description, it.SACCode, it.Quantity, it.UnitPriceMinor, int64(it.Quantity)*it.UnitPriceMinor); err != nil {
				return err
			}
		}
		return addEvent(ctx, tx, id, "issued", number)
	})
	if err != nil || verrs != nil {
		return Invoice{}, verrs, err
	}
	inv, err := r.Get(ctx, ownerID, id)
	return inv, nil, err
}

func (r *Repo) Get(ctx context.Context, ownerID, id string) (Invoice, error) {
	inv, err := r.scanInvoice(r.db.QueryRow(ctx, `SELECT `+invoiceCols+` FROM invoices i WHERE i.id::text = $1 AND i.owner_id = $2`, id, ownerID))
	if err != nil {
		return Invoice{}, err
	}
	return inv, r.loadItems(ctx, &inv)
}

// ByToken loads an invoice for its public, no-login page.
func (r *Repo) ByToken(ctx context.Context, token string) (Invoice, error) {
	if len(token) < 20 || len(token) > 64 {
		return Invoice{}, ErrNotFound
	}
	inv, err := r.scanInvoice(r.db.QueryRow(ctx, `SELECT `+invoiceCols+` FROM invoices i WHERE i.public_token = $1`, token))
	if err != nil {
		return Invoice{}, err
	}
	return inv, r.loadItems(ctx, &inv)
}

// RecordView logs that the public link was opened (at most once an hour).
func (r *Repo) RecordView(ctx context.Context, invoiceID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO invoice_events (invoice_id, kind)
		SELECT $1, 'viewed'
		WHERE NOT EXISTS (
			SELECT 1 FROM invoice_events WHERE invoice_id = $1 AND kind = 'viewed' AND created_at > now() - interval '1 hour'
		)`, invoiceID)
	return err
}

type ListFilter struct {
	Status string // issued | paid | void | overdue | ""
	FY     string
}

func (r *Repo) List(ctx context.Context, ownerID string, f ListFilter) ([]Invoice, error) {
	sql := `SELECT ` + invoiceCols + ` FROM invoices i WHERE i.owner_id = $1`
	args := []any{ownerID}
	switch f.Status {
	case "overdue":
		args = append(args, r.Today())
		sql += fmt.Sprintf(` AND i.status = 'issued' AND i.due_date < $%d`, len(args))
	case "issued", "paid", "void":
		args = append(args, f.Status)
		sql += fmt.Sprintf(` AND i.status = $%d`, len(args))
	}
	if f.FY != "" {
		args = append(args, f.FY)
		sql += fmt.Sprintf(` AND i.fy = $%d`, len(args))
	}
	sql += ` ORDER BY i.issue_date DESC, i.number DESC LIMIT 500`
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Invoice{}
	for rows.Next() {
		inv, err := r.scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Items for all listed invoices in one query.
	if len(out) > 0 {
		idx := map[string]int{}
		ids := make([]string, len(out))
		for i, inv := range out {
			idx[inv.ID], ids[i] = i, inv.ID
		}
		rows, err := r.db.Query(ctx, `
			SELECT invoice_id::text, description, sac_code, quantity, unit_price_minor, amount_minor
			FROM invoice_items WHERE invoice_id::text = ANY($1) ORDER BY invoice_id, position`, ids)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			var it Item
			if err := rows.Scan(&id, &it.Description, &it.SACCode, &it.Quantity, &it.UnitPriceMinor, &it.AmountMinor); err != nil {
				return nil, err
			}
			out[idx[id]].Items = append(out[idx[id]].Items, it)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// lockOwned locks an invoice row the owner can change.
func lockOwned(ctx context.Context, tx pgx.Tx, ownerID, id string) (string, Status, error) {
	var invID string
	var st Status
	err := tx.QueryRow(ctx, `SELECT id::text, status FROM invoices WHERE id::text = $1 AND owner_id = $2 FOR UPDATE`, id, ownerID).Scan(&invID, &st)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return invID, st, err
}

// Send emails the invoice link to the buyer (via the job queue).
func (r *Repo) Send(ctx context.Context, ownerID, id string) (Invoice, error) {
	inv, err := r.Get(ctx, ownerID, id)
	if err != nil {
		return Invoice{}, err
	}
	switch {
	case inv.Status != StatusIssued:
		return Invoice{}, ErrNotIssued
	case inv.BuyerEmail == "":
		return Invoice{}, ErrNoBuyerEmail
	case inv.SentAt != nil && r.now().Sub(*inv.SentAt) < 10*time.Minute:
		return Invoice{}, ErrSentRecently
	}
	err = pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, _, err := lockOwned(ctx, tx, ownerID, id); err != nil {
			return err
		}
		if err := email.Enqueue(ctx, tx, invoiceSentEmail(inv), ""); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE invoices SET sent_at = now(), updated_at = now() WHERE id = $1`, inv.ID); err != nil {
			return err
		}
		return addEvent(ctx, tx, inv.ID, "sent", inv.BuyerEmail)
	})
	if err != nil {
		return Invoice{}, err
	}
	return r.Get(ctx, ownerID, id)
}

// MarkPaid records that the creator received the money. A linked deal
// moves to "paid" in the same transaction.
func (r *Repo) MarkPaid(ctx context.Context, ownerID, id string, paidOn time.Time, reference string) (Invoice, error) {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		invID, st, err := lockOwned(ctx, tx, ownerID, id)
		if err != nil {
			return err
		}
		if st != StatusIssued {
			return ErrNotIssued
		}
		var dealID *string
		if err := tx.QueryRow(ctx, `
			UPDATE invoices SET status = 'paid', paid_at = $2, paid_reference = $3, updated_at = now()
			WHERE id = $1 RETURNING deal_id::text`, invID, paidOn, reference).Scan(&dealID); err != nil {
			return err
		}
		if err := addEvent(ctx, tx, invID, "paid", reference); err != nil {
			return err
		}
		if dealID != nil {
			return deal.MarkPaidTx(ctx, tx, ownerID, *dealID)
		}
		return nil
	})
	if err != nil {
		return Invoice{}, err
	}
	return r.Get(ctx, ownerID, id)
}

// ClaimPaid is the brand saying "I've paid" on the public page. The creator
// is emailed and confirms with MarkPaid; the invoice stays open until then.
func (r *Repo) ClaimPaid(ctx context.Context, token, reference string) (Invoice, error) {
	inv, err := r.ByToken(ctx, token)
	if err != nil {
		return Invoice{}, err
	}
	if inv.Status != StatusIssued {
		return Invoice{}, ErrNotIssued
	}
	if inv.PayerClaimedPaidAt != nil {
		return Invoice{}, ErrAlreadyClaims
	}
	err = pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var ownerEmail string
		tag, err := tx.Exec(ctx, `
			UPDATE invoices SET payer_claimed_paid_at = now(), payer_reference = $2, updated_at = now()
			WHERE id = $1 AND status = 'issued' AND payer_claimed_paid_at IS NULL`, inv.ID, reference)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrAlreadyClaims
		}
		if err := tx.QueryRow(ctx, `SELECT u.email FROM users u JOIN invoices i ON i.owner_id = u.id WHERE i.id = $1`, inv.ID).Scan(&ownerEmail); err != nil {
			return err
		}
		if err := addEvent(ctx, tx, inv.ID, "payer_claimed_paid", reference); err != nil {
			return err
		}
		return email.Enqueue(ctx, tx, payerClaimedEmail(inv, ownerEmail, reference, r.publicURL), "invoice:"+inv.ID+":claimed")
	})
	if err != nil {
		return Invoice{}, err
	}
	return r.ByToken(ctx, token)
}

func (r *Repo) Void(ctx context.Context, ownerID, id string) (Invoice, error) {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		invID, st, err := lockOwned(ctx, tx, ownerID, id)
		if err != nil {
			return err
		}
		if st != StatusIssued {
			return ErrNotIssued
		}
		if _, err := tx.Exec(ctx, `UPDATE invoices SET status = 'void', voided_at = now(), updated_at = now() WHERE id = $1`, invID); err != nil {
			return err
		}
		return addEvent(ctx, tx, invID, "voided", "")
	})
	if err != nil {
		return Invoice{}, err
	}
	return r.Get(ctx, ownerID, id)
}

type Event struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"kind"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *Repo) Events(ctx context.Context, ownerID, id string) ([]Event, error) {
	if _, err := r.Get(ctx, ownerID, id); err != nil {
		return nil, err
	}
	rows, _ := r.db.Query(ctx, `
		SELECT e.id, e.kind, e.detail, e.created_at FROM invoice_events e
		JOIN invoices i ON i.id = e.invoice_id WHERE i.id::text = $1 ORDER BY e.id DESC LIMIT 200`, id)
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Event])
}

func (r *Repo) Summary(ctx context.Context, ownerID string) (Summary, error) {
	today := r.Today()
	fy := FinancialYear(today)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, IST)
	fyStartYear := today.Year()
	if today.Month() < time.April {
		fyStartYear--
	}
	fyStart := time.Date(fyStartYear, time.April, 1, 0, 0, 0, 0, IST)
	s := Summary{FY: fy}
	err := r.db.QueryRow(ctx, `
		SELECT
			coalesce(sum(total_minor) FILTER (WHERE status = 'issued'), 0),
			count(*) FILTER (WHERE status = 'issued'),
			coalesce(sum(total_minor) FILTER (WHERE status = 'issued' AND due_date < $2), 0),
			count(*) FILTER (WHERE status = 'issued' AND due_date < $2),
			coalesce(sum(total_minor) FILTER (WHERE status = 'paid' AND paid_at >= $3), 0),
			coalesce(sum(total_minor) FILTER (WHERE status = 'paid' AND paid_at >= $4), 0),
			count(*) FILTER (WHERE status = 'issued' AND payer_claimed_paid_at IS NOT NULL)
		FROM invoices WHERE owner_id = $1`, ownerID, today, monthStart, fyStart).
		Scan(&s.OutstandingMinor, &s.OutstandingCount, &s.OverdueMinor, &s.OverdueCount,
			&s.PaidThisMonthMinor, &s.PaidThisFYMinor, &s.ClaimedPaidCount)
	return s, err
}
