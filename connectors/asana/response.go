package asana

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// maxErrorMessageLen caps the length of error messages included from Asana
// API responses to prevent unexpectedly large or raw response bodies (e.g.
// HTML error pages) from propagating through the system.
const maxErrorMessageLen = 512

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. Asana returns errors in the format:
// {"errors": [{"message": "...", "help": "..."}]}
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract Asana's structured error message. Only fall back to the
	// raw body if JSON parsing succeeds but yields no message — never include
	// the raw body verbatim to avoid leaking unexpected response content (e.g.
	// HTML error pages from intermediate proxies).
	var asanaErr struct {
		Errors []struct {
			Message string `json:"message"`
			Help    string `json:"help"`
		} `json:"errors"`
	}
	var msg string
	if json.Unmarshal(body, &asanaErr) == nil && len(asanaErr.Errors) > 0 {
		msg = truncateErrorMessage(asanaErr.Errors[0].Message)
		if asanaErr.Errors[0].Help != "" {
			msg += " (hint: " + truncateErrorMessage(asanaErr.Errors[0].Help) + ")"
		}
	}
	if msg == "" {
		msg = http.StatusText(statusCode)
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Asana API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Asana API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Asana API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Asana resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Asana API error: %s", msg)}
	}
}

// truncateErrorMessage caps error message length to prevent unexpectedly large
// Asana responses from bloating error payloads.
func truncateErrorMessage(msg string) string {
	return connectors.TruncateUTF8(msg, maxErrorMessageLen)
}
