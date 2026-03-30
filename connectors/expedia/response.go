package expedia

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. Expedia Rapid API returns errors
// as {"type": "...", "message": "..."}.
//
// Error mapping rationale:
//   - 400 → ValidationError: Expedia uses 400 for invalid parameters (unlike
//     GitHub which uses 422), so we map it to ValidationError.
//   - 401/403 → AuthError: Invalid or expired API key/signature.
//   - 404 → ValidationError: Resource not found (property, room, or itinerary).
//   - 429 → RateLimitError: With Retry-After header when available.
//   - Other 4xx/5xx → ExternalError: Includes HTTP status code for debugging.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract Expedia's error message.
	var rapidErr struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	msg := string(body)
	if json.Unmarshal(body, &rapidErr) == nil && rapidErr.Message != "" {
		msg = rapidErr.Message
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Expedia Rapid API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Expedia Rapid API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Expedia Rapid API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Expedia Rapid API resource not found: %s — verify the property, room, or itinerary ID", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Expedia Rapid API error: %s", msg)}
	}
}
