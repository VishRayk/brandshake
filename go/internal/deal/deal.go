// Package deal tracks brand–creator deals: terms, deliverables, status and
// history. Deals can be with someone on Brandshake or with an outside party.
// Brandshake never handles the money; amounts are recorded for tracking.
package deal

import (
	"net/mail"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Status string

const (
	StatusLead        Status = "lead"
	StatusNegotiating Status = "negotiating"
	StatusConfirmed   Status = "confirmed"
	StatusInProgress  Status = "in_progress"
	StatusPosted      Status = "posted"
	StatusPaid        Status = "paid"
	StatusLost        Status = "lost"
)

// Statuses in pipeline order.
var Statuses = []Status{StatusLead, StatusNegotiating, StatusConfirmed, StatusInProgress, StatusPosted, StatusPaid, StatusLost}

func (s Status) Valid() bool { return slices.Contains(Statuses, s) }

// Active means agreed and not yet finished: work or payment is outstanding.
func (s Status) Active() bool {
	return s == StatusConfirmed || s == StatusInProgress || s == StatusPosted
}

// Closed deals no longer need attention.
func (s Status) Closed() bool { return s == StatusPaid || s == StatusLost }

var Platforms = []string{"instagram", "youtube", "x", "linkedin", "facebook", "other"}

const (
	MaxDeliverables = 20
	dateLayout      = "2006-01-02"
)

// Counterparty is the other side of the deal.
type Counterparty struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	User  *UserLink `json:"user"` // set when they are on Brandshake
}

