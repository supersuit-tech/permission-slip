// Package linkedin implements the LinkedIn connector for the Permission Slip
// connector execution layer. It uses the LinkedIn Marketing & Community
// Management APIs with plain net/http and OAuth 2.0 access tokens provided
// by the platform.
package linkedin

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
	defaultV2BaseURL   = "https://api.linkedin.com/v2"
	defaultRestBaseURL = "https://api.linkedin.com/rest"
	defaultTimeout     = 30 * time.Second
	credKeyAccessToken = "access_token"
	linkedInVersion    = "202401"

	// defaultRetryAfter is used when the LinkedIn API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes caps how much data we read from the LinkedIn API to
	// prevent OOM from unexpectedly large responses.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// LinkedInConnector owns the shared HTTP client and base URLs used by all
// LinkedIn actions. Actions hold a pointer back to the connector to access
// these shared resources.
type LinkedInConnector struct {
	client      *http.Client
	v2BaseURL   string // legacy /v2 endpoints (e.g. userinfo)
	restBaseURL string // versioned /rest endpoints (e.g. posts)
}

// New creates a LinkedInConnector with sensible defaults.
func New() *LinkedInConnector {
	return &LinkedInConnector{
		client:      &http.Client{Timeout: defaultTimeout},
		v2BaseURL:   defaultV2BaseURL,
		restBaseURL: defaultRestBaseURL,
	}
}

// newForTest creates a LinkedInConnector that points at a test server.
func newForTest(client *http.Client, v2BaseURL, restBaseURL string) *LinkedInConnector {
	return &LinkedInConnector{
		client:      client,
		v2BaseURL:   v2BaseURL,
		restBaseURL: restBaseURL,
	}
}

// ID returns "linkedin", matching the connectors.id in the database.
func (c *LinkedInConnector) ID() string { return "linkedin" }

// Actions returns the registered action handlers keyed by action_type.
func (c *LinkedInConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"linkedin.create_post":         &createPostAction{conn: c},
		"linkedin.delete_post":         &deletePostAction{conn: c},
		"linkedin.add_comment":         &addCommentAction{conn: c},
		"linkedin.get_profile":         &getProfileAction{conn: c},
		"linkedin.get_post_analytics":  &getPostAnalyticsAction{conn: c},
		"linkedin.create_company_post": &createCompanyPostAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token. Since tokens are provided by the platform's
// OAuth infrastructure, format validation is minimal.
func (c *LinkedInConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	return nil
}

// linkedInErrorResponse is the LinkedIn API error envelope.
type linkedInErrorResponse struct {
	Status           int    `json:"status"`
	ServiceErrorCode int    `json:"serviceErrorCode,omitempty"`
	Message          string `json:"message"`
}

// do is the shared request lifecycle for all LinkedIn actions. It sends the
// request with Bearer token auth, handles rate limiting and timeouts, and
// unmarshals the response into dest. reqBody may be nil for GET/DELETE requests.
// useRestAPI controls whether the LinkedIn-Version header is set (required for
// /rest/ endpoints).
func (c *LinkedInConnector) do(ctx context.Context, creds connectors.Credentials, method, url string, reqBody, dest any, useRestAPI bool) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if useRestAPI {
		req.Header.Set("LinkedIn-Version", linkedInVersion)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return wrapHTTPError(err)
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
				Message:    "failed to decode LinkedIn API response",
			}
		}
	}

	return nil
}

// doWithHeaders is like do but also returns the response headers. This is
// needed for POST endpoints where LinkedIn returns the created resource ID
// in the x-restli-id header.
func (c *LinkedInConnector) doWithHeaders(ctx context.Context, creds connectors.Credentials, method, url string, reqBody, dest any, useRestAPI bool) (http.Header, error) {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return nil, &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if useRestAPI {
		req.Header.Set("LinkedIn-Version", linkedInVersion)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, wrapHTTPError(err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return nil, err
	}

	if dest != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, dest); err != nil {
			return nil, &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode LinkedIn API response",
			}
		}
	}

	return resp.Header, nil
}

// wrapHTTPError converts HTTP client errors into typed connector errors.
func wrapHTTPError(err error) error {
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("LinkedIn API request timed out: %v", err)}
	}
	if errors.Is(err, context.Canceled) {
		return &connectors.TimeoutError{Message: "LinkedIn API request canceled"}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("LinkedIn API request failed: %v", err)}
}

// checkResponse maps HTTP status codes to typed connector errors.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract a LinkedIn API error message and service error code.
	var apiErr linkedInErrorResponse
	msg := "LinkedIn API error"
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		msg = apiErr.Message
		if apiErr.ServiceErrorCode != 0 {
			msg = fmt.Sprintf("%s (service error %d)", msg, apiErr.ServiceErrorCode)
		}
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("LinkedIn auth error: %s", msg)}
	case http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("LinkedIn permission denied: %s", msg)}
	case http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("LinkedIn API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case http.StatusUnprocessableEntity:
		return &connectors.ValidationError{Message: fmt.Sprintf("LinkedIn validation error: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("LinkedIn API error (HTTP %d): %s", statusCode, msg),
		}
	}
}
