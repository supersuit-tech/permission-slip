package docusign

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// maxErrorBodyLen is the maximum number of bytes from a response body to
// include in an error message. Prevents log bloat from large responses.
const maxErrorBodyLen = 512

// truncateBody returns the response body as a string, truncated to
// maxErrorBodyLen characters to prevent oversized error messages.
func truncateBody(body []byte) string {
	return connectors.TruncateUTF8(string(body), maxErrorBodyLen)
}

// docuSignAPIError represents the standard DocuSign API error response.
type docuSignAPIError struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

// tryParseDocuSignError attempts to parse a DocuSign structured error from a
// response body. Returns the parsed error and true if successful, or a zero
// value and false if the body doesn't contain a recognized DocuSign error.
func tryParseDocuSignError(body []byte) (docuSignAPIError, bool) {
	var apiErr docuSignAPIError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.ErrorCode != "" {
		return apiErr, true
	}
	return docuSignAPIError{}, false
}

// mapDocuSignError converts a DocuSign API error to the appropriate connector
// error type. Provides actionable error messages for common DocuSign error codes
// so agents and users understand what went wrong and how to fix it.
func mapDocuSignError(statusCode int, apiErr docuSignAPIError) error {
	switch apiErr.ErrorCode {
	// Auth errors
	case "AUTHORIZATION_INVALID_TOKEN", "USER_AUTHENTICATION_FAILED",
		"ACCOUNT_NOT_AUTHORIZED", "USER_NOT_AUTHORIZED_FOR_ACCOUNT":
		return &connectors.AuthError{Message: fmt.Sprintf("DocuSign auth error: %s", apiErr.Message)}

	// Envelope state errors — include hints about what state is expected
	case "ENVELOPE_NOT_IN_CORRECT_STATE":
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("DocuSign: envelope is not in the correct state for this operation — %s. Check the envelope status with docusign.check_status first.", apiErr.Message),
		}
	case "ENVELOPE_IS_INCOMPLETE":
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("DocuSign: envelope is missing required information — %s. Ensure all recipients and document fields are filled.", apiErr.Message),
		}

	// Template errors
	case "TEMPLATE_NOT_FOUND":
		return &connectors.ValidationError{
			Message: "DocuSign: template not found. Use docusign.list_templates to browse available templates.",
		}

	// Recipient errors
	case "RECIPIENT_NOT_IN_SEQUENCE":
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("DocuSign: recipient routing order conflict — %s. Ensure routing_order values are sequential.", apiErr.Message),
		}
	case "INVALID_EMAIL_ADDRESS_FOR_RECIPIENT":
		return &connectors.ValidationError{
			Message: fmt.Sprintf("DocuSign: invalid email address for recipient — %s", apiErr.Message),
		}

	// Resource not found
	case "ENVELOPE_DOES_NOT_EXIST":
		return &connectors.ValidationError{
			Message: "DocuSign: envelope not found. Verify the envelope_id is correct.",
		}

	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("DocuSign API error: %s — %s", apiErr.ErrorCode, apiErr.Message),
		}
	}
}
