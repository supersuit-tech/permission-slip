package trello

import (
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// maxErrorMessageLen caps the length of error messages included from Trello
// API responses. Trello returns plain text errors that are usually short,
// but we truncate to prevent unexpectedly large error payloads from
// propagating through the system.
const maxErrorMessageLen = 512

// checkResponse maps Trello HTTP status codes to connector error types.
// Trello returns plain text error messages in the response body.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	msg := truncateErrorMessage(string(body))
	if msg == "" {
		msg = http.StatusText(statusCode)
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Trello API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Trello API auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Trello API forbidden: %s", msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Trello API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Trello API resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Trello API error: %s", msg),
		}
	}
}

// truncateErrorMessage caps error message length to prevent unexpectedly
// large Trello responses from bloating error payloads.
func truncateErrorMessage(msg string) string {
	if len(msg) <= maxErrorMessageLen {
		return msg
	}
	return msg[:maxErrorMessageLen] + "... (truncated)"
}
