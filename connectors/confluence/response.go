package confluence

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. Confluence Cloud returns structured
// error bodies with a "message" field and optional "statusCode".
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	msg := extractErrorMessage(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Confluence API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Confluence API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity:
		return &connectors.ValidationError{Message: fmt.Sprintf("Confluence API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Confluence API resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Confluence API error: %s", msg)}
	}
}

// extractErrorMessage tries to parse a Confluence error response body.
// Confluence v2 returns errors as: {"statusCode": N, "message": "..."}
// or sometimes: {"errors": [{"status": N, "title": "..."}]}
func extractErrorMessage(body []byte) string {
	if len(body) == 0 {
		return "(no error details returned)"
	}

	var cfErr struct {
		Message    string `json:"message"`
		StatusCode int    `json:"statusCode"`
		Errors     []struct {
			Status int    `json:"status"`
			Title  string `json:"title"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &cfErr) != nil {
		return truncate(string(body), 200)
	}

	if cfErr.Message != "" {
		return cfErr.Message
	}

	if len(cfErr.Errors) > 0 && cfErr.Errors[0].Title != "" {
		return cfErr.Errors[0].Title
	}

	return truncate(string(body), 200)
}

// truncate shortens s to maxLen characters, appending an ellipsis if truncated.
func truncate(s string, maxLen int) string {
	return connectors.TruncateUTF8(s, maxLen)
}
