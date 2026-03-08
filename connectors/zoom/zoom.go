// Package zoom implements the Zoom connector for the Permission Slip
// connector execution layer. It uses the Zoom REST API v2 with OAuth 2.0
// access tokens provided by the platform.
package zoom

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
	defaultBaseURL     = "https://api.zoom.us/v2"
	defaultTimeout     = 30 * time.Second
	credKeyAccessToken = "access_token"

	// defaultRetryAfter is used when Zoom returns a 429 without a
	// Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes caps the response body we'll read from Zoom APIs.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// ZoomConnector owns the shared HTTP client and base URL used by all
// Zoom actions. Actions hold a pointer back to the connector to access
// these shared resources.
type ZoomConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a ZoomConnector with sensible defaults.
func New() *ZoomConnector {
	return &ZoomConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a ZoomConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *ZoomConnector {
	return &ZoomConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "zoom", matching the connectors.id in the database.
func (c *ZoomConnector) ID() string { return "zoom" }

// Actions returns the registered action handlers keyed by action_type.
func (c *ZoomConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"zoom.list_meetings":              &listMeetingsAction{conn: c},
		"zoom.create_meeting":             &createMeetingAction{conn: c},
		"zoom.get_meeting":                &getMeetingAction{conn: c},
		"zoom.update_meeting":             &updateMeetingAction{conn: c},
		"zoom.delete_meeting":             &deleteMeetingAction{conn: c},
		"zoom.list_recordings":            &listRecordingsAction{conn: c},
		"zoom.get_meeting_participants":   &getMeetingParticipantsAction{conn: c},
		"zoom.add_registrant":             &addRegistrantAction{conn: c},
		"zoom.get_recording_transcript":   &getRecordingTranscriptAction{conn: c},
		"zoom.send_chat_message":          &sendChatMessageAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token.
func (c *ZoomConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	return nil
}

// doJSON is the shared request lifecycle for Zoom API calls that send and
// receive JSON. It marshals reqBody as JSON, sends the request with OAuth
// bearer auth, handles rate limiting and timeouts, and unmarshals the
// response into respBody.
//
// For DELETE requests that return 204 No Content, pass nil for respBody.
func (c *ZoomConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, url string, reqBody, respBody any) error {
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
			return &connectors.TimeoutError{Message: fmt.Sprintf("Zoom API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Zoom API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Zoom API request failed: %v", err)}
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
				Message:    "failed to decode Zoom API response",
			}
		}
	}

	return nil
}

// zoomAPIError represents the error response format from the Zoom API.
// Zoom returns {"code": <int>, "message": "<string>"} on errors.
type zoomAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// checkResponse maps HTTP status codes to typed connector errors.
// Zoom returns {"code": ..., "message": "..."} on errors.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract a Zoom API error message.
	var apiErr zoomAPIError
	msg := "Zoom API error"
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		msg = apiErr.Message
	}

	switch {
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Zoom auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Zoom permission denied: %s", msg)}
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Zoom API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Zoom API bad request: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Zoom API not found: %s", msg)}
	case statusCode == http.StatusConflict:
		return &connectors.ValidationError{Message: fmt.Sprintf("Zoom API conflict: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Zoom API error (HTTP %d): %s", statusCode, msg),
		}
	}
}
