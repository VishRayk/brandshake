package profile

import "testing"

func TestCheckHandle(t *testing.T) {
	ok := []string{"maya", "maya.rao", "maya_rao_99", "a1b", "abcdefghijklmnopqrstuvwxyz1234"}
	bad := []string{"", "ab", "Maya", "maya rao", ".maya", "maya.", "_maya", "ma..ya", "admin", "brandshake",
		"abcdefghijklmnopqrstuvwxyz12345", "maya-rao", "मायā"}
	for _, h := range ok {
		if msg := CheckHandle(h); msg != "" {
			t.Errorf("CheckHandle(%q) = %q, want ok", h, msg)
		}
	}
	for _, h := range bad {
		if CheckHandle(h) == "" {
			t.Errorf("CheckHandle(%q) accepted", h)
		}
	}
}

func TestNormalizeHandle(t *testing.T) {
	if got := NormalizeHandle("  @Maya.Rao "); got != "maya.rao" {
		t.Errorf("got %q", got)
	}
}

func TestHostMatches(t *testing.T) {
	yt := platformHosts["youtube"]
	for _, h := range []string{"youtube.com", "www.youtube.com", "m.youtube.com", "youtu.be", "music.youtube.com"} {
		if !hostMatches(h, yt) {
			t.Errorf("%s should match youtube", h)
		}
	}
	for _, h := range []string{"notyoutube.com", "youtube.com.evil.example", "evil.example"} {
		if hostMatches(h, yt) {
			t.Errorf("%s should not match youtube", h)
		}
	}
}

func TestValidateSocialsStripsClientVerification(t *testing.T) {
	in := []SocialAccount{{Platform: "Instagram", Handle: "@me", URL: "https://instagram.com/me", Followers: 5, Verified: true}}
	if errs := ValidateSocials(in); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if in[0].Verified || in[0].Platform != "instagram" || in[0].Handle != "me" {
		t.Errorf("not normalised: %+v", in[0])
	}
}

func TestValidateRatesCapsTypos(t *testing.T) {
	errs := ValidateRates([]Rate{{Deliverable: "Reel", PriceMinor: 1_00_00_000_01}})
	if errs["rates[0].price_minor"] == "" {
		t.Error("price above ₹1 crore should be rejected")
	}
}
