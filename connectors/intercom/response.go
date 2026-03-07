package intercom

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const maxErrorBodyPreview = 200

// intercomError represents the Intercom API error response format.
type intercomError struct {
	Type       string            `json:"type"`
	Message    string            `json:"message"`
	RequestID  string            `json:"request_id"`
	Errors     []intercomErrItem `json:"errors"`
}

type intercomErrItem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
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

	var icErr intercomError
	msg := truncateBody(body)
	if json.Unmarshal(body, &icErr) == nil {
		if icErr.Message != "" {
			msg = icErr.Message
		} else if len(icErr.Errors) > 0 && icErr.Errors[0].Message != "" {
			msg = icErr.Errors[0].Message
		}
		if icErr.RequestID != "" {
			msg = fmt.Sprintf("%s (request_id: %s)", msg, icErr.RequestID)
		}
	}

	if statusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 60*time.Second)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Intercom API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	}

	return mapStatusCodeError(statusCode, msg)
}

func mapStatusCodeError(statusCode int, msg string) error {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Intercom auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusUnprocessableEntity || statusCode == http.StatusBadRequest || statusCode == http.StatusConflict:
		return &connectors.ValidationError{Message: fmt.Sprintf("Intercom validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Intercom resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Intercom API error: %s", msg),
		}
	}
}
