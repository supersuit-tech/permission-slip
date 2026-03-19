package microsoft

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

// serverSideGraphErrorCodes is the set of Microsoft Graph error codes that
// indicate a server-side or infrastructure issue rather than a bad client
// request. Even when Graph returns HTTP 400 for these, they represent
// unexpected service conditions (file not yet ready, internal service errors,
// transient locks) and should be reported to Sentry as ExternalErrors rather
// than silently treated as client ValidationErrors.
//
// Reference: https://learn.microsoft.com/en-us/graph/errors
// All keys are lowercase; the lookup normalises the incoming code with
// strings.ToLower so that variant capitalisations (e.g. "FileCorruptTryRepair"
// observed in the wild) are handled without maintaining paired entries.
var serverSideGraphErrorCodes = map[string]bool{
	// File is not yet ready for editing (e.g. newly created workbook).
	"filecorrupttryrepair": true,
	// Transient service unavailability / internal errors.
	"servicenotavailable": true,
	"generalexception":    true,
	"notsupported":        true,
	"resourcemodified":    true,
	"lockmismatch":        true,
	"editmoderequired":    true,
}

// mapGraphError converts a Microsoft Graph API error response to the
// appropriate connector error type with actionable messages.
//
// Mapping:
//
//	401 → AuthError (token expired/invalid — suggest reconnecting)
//	403 → AuthError (insufficient scopes — suggest re-authorizing)
//	400 + server-side error code → ExternalError (Graph service issue, not client error)
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
		// Some Graph error codes indicate server-side / transient issues even on
		// HTTP 400. Map those to ExternalError so they are captured in Sentry
		// rather than silently treated as client validation errors.
		if serverSideGraphErrorCodes[strings.ToLower(code)] {
			return &connectors.ExternalError{
				StatusCode: statusCode,
				Message:    fmt.Sprintf("Microsoft Graph service error (%s): %s", code, ge.Error.Message),
			}
		}
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
