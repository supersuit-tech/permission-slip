// Package jira implements the Jira connector for the Permission Slip
// connector execution layer. It uses the Jira Cloud REST API v3 with
// basic auth (email + API token) via plain net/http.
package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const defaultTimeout = 30 * time.Second

// JiraConnector owns the shared HTTP client used by all Jira actions.
// The base URL is constructed per-request from the site credential
// (https://{site}.atlassian.net/rest/api/3/).
type JiraConnector struct {
	client  *http.Client
	baseURL string // empty for production (derived from credentials); set for tests
}

// New creates a JiraConnector with sensible defaults (30s timeout).
func New() *JiraConnector {
	return &JiraConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates a JiraConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *JiraConnector {
	return &JiraConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "jira", matching the connectors.id in the database.
func (c *JiraConnector) ID() string { return "jira" }

// Actions returns the registered action handlers keyed by action_type.
func (c *JiraConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{}
}

// ValidateCredentials checks that the provided credentials contain
// non-empty site, email, and api_token fields, which are required for
// all Jira API calls.
func (c *JiraConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	site, ok := creds.Get("site")
	if !ok || site == "" {
		return &connectors.ValidationError{Message: "missing required credential: site"}
	}
	email, ok := creds.Get("email")
	if !ok || email == "" {
		return &connectors.ValidationError{Message: "missing required credential: email"}
	}
	token, ok := creds.Get("api_token")
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_token"}
	}
	return nil
}

// apiBase returns the base URL for Jira REST API v3 calls. In test mode
// it returns the test server URL; in production it builds the URL from
// the site credential.
func (c *JiraConnector) apiBase(creds connectors.Credentials) (string, error) {
	if c.baseURL != "" {
		return c.baseURL, nil
	}
	site, ok := creds.Get("site")
	if !ok || site == "" {
		return "", &connectors.ValidationError{Message: "missing required credential: site"}
	}
	return "https://" + site + ".atlassian.net/rest/api/3", nil
}

// do is the shared request lifecycle for all Jira actions. It marshals
// reqBody as JSON, sends the request with basic auth headers, checks the
// response status, and unmarshals the response into respBody. Either
// reqBody or respBody may be nil.
func (c *JiraConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	base, err := c.apiBase(creds)
	if err != nil {
		return err
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	email, ok := creds.Get("email")
	if !ok || email == "" {
		return &connectors.ValidationError{Message: "email credential is missing or empty"}
	}
	token, ok := creds.Get("api_token")
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "api_token credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(email+":"+token)))

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Jira API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Jira API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Jira response: %v", err)}
		}
	}
	return nil
}
