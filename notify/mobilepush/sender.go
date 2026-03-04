package mobilepush

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/notify"
)

// ExpoAPIURL is the Expo Push Service endpoint. Overridable for testing.
var ExpoAPIURL = "https://exp.host/--/api/v2/push/send"

// Sender implements notify.Sender for the mobile push channel. It sends
// push messages via the Expo Push Service to all registered Expo push
// tokens for a recipient.
type Sender struct {
	db          db.DBTX
	accessToken string // optional Expo access token for authenticated requests
	httpClient  *http.Client
}

// New creates a mobile push sender. The accessToken is optional — when
// non-empty it is sent as a Bearer token in the Authorization header.
func New(d db.DBTX, accessToken string) *Sender {
	return &Sender{
		db:          d,
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns "mobile-push" — matches the notification_preferences.channel value.
func (s *Sender) Name() string { return "mobile-push" }

// expoMessage is a single push notification message for the Expo Push API.
type expoMessage struct {
	To         string `json:"to"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Sound      string `json:"sound,omitempty"`
	Priority   string `json:"priority,omitempty"`
	CategoryID string `json:"categoryId,omitempty"`
	Data       any    `json:"data,omitempty"`
}

// expoMessageData is the custom data payload attached to each push message.
type expoMessageData struct {
	URL        string `json:"url"`
	ApprovalID string `json:"approval_id"`
}

// expoTicketResponse represents a single push ticket from the Expo API.
type expoTicketResponse struct {
	Status  string `json:"status"` // "ok" or "error"
	ID      string `json:"id"`     // ticket ID (only when status=ok)
	Message string `json:"message"`
	Details *struct {
		Error string `json:"error"` // e.g. "DeviceNotRegistered"
	} `json:"details"`
}

// expoAPIResponse is the top-level response from POST /push/send.
type expoAPIResponse struct {
	Data   []expoTicketResponse `json:"data"`
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// Send delivers a push notification to all of the recipient's registered
// Expo push tokens. Invalid tokens (DeviceNotRegistered) are automatically
// cleaned up. Partial failures are logged but do not cause the overall
// Send to return an error — best-effort delivery across all devices.
func (s *Sender) Send(ctx context.Context, approval notify.Approval, recipient notify.Recipient) error {
	tokens, err := db.ListExpoPushTokensByUserID(ctx, s.db, recipient.UserID)
	if err != nil {
		return fmt.Errorf("list expo push tokens: %w", err)
	}
	if len(tokens) == 0 {
		return nil
	}

	msg := buildMessage(approval)

	var lastErr error
	for _, tok := range tokens {
		expoMsg := expoMessage{
			To:         tok.Token,
			Title:      msg.Title,
			Body:       msg.Body,
			Sound:      "default",
			Priority:   "high",
			CategoryID: "approval",
			Data: expoMessageData{
				URL:        msg.URL,
				ApprovalID: msg.ApprovalID,
			},
		}

		if err := s.sendOne(ctx, expoMsg, tok.Token); err != nil {
			log.Printf("mobilepush: failed to send to token %d: %v", tok.ID, err)
			lastErr = err
		}
	}

	return lastErr
}

// sendOne sends a single push message to the Expo Push Service and handles
// the response. It automatically cleans up tokens that are no longer valid.
func (s *Sender) sendOne(ctx context.Context, msg expoMessage, token string) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal expo push message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ExpoAPIURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.accessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("expo push API request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB limit
	if err != nil {
		return fmt.Errorf("read expo push API response: %w", err)
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf("expo push API returned %d: %s", resp.StatusCode, body)
	}

	var apiResp expoAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("unmarshal expo push API response: %w", err)
	}

	// Check for top-level API errors (invalid request, etc.)
	if len(apiResp.Errors) > 0 {
		return fmt.Errorf("expo push API error: %s: %s", apiResp.Errors[0].Code, apiResp.Errors[0].Message)
	}

	// Process per-ticket results
	for _, ticket := range apiResp.Data {
		if ticket.Status == "ok" {
			continue
		}

		if ticket.Details != nil && ticket.Details.Error == "DeviceNotRegistered" {
			log.Printf("mobilepush: token %q is no longer registered, removing", token)
			if err := db.DeleteExpoPushTokenByToken(ctx, s.db, token); err != nil {
				log.Printf("mobilepush: failed to delete invalid token: %v", err)
			}
			return nil // not an error worth propagating
		}

		return fmt.Errorf("expo push ticket error: %s", ticket.Message)
	}

	return nil
}

// buildMessage constructs the push notification content from the approval data.
// Delegates to notify.BuildPushContent for consistent messaging across channels.
func buildMessage(approval notify.Approval) notify.PushContent {
	return notify.BuildPushContent(approval)
}
