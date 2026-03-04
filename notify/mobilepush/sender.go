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

// defaultExpoAPIURL is the production Expo Push Service endpoint.
const defaultExpoAPIURL = "https://exp.host/--/api/v2/push/send"

// maxBatchSize is the maximum number of messages per Expo Push API request.
// The API supports up to 100 messages per batch.
const maxBatchSize = 100

// TokenStore abstracts the database operations the sender needs: listing
// tokens for a user and deleting invalid tokens. db.DBTX satisfies this
// indirectly via the db package functions — see dbTokenStore.
type TokenStore interface {
	ListTokens(ctx context.Context, userID string) ([]db.ExpoPushToken, error)
	DeleteToken(ctx context.Context, token string) error
}

// dbTokenStore adapts a db.DBTX to the TokenStore interface.
type dbTokenStore struct {
	dbtx db.DBTX
}

func (s dbTokenStore) ListTokens(ctx context.Context, userID string) ([]db.ExpoPushToken, error) {
	return db.ListExpoPushTokensByUserID(ctx, s.dbtx, userID)
}

func (s dbTokenStore) DeleteToken(ctx context.Context, token string) error {
	return db.DeleteExpoPushTokenByToken(ctx, s.dbtx, token)
}

// Sender implements notify.Sender for the mobile push channel. It sends
// push messages via the Expo Push Service to all registered Expo push
// tokens for a recipient, batching multiple tokens into a single API call.
type Sender struct {
	store       TokenStore
	accessToken string // optional Expo access token for authenticated requests
	apiURL      string // Expo Push API endpoint
	httpClient  *http.Client
}

// New creates a mobile push sender. The accessToken is optional — when
// non-empty it is sent as a Bearer token in the Authorization header
// for higher rate limits.
func New(d db.DBTX, accessToken string) *Sender {
	return &Sender{
		store:       dbTokenStore{dbtx: d},
		accessToken: accessToken,
		apiURL:      defaultExpoAPIURL,
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
// Expo push tokens. Messages are batched into a single API call (up to 100
// per request). Invalid tokens (DeviceNotRegistered) are automatically
// cleaned up. Partial failures are logged but do not cause the overall
// Send to return an error — best-effort delivery across all devices.
func (s *Sender) Send(ctx context.Context, approval notify.Approval, recipient notify.Recipient) error {
	tokens, err := s.store.ListTokens(ctx, recipient.UserID)
	if err != nil {
		return fmt.Errorf("list expo push tokens: %w", err)
	}
	if len(tokens) == 0 {
		return nil
	}

	content := buildMessage(approval)

	// Build messages for all tokens.
	messages := make([]expoMessage, len(tokens))
	tokenStrings := make([]string, len(tokens))
	for i, tok := range tokens {
		messages[i] = expoMessage{
			To:         tok.Token,
			Title:      content.Title,
			Body:       content.Body,
			Sound:      "default",
			Priority:   "high",
			CategoryID: "approval",
			Data: expoMessageData{
				URL:        content.URL,
				ApprovalID: content.ApprovalID,
			},
		}
		tokenStrings[i] = tok.Token
	}

	// Send in batches of maxBatchSize.
	var lastErr error
	for i := 0; i < len(messages); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(messages) {
			end = len(messages)
		}
		if err := s.sendBatch(ctx, messages[i:end], tokenStrings[i:end]); err != nil {
			log.Printf("mobilepush: batch send failed: %v", err)
			lastErr = err
		}
	}

	return lastErr
}

// sendBatch sends a batch of push messages to the Expo Push Service in a
// single HTTP request and processes the per-ticket results. Invalid tokens
// are automatically removed.
func (s *Sender) sendBatch(ctx context.Context, messages []expoMessage, tokens []string) error {
	payload, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("marshal expo push messages: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Truncate body in error message to avoid logging potentially sensitive data.
		errBody := string(body)
		if len(errBody) > 200 {
			errBody = errBody[:200] + "..."
		}
		return fmt.Errorf("expo push API returned %d: %s", resp.StatusCode, errBody)
	}

	var apiResp expoAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("unmarshal expo push API response: %w", err)
	}

	// Check for top-level API errors (invalid request, etc.)
	if len(apiResp.Errors) > 0 {
		return fmt.Errorf("expo push API error: %s: %s", apiResp.Errors[0].Code, apiResp.Errors[0].Message)
	}

	// Process per-ticket results — each ticket corresponds to the message
	// at the same index. Best-effort: log errors but continue processing.
	var lastErr error
	for i, ticket := range apiResp.Data {
		if ticket.Status == "ok" {
			continue
		}

		if i < len(tokens) && ticket.Details != nil && ticket.Details.Error == "DeviceNotRegistered" {
			log.Printf("mobilepush: token ending %q is no longer registered, removing", truncateTokenForLog(tokens[i]))
			if err := s.store.DeleteToken(ctx, tokens[i]); err != nil {
				log.Printf("mobilepush: failed to delete invalid token: %v", err)
			}
			continue
		}

		log.Printf("mobilepush: ticket error for token index %d: %s", i, ticket.Message)
		lastErr = fmt.Errorf("expo push ticket error: %s", ticket.Message)
	}

	return lastErr
}

// buildMessage constructs the push notification content from the approval data.
// Delegates to notify.BuildPushContent for consistent messaging across channels.
func buildMessage(approval notify.Approval) notify.PushContent {
	return notify.BuildPushContent(approval)
}

// truncateTokenForLog returns the last 8 characters of a token for logging,
// avoiding exposure of the full token string in log output.
func truncateTokenForLog(token string) string {
	if len(token) <= 8 {
		return token
	}
	return "..." + token[len(token)-8:]
}
