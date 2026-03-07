// Package monday implements the Monday.com connector for the Permission Slip
// connector execution layer. It uses the Monday.com GraphQL API with plain
// net/http (no third-party SDK) to keep the dependency footprint minimal.
package monday

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
	defaultBaseURL = "https://api.monday.com/v2"
	defaultTimeout = 30 * time.Second
	credKeyToken   = "api_key"

	// defaultRetryAfter is used when the Monday.com API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes caps the Monday.com API response body at 10 MB.
	maxResponseBytes = 10 << 20 // 10 MB
)

// MondayConnector owns the shared HTTP client and base URL used by all
// Monday.com actions.
type MondayConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a MondayConnector with sensible defaults (30s timeout,
// https://api.monday.com/v2 base URL).
func New() *MondayConnector {
	return &MondayConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a MondayConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *MondayConnector {
	return &MondayConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "monday", matching the connectors.id in the database.
func (c *MondayConnector) ID() string { return "monday" }

// Actions returns the registered action handlers keyed by action_type.
func (c *MondayConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"monday.create_item":        &createItemAction{conn: c},
		"monday.update_item":        &updateItemAction{conn: c},
		"monday.add_update":         &addUpdateAction{conn: c},
		"monday.create_subitem":     &createSubitemAction{conn: c},
		"monday.move_item_to_group": &moveItemToGroupAction{conn: c},
		"monday.search_items":       &searchItemsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key.
func (c *MondayConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
}

// graphQLRequest is the Monday.com GraphQL request body.
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLResponse is the common envelope for Monday.com GraphQL responses.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data,omitempty"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// query sends a GraphQL request to the Monday.com API and unmarshals the
// response data into dest.
func (c *MondayConnector) query(ctx context.Context, creds connectors.Credentials, gqlQuery string, variables map[string]any, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "api_key credential is missing or empty"}
	}

	reqBody := graphQLRequest{
		Query:     gqlQuery,
		Variables: variables,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("marshaling request body: %v", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("creating request: %v", err)}
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Monday.com API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Monday.com API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Monday.com API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Monday.com API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &connectors.AuthError{Message: "Monday.com auth error: invalid or expired API key"}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	// Handle HTTP 400 as a validation error (e.g. malformed query).
	if resp.StatusCode == http.StatusBadRequest {
		msg := extractErrorMessage(respBody)
		if msg == "" {
			msg = "Monday.com API rejected the request"
		}
		return &connectors.ValidationError{Message: fmt.Sprintf("Monday.com validation error: %s", msg)}
	}

	// Catch other non-2xx status codes not already handled above.
	if resp.StatusCode >= 400 {
		msg := extractErrorMessage(respBody)
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Monday.com API error: %s", msg),
		}
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Monday.com API response",
		}
	}

	// Check for top-level error fields (e.g. auth errors returned as 200).
	if gqlResp.ErrorCode != "" {
		return mapMondayError(gqlResp.ErrorCode, gqlResp.ErrorMessage)
	}

	// Check for GraphQL errors array.
	if len(gqlResp.Errors) > 0 {
		msg := gqlResp.Errors[0].Message
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Monday.com API error: %s", msg),
		}
	}

	if dest != nil && gqlResp.Data != nil {
		if err := json.Unmarshal(gqlResp.Data, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("failed to decode Monday.com response data: %v", err),
			}
		}
	}

	return nil
}

// isValidMondayID checks that an ID is a non-empty numeric string.
// Monday.com IDs are always numeric, so rejecting non-numeric values
// prevents unexpected API behavior.
func isValidMondayID(id string) bool {
	if id == "" {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// stringifyColumnValues marshals a column_values map to a JSON string,
// which is the format Monday.com's GraphQL API expects.
func stringifyColumnValues(cv map[string]any) (string, error) {
	data, err := json.Marshal(cv)
	if err != nil {
		return "", &connectors.ValidationError{Message: fmt.Sprintf("invalid column_values: %v", err)}
	}
	return string(data), nil
}

// extractErrorMessage tries to pull an error message from a Monday.com
// error response body. Returns empty string if the body can't be parsed.
func extractErrorMessage(body []byte) string {
	var envelope struct {
		ErrorMessage string `json:"error_message"`
		Errors       []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return ""
	}
	if envelope.ErrorMessage != "" {
		return envelope.ErrorMessage
	}
	if len(envelope.Errors) > 0 && envelope.Errors[0].Message != "" {
		return envelope.Errors[0].Message
	}
	return ""
}

// mapMondayError converts a Monday.com error code to the appropriate
// connector error type.
func mapMondayError(code, message string) error {
	if message == "" {
		message = code
	}
	switch code {
	case "UserUnauthorizedException", "NotAuthenticated", "invalid_token":
		return &connectors.AuthError{Message: fmt.Sprintf("Monday.com auth error: %s", message)}
	case "RateLimitExceeded", "ComplexityBudgetExhausted":
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Monday.com rate limit: %s", message),
			RetryAfter: defaultRetryAfter,
		}
	default:
		return &connectors.ExternalError{
			StatusCode: 200,
			Message:    fmt.Sprintf("Monday.com API error: %s", message),
		}
	}
}
