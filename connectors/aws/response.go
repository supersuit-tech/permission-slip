package aws

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
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

	code := extractErrorCode(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		return &connectors.RateLimitError{
			Message: fmt.Sprintf("AWS API rate limit exceeded: %s — retry after a brief delay", msg),
		}
	case statusCode == http.StatusForbidden:
		hint := "verify that the IAM user/role has the required permissions for this action"
		if code == "InvalidAccessKeyId" || code == "SignatureDoesNotMatch" {
			hint = "check that the access key ID and secret are correct and not expired"
		}
		return &connectors.AuthError{Message: fmt.Sprintf("AWS API auth error (403): %s — %s", msg, hint)}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{
			Message: fmt.Sprintf("AWS API auth error (401): %s — check that credentials are valid and not expired", msg),
		}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("AWS API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{
			Message: fmt.Sprintf("AWS resource not found: %s — verify the resource ID, region, and that the resource exists", msg),
		}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("AWS API error: %s", msg)}
	}
}

// extractErrorCode returns the AWS error code from the response body, or "".
func extractErrorCode(body []byte) string {
	var errResp awsErrorResponse
	if xml.Unmarshal(body, &errResp) == nil && errResp.Error.Code != "" {
		return errResp.Error.Code
	}
	var simpleErr awsError
	if xml.Unmarshal(body, &simpleErr) == nil && simpleErr.Code != "" {
		return simpleErr.Code
	}
	return ""
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
