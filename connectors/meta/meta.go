// Package meta implements the Meta connector for the Permission Slip
// connector execution layer. It covers Facebook Pages and Instagram
// (Business/Creator accounts) using the Meta Graph API with OAuth 2.0
// access tokens provided by the platform.
package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// graphIDPattern matches valid Meta Graph API object IDs. IDs are typically
// numeric (e.g., "123456789012345") or compound (e.g., "123456_789012" for
// page posts). This pattern prevents path traversal and query injection when
// IDs are interpolated into URL paths.
var graphIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// isValidGraphID returns true if s looks like a valid Meta Graph API ID.
func isValidGraphID(s string) bool {
	return graphIDPattern.MatchString(s)
}

const (
	defaultBaseURL = "https://graph.facebook.com/v19.0"
	defaultTimeout = 30 * time.Second

	credKeyAccessToken = "access_token"

	// defaultRetryAfter is used when the Meta API returns a rate limit
	// response without a usable Retry-After header.
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes prevents OOM from unexpectedly large responses.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// MetaConnector owns the shared HTTP client and base URL used by all
// Meta actions. Actions hold a pointer back to the connector to access
// these shared resources.
type MetaConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a MetaConnector with sensible defaults.
func New() *MetaConnector {
	return &MetaConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a MetaConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *MetaConnector {
	return &MetaConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "meta", matching the connectors.id in the database.
func (c *MetaConnector) ID() string { return "meta" }

// Actions returns the registered action handlers keyed by action_type.
func (c *MetaConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"meta.create_page_post":          &createPagePostAction{conn: c},
		"meta.delete_page_post":          &deletePagePostAction{conn: c},
		"meta.reply_page_comment":        &replyPageCommentAction{conn: c},
		"meta.create_instagram_post":     &createInstagramPostAction{conn: c},
		"meta.get_instagram_insights":    &getInstagramInsightsAction{conn: c},
		"meta.list_page_posts":           &listPagePostsAction{conn: c},
		"meta.create_instagram_story":    &createInstagramStoryAction{conn: c},
		"meta.get_page_insights":         &getPageInsightsAction{conn: c},
		"meta.list_instagram_posts":      &listInstagramPostsAction{conn: c},
		"meta.reply_instagram_comment":   &replyInstagramCommentAction{conn: c},
		"meta.create_ad":                 &createAdAction{conn: c},
		"meta.create_ad_campaign":        &createAdCampaignAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token. Since tokens are provided by the platform's
// OAuth infrastructure, format validation is minimal.
func (c *MetaConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	return nil
}

// doJSON is the shared request lifecycle for Meta Graph API calls. It marshals
// reqBody as JSON, sends the request with a Bearer token in the Authorization
// header, handles rate limiting and timeouts, and unmarshals the response into
// respBody.
// For GET requests, pass nil for reqBody.
func (c *MetaConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, rawURL string, reqBody, respBody any) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("marshaling request body: %v", err)}
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("creating request: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
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

	if respBody != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Meta API response",
			}
		}
	}

	return nil
}

// doDelete performs a DELETE request against the Meta Graph API.
func (c *MetaConnector) doDelete(ctx context.Context, creds connectors.Credentials, rawURL string) error {
	return c.doJSON(ctx, creds, http.MethodDelete, rawURL, nil, nil)
}

// doGet performs a GET request against the Meta Graph API and unmarshals
// the JSON response into respBody.
func (c *MetaConnector) doGet(ctx context.Context, creds connectors.Credentials, rawURL string, respBody any) error {
	return c.doJSON(ctx, creds, http.MethodGet, rawURL, nil, respBody)
}

// wrapHTTPError converts HTTP client errors into typed connector errors.
func wrapHTTPError(err error) error {
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("Meta API request timed out: %v", err)}
	}
	if errors.Is(err, context.Canceled) {
		return &connectors.CanceledError{Message: "Meta API request canceled"}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("Meta API request failed: %v", err)}
}

// metaAPIError is the error structure returned by the Meta Graph API.
type metaAPIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// checkResponse maps Meta Graph API error codes to typed connector errors.
// Meta returns errors as JSON: {"error": {"message": "...", "type": "OAuthException", "code": 190}}
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	var apiErr metaAPIError
	msg := "Meta API error"
	code := 0
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		msg = apiErr.Error.Message
		code = apiErr.Error.Code
	}

	// Map Meta-specific error codes:
	// 190 = invalid/expired access token
	// 4 = rate limit exceeded (app-level)
	// 17 = rate limit exceeded (user-level)
	// 100 = invalid parameter
	switch {
	case code == 190 || statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Meta auth error: %s", msg)}
	case code == 4 || code == 17 || statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Meta API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case code == 100:
		return &connectors.ValidationError{Message: fmt.Sprintf("Meta API validation error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Meta permission denied: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Meta API error (HTTP %d, code %d): %s", statusCode, code, msg),
		}
	}
}
