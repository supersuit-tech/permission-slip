// Package airtable implements the Airtable connector for the Permission Slip
// connector execution layer. It uses the Airtable REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package airtable

import (
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
	defaultBaseURL = "https://api.airtable.com/v0"
	metaBaseURL    = "https://api.airtable.com/v0/meta"
	defaultTimeout = 30 * time.Second
	credKeyToken   = "api_token"
	tokenPrefix    = "pat"

	// defaultRetryAfter is used when the Airtable API returns a rate limit
	// response without a Retry-After header.
	defaultRetryAfter = 30 * time.Second

	// maxResponseBytes caps the response body at 10 MB.
	maxResponseBytes = 10 << 20 // 10 MB
)

// AirtableConnector owns the shared HTTP client and base URL used by all
// Airtable actions.
type AirtableConnector struct {
	client  *http.Client
	baseURL string
	metaURL string
}

// New creates an AirtableConnector with sensible defaults.
func New() *AirtableConnector {
	return &AirtableConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
		metaURL: metaBaseURL,
	}
}

// newForTest creates an AirtableConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *AirtableConnector {
	return &AirtableConnector{
		client:  client,
		baseURL: baseURL,
		metaURL: baseURL + "/meta",
	}
}

// ID returns "airtable", matching the connectors.id in the database.
func (c *AirtableConnector) ID() string { return "airtable" }

// Actions returns the registered action handlers keyed by action_type.
func (c *AirtableConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"airtable.list_bases":      &listBasesAction{conn: c},
		"airtable.list_records":    &listRecordsAction{conn: c},
		"airtable.get_record":      &getRecordAction{conn: c},
		"airtable.create_records":  &createRecordsAction{conn: c},
		"airtable.update_records":  &updateRecordsAction{conn: c},
		"airtable.delete_records":  &deleteRecordsAction{conn: c},
		"airtable.search_records":  &searchRecordsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty personal access token with the required pat prefix.
func (c *AirtableConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_token"}
	}
	if len(token) < len(tokenPrefix) || token[:len(tokenPrefix)] != tokenPrefix {
		return &connectors.ValidationError{Message: "api_token must be a personal access token starting with \"pat\""}
	}
	return nil
}

// validateBaseID checks that a base_id looks like a valid Airtable base ID.
func validateBaseID(baseID string) error {
	if baseID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: base_id"}
	}
	if len(baseID) < 4 || baseID[:3] != "app" {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid base_id %q: expected an Airtable base ID starting with 'app'", baseID),
		}
	}
	return nil
}

// validateRecordID checks that a record_id looks like a valid Airtable record ID.
func validateRecordID(recordID string) error {
	if recordID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: record_id"}
	}
	if len(recordID) < 4 || recordID[:3] != "rec" {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid record_id %q: expected an Airtable record ID starting with 'rec'", recordID),
		}
	}
	return nil
}

// parseAndValidate unmarshals JSON parameters and validates them.
func parseAndValidate[T interface{ validate() error }](raw json.RawMessage, dest *T) error {
	if err := json.Unmarshal(raw, dest); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	return (*dest).validate()
}

// doRequest executes an HTTP request against the Airtable API with auth headers,
// handles rate limiting and timeouts, and unmarshals the response into dest.
func (c *AirtableConnector) doRequest(ctx context.Context, method, url string, creds connectors.Credentials, body io.Reader, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "api_token credential is missing or empty"}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
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
			return &connectors.TimeoutError{Message: fmt.Sprintf("Airtable API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Airtable API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Airtable API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Airtable API rate limit exceeded (5 requests/second per base)",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return &connectors.AuthError{
			Message: "Airtable authentication failed — check that your personal access token is valid and not expired",
		}
	}
	if resp.StatusCode == http.StatusForbidden {
		return &connectors.AuthError{
			Message: "Airtable permission denied — your token may lack the required scopes for this base or table",
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapAirtableError(resp.StatusCode, respBody)
	}

	if dest != nil {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Airtable API response",
			}
		}
	}

	return nil
}

// airtableErrorResponse is the error envelope from the Airtable API.
type airtableErrorResponse struct {
	Error *airtableErrorDetail `json:"error,omitempty"`
}

type airtableErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// mapAirtableError converts an Airtable API error response to the appropriate
// connector error type.
func mapAirtableError(statusCode int, body []byte) error {
	var errResp airtableErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil || errResp.Error == nil {
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Airtable API error (HTTP %d): %s", statusCode, string(body)),
		}
	}

	msg := fmt.Sprintf("Airtable API error: %s — %s", errResp.Error.Type, errResp.Error.Message)

	switch errResp.Error.Type {
	case "AUTHENTICATION_REQUIRED":
		return &connectors.AuthError{Message: "Airtable authentication required — check that your personal access token is valid"}
	case "INVALID_PERMISSIONS_OR_MODEL_NOT_FOUND":
		return &connectors.AuthError{
			Message: "Airtable permission denied or resource not found — verify the base ID exists and your token has access to it",
		}
	case "NOT_FOUND", "TABLE_NOT_FOUND", "ROW_DOES_NOT_EXIST":
		return &connectors.ExternalError{StatusCode: 404, Message: msg}
	case "VIEW_NOT_FOUND":
		return &connectors.ExternalError{StatusCode: 404, Message: "Airtable view not found — check the view name or ID"}
	case "INVALID_REQUEST_UNKNOWN", "INVALID_VALUE_FOR_COLUMN", "CANNOT_UPDATE_COMPUTED_FIELD",
		"FIELD_NOT_FOUND", "UNKNOWN_FIELD_NAME", "CANNOT_CREATE_DUPLICATE_RECORD":
		return &connectors.ValidationError{Message: msg}
	case "INVALID_FILTER_BY_FORMULA":
		return &connectors.ValidationError{
			Message: fmt.Sprintf("Airtable formula syntax error — %s. See https://support.airtable.com/docs/formula-field-reference", errResp.Error.Message),
		}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: msg}
	}
}
