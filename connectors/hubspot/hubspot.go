// Package hubspot implements the HubSpot connector for the Permission Slip
// connector execution layer. It uses the HubSpot CRM API v3 with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
//
// Auth: HubSpot private app access tokens (API key auth).
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

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://api.hubapi.com"
	defaultTimeout = 30 * time.Second
)

// HubSpotConnector owns the shared HTTP client and base URL used by all
// HubSpot actions. Actions hold a pointer back to the connector to access
// these shared resources.
type HubSpotConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a HubSpotConnector with sensible defaults (30s timeout,
// https://api.hubapi.com base URL).
func New() *HubSpotConnector {
	return &HubSpotConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
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

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup. Actions are registered in Phase 2.
func (c *HubSpotConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "hubspot",
		Name:        "HubSpot",
		Description: "HubSpot CRM integration for contacts, deals, tickets, and notes",
		Actions:     []connectors.ManifestAction{},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "hubspot",
				AuthType:        "api_key",
				InstructionsURL: "https://developers.hubspot.com/docs/api/private-apps",
			},
		},
		Templates: []connectors.ManifestTemplate{},
	}
}

// Actions returns the registered action handlers keyed by action_type.
// Actions are added in Phase 2.
func (c *HubSpotConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key, which is required for all HubSpot API calls.
func (c *HubSpotConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
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

	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "api_key credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("HubSpot API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("HubSpot API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
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
