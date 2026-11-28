package httpapi

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/VishRayk/brandshake/internal/aiclient"
	"github.com/VishRayk/brandshake/internal/deal"
)

// IST is used for "today" in summaries until users can pick a timezone.
var ist = time.FixedZone("IST", 5*60*60+30*60)

func writeDealError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, deal.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Deal not found.")
	case errors.Is(err, deal.ErrDeliverableNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Deliverable not found.")
	case errors.Is(err, deal.ErrCounterpartyNotFound):
		writeValidation(w, map[string]string{"counterparty_handle": "No one on Brandshake has this handle."})
	case errors.Is(err, deal.ErrCounterpartySameRole):
		writeValidation(w, map[string]string{"counterparty_handle": "Deals are between a creator and a brand."})
	case errors.Is(err, deal.ErrCounterpartySelf):
		writeValidation(w, map[string]string{"counterparty_handle": "You can't make a deal with yourself."})
	default:
		writeStoreError(w, r, err)
	}
}

func (a *api) decodeDealInput(w http.ResponseWriter, r *http.Request) (deal.Input, bool) {
	var in deal.Input
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return in, false
	}
	in.Normalize()
	if errs := in.Validate(); len(errs) > 0 {
		writeValidation(w, errs)
		return in, false
	}
	return in, true
}

func (a *api) listDeals(w http.ResponseWriter, r *http.Request) {
	f := deal.ListFilter{Status: deal.Status(r.URL.Query().Get("status")), Query: r.URL.Query().Get("q")}
	if f.Status != "" && !f.Status.Valid() {
		writeValidation(w, map[string]string{"status": "Unknown status."})
		return
	}
	deals, err := a.Deals.List(r.Context(), principal(r).UserID, f)
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deals": deals})
}

func (a *api) createDeal(w http.ResponseWriter, r *http.Request) {
	in, ok := a.decodeDealInput(w, r)
	if !ok {
		return
	}
	p := principal(r)
	d, err := a.Deals.Create(r.Context(), p.UserID, p.Role, in)
	if err != nil {
		writeDealError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]deal.Deal{"deal": d})
}

func (a *api) getDeal(w http.ResponseWriter, r *http.Request) {
	d, err := a.Deals.Get(r.Context(), principal(r).UserID, r.PathValue("id"))
	if err != nil {
		writeDealError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]deal.Deal{"deal": d})
}

func (a *api) updateDeal(w http.ResponseWriter, r *http.Request) {
	in, ok := a.decodeDealInput(w, r)
	if !ok {
		return
	}
	p := principal(r)
	d, err := a.Deals.Update(r.Context(), p.UserID, p.Role, r.PathValue("id"), in)
	if err != nil {
		writeDealError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]deal.Deal{"deal": d})
}

func (a *api) setDealStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status deal.Status `json:"status"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if !req.Status.Valid() {
		writeValidation(w, map[string]string{"status": "Unknown status."})
		return
	}
	d, err := a.Deals.SetStatus(r.Context(), principal(r).UserID, r.PathValue("id"), req.Status)
	if err != nil {
		writeDealError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]deal.Deal{"deal": d})
}

func (a *api) setDeliverableDone(w http.ResponseWriter, r *http.Request) {
	pos, err := strconv.Atoi(r.PathValue("position"))
	if err != nil || pos < 0 || pos >= deal.MaxDeliverables {
		writeError(w, http.StatusNotFound, "not_found", "Deliverable not found.")
		return
	}
	var req struct {
		Done bool `json:"done"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	d, err := a.Deals.SetDeliverableDone(r.Context(), principal(r).UserID, r.PathValue("id"), pos, req.Done)
	if err != nil {
		writeDealError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]deal.Deal{"deal": d})
}

