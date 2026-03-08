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
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

var (
	// urnPattern validates LinkedIn URN format (e.g. urn:li:share:123456).
	urnPattern = regexp.MustCompile(`^urn:li:[a-zA-Z]+:\d+$`)

	// numericPattern validates that organization IDs are numeric.
	numericPattern = regexp.MustCompile(`^\d+$`)

	// personURNPattern validates LinkedIn person URNs specifically.
	// Only urn:li:person:{numeric_id} is accepted — share/post URNs are rejected.
	personURNPattern = regexp.MustCompile(`^urn:li:person:\d+$`)
)

const (
	defaultV2BaseURL   = "https://api.linkedin.com/v2"
	defaultRestBaseURL = "https://api.linkedin.com/rest"
	defaultTimeout     = 30 * time.Second
	credKeyAccessToken = "access_token"

	// linkedInVersion is sent via the LinkedIn-Version header on /rest/ endpoints.
	// LinkedIn uses calendar versioning (YYYYMM). Bump this when adopting a newer
	// API version: https://learn.microsoft.com/en-us/linkedin/marketing/versioning
	linkedInVersion = "202401"

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
		"linkedin.send_message":        &sendMessageAction{conn: c},
		"linkedin.search_people":       &searchPeopleAction{conn: c},
		"linkedin.search_companies":    &searchCompaniesAction{conn: c},
		"linkedin.get_company":         &getCompanyAction{conn: c},
		"linkedin.list_connections":    &listConnectionsAction{conn: c},
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

// validateArticleURL checks that the article URL uses http or https.
// This prevents injection of dangerous schemes (javascript:, data:, file:)
// when the URL is rendered in LinkedIn posts.
func validateArticleURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return &connectors.ValidationError{Message: "article_url must be a valid URL"}
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return &connectors.ValidationError{Message: "article_url must use http or https scheme"}
	}
	return nil
}

// validatePostURN checks that a post URN matches the expected LinkedIn format
// (urn:li:{type}:{numeric_id}). This prevents path traversal when the URN is
// URL-encoded and interpolated into API paths.
func validatePostURN(urn string) error {
	if !urnPattern.MatchString(urn) {
		return &connectors.ValidationError{Message: "post_urn must be a valid LinkedIn URN (e.g. urn:li:share:123456)"}
	}
	return nil
}

// validateOrganizationID checks that the organization ID is numeric. The ID
// is interpolated into "urn:li:organization:{id}" and sent as the post author.
func validateOrganizationID(id string) error {
	if !numericPattern.MatchString(id) {
		return &connectors.ValidationError{Message: "organization_id must be numeric"}
	}
	return nil
}

// validatePersonURN checks that a recipient URN is a valid LinkedIn person URN
// (urn:li:person:{numeric_id}). Only person URNs are accepted — share or post
// URNs are rejected to prevent accidentally messaging the wrong entity type.
func validatePersonURN(urn string) error {
	if !personURNPattern.MatchString(urn) {
		return &connectors.ValidationError{Message: "recipient_urn must be a valid LinkedIn person URN (e.g. urn:li:person:123456)"}
	}
	return nil
}

// linkedInErrorResponse is the LinkedIn API error envelope. ServiceErrorCode is
// a LinkedIn-specific numeric code that provides more detail than the HTTP
// status alone (e.g. 65600 = "Invalid access token").
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
	_, err := c.doWithHeaders(ctx, creds, method, url, reqBody, dest, useRestAPI)
	return err
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

// getPersonURN fetches the authenticated user's person URN via the userinfo
// endpoint and returns it in the format "urn:li:person:{sub}".
func (c *LinkedInConnector) getPersonURN(ctx context.Context, creds connectors.Credentials) (string, error) {
	var resp userinfoResponse
	url := c.v2BaseURL + "/userinfo"
	if err := c.do(ctx, creds, http.MethodGet, url, nil, &resp, false); err != nil {
		return "", err
	}
	if resp.Sub == "" {
		return "", &connectors.ExternalError{Message: "LinkedIn userinfo returned empty sub"}
	}
	return "urn:li:person:" + resp.Sub, nil
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

// checkResponse maps HTTP status codes to typed connector errors. The mapping
// ensures the execution layer can distinguish auth failures (trigger re-auth),
// rate limits (schedule retry), and validation errors (surface to user) from
// unexpected server errors.
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
