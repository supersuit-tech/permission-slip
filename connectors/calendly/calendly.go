// Package calendly implements the Calendly connector for the Permission Slip
// connector execution layer. It uses the Calendly REST API v2 with either
// OAuth2 access tokens (preferred) or personal access tokens (API key).
package calendly

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

const (
	defaultBaseURL = "https://api.calendly.com"
	defaultTimeout = 30 * time.Second

	// credKeyAPIKey is the credential key for Calendly personal access tokens.
	credKeyAPIKey = "api_key"

	// credKeyAccessToken is the credential key for OAuth2 access tokens,
	// set by the OAuth credential resolution path.
	credKeyAccessToken = "access_token"

	// defaultRetryAfter is used when Calendly returns a 429 without a
	// Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes caps the response body we'll read from Calendly APIs.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// CalendlyConnector owns the shared HTTP client and base URL used by all
// Calendly actions. Actions hold a pointer back to the connector to access
// these shared resources.
type CalendlyConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a CalendlyConnector with sensible defaults.
func New() *CalendlyConnector {
	return &CalendlyConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a CalendlyConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *CalendlyConnector {
	return &CalendlyConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "calendly", matching the connectors.id in the database.
func (c *CalendlyConnector) ID() string { return "calendly" }

// Actions returns the registered action handlers keyed by action_type.
func (c *CalendlyConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"calendly.list_event_types":      &listEventTypesAction{conn: c},
		"calendly.create_scheduling_link": &createSchedulingLinkAction{conn: c},
		"calendly.list_scheduled_events":  &listScheduledEventsAction{conn: c},
		"calendly.cancel_event":           &cancelEventAction{conn: c},
		"calendly.get_event":              &getEventAction{conn: c},
		"calendly.list_available_times":   &listAvailableTimesAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain either a
// non-empty access_token (OAuth2) or api_key (personal access token), and
// tests them against GET /users/me.
func (c *CalendlyConnector) ValidateCredentials(ctx context.Context, creds connectors.Credentials) error {
	if token, ok := creds.Get(credKeyAccessToken); ok && token != "" {
		// OAuth path — test against /users/me.
		var resp usersmeResponse
		if err := c.doJSON(ctx, creds, http.MethodGet, c.baseURL+"/users/me", nil, &resp); err != nil {
			return err
		}
		if resp.Resource.URI == "" {
			return &connectors.ValidationError{Message: "Calendly API returned empty user URI"}
		}
		return nil
	}
	if key, ok := creds.Get(credKeyAPIKey); ok && key != "" {
		// API key path — test against /users/me.
		var resp usersmeResponse
		if err := c.doJSON(ctx, creds, http.MethodGet, c.baseURL+"/users/me", nil, &resp); err != nil {
			return err
		}
		if resp.Resource.URI == "" {
			return &connectors.ValidationError{Message: "Calendly API returned empty user URI"}
		}
		return nil
	}
	return &connectors.ValidationError{Message: "missing required credential: access_token (OAuth) or api_key"}
}

// usersmeResponse is the Calendly API response from GET /users/me.
type usersmeResponse struct {
	Resource struct {
		URI  string `json:"uri"`
		Name string `json:"name"`
	} `json:"resource"`
}

// getUserURI calls GET /users/me and returns the authenticated user's URI.
// Most Calendly list endpoints require this URI as a filter parameter.
func (c *CalendlyConnector) getUserURI(ctx context.Context, creds connectors.Credentials) (string, error) {
	var resp usersmeResponse
	if err := c.doJSON(ctx, creds, http.MethodGet, c.baseURL+"/users/me", nil, &resp); err != nil {
		return "", err
	}
	if resp.Resource.URI == "" {
		return "", &connectors.ExternalError{Message: "Calendly API returned empty user URI from /users/me"}
	}
	return resp.Resource.URI, nil
}

// doJSON is the shared request lifecycle for Calendly API calls that send and
// receive JSON. It marshals reqBody as JSON, sends the request with Bearer
// token auth, handles rate limiting and timeouts, and unmarshals the
// response into respBody.
func (c *CalendlyConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, url string, reqBody, respBody any) error {
	// Prefer OAuth access_token; fall back to personal access token (api_key).
	token, _ := creds.Get(credKeyAccessToken)
	if token == "" {
		token, _ = creds.Get(credKeyAPIKey)
	}
	if token == "" {
		return &connectors.ValidationError{Message: "access_token (OAuth) or api_key credential is missing or empty"}
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("marshaling request body: %v", err)}
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("creating request: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Calendly API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.CanceledError{Message: "Calendly API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Calendly API request failed: %v", err)}
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
				Message:    "failed to decode Calendly API response",
			}
		}
	}

	return nil
}

// uuidPattern validates that an event UUID contains only safe characters.
// Calendly UUIDs are alphanumeric with hyphens (e.g., "abc-123-def-456").
var uuidPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateUUID checks that the given string is a safe UUID for use in URL paths.
// This prevents path traversal attacks (e.g., "../../other-endpoint").
func validateUUID(uuid string) error {
	if !uuidPattern.MatchString(uuid) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid event_uuid format: must contain only alphanumeric characters, hyphens, and underscores; got %q", uuid),
		}
	}
	return nil
}

// resolveUserURI returns the user URI to use for API calls. If userURI is
// provided (non-empty), it is returned directly — this allows callers to
// skip the extra GET /users/me round-trip when they already have the URI.
// Otherwise, it fetches the URI from the API.
func (c *CalendlyConnector) resolveUserURI(ctx context.Context, creds connectors.Credentials, userURI string) (string, error) {
	if userURI != "" {
		return userURI, nil
	}
	return c.getUserURI(ctx, creds)
}

// calendlyAPIError represents the error response format from the Calendly API.
// Calendly returns {"title": "<string>", "message": "<string>"} on errors.
type calendlyAPIError struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// checkResponse maps HTTP status codes to typed connector errors.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	var apiErr calendlyAPIError
	msg := "Calendly API error"
	if err := json.Unmarshal(body, &apiErr); err == nil {
		if apiErr.Message != "" {
			msg = apiErr.Message
		} else if apiErr.Title != "" {
			msg = apiErr.Title
		}
	}

	switch {
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Calendly auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Calendly permission denied: %s", msg)}
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Calendly API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Calendly API bad request: %s", msg)}
	case statusCode == http.StatusUnprocessableEntity:
		return &connectors.ValidationError{Message: fmt.Sprintf("Calendly API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Calendly API not found: %s", msg)}
	case statusCode == http.StatusConflict:
		return &connectors.ValidationError{Message: fmt.Sprintf("Calendly API conflict: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Calendly API error (HTTP %d): %s", statusCode, msg),
		}
	}
}
