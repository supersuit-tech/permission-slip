// Package cloudflare implements the Cloudflare connector for the Permission
// Slip connector execution layer. It uses the Cloudflare API v4 with API
// tokens for authentication.
package cloudflare

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
	defaultBaseURL = "https://api.cloudflare.com/client/v4"
	defaultTimeout = 30 * time.Second
	// maxResponseBytes caps the response body we'll read from Cloudflare APIs.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// CloudflareConnector owns the shared HTTP client and base URL used by all
// Cloudflare actions.
type CloudflareConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a CloudflareConnector with sensible defaults.
func New() *CloudflareConnector {
	return &CloudflareConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a CloudflareConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *CloudflareConnector {
	return &CloudflareConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "cloudflare", matching the connectors.id in the database.
func (c *CloudflareConnector) ID() string { return "cloudflare" }

// Actions returns the registered action handlers keyed by action_type.
func (c *CloudflareConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"cloudflare.list_zones":          &listZonesAction{conn: c},
		"cloudflare.get_zone":            &getZoneAction{conn: c},
		"cloudflare.list_dns_records":    &listDNSRecordsAction{conn: c},
		"cloudflare.create_dns_record":   &createDNSRecordAction{conn: c},
		"cloudflare.update_dns_record":   &updateDNSRecordAction{conn: c},
		"cloudflare.delete_dns_record":   &deleteDNSRecordAction{conn: c},
		"cloudflare.list_tunnels":        &listTunnelsAction{conn: c},
		"cloudflare.create_tunnel":       &createTunnelAction{conn: c},
		"cloudflare.delete_tunnel":       &deleteTunnelAction{conn: c},
		"cloudflare.get_tunnel":          &getTunnelAction{conn: c},
		"cloudflare.list_tunnel_configs":  &listTunnelConfigsAction{conn: c},
		"cloudflare.update_tunnel_config": &updateTunnelConfigAction{conn: c},
		"cloudflare.check_domain":        &checkDomainAction{conn: c},
		"cloudflare.register_domain":     &registerDomainAction{conn: c},
		"cloudflare.purge_cache":         &purgeCacheAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key (API token).
func (c *CloudflareConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
}

// execRequest is the shared HTTP lifecycle: sets auth, sends the request,
// reads the response body, and maps HTTP errors to typed connector errors.
func (c *CloudflareConnector) execRequest(ctx context.Context, creds connectors.Credentials, req *http.Request) ([]byte, error) {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return nil, &connectors.ValidationError{Message: "api_key credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("Cloudflare API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return nil, &connectors.CanceledError{Message: "Cloudflare API request canceled"}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("Cloudflare API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return nil, err
	}

	return respBytes, nil
}

// doJSON sends a JSON request and unmarshals the result field from the
// Cloudflare API envelope into respBody.
func (c *CloudflareConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, path string, reqBody any, respBody any) error {
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

	respBytes, err := c.execRequest(ctx, creds, req)
	if err != nil {
		return err
	}

	if respBody != nil {
		var envelope cfEnvelope
		if err := json.Unmarshal(respBytes, &envelope); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Cloudflare response: %v", err)}
		}
		if err := json.Unmarshal(envelope.Result, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Cloudflare result: %v", err)}
		}
	}
	return nil
}

// doGet is a convenience for GET requests (no body).
func (c *CloudflareConnector) doGet(ctx context.Context, creds connectors.Credentials, path string, respBody any) error {
	return c.doJSON(ctx, creds, http.MethodGet, path, nil, respBody)
}
