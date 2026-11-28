// Package invoice lets creators issue GST-ready invoices and track payment.
// Brands pay creators directly; Brandshake only records that it happened.
package invoice

import (
	"net/mail"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Errors map[string]string

func (e Errors) add(field, msg string) {
	if _, ok := e[field]; !ok {
		e[field] = msg
	}
}

func runes(s string) int { return utf8.RuneCountInString(s) }

// Billing is a creator's invoicing details.
type Billing struct {
	LegalName         string    `json:"legal_name"`
	Address           string    `json:"address"`
	State             string    `json:"state"`
	PAN               string    `json:"pan"`
	GSTRegistered     bool      `json:"gst_registered"`
	GSTIN             string    `json:"gstin"`
	GSTRate           int       `json:"gst_rate"`
	UPIID             string    `json:"upi_id"`
	BankAccountName   string    `json:"bank_account_name"`
	BankAccountNumber string    `json:"bank_account_number"`
	BankIFSC          string    `json:"bank_ifsc"`
	BankName          string    `json:"bank_name"`
	InvoicePrefix     string    `json:"invoice_prefix"`
	PaymentTermsDays  int       `json:"payment_terms_days"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (b *Billing) Normalize() {
	b.LegalName = strings.TrimSpace(b.LegalName)
	b.Address = strings.TrimSpace(b.Address)
	b.State = strings.TrimSpace(b.State)
	b.PAN = strings.ToUpper(strings.TrimSpace(b.PAN))
	b.GSTIN = strings.ToUpper(strings.TrimSpace(b.GSTIN))
	b.UPIID = strings.TrimSpace(b.UPIID)
	b.BankAccountName = strings.TrimSpace(b.BankAccountName)
	b.BankAccountNumber = strings.ReplaceAll(strings.TrimSpace(b.BankAccountNumber), " ", "")
	b.BankIFSC = strings.ToUpper(strings.TrimSpace(b.BankIFSC))
	b.BankName = strings.TrimSpace(b.BankName)
	b.InvoicePrefix = strings.ToUpper(strings.TrimSpace(b.InvoicePrefix))
	if b.InvoicePrefix == "" {
		b.InvoicePrefix = "INV"
	}
	if !b.GSTRegistered {
		b.GSTIN = ""
	}
	if b.GSTRegistered && b.PAN == "" && len(b.GSTIN) == 15 {
		b.PAN = b.GSTIN[2:12]
	}
}

func (b *Billing) Validate() Errors {
	errs := Errors{}
	if b.LegalName == "" || runes(b.LegalName) > 120 {
		errs.add("legal_name", "Enter the name to show on invoices (your name or business name).")
	}
	if runes(b.Address) > 300 {
		errs.add("address", "Address is too long.")
	}
	if _, ok := StateCodes[b.State]; !ok {
		errs.add("state", "Choose your state.")
	}
	if b.PAN != "" && !ValidPAN(b.PAN) {
		errs.add("pan", "Enter a valid PAN, e.g. ABCDE1234F.")
	}
	if b.GSTRegistered {
		switch {
		case !ValidGSTIN(b.GSTIN):
			errs.add("gstin", "Enter a valid 15-character GSTIN.")
		case StateCodes[b.State] != b.GSTIN[:2]:
			errs.add("gstin", "This GSTIN is registered in a different state than the one you chose.")
		case b.PAN != "" && b.PAN != b.GSTIN[2:12]:
			errs.add("pan", "PAN doesn't match the one inside your GSTIN.")
		}
		if !slices.Contains(GSTRates, b.GSTRate) {
			errs.add("gst_rate", "Choose a GST rate.")
		}
	}
	if b.UPIID != "" && !ValidUPI(b.UPIID) {
		errs.add("upi_id", "Enter a valid UPI ID, e.g. yourname@okhdfcbank.")
	}
	hasBank := b.BankAccountNumber != "" || b.BankIFSC != "" || b.BankAccountName != ""
	if hasBank {
		if !acctRe.MatchString(b.BankAccountNumber) {
			errs.add("bank_account_number", "Enter a 9–18 digit account number.")
		}
		if !ValidIFSC(b.BankIFSC) {
			errs.add("bank_ifsc", "Enter a valid IFSC, e.g. HDFC0001234.")
		}
		if b.BankAccountName == "" || runes(b.BankAccountName) > 120 {
			errs.add("bank_account_name", "Enter the account holder's name.")
		}
	}
	if b.UPIID == "" && !hasBank {
		errs.add("upi_id", "Add a UPI ID or bank account so brands can pay you.")
	}
	if runes(b.BankName) > 80 {
		errs.add("bank_name", "Bank name is too long.")
	}
	if !prefRe.MatchString(b.InvoicePrefix) {
		errs.add("invoice_prefix", "Use 1–5 capital letters or digits.")
	}
	if b.PaymentTermsDays < 0 || b.PaymentTermsDays > 120 {
		errs.add("payment_terms_days", "Payment terms must be 0–120 days.")
	}
	return errs
}

// Seller is the snapshot of billing details stored on each invoice.
type Seller struct {
	LegalName         string `json:"legal_name"`
	Address           string `json:"address"`
	State             string `json:"state"`
	PAN               string `json:"pan"`
	GSTIN             string `json:"gstin"`
	UPIID             string `json:"upi_id"`
	BankAccountName   string `json:"bank_account_name"`
	BankAccountNumber string `json:"bank_account_number"`
	BankIFSC          string `json:"bank_ifsc"`
	BankName          string `json:"bank_name"`
}

type Item struct {
	Description    string `json:"description"`
	SACCode        string `json:"sac_code"`
	Quantity       int    `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	AmountMinor    int64  `json:"amount_minor"`
}

type Status string

const (
	StatusIssued Status = "issued"
	StatusPaid   Status = "paid"
	StatusVoid   Status = "void"
)

type Invoice struct {
	ID                 string     `json:"id"`
	Number             string     `json:"number"`
	FY                 string     `json:"fy"`
	Status             Status     `json:"status"`
	Overdue            bool       `json:"overdue"`
	IssueDate          string     `json:"issue_date"`
	DueDate            string     `json:"due_date"`
	Seller             Seller     `json:"seller"`
	BuyerName          string     `json:"buyer_name"`
	BuyerEmail         string     `json:"buyer_email"`
	BuyerAddress       string     `json:"buyer_address"`
	BuyerState         string     `json:"buyer_state"`
	BuyerGSTIN         string     `json:"buyer_gstin"`
	GSTMode            GSTMode    `json:"gst_mode"`
	GSTRate            int        `json:"gst_rate"`
	Items              []Item     `json:"items"`
	Totals                        // subtotal, cgst, sgst, igst, total
	Notes              string     `json:"notes"`
	DealID             *string    `json:"deal_id,omitempty"`
	PublicToken        string     `json:"-"`
	PublicURL          string     `json:"public_url,omitempty"`
	UPIURI             string     `json:"upi_uri,omitempty"`
	SentAt             *time.Time `json:"sent_at"`
	PayerClaimedPaidAt *time.Time `json:"payer_claimed_paid_at"`
	PayerReference     string     `json:"payer_reference"`
	PaidAt             *time.Time `json:"paid_at"`
	PaidReference      string     `json:"paid_reference"`
	VoidedAt           *time.Time `json:"voided_at"`
	RemindersSent      int        `json:"reminders_sent"`
	CreatedAt          time.Time  `json:"created_at"`
}

// Input is what a creator sends to issue an invoice.
type Input struct {
	DealID       string      `json:"deal_id"`
	BuyerName    string      `json:"buyer_name"`
	BuyerEmail   string      `json:"buyer_email"`
	BuyerAddress string      `json:"buyer_address"`
	BuyerState   string      `json:"buyer_state"`
	BuyerGSTIN   string      `json:"buyer_gstin"`
	IssueDate    string      `json:"issue_date"` // "" = today (IST)
	DueDate      string      `json:"due_date"`   // "" = issue date + payment terms
	Items        []ItemInput `json:"items"`
	Notes        string      `json:"notes"`
}

type ItemInput struct {
	Description    string `json:"description"`
	SACCode        string `json:"sac_code"`
	Quantity       int    `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
}

const (
	MaxItems       = 20
	dateLayout     = "2006-01-02"
	maxTotalMinor  = 10_00_00_000_00 // ₹10 crore
	maxBackdateDay = 365
)

func (in *Input) Normalize() {
	in.DealID = strings.TrimSpace(in.DealID)
	in.BuyerName = strings.TrimSpace(in.BuyerName)
	in.BuyerEmail = strings.ToLower(strings.TrimSpace(in.BuyerEmail))
	in.BuyerAddress = strings.TrimSpace(in.BuyerAddress)
	in.BuyerState = strings.TrimSpace(in.BuyerState)
	in.BuyerGSTIN = strings.ToUpper(strings.TrimSpace(in.BuyerGSTIN))
	in.Notes = strings.TrimSpace(in.Notes)
	if in.BuyerGSTIN != "" && in.BuyerState == "" && len(in.BuyerGSTIN) >= 2 {
		in.BuyerState = StateForCode(in.BuyerGSTIN[:2])
	}
	for i := range in.Items {
		it := &in.Items[i]
		it.Description = strings.TrimSpace(it.Description)
		it.SACCode = strings.TrimSpace(it.SACCode)
		if it.SACCode == "" {
			it.SACCode = DefaultSAC
		}
		if it.Quantity == 0 {
			it.Quantity = 1
		}
	}
}

// Validate checks the input for a seller and resolves the issue and due
// dates. today is the current date in IST.
func (in *Input) Validate(b Billing, today time.Time) (issue, due time.Time, errs Errors) {
	errs = Errors{}
	if in.BuyerName == "" || runes(in.BuyerName) > 120 {
		errs.add("buyer_name", "Who is this invoice for?")
	}
	if in.BuyerEmail != "" {
		if a, err := mail.ParseAddress(in.BuyerEmail); err != nil || a.Address != in.BuyerEmail {
			errs.add("buyer_email", "Enter a valid email address.")
		}
	}
	if runes(in.BuyerAddress) > 300 {
		errs.add("buyer_address", "Address is too long.")
	}
	if in.BuyerState != "" {
		if _, ok := StateCodes[in.BuyerState]; !ok {
			errs.add("buyer_state", "Choose a state.")
		}
	} else if b.GSTRegistered {
		errs.add("buyer_state", "Choose the brand's state. It decides whether CGST + SGST or IGST applies.")
	}
	if in.BuyerGSTIN != "" {
		switch {
		case !ValidGSTIN(in.BuyerGSTIN):
			errs.add("buyer_gstin", "Enter a valid 15-character GSTIN.")
		case in.BuyerState != "" && StateCodes[in.BuyerState] != in.BuyerGSTIN[:2]:
			errs.add("buyer_gstin", "This GSTIN belongs to a different state.")
		}
	}
	if runes(in.Notes) > 1000 {
		errs.add("notes", "Notes are limited to 1,000 characters.")
	}

	issue = today
	if in.IssueDate != "" {
		t, err := time.ParseInLocation(dateLayout, in.IssueDate, today.Location())
		switch {
		case err != nil:
			errs.add("issue_date", "Use a date like 2026-12-01.")
		case t.After(today):
			errs.add("issue_date", "Invoices can't be dated in the future.")
		case today.Sub(t) > maxBackdateDay*24*time.Hour:
			errs.add("issue_date", "Invoices can't be backdated by more than a year.")
		default:
			issue = t
		}
	}
	due = issue.AddDate(0, 0, b.PaymentTermsDays)
	if in.DueDate != "" {
		t, err := time.ParseInLocation(dateLayout, in.DueDate, today.Location())
		switch {
		case err != nil:
			errs.add("due_date", "Use a date like 2026-12-15.")
		case t.Before(issue):
			errs.add("due_date", "The due date can't be before the invoice date.")
		default:
			due = t
		}
	}

	if len(in.Items) == 0 {
		errs.add("items", "Add at least one line item.")
	}
	if len(in.Items) > MaxItems {
		errs.add("items", "Add up to 20 line items.")
		return issue, due, errs
	}
	var subtotal int64
	for i, it := range in.Items {
		f := func(n string) string { return "items[" + strconv.Itoa(i) + "]." + n }
		if it.Description == "" || runes(it.Description) > 200 {
			errs.add(f("description"), "Describe the work, e.g. \"Instagram Reel for FitBar launch\".")
		}
		if !sacRe.MatchString(it.SACCode) {
			errs.add(f("sac_code"), "SAC code must be 4–8 digits.")
		}
		if it.Quantity < 1 || it.Quantity > 1000 {
			errs.add(f("quantity"), "Quantity must be 1–1,000.")
		}
		if it.UnitPriceMinor < 0 || it.UnitPriceMinor > maxTotalMinor {
			errs.add(f("unit_price_minor"), "Enter a price between ₹0 and ₹10,00,00,000.")
		}
		subtotal += int64(max(it.Quantity, 0)) * max(it.UnitPriceMinor, 0)
	}
	if subtotal > maxTotalMinor {
		errs.add("items", "The invoice total can't exceed ₹10,00,00,000.")
	}
	return issue, due, errs
}

// ModeFor decides how GST applies between a seller and a place of supply.
func ModeFor(b Billing, buyerState string) GSTMode {
	switch {
	case !b.GSTRegistered:
		return GSTNone
	case buyerState == b.State:
		return GSTIntra
	default:
		return GSTInter
	}
}

// Summary is the creator's earnings overview.
type Summary struct {
	OutstandingMinor   int64  `json:"outstanding_minor"`
	OutstandingCount   int    `json:"outstanding_count"`
	OverdueMinor       int64  `json:"overdue_minor"`
	OverdueCount       int    `json:"overdue_count"`
	PaidThisMonthMinor int64  `json:"paid_this_month_minor"`
	PaidThisFYMinor    int64  `json:"paid_this_fy_minor"`
	FY                 string `json:"fy"`
	ClaimedPaidCount   int    `json:"claimed_paid_count"` // brand says paid, awaiting your confirmation
}
