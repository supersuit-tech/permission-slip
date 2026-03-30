package pagerduty

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	var pdErr struct {
		Error struct {
			Message string   `json:"message"`
			Errors  []string `json:"errors"`
		} `json:"error"`
	}
	msg := string(body)
	if json.Unmarshal(body, &pdErr) == nil && pdErr.Error.Message != "" {
		msg = pdErr.Error.Message
		if len(pdErr.Error.Errors) > 0 {
			msg = fmt.Sprintf("%s: %s", pdErr.Error.Message, pdErr.Error.Errors[0])
		}
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("PagerDuty API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("PagerDuty API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("PagerDuty resource not found: %s", msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("PagerDuty API validation error: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("PagerDuty API error: %s", msg)}
	}
}
