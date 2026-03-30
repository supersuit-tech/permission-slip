// Package hubspot implements the HubSpot connector for the Permission Slip
// connector execution layer. It uses the HubSpot CRM API v3 with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
//
// Auth: OAuth2 access tokens (preferred) or HubSpot private app access tokens (API key).
// Base URL: https://api.hubapi.com
package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultBaseURL  = "https://api.hubapi.com"
	defaultTimeout  = 30 * time.Second
	maxResponseBody = 10 << 20 // 10 MB — guard against unexpectedly large responses

	// credKeyAPIKey is the credential key for HubSpot private app access tokens.
	credKeyAPIKey = "api_key"

	// credKeyAccessToken is the credential key for OAuth2 access tokens,
	// set by the OAuth credential resolution path.
	credKeyAccessToken = "access_token"
)

// HubSpotConnector owns the shared HTTP client and base URL used by all
// HubSpot actions. Actions hold a pointer back to the connector to access
// these shared resources.
type HubSpotConnector struct {
	client  *http.Client
	baseURL string
	nowFunc func() time.Time // defaults to time.Now; override in tests for deterministic timestamps
}

// New creates a HubSpotConnector with sensible defaults (30s timeout,
// https://api.hubapi.com base URL).
func New() *HubSpotConnector {
	return &HubSpotConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// now returns the current time, using the connector's nowFunc if set.
func (c *HubSpotConnector) now() time.Time {
	if c.nowFunc != nil {
		return c.nowFunc()
	}
	return time.Now()
}

// newForTest creates a HubSpotConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *HubSpotConnector {
	return &HubSpotConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "hubspot", matching the connectors.id in the database.
func (c *HubSpotConnector) ID() string { return "hubspot" }

// Actions returns the registered action handlers keyed by action_type.
func (c *HubSpotConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"hubspot.create_contact":        &createContactAction{conn: c},
		"hubspot.update_contact":        &updateContactAction{conn: c},
		"hubspot.list_contacts":         &listContactsAction{conn: c},
		"hubspot.get_contact":           &getContactAction{conn: c},
		"hubspot.delete_contact":        &deleteContactAction{conn: c},
		"hubspot.create_deal":           &createDealAction{conn: c},
		"hubspot.delete_deal":           &deleteDealAction{conn: c},
		"hubspot.create_ticket":         &createTicketAction{conn: c},
		"hubspot.add_note":              &addNoteAction{conn: c},
		"hubspot.search":                &searchAction{conn: c},
		"hubspot.list_deals":            &listDealsAction{conn: c},
		"hubspot.update_deal_stage":     &updateDealStageAction{conn: c},
		"hubspot.enroll_in_workflow":    &enrollInWorkflowAction{conn: c},
		"hubspot.create_email_campaign": &createEmailCampaignAction{conn: c},
		"hubspot.get_analytics":         &getAnalyticsAction{conn: c},
		"hubspot.create_company":        &createCompanyAction{conn: c},
		"hubspot.update_company":        &updateCompanyAction{conn: c},
		"hubspot.list_pipelines":        &listPipelinesAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access token. Accepts either an OAuth2 access_token or a
// private app api_key — both are used as Bearer tokens against the HubSpot API.
func (c *HubSpotConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	if token, ok := creds.Get(credKeyAccessToken); ok && token != "" {
		return nil
	}
	if key, ok := creds.Get(credKeyAPIKey); ok && key != "" {
		return nil
	}
	return &connectors.ValidationError{Message: "missing required credential: api_key or access_token (OAuth)"}
}

// maxAssociations limits the number of associations per request to prevent
// excessive API calls against HubSpot's rate limit (100 req / 10s).
const maxAssociations = 50

// isValidHubSpotID checks that an ID string is safe to interpolate into a
// URL path. HubSpot object IDs are always numeric strings. Rejecting
// non-numeric values prevents path traversal (e.g., "../../admin/endpoint")
// from redirecting requests to unintended API endpoints.
func isValidHubSpotID(id string) bool {
	if id == "" {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// do is the shared request lifecycle for all HubSpot actions. It marshals
// reqBody as JSON, sends the request with auth headers, checks the response
// status, and unmarshals the response into respBody. Either reqBody or
// respBody may be nil.
func (c *HubSpotConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody any) error {
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

	// Prefer OAuth access_token; fall back to private app api_key.
	token, _ := creds.Get(credKeyAccessToken)
	if token == "" {
		token, _ = creds.Get(credKeyAPIKey)
	}
	if token == "" {
		return &connectors.ValidationError{Message: "api_key or access_token credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("HubSpot API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("HubSpot API request failed: %v", err)}
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
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing HubSpot response: %v", err)}
		}
	}
	return nil
}
