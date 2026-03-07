package asana

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. Asana returns errors in the format:
// {"errors": [{"message": "...", "help": "..."}]}
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract Asana's error message.
	var asanaErr struct {
		Errors []struct {
			Message string `json:"message"`
			Help    string `json:"help"`
		} `json:"errors"`
	}
	msg := string(body)
	if json.Unmarshal(body, &asanaErr) == nil && len(asanaErr.Errors) > 0 {
		msg = asanaErr.Errors[0].Message
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
