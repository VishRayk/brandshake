// Package profile manages public creator and brand profiles: the profile
// itself, social accounts, a creator's rate card and work samples.
package profile

import (
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Profile struct {
	UserID        string    `json:"-"`
	Handle        string    `json:"handle"`
	Headline      string    `json:"headline"`
	Bio           string    `json:"bio"`
	City          string    `json:"city"`
	State         string    `json:"state"`
	Website       string    `json:"website"`
	Niches        []string  `json:"niches"`
	Languages     []string  `json:"languages"`
	OpenToCollabs bool      `json:"open_to_collabs"`
	Industry      string    `json:"industry"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SocialAccount struct {
	Platform  string `json:"platform"`
	Handle    string `json:"handle"`
	URL       string `json:"url"`
	Followers int    `json:"followers"`
	Verified  bool   `json:"verified"`
}

type Rate struct {
	Deliverable string `json:"deliverable"`
	PriceMinor  int64  `json:"price_minor"`
	Note        string `json:"note"`
}

type Sample struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Platform  string `json:"platform"`
	BrandName string `json:"brand_name"`
}

// Public is everything shown on a public profile page. It never includes
// the account email.
type Public struct {
	Profile
	Role        string          `json:"role"`
	DisplayName string          `json:"display_name"`
	MemberSince time.Time       `json:"member_since"`
	Socials     []SocialAccount `json:"socials"`
	Rates       []Rate          `json:"rates"`
	Samples     []Sample        `json:"samples"`
}

// Vocabularies are fixed so that search and filters stay consistent.
var (
	Niches = []string{
		"art", "automotive", "beauty", "business", "comedy", "education", "fashion", "finance",
		"fitness", "food", "gaming", "health", "lifestyle", "music", "parenting", "photography",
		"sports", "tech", "travel", "other",
	}
	Industries = []string{
		"automotive", "beauty_personal_care", "education", "fashion_apparel", "finance",
		"food_beverage", "gaming", "health_fitness", "home_living", "media_entertainment",
		"tech_apps", "travel_hospitality", "other",
	}
	Languages = []string{"en", "hi", "bn", "gu", "kn", "ml", "mr", "or", "pa", "ta", "te", "ur"}
	Platforms = []string{"instagram", "youtube", "x", "linkedin", "facebook"}

	platformHosts = map[string][]string{
		"instagram": {"instagram.com"},
		"youtube":   {"youtube.com", "youtu.be"},
		"x":         {"x.com", "twitter.com"},
		"linkedin":  {"linkedin.com"},
		"facebook":  {"facebook.com", "fb.com"},
	}

	handleRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_.]{1,28}[a-z0-9]$`)

	// Handles that would collide with app routes or impersonate the platform.
	reservedHandles = []string{
		"about", "admin", "api", "app", "b", "blog", "brands", "brandshake", "careers", "contact",
		"creators", "dashboard", "deals", "explore", "help", "invoices", "login", "logout", "me",
		"messages", "privacy", "pricing", "root", "security", "settings", "signup", "status",
		"support", "terms", "www",
	}
)

const (
	MaxNiches    = 5
	MaxLanguages = 6
	MaxSocials   = 5 // one per platform
	MaxRates     = 10
	MaxSamples   = 12
)

// Errors maps field paths (e.g. "handle", "rates[2].price_minor") to messages.
type Errors map[string]string

func (e Errors) add(field, msg string) {
	if _, ok := e[field]; !ok {
		e[field] = msg
	}
}

// NormalizeHandle trims, strips a leading @ and lowercases.
func NormalizeHandle(h string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(h), "@"))
}

// CheckHandle returns a user-facing problem with h, or "" if it is usable.
func CheckHandle(h string) string {
	switch {
	case !handleRe.MatchString(h):
		return "Use 3–30 lowercase letters, numbers, dots or underscores. Start and end with a letter or number."
	case strings.Contains(h, ".."):
		return "Handles can't contain two dots in a row."
	case slices.Contains(reservedHandles, h):
		return "This handle is reserved."
	}
	return ""
}

func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }

func validWebURL(raw string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || len(raw) > 300 {
		return nil, false
	}
	return u, true
}

