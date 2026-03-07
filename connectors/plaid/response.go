package plaid

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses.
func checkResponse(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Plaid error response format.
	var plaidErr struct {
		ErrorType    string `json:"error_type"`
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
		DisplayMsg   string `json:"display_message"`
	}

	const maxErrBody = 512
	msg := string(body)
	if len(msg) > maxErrBody {
		msg = msg[:maxErrBody] + "...(truncated)"
	}
	if json.Unmarshal(body, &plaidErr) == nil && plaidErr.ErrorMessage != "" {
		msg = fmt.Sprintf("[%s/%s] %s", plaidErr.ErrorType, plaidErr.ErrorCode, plaidErr.ErrorMessage)
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		return &connectors.RateLimitError{
			Message: fmt.Sprintf("Plaid API rate limit exceeded: %s", msg),
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Plaid API auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Plaid API forbidden: %s", msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Plaid API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Plaid API resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Plaid API error: %s", msg)}
	}
}
