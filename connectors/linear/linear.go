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
	defaultBaseURL = "https://api.linear.app/graphql"
	defaultTimeout = 30 * time.Second
	credKeyAPIKey  = "api_key"

	// defaultRetryAfter is used when Linear returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second
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
// Phase 1 returns an empty map — actions are added in Phase 2.
func (c *LinearConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key. Linear personal API keys are opaque strings with
// no fixed prefix, so we only validate presence.
func (c *LinearConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
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

// doGraphQL sends a GraphQL request to the Linear API and unmarshals the
// response data into dest. It handles auth, rate limiting, timeouts, and
// maps Linear GraphQL errors to connector error types.
func (c *LinearConnector) doGraphQL(ctx context.Context, creds connectors.Credentials, query string, variables map[string]any, dest any) error {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "api_key credential is missing or empty"}
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
	// Linear uses "Authorization: {api_key}" — no "Bearer" prefix.
	req.Header.Set("Authorization", key)
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

	body, err := io.ReadAll(resp.Body)
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
