package microsoft

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// graphError represents a Microsoft Graph API error response.
// See: https://learn.microsoft.com/en-us/graph/errors
type graphError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// mapGraphError converts a Microsoft Graph API error response to the
// appropriate connector error type with actionable messages.
//
// Mapping:
//
//	401 → AuthError (token expired/invalid — suggest reconnecting)
//	403 → AuthError (insufficient scopes — suggest re-authorizing)
//	400 → ValidationError (bad request — surface the Graph error message)
//	404 → ExternalError (resource not found)
//	other → ExternalError (generic Graph API failure)
func mapGraphError(statusCode int, body []byte) error {
	var ge graphError
	if json.Unmarshal(body, &ge) != nil || ge.Error.Message == "" {
		ge.Error.Message = string(body)
	}

	code := ge.Error.Code

	switch statusCode {
	case http.StatusUnauthorized:
		return &connectors.AuthError{
			Message: fmt.Sprintf("Microsoft Graph authentication failed (%s): %s — the user may need to reconnect their Microsoft account", code, ge.Error.Message),
		}
	case http.StatusForbidden:
		return &connectors.AuthError{
			Message: fmt.Sprintf("Microsoft Graph permission denied (%s): %s — ensure the required OAuth scopes are granted", code, ge.Error.Message),
		}
	case http.StatusNotFound:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Microsoft Graph resource not found (%s): %s", code, ge.Error.Message),
		}
	case http.StatusBadRequest:
		return &connectors.ValidationError{
			Message: fmt.Sprintf("Microsoft Graph rejected the request (%s): %s", code, ge.Error.Message),
		}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Microsoft Graph API error (%d/%s): %s", statusCode, code, ge.Error.Message),
		}
	}
}
