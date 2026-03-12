package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// validateActionParameters checks the supplied parameters against the
// connector's parameters_schema for the given action type. It validates
// that all required fields are present.
//
// Behaviour:
//   - Fail-open: if no schema exists for the action type, validation is skipped.
//   - Only required-field validation is performed (not full JSON Schema).
//   - Returns true if validation passed (or was skipped). Returns false if
//     validation failed and an error response was already written.
func validateActionParameters(
	w http.ResponseWriter,
	r *http.Request,
	d db.DBTX,
	actionType string,
	parameters json.RawMessage,
) bool {
	// Look up the schema from the database.
	schema, err := db.GetActionParametersSchema(r.Context(), d, actionType)
	if err != nil {
		log.Printf("[%s] validateActionParameters: schema lookup: %v", TraceID(r.Context()), err)
		// Fail-open on DB errors — don't block the request.
		return true
	}
	if len(schema) == 0 {
		// No schema defined for this action — fail-open.
		return true
	}

	// Parse the schema to extract required fields.
	var schemaDef struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(schema, &schemaDef); err != nil {
		log.Printf("[%s] validateActionParameters: parse schema: %v", TraceID(r.Context()), err)
		// Malformed schema — fail-open.
		return true
	}
	if len(schemaDef.Required) == 0 {
		// No required fields — nothing to validate.
		return true
	}

	// Parse the supplied parameters.
	var params map[string]json.RawMessage
	if len(parameters) > 0 {
		if err := json.Unmarshal(parameters, &params); err != nil {
			// Parameters aren't a valid JSON object — report as missing all required fields.
			params = nil
		}
	}

	// Check for missing required fields.
	var missing []string
	for _, field := range schemaDef.Required {
		val, exists := params[field]
		if !exists || isRawJSONNull(val) {
			missing = append(missing, field)
		}
	}

	if len(missing) == 0 {
		return true
	}

	// Build connector hint from the action type (e.g., "github.create_issue" → "github").
	hint := "see GET /connectors for the full parameter schema"
	if parts := strings.SplitN(actionType, ".", 2); len(parts) == 2 && parts[0] != "" {
		hint = fmt.Sprintf("see GET /connectors/%s for the full parameter schema", parts[0])
	}

	resp := BadRequest(ErrInvalidParameters, "action parameters are missing required fields")
	resp.Error.Details = map[string]any{
		"missing_fields": missing,
		"hint":           hint,
	}
	RespondError(w, r, http.StatusBadRequest, resp)
	return false
}
