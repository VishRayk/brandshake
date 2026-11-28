// Package aiclient calls the Brandshake AI service (python/ai).
package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrUnavailable means the AI service could not produce a result. Callers
// should fall back to manual entry, never fail the user's whole flow.
var ErrUnavailable = errors.New("ai service unavailable")

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

type DraftDeliverable struct {
	Platform string  `json:"platform"`
	Format   string  `json:"format"`
	Quantity int     `json:"quantity"`
	DueDate  *string `json:"due_date"`
}

type UsageRights struct {
	Granted      bool     `json:"granted"`
	Channels     []string `json:"channels"`
	DurationDays *int     `json:"duration_days"`
	Perpetual    bool     `json:"perpetual"`
}

type Draft struct {
	CounterpartyName  *string            `json:"counterparty_name"`
	CounterpartyEmail *string            `json:"counterparty_email"`
	Title             string             `json:"title"`
	Deliverables      []DraftDeliverable `json:"deliverables"`
	PriceAmount       *float64           `json:"price_amount"`
	Currency          *string            `json:"currency"`
	Revisions         *int               `json:"revisions"`
	UsageRights       UsageRights        `json:"usage_rights"`
	ExclusivityDays   *int               `json:"exclusivity_days"`
	OpenQuestions     []string           `json:"open_questions"`
	Summary           string             `json:"summary"`
}

type RiskFlag struct {
	Severity string `json:"severity"`
	Term     string `json:"term"`
	Message  string `json:"message"`
}

type DraftResult struct {
	Draft     Draft      `json:"draft"`
	RiskFlags []RiskFlag `json:"risk_flags"`
	Provider  string     `json:"provider"`
}

// DraftDeal asks the AI service to draft a deal from pasted text.
func (c *Client) DraftDeal(ctx context.Context, text, ownerRole string) (DraftResult, error) {
	var out DraftResult
	err := c.post(ctx, "/v1/deals/draft", map[string]string{"text": text, "owner_role": ownerRole}, &out)
	return out, err
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%w: status %d: %s", ErrUnavailable, resp.StatusCode, msg)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out); err != nil {
		return fmt.Errorf("%w: decode: %v", ErrUnavailable, err)
	}
	return nil
}
