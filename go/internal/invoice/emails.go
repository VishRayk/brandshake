package invoice

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/VishRayk/brandshake/internal/email"
)

// FormatINR formats paise with Indian digit grouping: 177000050 → "₹17,70,000.50".
func FormatINR(paise int64) string {
	neg := paise < 0
	if neg {
		paise = -paise
	}
	rupees, p := paise/100, paise%100
	s := strconv.FormatInt(rupees, 10)
	if len(s) > 3 {
		head, tail := s[:len(s)-3], s[len(s)-3:]
		var parts []string
		for len(head) > 2 {
			parts = append([]string{head[len(head)-2:]}, parts...)
			head = head[:len(head)-2]
		}
		parts = append([]string{head}, parts...)
		s = strings.Join(parts, ",") + "," + tail
	}
	out := "₹" + s
	if p != 0 {
		out += fmt.Sprintf(".%02d", p)
	}
	if neg {
		out = "-" + out
	}
	return out
}

func humanDate(iso string) string {
	t, err := time.Parse(dateLayout, iso)
	if err != nil {
		return iso
	}
	return t.Format("2 Jan 2006")
}

func paymentLines(inv Invoice) string {
	var b strings.Builder
	if inv.Seller.UPIID != "" {
		fmt.Fprintf(&b, "UPI: %s\n", inv.Seller.UPIID)
	}
	if inv.Seller.BankAccountNumber != "" {
		fmt.Fprintf(&b, "Bank transfer: %s, A/c %s, IFSC %s\n", inv.Seller.BankAccountName, inv.Seller.BankAccountNumber, inv.Seller.BankIFSC)
	}
	return b.String()
}

func invoiceSentEmail(inv Invoice) email.Message {
	return email.Message{
		To:      inv.BuyerEmail,
		Kind:    "invoice_sent",
		Subject: fmt.Sprintf("Invoice %s from %s: %s due %s", inv.Number, inv.Seller.LegalName, FormatINR(inv.Total), humanDate(inv.DueDate)),
		Text: fmt.Sprintf(`Hi %s,

%s has sent you invoice %s for %s, due on %s.

View the invoice and pay: %s

Pay %s directly:
%s
After paying, open the link and tap "I've paid" so %s knows.

Sent with Brandshake. Brandshake does not handle payments.
`, inv.BuyerName, inv.Seller.LegalName, inv.Number, FormatINR(inv.Total), humanDate(inv.DueDate),
			inv.PublicURL, inv.Seller.LegalName, paymentLines(inv), inv.Seller.LegalName),
	}
}

func invoiceReminderEmail(inv Invoice, daysOverdue int) email.Message {
	return email.Message{
		To:      inv.BuyerEmail,
		Kind:    "invoice_overdue",
		Subject: fmt.Sprintf("Reminder: invoice %s from %s is %d day%s overdue", inv.Number, inv.Seller.LegalName, daysOverdue, plural(daysOverdue)),
		Text: fmt.Sprintf(`Hi %s,

A friendly reminder that invoice %s from %s for %s was due on %s.

View the invoice and pay: %s

%s
If you've already paid, open the link and tap "I've paid". Thank you!

Sent with Brandshake. Brandshake does not handle payments.
`, inv.BuyerName, inv.Number, inv.Seller.LegalName, FormatINR(inv.Total), humanDate(inv.DueDate), inv.PublicURL, paymentLines(inv)),
	}
}

func payerClaimedEmail(inv Invoice, ownerEmail, reference, publicURL string) email.Message {
	ref := ""
	if reference != "" {
		ref = fmt.Sprintf("\nPayment reference they gave: %s\n", reference)
	}
	return email.Message{
		To:      ownerEmail,
		Kind:    "invoice_payer_claimed",
		Subject: fmt.Sprintf("%s says they paid invoice %s", inv.BuyerName, inv.Number),
		Text: fmt.Sprintf(`%s says they've paid invoice %s (%s).
%s
Check your bank or UPI app, then mark the invoice as paid:
%s/invoices/%s
`, inv.BuyerName, inv.Number, FormatINR(inv.Total), ref, publicURL, inv.ID),
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
