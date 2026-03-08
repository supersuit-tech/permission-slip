// Package linear implements the Linear connector for the Permission Slip
// connector execution layer. It uses the Linear GraphQL API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package linear

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
	defaultBaseURL      = "https://api.linear.app/graphql"
	defaultTimeout      = 30 * time.Second
	credKeyAPIKey       = "api_key"
	credKeyAccessToken  = "access_token"

	// defaultRetryAfter is used when Linear returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes limits how much data we read from the Linear API
	// to prevent OOM from unexpectedly large responses.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// LinearConnector owns the shared HTTP client and base URL used by all
// Linear actions. Actions hold a pointer back to the connector to access
// these shared resources.
type LinearConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a LinearConnector with sensible defaults (30s timeout,
// https://api.linear.app/graphql base URL).
func New() *LinearConnector {
	return &LinearConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a LinearConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *LinearConnector {
	return &LinearConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "linear", matching the connectors.id in the database.
func (c *LinearConnector) ID() string { return "linear" }

// Actions returns the registered action handlers keyed by action_type.
func (c *LinearConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"linear.create_issue":   &createIssueAction{conn: c},
		"linear.update_issue":   &updateIssueAction{conn: c},
		"linear.add_comment":    &addCommentAction{conn: c},
		"linear.create_project": &createProjectAction{conn: c},
		"linear.search_issues":  &searchIssuesAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain either a
// non-empty api_key (for API key auth) or a non-empty access_token (for OAuth).
// At least one must be present.
func (c *LinearConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	apiKey, hasAPIKey := creds.Get(credKeyAPIKey)
	accessToken, hasAccessToken := creds.Get(credKeyAccessToken)

	if (!hasAPIKey || apiKey == "") && (!hasAccessToken || accessToken == "") {
		return &connectors.ValidationError{Message: "missing required credential: api_key or access_token (OAuth)"}
	}
	return nil
}

// validatePriority checks that a priority value (if set) is within the valid
// Linear range of 0–4. Shared by create_issue and update_issue.
func validatePriority(p *int) error {
	if p != nil && (*p < 0 || *p > 4) {
		return &connectors.ValidationError{Message: "priority must be 0 (none), 1 (urgent), 2 (high), 3 (medium), or 4 (low)"}
	}
	return nil
}

// graphQLRequest is the standard GraphQL request envelope.
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLResponse is the standard GraphQL response envelope.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors,omitempty"`
}

// graphQLError represents a single error in the GraphQL errors array.
type graphQLError struct {
	Message    string                 `json:"message"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// resolveAuthHeader returns the Authorization header value from the provided
// credentials. OAuth access tokens use "Bearer {token}" format, while API keys
// use the raw key (no prefix), matching Linear's API conventions.
func resolveAuthHeader(creds connectors.Credentials) (string, error) {
	// Prefer OAuth access_token over API key when both are present.
	if token, ok := creds.Get(credKeyAccessToken); ok && token != "" {
		return "Bearer " + token, nil
	}
	if key, ok := creds.Get(credKeyAPIKey); ok && key != "" {
		return key, nil
	}
	return "", &connectors.ValidationError{Message: "api_key or access_token credential is missing or empty"}
}

// doGraphQL sends a GraphQL request to the Linear API and unmarshals the
// response data into dest. It handles auth, rate limiting, timeouts, and
// maps Linear GraphQL errors to connector error types.
func (c *LinearConnector) doGraphQL(ctx context.Context, creds connectors.Credentials, query string, variables map[string]any, dest any) error {
	authHeader, err := resolveAuthHeader(creds)
	if err != nil {
		return err
	}

	gqlReq := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	payload, err := json.Marshal(gqlReq)
	if err != nil {
		return fmt.Errorf("marshaling GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Linear API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Linear API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Linear API request failed: %v", err)}
	}
	defer resp.Body.Close()

	// Handle HTTP-level rate limiting.
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Linear API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	// Handle HTTP-level auth errors.
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &connectors.AuthError{Message: fmt.Sprintf("Linear API auth error (HTTP %d)", resp.StatusCode)}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Linear API response",
		}
	}

	// Check for GraphQL-level errors.
	if len(gqlResp.Errors) > 0 {
		return mapGraphQLErrors(gqlResp.Errors)
	}

	// Unmarshal the data field into the caller's destination.
	if dest != nil && gqlResp.Data != nil {
		if err := json.Unmarshal(gqlResp.Data, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("failed to decode Linear API response data: %v", err),
			}
		}
	}

	return nil
}

// mapGraphQLErrors converts Linear GraphQL errors to the appropriate
// connector error type using the extensions.type field.
func mapGraphQLErrors(errs []graphQLError) error {
	if len(errs) == 0 {
		return nil
	}

	first := errs[0]
	extType := graphQLExtensionType(first)

	switch extType {
	case "authentication_error":
		return &connectors.AuthError{Message: fmt.Sprintf("Linear auth error: %s", first.Message)}
	case "forbidden":
		return &connectors.AuthError{Message: fmt.Sprintf("Linear forbidden: %s", first.Message)}
	case "ratelimited":
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Linear rate limit: %s", first.Message),
			RetryAfter: defaultRetryAfter,
		}
	case "validation_error":
		return &connectors.ValidationError{Message: fmt.Sprintf("Linear validation error: %s", first.Message)}
	default:
		return &connectors.ExternalError{
			StatusCode: 200,
			Message:    fmt.Sprintf("Linear GraphQL error: %s", first.Message),
		}
	}
}

// graphQLExtensionType extracts the "type" field from a GraphQL error's
// extensions map, returning an empty string if not present.
func graphQLExtensionType(e graphQLError) string {
	if e.Extensions == nil {
		return ""
	}
	t, ok := e.Extensions["type"].(string)
	if !ok {
		return ""
	}
	return t
}
