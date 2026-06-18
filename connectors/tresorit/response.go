package tresorit

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type s3ErrorResponse struct {
	XMLName xml.Name `xml:"ErrorResponse"`
	Error   struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	} `xml:"Error"`
}

type s3Error struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

func checkResponse(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	msg := extractErrorMessage(body)
	code := extractErrorCode(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		return &connectors.RateLimitError{
			Message: fmt.Sprintf("Tresorit gateway rate limit exceeded: %s", msg),
		}
	case statusCode == http.StatusForbidden:
		hint := "verify access_key and secret_key from credentials.json"
		if code == "InvalidAccessKeyId" || code == "SignatureDoesNotMatch" {
			hint = "check that access_key and secret_key match the gateway credentials"
		}
		return &connectors.AuthError{Message: fmt.Sprintf("Tresorit gateway auth error (403): %s — %s", msg, hint)}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{
			Message: fmt.Sprintf("Tresorit gateway auth error (401): %s", msg),
		}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Tresorit gateway validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{
			Message: fmt.Sprintf("Tresorit resource not found: %s — verify the tresor name and file path", msg),
		}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Tresorit gateway error: %s", msg)}
	}
}

func extractErrorCode(body []byte) string {
	var errResp s3ErrorResponse
	if xml.Unmarshal(body, &errResp) == nil && errResp.Error.Code != "" {
		return errResp.Error.Code
	}
	var simpleErr s3Error
	if xml.Unmarshal(body, &simpleErr) == nil && simpleErr.Code != "" {
		return simpleErr.Code
	}
	return ""
}

func extractErrorMessage(body []byte) string {
	var errResp s3ErrorResponse
	if xml.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		if errResp.Error.Code != "" {
			return fmt.Sprintf("%s: %s", errResp.Error.Code, errResp.Error.Message)
		}
		return errResp.Error.Message
	}

	var simpleErr s3Error
	if xml.Unmarshal(body, &simpleErr) == nil && simpleErr.Message != "" {
		if simpleErr.Code != "" {
			return fmt.Sprintf("%s: %s", simpleErr.Code, simpleErr.Message)
		}
		return simpleErr.Message
	}

	return string(body)
}
