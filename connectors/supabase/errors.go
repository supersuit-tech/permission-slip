package supabase

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// postgrestErrorResponse is the error envelope from PostgREST.
type postgrestErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Details string `json:"details"`
	Hint    string `json:"hint"`
}

// mapSupabaseError converts a PostgREST error response to the appropriate
// connector error type.
func mapSupabaseError(statusCode int, body []byte) error {
	var errResp postgrestErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		snippet := connectors.TruncateUTF8(string(body), 500)
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Supabase PostgREST error (HTTP %d): %s", statusCode, snippet),
		}
	}

	msg := fmt.Sprintf("Supabase PostgREST error: %s", errResp.Message)
	if errResp.Details != "" {
		msg += " — " + errResp.Details
	}
	if errResp.Hint != "" {
		msg += " (hint: " + errResp.Hint + ")"
	}

	// Map PostgreSQL/PostgREST error codes to connector error types.
	switch {
	case errResp.Code == "PGRST301" || errResp.Code == "PGRST302":
		// JWT expired or invalid
		return &connectors.AuthError{Message: msg}
	case errResp.Code == "42501":
		// insufficient_privilege
		return &connectors.AuthError{Message: msg}
	case strings.HasPrefix(errResp.Code, "PGRST"):
		// Other PostgREST-specific errors are validation issues
		return &connectors.ValidationError{Message: msg}
	case errResp.Code == "42P01":
		// undefined_table
		return &connectors.ValidationError{Message: msg}
	case errResp.Code == "42703":
		// undefined_column
		return &connectors.ValidationError{Message: msg}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: msg}
	}
}
