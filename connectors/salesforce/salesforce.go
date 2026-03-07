// Package salesforce implements the Salesforce connector for the Permission Slip
// connector execution layer. It uses the Salesforce REST API with OAuth 2.0
// access tokens provided by the platform.
package salesforce

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

// sfIDPattern matches Salesforce 15 or 18-character alphanumeric record IDs.
var sfIDPattern = regexp.MustCompile(`^[a-zA-Z0-9]{15}([a-zA-Z0-9]{3})?$`)

const (
	// apiVersion is the pinned Salesforce REST API version.
	apiVersion = "v62.0"

	defaultTimeout = 30 * time.Second

	credKeyAccessToken = "access_token"
	credKeyInstanceURL = "instance_url"

	// defaultRetryAfter is used when Salesforce returns a rate limit response
	// without a Retry-After header.
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes prevents OOM from unexpectedly large responses.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// SalesforceConnector owns the shared HTTP client used by all Salesforce
// actions. Unlike most connectors, the base URL is dynamic — it comes from
// the user's instance_url credential (extracted from the OAuth token response).
type SalesforceConnector struct {
	client *http.Client
	// baseURLOverride is used only in tests to point at httptest servers.
	baseURLOverride string
}

// New creates a SalesforceConnector with sensible defaults.
func New() *SalesforceConnector {
	return &SalesforceConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates a SalesforceConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *SalesforceConnector {
	return &SalesforceConnector{
		client:          client,
		baseURLOverride: baseURL,
	}
}

// ID returns "salesforce", matching the connectors.id in the database.
func (c *SalesforceConnector) ID() string { return "salesforce" }

// Actions returns the registered action handlers keyed by action_type.
func (c *SalesforceConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"salesforce.create_record": &createRecordAction{conn: c},
		"salesforce.update_record": &updateRecordAction{conn: c},
		"salesforce.query":         &queryAction{conn: c},
		"salesforce.create_task":   &createTaskAction{conn: c},
		"salesforce.add_note":      &addNoteAction{conn: c},
	}
}

// ValidateCredentials checks that access_token and instance_url are present
// and that instance_url points to a valid Salesforce domain over HTTPS.
func (c *SalesforceConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	instanceURL, ok := creds.Get(credKeyInstanceURL)
	if !ok || instanceURL == "" {
		return &connectors.ValidationError{Message: "missing required credential: instance_url"}
	}
	if c.baseURLOverride == "" {
		if err := validateInstanceURL(instanceURL); err != nil {
			return err
		}
	}
	return nil
}

// apiBaseURL returns the Salesforce REST API base URL for the given credentials.
// In tests, the baseURLOverride is used instead.
func (c *SalesforceConnector) apiBaseURL(creds connectors.Credentials) (string, error) {
	if c.baseURLOverride != "" {
		return c.baseURLOverride + "/services/data/" + apiVersion, nil
	}
	instanceURL, ok := creds.Get(credKeyInstanceURL)
	if !ok || instanceURL == "" {
		return "", &connectors.ValidationError{Message: "instance_url credential is missing or empty"}
	}
	if err := validateInstanceURL(instanceURL); err != nil {
		return "", err
	}
	return instanceURL + "/services/data/" + apiVersion, nil
}

// validateInstanceURL ensures the Salesforce instance URL is a valid HTTPS URL
// pointing to a *.salesforce.com or *.force.com domain. This prevents SSRF
// attacks where a tampered instance_url could redirect API calls to an
// attacker-controlled server.
func validateInstanceURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("instance_url is not a valid URL: %v", err)}
	}
	if u.Scheme != "https" {
		return &connectors.ValidationError{Message: "instance_url must use HTTPS"}
	}
	host := strings.ToLower(u.Hostname())
	if !strings.HasSuffix(host, ".salesforce.com") && !strings.HasSuffix(host, ".force.com") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("instance_url host %q is not a valid Salesforce domain (must end in .salesforce.com or .force.com)", host),
		}
	}
	return nil
}

// recordURL builds a user-facing Salesforce URL for a record. Returns empty
// string if instance_url is not available (e.g. in tests with baseURLOverride).
func recordURL(creds connectors.Credentials, recordID string) string {
	instanceURL, ok := creds.Get(credKeyInstanceURL)
	if !ok || instanceURL == "" {
		return ""
	}
	return instanceURL + "/" + recordID
}

// validateRecordID checks that id looks like a valid Salesforce record ID
// (15 or 18 alphanumeric characters). fieldName is used in the error message.
func validateRecordID(id, fieldName string) error {
	if !sfIDPattern.MatchString(id) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid %s: expected 15 or 18 alphanumeric characters, got %q", fieldName, id),
		}
	}
	return nil
}

// doJSON is the shared request lifecycle for Salesforce API calls that send
// and receive JSON. It marshals reqBody, sends the request with OAuth bearer
// auth, handles error mapping, and unmarshals the response into respBody.
// respBody may be nil for requests that return no body (e.g. PATCH → 204).
func (c *SalesforceConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, url string, reqBody, respBody any) error {
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
	req.Header.Set("Accept", "application/json")
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
				Message:    "failed to decode Salesforce API response",
			}
		}
	}

	return nil
}

// sfAPIError represents a single error entry from the Salesforce REST API.
type sfAPIError struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

// wrapHTTPError converts HTTP client errors into typed connector errors.
func wrapHTTPError(err error) error {
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("Salesforce API request timed out: %v", err)}
	}
	if errors.Is(err, context.Canceled) {
		return &connectors.TimeoutError{Message: "Salesforce API request canceled"}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("Salesforce API request failed: %v", err)}
}

// checkResponse maps Salesforce HTTP status codes and error codes to typed
// connector errors. Salesforce returns errors as [{"errorCode":"...","message":"..."}].
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract Salesforce API error details.
	msg := "Salesforce API error"
	errorCode := ""
	var sfErrors []sfAPIError
	if err := json.Unmarshal(body, &sfErrors); err == nil && len(sfErrors) > 0 {
		msg = sfErrors[0].Message
		errorCode = sfErrors[0].ErrorCode
	}

	// Map known Salesforce error codes to typed errors.
	switch errorCode {
	case "INVALID_SESSION_ID":
		return &connectors.AuthError{Message: fmt.Sprintf("Salesforce auth error: %s", msg)}
	case "REQUEST_LIMIT_EXCEEDED":
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Salesforce API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case "MALFORMED_QUERY", "INVALID_FIELD", "INVALID_TYPE", "REQUIRED_FIELD_MISSING",
		"INVALID_OR_NULL_FOR_RESTRICTED_PICKLIST", "STRING_TOO_LONG", "FIELD_CUSTOM_VALIDATION_EXCEPTION",
		"DUPLICATE_VALUE", "INVALID_CROSS_REFERENCE_KEY":
		return &connectors.ValidationError{Message: fmt.Sprintf("Salesforce validation error: %s", msg)}
	}

	// Fall back to HTTP status code mapping.
	switch statusCode {
	case http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Salesforce auth error: %s", msg)}
	case http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Salesforce permission denied: %s", msg)}
	case http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Salesforce API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Salesforce API error (HTTP %d): %s", statusCode, msg),
		}
	}
}
