package square

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// squareErrorResponse represents the Square API error envelope.
// Square returns errors as: {"errors": [{"category": "...", "code": "...", "detail": "..."}]}
type squareErrorResponse struct {
	Errors []squareError `json:"errors"`
}

type squareError struct {
	Category string `json:"category"`
	Code     string `json:"code"`
	Detail   string `json:"detail"`
	Field    string `json:"field,omitempty"`
}

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. Square errors use a structured
// {"errors": [...]} format with category/code/detail fields.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Parse once and reuse for both message extraction and category checks.
	parsed := parseSquareErrors(body)
	msg := formatErrorMessage(parsed, body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Square API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Square API auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		// Square uses AUTHENTICATION_ERROR category for auth issues on 403.
		if parsedHasCategory(parsed, "AUTHENTICATION_ERROR") {
			return &connectors.AuthError{Message: fmt.Sprintf("Square API auth error: %s", msg)}
		}
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Square API error: %s", msg)}
	case statusCode == http.StatusBadRequest:
		// Square uses INVALID_REQUEST_ERROR for validation failures.
		return &connectors.ValidationError{Message: fmt.Sprintf("Square API validation error: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Square API error: %s", msg)}
	}
}

// maxErrorBodyLen caps how much of the raw response body is included in error
// messages when the structured error envelope can't be parsed. Prevents large
// or unexpected response data from leaking into logs and user-facing errors.
const maxErrorBodyLen = 512

// parseSquareErrors attempts to unmarshal the Square error envelope.
// Returns nil if the body isn't valid JSON or has no errors.
func parseSquareErrors(body []byte) []squareError {
	var resp squareErrorResponse
	if json.Unmarshal(body, &resp) != nil || len(resp.Errors) == 0 {
		return nil
	}
	return resp.Errors
}

// formatErrorMessage builds a human-readable message from parsed Square errors.
// Falls back to a truncated raw body when errors is nil (unparseable response).
func formatErrorMessage(errs []squareError, rawBody []byte) string {
	if errs == nil {
		return truncateBody(rawBody)
	}

	details := make([]string, 0, len(errs))
	for _, e := range errs {
		part := e.Detail
		if part == "" {
			part = e.Code
		}
		if part == "" {
			continue
		}
		// Prefix with the field name when available so developers can
		// immediately see which parameter caused the error.
		if e.Field != "" {
			part = e.Field + ": " + part
		}
		// Append the error code in parentheses when we have both a detail
		// and a code — the code is machine-readable and useful for debugging.
		if e.Detail != "" && e.Code != "" {
			part += " (" + e.Code + ")"
		}
		details = append(details, part)
	}
	if len(details) == 0 {
		return truncateBody(rawBody)
	}
	return strings.Join(details, "; ")
}

// parsedHasCategory checks whether the parsed errors contain a specific category.
func parsedHasCategory(errs []squareError, category string) bool {
	for _, e := range errs {
		if e.Category == category {
			return true
		}
	}
	return false
}

// truncateBody returns the raw body as a string, capped at maxErrorBodyLen.
// This prevents large or unexpected response payloads from leaking into
// error messages that may surface in logs or API responses.
func truncateBody(body []byte) string {
	return connectors.TruncateUTF8(string(body), maxErrorBodyLen)
}
