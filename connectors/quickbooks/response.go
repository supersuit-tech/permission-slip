package quickbooks

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// qboFault represents the error envelope returned by the QuickBooks Online API.
// See https://developer.intuit.com/app/developer/qbo/docs/develop/troubleshooting/error-codes
type qboFault struct {
	Fault struct {
		Error []struct {
			Message string `json:"Message"`
			Detail  string `json:"Detail"`
			Code    string `json:"code"`
		} `json:"Error"`
		Type string `json:"type"`
	} `json:"Fault"`
}

// checkResponse inspects the HTTP status code and QuickBooks error body,
// returning an appropriate typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract QuickBooks' structured fault.
	var fault qboFault
	msg := truncate(string(body), maxErrorMessageBytes)
	if json.Unmarshal(body, &fault) == nil && len(fault.Fault.Error) > 0 {
		first := fault.Fault.Error[0]
		msg = first.Message
		if first.Detail != "" {
			msg = fmt.Sprintf("%s: %s", msg, first.Detail)
		}
		if first.Code != "" {
			msg = fmt.Sprintf("%s (code: %s)", msg, first.Code)
		}
	}

	// Rate limit.
	if statusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("QuickBooks API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	}

	// Map by HTTP status code.
	switch {
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("QuickBooks auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("QuickBooks auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("QuickBooks validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("QuickBooks validation error (resource not found): %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("QuickBooks API error: %s", msg)}
	}
}
