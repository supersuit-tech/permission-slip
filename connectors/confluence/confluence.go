// Package confluence implements the Confluence connector for the Permission Slip
// connector execution layer. It uses the Confluence Cloud REST API v2 with
// basic auth (email + API token) via plain net/http.
package confluence

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validSite matches Atlassian site subdomains: alphanumeric with hyphens.
// Prevents SSRF by ensuring the site value cannot contain path separators,
// fragments, or other characters that would alter the target host.
var validSite = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)

const (
	defaultTimeout = 30 * time.Second
	// maxResponseBody is the maximum response body size (10 MB) to prevent OOM
	// from malicious or buggy API responses.
	maxResponseBody = 10 << 20
)

// ConfluenceConnector owns the shared HTTP client used by all Confluence actions.
// The base URL is constructed per-request from the site credential
// (https://{site}.atlassian.net/wiki/api/v2/).
type ConfluenceConnector struct {
	client  *http.Client
	baseURL string // empty for production (derived from credentials); set for tests
}

// New creates a ConfluenceConnector with sensible defaults (30s timeout).
func New() *ConfluenceConnector {
	return &ConfluenceConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates a ConfluenceConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *ConfluenceConnector {
	return &ConfluenceConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "confluence", matching the connectors.id in the database.
func (c *ConfluenceConnector) ID() string { return "confluence" }

// Actions returns the registered action handlers keyed by action_type.
func (c *ConfluenceConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"confluence.create_page":     &createPageAction{conn: c},
		"confluence.update_page":     &updatePageAction{conn: c},
		"confluence.get_page":        &getPageAction{conn: c},
		"confluence.search":          &searchAction{conn: c},
		"confluence.add_comment":     &addCommentAction{conn: c},
		"confluence.list_spaces":     &listSpacesAction{conn: c},
		"confluence.list_pages":      &listPagesAction{conn: c},
		"confluence.delete_page":     &deletePageAction{conn: c},
		"confluence.get_attachments": &getAttachmentsAction{conn: c},
		"confluence.add_attachment":  &addAttachmentAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain
// non-empty site, email, and api_token fields, which are required for
// all Confluence API calls.
func (c *ConfluenceConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
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

// apiBase returns the base URL for Confluence REST API v2 calls. In test mode
// it returns the test server URL; in production it builds the URL from
// the site credential.
func (c *ConfluenceConnector) apiBase(creds connectors.Credentials) (string, error) {
	if c.baseURL != "" {
		return c.baseURL, nil
	}
	site, ok := creds.Get("site")
	if !ok || site == "" {
		return "", &connectors.ValidationError{Message: "missing required credential: site"}
	}
	if !validSite.MatchString(site) {
		return "", &connectors.ValidationError{
			Message: "invalid site credential: must contain only alphanumeric characters and hyphens (e.g. \"my-company\")",
		}
	}
	return "https://" + site + ".atlassian.net/wiki/api/v2", nil
}

// do is the shared request lifecycle for all Confluence actions. It marshals
// reqBody as JSON, sends the request with basic auth headers, checks the
// response status, and unmarshals the response into respBody. Either
// reqBody or respBody may be nil.
func (c *ConfluenceConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
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
		if connectors.IsTimeout(err) || errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Confluence API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Confluence API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Confluence response: %v", err)}
		}
	}
	return nil
}
