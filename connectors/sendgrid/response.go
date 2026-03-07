package sendgrid

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract SendGrid's error message.
	// SendGrid v3 returns: {"errors": [{"message": "...", "field": "...", "help": "..."}]}
	var sgErr struct {
		Errors []struct {
			Message string `json:"message"`
			Field   string `json:"field"`
		} `json:"errors"`
	}

	const maxErrBody = 512
	msg := string(body)
	if len(msg) > maxErrBody {
		msg = msg[:maxErrBody] + "...(truncated)"
	}

	if json.Unmarshal(body, &sgErr) == nil && len(sgErr.Errors) > 0 {
		first := sgErr.Errors[0]
		if first.Field != "" {
			msg = fmt.Sprintf("%s (field: %s)", first.Message, first.Field)
		} else {
			msg = first.Message
		}
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("SendGrid API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("SendGrid API auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("SendGrid API forbidden: %s", msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("SendGrid API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("SendGrid API resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("SendGrid API error: %s", msg)}
	}
}
