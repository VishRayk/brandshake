package invoice

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GST state codes (first two digits of a GSTIN). Names match the web app's
// state list.
var StateCodes = map[string]string{
	"Jammu and Kashmir": "01", "Himachal Pradesh": "02", "Punjab": "03", "Chandigarh": "04",
	"Uttarakhand": "05", "Haryana": "06", "Delhi": "07", "Rajasthan": "08", "Uttar Pradesh": "09",
	"Bihar": "10", "Sikkim": "11", "Arunachal Pradesh": "12", "Nagaland": "13", "Manipur": "14",
	"Mizoram": "15", "Tripura": "16", "Meghalaya": "17", "Assam": "18", "West Bengal": "19",
	"Jharkhand": "20", "Odisha": "21", "Chhattisgarh": "22", "Madhya Pradesh": "23", "Gujarat": "24",
	"Dadra and Nagar Haveli and Daman and Diu": "26", "Maharashtra": "27", "Karnataka": "29",
	"Goa": "30", "Lakshadweep": "31", "Kerala": "32", "Tamil Nadu": "33", "Puducherry": "34",
	"Andaman and Nicobar Islands": "35", "Telangana": "36", "Andhra Pradesh": "37", "Ladakh": "38",
}

// StateForCode returns the state name for a GST state code, or "".
func StateForCode(code string) string {
	for name, c := range StateCodes {
		if c == code {
			return name
		}
	}
	return ""
}

var (
	panRe   = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)
	gstinRe = regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$`)
	ifscRe  = regexp.MustCompile(`^[A-Z]{4}0[A-Z0-9]{6}$`)
	upiRe   = regexp.MustCompile(`^[a-zA-Z0-9._-]{2,256}@[a-zA-Z][a-zA-Z0-9.-]{1,63}$`)
	acctRe  = regexp.MustCompile(`^[0-9]{9,18}$`)
	sacRe   = regexp.MustCompile(`^[0-9]{4,8}$`)
	prefRe  = regexp.MustCompile(`^[A-Z0-9]{1,5}$`)
)

func ValidPAN(s string) bool  { return panRe.MatchString(s) }
func ValidIFSC(s string) bool { return ifscRe.MatchString(s) }
func ValidUPI(s string) bool  { return upiRe.MatchString(s) }

const gstinChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// ValidGSTIN checks the format, state code and check digit of a GSTIN.
func ValidGSTIN(s string) bool {
	if !gstinRe.MatchString(s) || StateForCode(s[:2]) == "" {
		return false
	}
	sum := 0
	for i := range 14 {
		v := strings.IndexByte(gstinChars, s[i])
		factor := 1
		if i%2 == 1 {
			factor = 2
		}
		p := v * factor
		sum += p/36 + p%36
	}
	check := (36 - sum%36) % 36
	return s[14] == gstinChars[check]
}

// FinancialYear returns the Indian financial year (April–March) for a date,
// e.g. 2026-11-28 → "2026-27".
func FinancialYear(d time.Time) string {
	start := d.Year()
	if d.Month() < time.April {
		start--
	}
	return fmt.Sprintf("%d-%02d", start, (start+1)%100)
}

// InvoiceNumber formats a GST-compliant invoice number (≤ 16 characters,
// letters, digits, "/" and "-"), unique per financial year:
// prefix "INV", FY "2026-27", sequence 7 → "INV/26-27/0007".
func InvoiceNumber(prefix, fy string, seq int) string {
	return fmt.Sprintf("%s/%s/%04d", prefix, fy[2:], seq)
}

// UPIPayURI builds a UPI deep link that any UPI app can open (also the
// content of the payment QR code).
func UPIPayURI(vpa, payee string, amountMinor int64, note string) string {
	q := url.Values{}
	q.Set("pa", vpa)
	q.Set("pn", payee)
	q.Set("am", strconv.FormatInt(amountMinor/100, 10)+"."+fmt.Sprintf("%02d", amountMinor%100))
	q.Set("cu", "INR")
	q.Set("tn", note)
	return "upi://pay?" + strings.ReplaceAll(q.Encode(), "+", "%20")
}

// Allowed GST rates (percent) after the 2025 rate rationalisation. Most
// creator services (advertising, SAC 998361) are 18%. Creators should
// confirm their rate with a CA; Brandshake does not give tax advice.
var GSTRates = []int{0, 5, 18, 40}

const DefaultSAC = "998361" // advertising services

// roundDiv divides and rounds half up (non-negative inputs).
func roundDiv(n, d int64) int64 { return (2*n + d) / (2 * d) }

type Totals struct {
	Subtotal int64 `json:"subtotal_minor"`
	CGST     int64 `json:"cgst_minor"`
	SGST     int64 `json:"sgst_minor"`
	IGST     int64 `json:"igst_minor"`
	Total    int64 `json:"total_minor"`
}

// GSTMode is how tax applies: none (seller not registered), intra-state
// (CGST + SGST) or inter-state (IGST).
type GSTMode string

const (
	GSTNone  GSTMode = "none"
	GSTIntra GSTMode = "intra"
	GSTInter GSTMode = "inter"
)

// Compute returns the invoice totals. CGST and SGST are each half the rate
// on the taxable value, rounded to the paisa.
func Compute(subtotal int64, mode GSTMode, ratePercent int) Totals {
	t := Totals{Subtotal: subtotal}
	bp := int64(ratePercent) * 100 // basis points
	switch mode {
	case GSTIntra:
		t.CGST = roundDiv(subtotal*bp/2, 10_000)
		t.SGST = t.CGST
	case GSTInter:
		t.IGST = roundDiv(subtotal*bp, 10_000)
	}
	t.Total = t.Subtotal + t.CGST + t.SGST + t.IGST
	return t
}
