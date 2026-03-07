package zendesk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const maxErrorBodyPreview = 200

// zendeskError represents the Zendesk API error response format.
type zendeskError struct {
	Error       string            `json:"error"`
	Description string            `json:"description"`
	Details     map[string][]any  `json:"details"`
	Message     string            `json:"message"`
}

// truncateBody returns a safe preview of the response body for error messages.
func truncateBody(body []byte) string {
	if len(body) == 0 {
		return "(empty response)"
	}
	if len(body) <= maxErrorBodyPreview {
		return string(body)
	}
	return string(body[:maxErrorBodyPreview]) + "... (truncated)"
}

// checkResponse inspects the HTTP status code and response body, returning
// an appropriate typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	var zdErr zendeskError
	msg := truncateBody(body)
	if json.Unmarshal(body, &zdErr) == nil {
		if zdErr.Description != "" {
			msg = zdErr.Description
		} else if zdErr.Error != "" {
			msg = zdErr.Error
		} else if zdErr.Message != "" {
			msg = zdErr.Message
		}
	}

	if statusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 60*time.Second)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Zendesk API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	}

	return mapStatusCodeError(statusCode, msg)
}

func mapStatusCodeError(statusCode int, msg string) error {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Zendesk auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusUnprocessableEntity || statusCode == http.StatusBadRequest || statusCode == http.StatusConflict:
		return &connectors.ValidationError{Message: fmt.Sprintf("Zendesk validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Zendesk resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Zendesk API error: %s", msg),
		}
	}
}