type UserLink struct {
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type Deliverable struct {
	Title    string     `json:"title"`
	Platform string     `json:"platform"`
	DueDate  *string    `json:"due_date"` // YYYY-MM-DD
	Done     bool       `json:"done"`
	DoneAt   *time.Time `json:"done_at"`
}

type Deal struct {
	ID              string        `json:"id"`
	Title           string        `json:"title"`
	Status          Status        `json:"status"`
	Counterparty    Counterparty  `json:"counterparty"`
	AmountMinor     *int64        `json:"amount_minor"`
	Currency        string        `json:"currency"`
	Revisions       *int          `json:"revisions"`
	UsageRightsDays *int          `json:"usage_rights_days"`
	UsagePerpetual  bool          `json:"usage_perpetual"`
	ExclusivityDays *int          `json:"exclusivity_days"`
	Notes           string        `json:"notes"`
	Source          string        `json:"source"`
	Deliverables    []Deliverable `json:"deliverables"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	StatusChangedAt time.Time     `json:"status_changed_at"`
}

type Event struct {
	ID         int64     `json:"id"`
	Kind       string    `json:"kind"`
	FromStatus *string   `json:"from_status"`
	ToStatus   *string   `json:"to_status"`
	Detail     string    `json:"detail"`
	CreatedAt  time.Time `json:"created_at"`
}

// Input is what a client sends to create or fully update a deal.
type Input struct {
	Title              string             `json:"title"`
	Status             Status             `json:"status"`
	CounterpartyName   string             `json:"counterparty_name"`
	CounterpartyEmail  string             `json:"counterparty_email"`
	CounterpartyHandle string             `json:"counterparty_handle"`
	AmountMinor        *int64             `json:"amount_minor"`
	Revisions          *int               `json:"revisions"`
	UsageRightsDays    *int               `json:"usage_rights_days"`
	UsagePerpetual     bool               `json:"usage_perpetual"`
	ExclusivityDays    *int               `json:"exclusivity_days"`
	Notes              string             `json:"notes"`
	Source             string             `json:"source"`
	Deliverables       []DeliverableInput `json:"deliverables"`
}

type DeliverableInput struct {
	Title    string `json:"title"`
	Platform string `json:"platform"`
	DueDate  string `json:"due_date"` // "" or YYYY-MM-DD
	Done     bool   `json:"done"`
}

type Errors map[string]string

func (e Errors) add(field, msg string) {
	if _, ok := e[field]; !ok {
		e[field] = msg
	}
}

func runes(s string) int { return utf8.RuneCountInString(s) }

// Normalize trims text and fills defaults in place.
func (in *Input) Normalize() {
	in.Title = strings.TrimSpace(in.Title)
	in.CounterpartyName = strings.TrimSpace(in.CounterpartyName)
	in.CounterpartyEmail = strings.ToLower(strings.TrimSpace(in.CounterpartyEmail))
	in.CounterpartyHandle = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(in.CounterpartyHandle), "@"))
	in.Notes = strings.TrimSpace(in.Notes)
	if in.Status == "" {
		in.Status = StatusLead
	}
	if in.Source == "" {
		in.Source = "manual"
	}
	if in.UsagePerpetual {
		in.UsageRightsDays = nil
	}
	for i := range in.Deliverables {
		d := &in.Deliverables[i]
		d.Title = strings.TrimSpace(d.Title)
		d.Platform = strings.ToLower(strings.TrimSpace(d.Platform))
		d.DueDate = strings.TrimSpace(d.DueDate)
		if d.Platform == "" {
			d.Platform = "other"
		}
	}
}

func (in *Input) Validate() Errors {
	errs := Errors{}
	if in.Title == "" || runes(in.Title) > 120 {
		errs.add("title", "Give the deal a title (up to 120 characters).")
	}
	if !in.Status.Valid() {
		errs.add("status", "Unknown status.")
	}
	if in.CounterpartyName == "" && in.CounterpartyHandle == "" {
		errs.add("counterparty_name", "Who is this deal with?")
	}
	if runes(in.CounterpartyName) > 120 {
		errs.add("counterparty_name", "Name is too long.")
	}
	if in.CounterpartyEmail != "" {
		if a, err := mail.ParseAddress(in.CounterpartyEmail); err != nil || a.Address != in.CounterpartyEmail || len(in.CounterpartyEmail) > 254 {
			errs.add("counterparty_email", "Enter a valid email address.")
		}
	}
	// ₹10 crore cap catches typos with extra zeros.
	if in.AmountMinor != nil && (*in.AmountMinor < 0 || *in.AmountMinor > 10_00_00_000_00) {
		errs.add("amount_minor", "Enter an amount between ₹0 and ₹10,00,00,000.")
	}
	checkInt := func(field string, v *int, max int, msg string) {
		if v != nil && (*v < 0 || *v > max) {
			errs.add(field, msg)
		}
	}
	checkInt("revisions", in.Revisions, 20, "Revisions must be between 0 and 20.")
	checkInt("usage_rights_days", in.UsageRightsDays, 3650, "Usage rights must be between 0 and 3,650 days.")
	checkInt("exclusivity_days", in.ExclusivityDays, 3650, "Exclusivity must be between 0 and 3,650 days.")
	if runes(in.Notes) > 5000 {
		errs.add("notes", "Notes are limited to 5,000 characters.")
	}
	if in.Source != "manual" && in.Source != "ai_draft" {
		errs.add("source", "Unknown source.")
	}
	if len(in.Deliverables) > MaxDeliverables {
		errs.add("deliverables", "Add up to 20 deliverables.")
		return errs
	}
	for i, d := range in.Deliverables {
		f := func(name string) string { return "deliverables[" + strconv.Itoa(i) + "]." + name }
		if d.Title == "" || runes(d.Title) > 120 {
			errs.add(f("title"), "Describe the deliverable.")
		}
		if !slices.Contains(Platforms, d.Platform) {
			errs.add(f("platform"), "Unknown platform.")
		}
		if d.DueDate != "" {
			if _, err := time.Parse(dateLayout, d.DueDate); err != nil {
				errs.add(f("due_date"), "Use a date like 2026-12-31.")
			}
		}
	}
	return errs
}

// Summary powers the dashboard home.
type Summary struct {
	Counts           map[Status]int `json:"counts"`
	ActiveCount      int            `json:"active_count"`
	ActiveValueMinor int64          `json:"active_value_minor"`
	AwaitingPayment  int64          `json:"awaiting_payment_minor"` // posted, not yet marked paid
	PaidThisMonth    int64          `json:"paid_this_month_minor"`
	Overdue          int            `json:"overdue_deliverables"`
	DueSoon          []DueItem      `json:"due_soon"`
}

type DueItem struct {
	DealID      string `json:"deal_id"`
	DealTitle   string `json:"deal_title"`
	Counterpart string `json:"counterparty_name"`
	Title       string `json:"title"`
	Platform    string `json:"platform"`
	DueDate     string `json:"due_date"`
	Overdue     bool   `json:"overdue"`
}
