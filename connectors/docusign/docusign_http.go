package docusign

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

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// resolveBaseURL returns the base URL to use for API requests. If the user
// has configured a base_url credential (for production or a specific region),
// that takes precedence over the connector's default (demo sandbox).
func (c *DocuSignConnector) resolveBaseURL(creds connectors.Credentials) (string, error) {
	if customURL, ok := creds.Get(credKeyBaseURL); ok && customURL != "" {
		if err := validateBaseURL(customURL); err != nil {
			return "", err
		}
		return strings.TrimRight(customURL, "/"), nil
	}
	return c.baseURL, nil
}

// validateBaseURL checks that a user-provided base URL points to a legitimate
// DocuSign API endpoint. This prevents SSRF by ensuring we only send credentials
// to DocuSign-owned hosts.
func validateBaseURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid base_url: %v", err),
		}
	}
	if parsed.Scheme != "https" {
		return &connectors.ValidationError{
			Message: "base_url must use HTTPS",
		}
	}
	host := strings.ToLower(parsed.Hostname())
	if !strings.HasSuffix(host, ".docusign.net") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("base_url host %q is not a DocuSign domain (must end with .docusign.net)", host),
		}
	}
	return nil
}

// doJSON sends a JSON request to the DocuSign API and unmarshals the response.
func (c *DocuSignConnector) doJSON(ctx context.Context, method, path string, creds connectors.Credentials, body any, dest any) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	baseURL, err := c.resolveBaseURL(creds)
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("DocuSign API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.CanceledError{Message: "DocuSign API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("DocuSign API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "DocuSign API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return &connectors.AuthError{Message: "DocuSign auth error: invalid or expired access token"}
	}

	if resp.StatusCode == http.StatusNotFound {
		if apiErr, ok := tryParseDocuSignError(respBody); ok {
			return mapDocuSignError(resp.StatusCode, apiErr)
		}
		return &connectors.ValidationError{
			Message: "DocuSign: resource not found (HTTP 404) — verify the envelope_id or document_id is correct",
		}
	}

	if resp.StatusCode >= 400 {
		if apiErr, ok := tryParseDocuSignError(respBody); ok {
			return mapDocuSignError(resp.StatusCode, apiErr)
		}
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("DocuSign API error (HTTP %d): %s", resp.StatusCode, truncateBody(respBody)),
		}
	}

	if dest != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode DocuSign API response",
			}
		}
	}

	return nil
}

// doRaw sends a request and returns the raw response body (for binary content like PDFs).
func (c *DocuSignConnector) doRaw(ctx context.Context, method, path string, creds connectors.Credentials) ([]byte, error) {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return nil, &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	baseURL, err := c.resolveBaseURL(creds)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/pdf")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("DocuSign API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return nil, &connectors.CanceledError{Message: "DocuSign API request canceled"}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("DocuSign API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return nil, &connectors.RateLimitError{
			Message:    "DocuSign API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &connectors.AuthError{Message: "DocuSign auth error: invalid or expired access token"}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &connectors.ValidationError{
			Message: "DocuSign: resource not found (HTTP 404) — verify the envelope_id or document_id is correct",
		}
	}

	if resp.StatusCode >= 400 {
		return nil, &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("DocuSign API error (HTTP %d): %s", resp.StatusCode, truncateBody(body)),
		}
	}

	return body, nil
}
