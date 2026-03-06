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

	msg := extractErrorMessage(body)

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
		if hasErrorCategory(body, "AUTHENTICATION_ERROR") {
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

// extractErrorMessage builds a human-readable message from the Square error
// response. It joins the detail fields from all errors in the array.
func extractErrorMessage(body []byte) string {
	var sqErr squareErrorResponse
	if json.Unmarshal(body, &sqErr) != nil || len(sqErr.Errors) == 0 {
		// Fall back to raw body if we can't parse the error envelope.
		return string(body)
	}

	details := make([]string, 0, len(sqErr.Errors))
	for _, e := range sqErr.Errors {
		if e.Detail != "" {
			details = append(details, e.Detail)
		} else if e.Code != "" {
			details = append(details, e.Code)
		}
	}
	if len(details) == 0 {
		return string(body)
	}
	return strings.Join(details, "; ")
}

// hasErrorCategory checks whether the Square error response contains an
// error with the given category.
func hasErrorCategory(body []byte, category string) bool {
	var sqErr squareErrorResponse
	if json.Unmarshal(body, &sqErr) != nil {
		return false
	}
	for _, e := range sqErr.Errors {
		if e.Category == category {
			return true
		}
	}
	return false
}
