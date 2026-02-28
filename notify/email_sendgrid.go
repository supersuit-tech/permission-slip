package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SendGridSender sends email notifications via the SendGrid v3 HTTP API.
// It uses /v3/mail/send (not SMTP relay) for simplicity and reliability.
type SendGridSender struct {
	apiKey  string
	from    string
	client  *http.Client
	baseURL string // override for testing; empty uses the default SendGrid URL
}

// NewSendGridSender creates a SendGridSender with the given API key and from address.
func NewSendGridSender(apiKey, from string) *SendGridSender {
	return &SendGridSender{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *SendGridSender) Name() string { return "email" }

func (s *SendGridSender) Send(ctx context.Context, approval Approval, recipient Recipient) error {
	url := defaultSendGridURL
	if s.baseURL != "" {
		url = s.baseURL
	}
	return s.sendToURL(ctx, approval, recipient, url)
}

func (s *SendGridSender) sendToURL(ctx context.Context, approval Approval, recipient Recipient, url string) error {
	if recipient.Email == nil || *recipient.Email == "" {
		return fmt.Errorf("recipient %s has no email address", recipient.UserID)
	}

	subject := buildEmailSubject(approval)
	plainBody := buildEmailPlainBody(approval)
	htmlBody := buildEmailHTMLBody(approval)

	payload := sendGridPayload{
		Personalizations: []sendGridPersonalization{{
			To: []sendGridAddress{{Email: *recipient.Email}},
		}},
		From:    sendGridAddress{Email: s.from},
		Subject: subject,
		Content: []sendGridContent{
			{Type: "text/plain", Value: plainBody},
			{Type: "text/html", Value: htmlBody},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sendgrid payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create sendgrid request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sendgrid request failed: %w", err)
	}
	defer resp.Body.Close()

	// SendGrid returns 202 Accepted on success.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Read a limited portion of the error body for diagnostics.
	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("sendgrid returned %d: %s", resp.StatusCode, string(errBody))
}

const defaultSendGridURL = "https://api.sendgrid.com/v3/mail/send"

// SendGrid v3 API request types — only the fields we need.

type sendGridPayload struct {
	Personalizations []sendGridPersonalization `json:"personalizations"`
	From             sendGridAddress           `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []sendGridContent         `json:"content"`
}

type sendGridPersonalization struct {
	To []sendGridAddress `json:"to"`
}

type sendGridAddress struct {
	Email string `json:"email"`
}

type sendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
