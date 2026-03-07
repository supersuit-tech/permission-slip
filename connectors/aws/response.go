package aws

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// awsErrorResponse represents the standard AWS XML error response.
type awsErrorResponse struct {
	XMLName xml.Name `xml:"ErrorResponse"`
	Error   struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	} `xml:"Error"`
}

// awsError is a simpler error format used by some AWS services (e.g. S3).
type awsError struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses.
func checkResponse(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to parse AWS XML error response.
	msg := extractErrorMessage(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		return &connectors.RateLimitError{
			Message: fmt.Sprintf("AWS API rate limit exceeded: %s", msg),
		}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("AWS API auth error (403): %s", msg)}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("AWS API auth error (401): %s", msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("AWS API validation error: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("AWS API error: %s", msg)}
	}
}

// extractErrorMessage attempts to parse the AWS error response body.
// It tries the ErrorResponse format first, then the simpler Error format,
// and falls back to the raw body string.
func extractErrorMessage(body []byte) string {
	var errResp awsErrorResponse
	if xml.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		if errResp.Error.Code != "" {
			return fmt.Sprintf("%s: %s", errResp.Error.Code, errResp.Error.Message)
		}
		return errResp.Error.Message
	}

	var simpleErr awsError
	if xml.Unmarshal(body, &simpleErr) == nil && simpleErr.Message != "" {
		if simpleErr.Code != "" {
			return fmt.Sprintf("%s: %s", simpleErr.Code, simpleErr.Message)
		}
		return simpleErr.Message
	}

	return string(body)
}