func (a *api) deleteDeal(w http.ResponseWriter, r *http.Request) {
	if err := a.Deals.Delete(r.Context(), principal(r).UserID, r.PathValue("id")); err != nil {
		writeDealError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) dealEvents(w http.ResponseWriter, r *http.Request) {
	events, err := a.Deals.Events(r.Context(), principal(r).UserID, r.PathValue("id"))
	if err != nil {
		writeDealError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (a *api) dealSummary(w http.ResponseWriter, r *http.Request) {
	s, err := a.Deals.Summary(r.Context(), principal(r).UserID, time.Now().In(ist))
	if err != nil {
		writeInternal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]deal.Summary{"summary": s})
}

// draftDeal turns pasted text into a suggested deal. Nothing is saved: the
// client shows the suggestion in the create form for the user to confirm.
func (a *api) draftDeal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" || utf8.RuneCountInString(req.Text) > 20_000 {
		writeValidation(w, map[string]string{"text": "Paste an email or chat (up to 20,000 characters)."})
		return
	}
	if a.AI == nil {
		writeError(w, http.StatusServiceUnavailable, "ai_unavailable", "Drafting from text is unavailable right now. Fill in the deal manually.")
		return
	}
	res, err := a.AI.DraftDeal(r.Context(), req.Text, principal(r).Role)
	if err != nil {
		writeAIUnavailable(w, r, err)
		return
	}
	suggestion, notes := suggestionFromDraft(res.Draft)
	writeJSON(w, http.StatusOK, map[string]any{
		"suggestion":     suggestion,
		"summary":        res.Draft.Summary,
		"open_questions": notes,
		"risk_flags":     nonNilSlice(res.RiskFlags),
	})
}

func writeAIUnavailable(w http.ResponseWriter, r *http.Request, err error) {
	logWarn(r, "ai draft failed", err)
	writeError(w, http.StatusServiceUnavailable, "ai_unavailable", "Drafting from text is unavailable right now. Fill in the deal manually.")
}

var platformLabels = map[string]string{
	"instagram": "Instagram", "youtube": "YouTube", "x": "X", "linkedin": "LinkedIn", "facebook": "Facebook",
}

// suggestionFromDraft maps the AI draft onto a deal.Input, keeping only
// values that pass the same validation as user input.
func suggestionFromDraft(d aiclient.Draft) (deal.Input, []string) {
	notes := append([]string{}, d.OpenQuestions...)
	in := deal.Input{Title: d.Title, Status: deal.StatusNegotiating, Source: "ai_draft"}
	if d.CounterpartyName != nil {
		in.CounterpartyName = *d.CounterpartyName
	}
	if d.CounterpartyEmail != nil {
		if a, err := mail.ParseAddress(*d.CounterpartyEmail); err == nil {
			in.CounterpartyEmail = a.Address
		}
	}
	if d.PriceAmount != nil && *d.PriceAmount >= 0 {
		cur := "INR"
		if d.Currency != nil {
			cur = strings.ToUpper(*d.Currency)
		}
		if cur == "INR" {
			v := int64(math.Round(*d.PriceAmount * 100))
			in.AmountMinor = &v
		} else {
			notes = append(notes, fmt.Sprintf("The price was mentioned in %s (%.2f). Enter the INR amount.", cur, *d.PriceAmount))
		}
	}
	in.Revisions, in.ExclusivityDays = d.Revisions, d.ExclusivityDays
	if d.UsageRights.Granted {
		in.UsagePerpetual = d.UsageRights.Perpetual
		in.UsageRightsDays = d.UsageRights.DurationDays
	}

	for _, x := range d.Deliverables {
		platform := x.Platform
		if _, ok := platformLabels[platform]; !ok {
			platform = "other"
		}
		label := strings.TrimSpace(platformLabels[platform] + " " + x.Format)
		qty := min(max(x.Quantity, 1), deal.MaxDeliverables)
		due := ""
		if x.DueDate != nil {
			if _, err := time.Parse("2006-01-02", *x.DueDate); err == nil {
				due = *x.DueDate
			}
		}
		for i := 1; i <= qty && len(in.Deliverables) < deal.MaxDeliverables; i++ {
			title := label
			if qty > 1 {
				title = fmt.Sprintf("%s %d of %d", label, i, qty)
			}
			in.Deliverables = append(in.Deliverables, deal.DeliverableInput{Title: title, Platform: platform, DueDate: due})
		}
	}

	in.Normalize()
	if in.Title == "" {
		in.Title = "New deal"
	}
	// Drop anything the AI produced that would fail validation, rather than
	// handing the user a form full of errors.
	errs := in.Validate()
	if _, bad := errs["counterparty_email"]; bad {
		in.CounterpartyEmail = ""
	}
	if _, bad := errs["amount_minor"]; bad {
		in.AmountMinor = nil
	}
	for _, f := range []struct {
		key string
		v   **int
	}{{"revisions", &in.Revisions}, {"usage_rights_days", &in.UsageRightsDays}, {"exclusivity_days", &in.ExclusivityDays}} {
		if _, bad := errs[f.key]; bad {
			*f.v = nil
		}
	}
	if utf8.RuneCountInString(in.Title) > 120 {
		in.Title = string([]rune(in.Title)[:120])
	}
	return in, notes
}
