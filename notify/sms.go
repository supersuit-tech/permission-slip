package notify

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SMSConfig holds the Twilio credentials and sender number needed to
// deliver SMS notifications.
type SMSConfig struct {
	AccountSID string
	AuthToken  string
	FromNumber string
}

// SMSSender delivers approval notifications via Twilio's REST API.
// It is safe for concurrent use.
type SMSSender struct {
	cfg     SMSConfig
	client  *http.Client
	baseURL string // override for testing; empty uses the real Twilio API
}

// NewSMSSender creates a Sender that delivers SMS via Twilio.
func NewSMSSender(cfg SMSConfig) *SMSSender {
	return &SMSSender{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the channel identifier used for preference matching.
func (s *SMSSender) Name() string { return "sms" }

// Send delivers an SMS notification for the given approval to the recipient.
// If the recipient has no phone number, it logs and returns nil (no error)
// so other channels can still deliver.
func (s *SMSSender) Send(ctx context.Context, approval Approval, recipient Recipient) error {
	if recipient.Phone == nil || *recipient.Phone == "" {
		log.Printf("notify/sms: recipient %s has no phone number, skipping", recipient.UserID)
		return nil
	}

	body := formatSMSBody(approval)
	return s.sendViaTwilio(ctx, *recipient.Phone, body)
}

// formatSMSBody constructs a concise SMS message. We aim for ≤160 characters
// (single SMS segment) but allow overflow when the content requires it.
func formatSMSBody(a Approval) string {
	agentName := AgentDisplayName(a.AgentName, a.AgentID)
	actionSummary := SummarizeAction(a.Action)

	msg := fmt.Sprintf("[Permission Slip] %s requests: %s", agentName, actionSummary)

	if a.ApprovalURL != "" {
		return fmt.Sprintf("%s Approve: %s", msg, a.ApprovalURL)
	}

	return msg
}

// sendViaTwilio posts a message to Twilio's Messages REST API.
func (s *SMSSender) sendViaTwilio(ctx context.Context, to, body string) error {
	base := s.baseURL
	if base == "" {
		base = "https://api.twilio.com"
	}
	apiURL := fmt.Sprintf(
		"%s/2010-04-01/Accounts/%s/Messages.json",
		base, url.PathEscape(s.cfg.AccountSID),
	)

	form := url.Values{}
	form.Set("To", to)
	form.Set("From", s.cfg.FromNumber)
	form.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("notify/sms: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.cfg.AccountSID, s.cfg.AuthToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("notify/sms: twilio request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Read error body for logging (cap at 1KB to avoid memory issues).
	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("notify/sms: twilio returned %d: %s", resp.StatusCode, string(errBody))
}