func hostMatches(host string, allowed []string) bool {
	host = strings.ToLower(strings.TrimPrefix(host, "www."))
	host = strings.TrimPrefix(host, "m.")
	for _, a := range allowed {
		if host == a || strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}

// Normalize trims text fields and normalises list fields in place.
func (p *Profile) Normalize() {
	p.Handle = NormalizeHandle(p.Handle)
	for _, f := range []*string{&p.Headline, &p.Bio, &p.City, &p.State, &p.Website, &p.Industry} {
		*f = strings.TrimSpace(*f)
	}
	p.Niches = dedupe(p.Niches)
	p.Languages = dedupe(p.Languages)
}

func dedupe(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "" && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// Validate checks a normalised profile for the given role.
func (p *Profile) Validate(role string) Errors {
	errs := Errors{}
	if msg := CheckHandle(p.Handle); msg != "" {
		errs.add("handle", msg)
	}
	if tooLong(p.Headline, 120) {
		errs.add("headline", "Keep your headline under 120 characters.")
	}
	if tooLong(p.Bio, 1000) {
		errs.add("bio", "Keep your bio under 1,000 characters.")
	}
	if tooLong(p.City, 60) {
		errs.add("city", "City is too long.")
	}
	if tooLong(p.State, 60) {
		errs.add("state", "State is too long.")
	}
	if p.Website != "" {
		if _, ok := validWebURL(p.Website); !ok || len(p.Website) > 200 {
			errs.add("website", "Enter a full web address starting with https://")
		}
	}

	switch role {
	case "creator":
		if p.Industry != "" {
			errs.add("industry", "Only brands have an industry.")
		}
		if len(p.Niches) > MaxNiches {
			errs.add("niches", "Pick up to 5 niches.")
		}
		for _, n := range p.Niches {
			if !slices.Contains(Niches, n) {
				errs.add("niches", "Unknown niche: "+n)
			}
		}
		if len(p.Languages) > MaxLanguages {
			errs.add("languages", "Pick up to 6 languages.")
		}
		for _, l := range p.Languages {
			if !slices.Contains(Languages, l) {
				errs.add("languages", "Unknown language: "+l)
			}
		}
	case "brand":
		if len(p.Niches) > 0 || len(p.Languages) > 0 {
			errs.add("niches", "Niches and languages are for creator profiles.")
		}
		if p.Industry != "" && !slices.Contains(Industries, p.Industry) {
			errs.add("industry", "Choose an industry from the list.")
		}
		p.OpenToCollabs = false
	}
	return errs
}

func ValidateSocials(in []SocialAccount) Errors {
	errs := Errors{}
	if len(in) > MaxSocials {
		errs.add("socials", "Add at most one account per platform.")
		return errs
	}
	seen := map[string]bool{}
	for i := range in {
		s := &in[i]
		s.Platform = strings.ToLower(strings.TrimSpace(s.Platform))
		s.Handle = strings.TrimPrefix(strings.TrimSpace(s.Handle), "@")
		s.URL = strings.TrimSpace(s.URL)
		s.Verified = false // only set by platform verification, never by the client
		field := func(name string) string { return "socials[" + strconv.Itoa(i) + "]." + name }

		hosts, ok := platformHosts[s.Platform]
		switch {
		case !ok:
			errs.add(field("platform"), "Unsupported platform.")
			continue
		case seen[s.Platform]:
			errs.add(field("platform"), "You already added this platform.")
		}
		seen[s.Platform] = true
		if s.Handle == "" || tooLong(s.Handle, 60) {
			errs.add(field("handle"), "Enter your handle.")
		}
		if u, ok := validWebURL(s.URL); !ok || len(s.URL) > 200 || !hostMatches(u.Host, hosts) {
			errs.add(field("url"), "Enter the link to your "+s.Platform+" profile.")
		}
		if s.Followers < 0 || s.Followers > 1_000_000_000 {
			errs.add(field("followers"), "Enter a follower count.")
		}
	}
	return errs
}

func ValidateRates(in []Rate) Errors {
	errs := Errors{}
	if len(in) > MaxRates {
		errs.add("rates", "Add up to 10 rates.")
		return errs
	}
	for i := range in {
		r := &in[i]
		r.Deliverable = strings.TrimSpace(r.Deliverable)
		r.Note = strings.TrimSpace(r.Note)
		field := func(name string) string { return "rates[" + strconv.Itoa(i) + "]." + name }
		if r.Deliverable == "" || tooLong(r.Deliverable, 80) {
			errs.add(field("deliverable"), "Describe the deliverable, e.g. \"Instagram Reel\".")
		}
		// ₹1 crore cap keeps typos (extra zeros) out of public rate cards.
		if r.PriceMinor < 0 || r.PriceMinor > 1_00_00_000_00 {
			errs.add(field("price_minor"), "Enter a price between ₹0 and ₹1,00,00,000.")
		}
		if tooLong(r.Note, 120) {
			errs.add(field("note"), "Keep the note under 120 characters.")
		}
	}
	return errs
}

func ValidateSamples(in []Sample) Errors {
	errs := Errors{}
	if len(in) > MaxSamples {
		errs.add("samples", "Add up to 12 work samples.")
		return errs
	}
	for i := range in {
		s := &in[i]
		s.Title = strings.TrimSpace(s.Title)
		s.URL = strings.TrimSpace(s.URL)
		s.BrandName = strings.TrimSpace(s.BrandName)
		s.Platform = strings.ToLower(strings.TrimSpace(s.Platform))
		field := func(name string) string { return "samples[" + strconv.Itoa(i) + "]." + name }
		if s.Title == "" || tooLong(s.Title, 100) {
			errs.add(field("title"), "Give this sample a title.")
		}
		u, ok := validWebURL(s.URL)
		if !ok {
			errs.add(field("url"), "Enter a full link starting with https://")
		}
		if s.Platform == "" && ok {
			for p, hosts := range platformHosts {
				if hostMatches(u.Host, hosts) {
					s.Platform = p
				}
			}
		}
		if s.Platform != "" && !slices.Contains(Platforms, s.Platform) {
			s.Platform = ""
		}
		if tooLong(s.BrandName, 80) {
			errs.add(field("brand_name"), "Brand name is too long.")
		}
	}
	return errs
}
