// Package intercom implements the Intercom connector for the Permission Slip
// connector execution layer. It uses the Intercom API v2 with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
//
// Auth: Bearer token (Intercom Access Token).
// Base URL: https://api.intercom.io
package intercom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL  = "https://api.intercom.io"
	defaultTimeout  = 30 * time.Second
	maxResponseBody = 10 << 20 // 10 MB

	// credKeyAccessToken is the credential key for Intercom access tokens.
	credKeyAccessToken = "access_token"
)

// IntercomConnector owns the shared HTTP client and base URL used by all
// Intercom actions.
type IntercomConnector struct {
	client  *http.Client
	baseURL string
}

// New creates an IntercomConnector with sensible defaults.
func New() *IntercomConnector {
	return &IntercomConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates an IntercomConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *IntercomConnector {
	return &IntercomConnector{client: client, baseURL: baseURL}
}

// ID returns "intercom", matching the connectors.id in the database.
func (c *IntercomConnector) ID() string { return "intercom" }

// Actions returns the registered action handlers keyed by action_type.
func (c *IntercomConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"intercom.create_ticket":      &createTicketAction{conn: c},
		"intercom.reply_ticket":       &replyTicketAction{conn: c},
		"intercom.update_ticket":      &updateTicketAction{conn: c},
		"intercom.assign_ticket":      &assignTicketAction{conn: c},
		"intercom.search_tickets":     &searchTicketsAction{conn: c},
		"intercom.list_tags":          &listTagsAction{conn: c},
		"intercom.tag_ticket":         &tagTicketAction{conn: c},
		"intercom.create_contact":     &createContactAction{conn: c},
		"intercom.update_contact":     &updateContactAction{conn: c},
		"intercom.search_contacts":    &searchContactsAction{conn: c},
		"intercom.send_message":       &sendMessageAction{conn: c},
		"intercom.list_conversations": &listConversationsAction{conn: c},
		"intercom.create_article":     &createArticleAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token, which is required for all Intercom API calls.
func (c *IntercomConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get(credKeyAccessToken)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	return nil
}

// do is the shared request lifecycle for all Intercom actions.
func (c *IntercomConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody any) error {
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Intercom-Version", "2.11")

	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Intercom API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.CanceledError{Message: "Intercom API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Intercom API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Intercom response: %v", err)}
		}
	}
	return nil
}
