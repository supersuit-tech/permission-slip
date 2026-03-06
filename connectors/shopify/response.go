package shopify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. See extractErrorMessage for the
// three Shopify error formats that are parsed.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	msg := extractErrorMessage(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Shopify API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Shopify API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Shopify API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusUnprocessableEntity:
		return &connectors.ValidationError{Message: fmt.Sprintf("Shopify API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Shopify API resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Shopify API error: %s", msg)}
	}
}

// extractErrorMessage tries to parse Shopify's error response formats.
// Shopify can return:
//   - {"errors": "Not Found"}                        (string)
//   - {"errors": {"title": ["can't be blank"]}}      (object of arrays)
//   - {"error": "Not Found"}                         (singular key, used by some endpoints)
//
// Falls back to the raw body if parsing fails.
func extractErrorMessage(body []byte) string {
	// Try {"errors": ...} first (most common).
	var errorsEnvelope struct {
		Errors json.RawMessage `json:"errors"`
	}
	if json.Unmarshal(body, &errorsEnvelope) == nil && len(errorsEnvelope.Errors) > 0 {
		// Try as plain string: {"errors": "Not Found"}
		var s string
		if json.Unmarshal(errorsEnvelope.Errors, &s) == nil && s != "" {
			return s
		}

		// Try as object of arrays: {"errors": {"title": ["can't be blank"]}}
		var fields map[string][]string
		if json.Unmarshal(errorsEnvelope.Errors, &fields) == nil && len(fields) > 0 {
			// Sort field names for deterministic error messages.
			sortedFields := make([]string, 0, len(fields))
			for field := range fields {
				sortedFields = append(sortedFields, field)
			}
			sort.Strings(sortedFields)

			var parts []string
			for _, field := range sortedFields {
				for _, m := range fields[field] {
					parts = append(parts, fmt.Sprintf("%s %s", field, m))
				}
			}
			return strings.Join(parts, "; ")
		}
	}

	// Try {"error": "..."} (singular key, used by some Shopify endpoints).
	var singleError struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &singleError) == nil && singleError.Error != "" {
		return singleError.Error
	}

	// Fallback to raw body.
	if len(body) > 0 {
		return string(body)
	}
	return "unknown error"
}
