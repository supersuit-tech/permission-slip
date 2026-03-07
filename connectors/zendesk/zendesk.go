// Package zendesk implements the Zendesk connector for the Permission Slip
// connector execution layer. It uses the Zendesk Support API v2 with plain
// net/http (no third-party SDK) to keep the dependency footprint minimal.
//
// Auth: API token (email/token pair) via Basic auth, or OAuth 2.0 bearer token.
// Base URL: https://{subdomain}.zendesk.com/api/v2/
package zendesk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultTimeout  = 30 * time.Second
	maxResponseBody = 10 << 20 // 10 MB

	// Credential keys — keep in sync with ValidateCredentials and do().
	credKeySubdomain = "subdomain"
	credKeyAPIToken  = "api_token"
	credKeyEmail     = "email"
)

// subdomainPattern validates Zendesk subdomains: alphanumeric with hyphens.
var subdomainPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,62}$`)

// ZendeskConnector owns the shared HTTP client used by all Zendesk actions.
type ZendeskConnector struct {
	client          *http.Client
	baseURLOverride string // empty in production; set in tests to point at httptest.Server
}

// New creates a ZendeskConnector with sensible defaults.
func New() *ZendeskConnector {
	return &ZendeskConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates a ZendeskConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *ZendeskConnector {
	return &ZendeskConnector{client: client, baseURLOverride: baseURL}
}

// ID returns "zendesk", matching the connectors.id in the database.
func (c *ZendeskConnector) ID() string { return "zendesk" }

// Actions returns the registered action handlers keyed by action_type.
func (c *ZendeskConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"zendesk.create_ticket":  &createTicketAction{conn: c},
		"zendesk.reply_ticket":   &replyTicketAction{conn: c},
		"zendesk.update_ticket":  &updateTicketAction{conn: c},
		"zendesk.assign_ticket":  &assignTicketAction{conn: c},
		"zendesk.merge_tickets":  &mergeTicketsAction{conn: c},
		"zendesk.search_tickets": &searchTicketsAction{conn: c},
		"zendesk.list_tags":      &listTagsAction{conn: c},
		"zendesk.update_tags":    &updateTagsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain subdomain,
// email, and api_token which are required for all Zendesk API calls.
func (c *ZendeskConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	subdomain, ok := creds.Get(credKeySubdomain)
	if !ok || subdomain == "" {
		return &connectors.ValidationError{Message: "missing required credential: subdomain"}
	}
	if !subdomainPattern.MatchString(subdomain) {
		return &connectors.ValidationError{Message: "invalid subdomain format: must be alphanumeric with hyphens"}
	}
	email, ok := creds.Get(credKeyEmail)
	if !ok || email == "" {
		return &connectors.ValidationError{Message: "missing required credential: email"}
	}
	token, ok := creds.Get(credKeyAPIToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_token"}
	}
	return nil
}

// baseURL builds the Zendesk API base URL from the subdomain credential.
func baseURL(creds connectors.Credentials) string {
	subdomain, _ := creds.Get(credKeySubdomain)
	return fmt.Sprintf("https://%s.zendesk.com/api/v2", subdomain)
}

// do is the shared request lifecycle for all Zendesk actions. It sends
// the request with auth headers, checks the response, and unmarshals.
func (c *ZendeskConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody any) error {
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	base := c.baseURLOverride
	if base == "" {
		base = baseURL(creds)
	}
	url := base + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Zendesk API token auth: {email}/token:{api_token}
	email, _ := creds.Get(credKeyEmail)
	token, _ := creds.Get(credKeyAPIToken)
	req.SetBasicAuth(email+"/token", token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Zendesk API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Zendesk API request failed: %v", err)}
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
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Zendesk response: %v", err)}
		}
	}
	return nil
}
