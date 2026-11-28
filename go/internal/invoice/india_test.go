package invoice

import (
	"testing"
	"time"
)

func gstinWithCheck(prefix14 string) string {
	sum := 0
	for i := range 14 {
		v := indexOf(prefix14[i])
		f := 1
		if i%2 == 1 {
			f = 2
		}
		p := v * f
		sum += p/36 + p%36
	}
	return prefix14 + string(gstinChars[(36-sum%36)%36])
}

func indexOf(c byte) int {
	for i := range len(gstinChars) {
		if gstinChars[i] == c {
			return i
		}
	}
	return -1
}

func TestValidGSTIN(t *testing.T) {
	// A widely published sample GSTIN.
	if !ValidGSTIN("27AAPFU0939F1ZV") {
		t.Error("27AAPFU0939F1ZV should be valid")
	}
	good := gstinWithCheck("29ABCDE1234F1Z")
	if !ValidGSTIN(good) {
		t.Errorf("%s should be valid", good)
	}
	for _, bad := range []string{
		"27AAPFU0939F1ZW", // wrong check digit
		"99AAPFU0939F1ZV", // unknown state code
		"27aapfu0939f1zv", // lowercase
		"27AAPFU0939F1Z",  // too short
		"27AAPFU0939F1AV", // 14th char must be Z
		"",
	} {
		if ValidGSTIN(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestIdentifiers(t *testing.T) {
	cases := []struct {
		fn   func(string) bool
		good []string
		bad  []string
	}{
		{ValidPAN, []string{"ABCDE1234F"}, []string{"ABCD1234F", "abcde1234f", "ABCDE12345"}},
		{ValidIFSC, []string{"HDFC0001234", "SBIN0ABC123"}, []string{"HDFC1001234", "HDF0001234", "hdfc0001234"}},
		{ValidUPI, []string{"maya.rao@okhdfcbank", "9876543210@ybl"}, []string{"maya", "@ybl", "maya@", "ma ya@ybl"}},
	}
	for _, c := range cases {
		for _, g := range c.good {
			if !c.fn(g) {
				t.Errorf("%q should be valid", g)
			}
		}
		for _, b := range c.bad {
			if c.fn(b) {
				t.Errorf("%q should be invalid", b)
			}
		}
	}
}

func TestFinancialYearAndNumbering(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	for date, want := range map[string]string{
		"2026-11-28": "2026-27", "2027-03-31": "2026-27", "2027-04-01": "2027-28", "2099-12-01": "2099-00",
	} {
		if got := FinancialYear(d(date)); got != want {
			t.Errorf("FinancialYear(%s) = %s, want %s", date, got, want)
		}
	}
	n := InvoiceNumber("MAYA", "2026-27", 7)
	if n != "MAYA/26-27/0007" || len(InvoiceNumber("ABCDE", "2026-27", 9999)) > 16 {
		t.Errorf("number = %s", n)
	}
}

func TestComputeGST(t *testing.T) {
	cases := []struct {
		sub  int64
		mode GSTMode
		rate int
		want Totals
	}{
		{1_500_000, GSTNone, 18, Totals{Subtotal: 1_500_000, Total: 1_500_000}},
		{1_500_000, GSTIntra, 18, Totals{Subtotal: 1_500_000, CGST: 135_000, SGST: 135_000, Total: 1_770_000}},
		{1_500_000, GSTInter, 18, Totals{Subtotal: 1_500_000, IGST: 270_000, Total: 1_770_000}},
		// ₹333.33 at 5% intra: 2.5% each = 8.33325 → ₹8.33 each
		{33_333, GSTIntra, 5, Totals{Subtotal: 33_333, CGST: 833, SGST: 833, Total: 34_999}},
		// half-up rounding: ₹1.25 at 18% inter = 22.5 paise → 23
		{125, GSTInter, 18, Totals{Subtotal: 125, IGST: 23, Total: 148}},
		{1_00_00_000_00, GSTInter, 40, Totals{Subtotal: 1_00_00_000_00, IGST: 40_00_000_00, Total: 1_40_00_000_00}},
	}
	for _, c := range cases {
		if got := Compute(c.sub, c.mode, c.rate); got != c.want {
			t.Errorf("Compute(%d, %s, %d) = %+v, want %+v", c.sub, c.mode, c.rate, got, c.want)
		}
	}
}

func TestUPIPayURI(t *testing.T) {
	got := UPIPayURI("maya.rao@okhdfcbank", "Maya Rao", 1_770_050, "INV/26-27/0001")
	want := "upi://pay?am=17700.50&cu=INR&pa=maya.rao%40okhdfcbank&pn=Maya%20Rao&tn=INV%2F26-27%2F0001"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
