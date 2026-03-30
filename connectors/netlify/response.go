package netlify

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. The mapping follows the same
// conventions as other connectors in the codebase:
//   - 429 → RateLimitError (with Retry-After parsing)
//   - 401/403 → AuthError (invalid or expired token)
//   - 400/422 → ValidationError (malformed request)
//   - 404 → ValidationError (resource not found)
//   - Other → ExternalError (unexpected API failure)
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract Netlify's error message.
	var netlifyErr struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}
	msg := string(body)
	if json.Unmarshal(body, &netlifyErr) == nil && netlifyErr.Message != "" {
		msg = netlifyErr.Message
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Netlify API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Netlify API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity:
		return &connectors.ValidationError{Message: fmt.Sprintf("Netlify API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Netlify API resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Netlify API error: %s", msg)}
	}
}
