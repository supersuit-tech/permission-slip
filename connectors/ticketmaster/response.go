package ticketmaster

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// checkResponse maps Discovery API HTTP status codes to typed connector errors.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	msg := extractErrorMessage(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), time.Second)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Ticketmaster Discovery API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Ticketmaster Discovery API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Ticketmaster Discovery API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Ticketmaster Discovery API resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Ticketmaster Discovery API error: %s", msg),
		}
	}
}

func extractErrorMessage(body []byte) string {
	var env struct {
		Fault struct {
			FaultString string `json:"faultstring"`
		} `json:"fault"`
		Errors []struct {
			Detail string `json:"detail"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &env) == nil {
		if env.Fault.FaultString != "" {
			return env.Fault.FaultString
		}
		if len(env.Errors) > 0 && env.Errors[0].Detail != "" {
			return env.Errors[0].Detail
		}
	}
	s := string(body)
	if len(s) > 500 {
		s = s[:500] + "... (truncated)"
	}
	return s
}
