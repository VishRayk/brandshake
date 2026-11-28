package httpapi

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/VishRayk/brandshake/internal/invoice"
)

const creatorsOnlyInvoices = "Invoices are issued by creators."

func writeInvoiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, invoice.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Invoice not found.")
	case errors.Is(err, invoice.ErrNoBilling):
		writeError(w, http.StatusUnprocessableEntity, "billing_required", "Add your billing details before creating an invoice.")
	case errors.Is(err, invoice.ErrDealNotFound):
		writeValidation(w, map[string]string{"deal_id": "Deal not found."})
	case errors.Is(err, invoice.ErrNotIssued):
		writeError(w, http.StatusConflict, "invoice_closed", "This invoice is already paid or void.")
	case errors.Is(err, invoice.ErrNoBuyerEmail):
		writeValidation(w, map[string]string{"buyer_email": "This invoice has no email address to send to."})
	case errors.Is(err, invoice.ErrSentRecently):
		writeError(w, http.StatusTooManyRequests, "sent_recently", "This invoice was just sent. Try again in a few minutes.")
	case errors.Is(err, invoice.ErrAlreadyClaims):
		writeError(w, http.StatusConflict, "already_reported", "Payment for this invoice has already been reported.")
	default:
		writeStoreError(w, r, err)
	}
}

func (a *api) getBilling(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	b, err := a.Invoices.Billing(r.Context(), principal(r).UserID)
	if errors.Is(err, invoice.ErrNoBilling) {
		writeJSON(w, http.StatusOK, map[string]any{"billing": nil})
		return
	}
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"billing": b})
}

func (a *api) putBilling(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	var b invoice.Billing
	if err := decodeJSON(w, r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	b.Normalize()
	if errs := b.Validate(); len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	saved, err := a.Invoices.SaveBilling(r.Context(), principal(r).UserID, b)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"billing": saved})
}

var fyRe = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}$`)

func (a *api) listInvoices(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	f := invoice.ListFilter{Status: r.URL.Query().Get("status"), FY: r.URL.Query().Get("fy")}
	switch f.Status {
	case "", "issued", "paid", "void", "overdue":
	default:
		writeValidation(w, map[string]string{"status": "Unknown status."})
		return
	}
	if f.FY != "" && !fyRe.MatchString(f.FY) {
		writeValidation(w, map[string]string{"fy": "Use a financial year like 2026-27."})
		return
	}
	list, err := a.Invoices.List(r.Context(), principal(r).UserID, f)
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invoices": list})
}

func (a *api) createInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	var in invoice.Input
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	in.Normalize()
	inv, errs, err := a.Invoices.Create(r.Context(), principal(r).UserID, in)
	switch {
	case err != nil:
		writeInvoiceError(w, r, err)
	case len(errs) > 0:
		writeValidation(w, errs)
	default:
		writeJSON(w, http.StatusCreated, map[string]invoice.Invoice{"invoice": inv})
	}
}

func (a *api) getInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	inv, err := a.Invoices.Get(r.Context(), principal(r).UserID, r.PathValue("id"))
	if err != nil {
		writeInvoiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]invoice.Invoice{"invoice": inv})
}

func (a *api) invoiceEvents(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	events, err := a.Invoices.Events(r.Context(), principal(r).UserID, r.PathValue("id"))
	if err != nil {
		writeInvoiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (a *api) sendInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	inv, err := a.Invoices.Send(r.Context(), principal(r).UserID, r.PathValue("id"))
	if err != nil {
		writeInvoiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]invoice.Invoice{"invoice": inv})
}

func (a *api) markInvoicePaid(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	var req struct {
		PaidOn    string `json:"paid_on"`
		Reference string `json:"reference"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	today := a.Invoices.Today()
	paidOn := today
	if req.PaidOn != "" {
		t, err := time.ParseInLocation("2006-01-02", req.PaidOn, invoice.IST)
		if err != nil || t.After(today) {
			writeValidation(w, map[string]string{"paid_on": "Enter the date you received the money (not in the future)."})
			return
		}
		paidOn = t
	}
	req.Reference = strings.TrimSpace(req.Reference)
	if utf8.RuneCountInString(req.Reference) > 100 {
		writeValidation(w, map[string]string{"reference": "Keep the reference under 100 characters."})
		return
	}
	inv, err := a.Invoices.MarkPaid(r.Context(), principal(r).UserID, r.PathValue("id"), paidOn, req.Reference)
	if err != nil {
		writeInvoiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]invoice.Invoice{"invoice": inv})
}

