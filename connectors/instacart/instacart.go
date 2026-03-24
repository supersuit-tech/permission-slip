// Package instacart implements the Instacart connector for the Permission Slip
// connector execution layer. It uses the Instacart Developer Platform REST API
// with plain net/http (no third-party SDK).
//
// See https://docs.instacart.com/developer_platform_api/
package instacart

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL    = "https://connect.instacart.com"
	defaultTimeout    = 30 * time.Second
	defaultRetryAfter = 30 * time.Second

	credKeyAPIKey  = "api_key"
	credKeyBaseURL = "base_url"

	// maxResponseBytes caps response size to avoid OOM from misbehaving upstream.
	maxResponseBytes = 2 * 1024 * 1024 // 2 MiB
)

// allowedBaseHosts is the allowlist for credential base_url. Only these hosts
// are accepted so a compromised or mistyped credential cannot be used to
// redirect API calls to an arbitrary origin (SSRF-style abuse).
var allowedBaseHosts = map[string]struct{}{
	"connect.instacart.com":       {},
	"connect.dev.instacart.tools": {},
}

// InstacartConnector owns the shared HTTP client used by all Instacart actions.
type InstacartConnector struct {
	client  *http.Client
	baseURL string
}

// New creates an InstacartConnector with sensible defaults.
func New() *InstacartConnector {
	return &InstacartConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates an InstacartConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *InstacartConnector {
	return &InstacartConnector{
		client:  client,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// ID returns "instacart", matching the connectors.id in the database.
func (c *InstacartConnector) ID() string { return "instacart" }

// Actions returns the registered action handlers keyed by action_type.
func (c *InstacartConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"instacart.create_products_link": &createProductsLinkAction{conn: c},
	}
}

// ValidateCredentials checks api_key and optional base_url (sandbox vs production).
func (c *InstacartConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key (Instacart Developer Platform API key from https://www.instacart.com/developer)"}
	}
	if len(key) < 8 {
		return &connectors.ValidationError{Message: "api_key looks invalid (too short); use the full key from the Instacart developer portal"}
	}
	if raw, ok := creds.Get(credKeyBaseURL); ok && raw != "" {
		if err := validateBaseURL(raw); err != nil {
			return err
		}
	}
	return nil
}

func validateBaseURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return &connectors.ValidationError{Message: "base_url must be a valid https URL"}
	}
	host := strings.ToLower(u.Hostname())
	if _, ok := allowedBaseHosts[host]; !ok {
		return &connectors.ValidationError{Message: "base_url must be https://connect.instacart.com (production) or https://connect.dev.instacart.tools (sandbox)"}
	}
	return nil
}

func (c *InstacartConnector) resolveBaseURL(creds connectors.Credentials) (string, error) {
	if raw, ok := creds.Get(credKeyBaseURL); ok && raw != "" {
		if err := validateBaseURL(raw); err != nil {
			return "", err
		}
		return strings.TrimSuffix(strings.TrimSpace(raw), "/"), nil
	}
	return strings.TrimSuffix(c.baseURL, "/"), nil
}

// do sends a JSON request with Bearer api_key auth.
func (c *InstacartConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, dest any) error {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "api_key credential is missing or empty (add your Instacart Developer Platform API key)"}
	}

	base, err := c.resolveBaseURL(creds)
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

	reqURL := base + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Instacart API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Instacart API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Instacart API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if dest != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("parsing Instacart API response: %v", err),
			}
		}
	}
	return nil
}