func (a *api) voidInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	inv, err := a.Invoices.Void(r.Context(), principal(r).UserID, r.PathValue("id"))
	if err != nil {
		writeInvoiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]invoice.Invoice{"invoice": inv})
}

func (a *api) invoiceSummary(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	s, err := a.Invoices.Summary(r.Context(), principal(r).UserID)
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]invoice.Summary{"summary": s})
}

// csvSafe stops spreadsheet apps from treating a cell as a formula.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func rupees(paise int64) string { return fmt.Sprintf("%d.%02d", paise/100, paise%100) }

func (a *api) exportInvoicesCSV(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "creator", creatorsOnlyInvoices) {
		return
	}
	fy := r.URL.Query().Get("fy")
	if fy == "" {
		fy = invoice.FinancialYear(a.Invoices.Today())
	}
	if !fyRe.MatchString(fy) {
		writeValidation(w, map[string]string{"fy": "Use a financial year like 2026-27."})
		return
	}
	list, err := a.Invoices.List(r.Context(), principal(r).UserID, invoice.ListFilter{FY: fy})
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="brandshake-invoices-%s.csv"`, fy))
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"Invoice number", "Invoice date", "Due date", "Status", "Buyer", "Buyer GSTIN", "Place of supply",
		"Taxable value (INR)", "CGST (INR)", "SGST (INR)", "IGST (INR)", "Total (INR)", "Paid on", "Payment reference"})
	// Oldest first reads naturally in a ledger.
	for i := len(list) - 1; i >= 0; i-- {
		inv := list[i]
		paidOn := ""
		if inv.PaidAt != nil {
			paidOn = inv.PaidAt.In(invoice.IST).Format("2006-01-02")
		}
		_ = cw.Write([]string{
			inv.Number, inv.IssueDate, inv.DueDate, string(inv.Status), csvSafe(inv.BuyerName), inv.BuyerGSTIN, inv.BuyerState,
			rupees(inv.Subtotal), rupees(inv.CGST), rupees(inv.SGST), rupees(inv.IGST), rupees(inv.Total), paidOn, csvSafe(inv.PaidReference),
		})
	}
	cw.Flush()
}

// publicInvoice is the brand's no-login view of an invoice.
func (a *api) publicInvoice(w http.ResponseWriter, r *http.Request) {
	inv, err := a.Invoices.ByToken(r.Context(), r.PathValue("token"))
	if err != nil {
		writeInvoiceError(w, r, err)
		return
	}
	if err := a.Invoices.RecordView(r.Context(), inv.ID); err != nil {
		logWarn(r, "record invoice view", err)
	}
	inv.DealID, inv.RemindersSent = nil, 0 // creator-only details
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex")
	writeJSON(w, http.StatusOK, map[string]invoice.Invoice{"invoice": inv})
}

func (a *api) claimInvoicePaid(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reference string `json:"reference"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	req.Reference = strings.TrimSpace(req.Reference)
	if utf8.RuneCountInString(req.Reference) > 100 {
		writeValidation(w, map[string]string{"reference": "Keep the reference under 100 characters."})
		return
	}
	inv, err := a.Invoices.ClaimPaid(r.Context(), r.PathValue("token"), req.Reference)
	if err != nil {
		writeInvoiceError(w, r, err)
		return
	}
	inv.DealID, inv.RemindersSent = nil, 0
	writeJSON(w, http.StatusOK, map[string]invoice.Invoice{"invoice": inv})
}
