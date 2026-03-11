package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// ConfigValidationResult holds the validated action configuration after
// all checks pass. Handlers use this to proceed with execution.
type ConfigValidationResult struct {
	Config *db.ActionConfiguration
}

// ValidateConfigurationReference validates a configuration_id supplied by an
// agent. It performs the full validation chain documented in the OpenAPI spec:
//
//  1. The configuration exists and belongs to this agent
//  2. The configuration is active (not disabled)
//  3. The action type matches the configuration's action_type
//  4. The action parameters comply with the configuration constraints:
//     fixed parameters must match exactly, wildcard parameters accept any value
//
// On success, returns the validated configuration. On failure, writes the
// appropriate HTTP error response and returns nil — the caller should return
// immediately when nil is returned.
func ValidateConfigurationReference(
	w http.ResponseWriter,
	r *http.Request,
	deps *Deps,
	configID string,
	agentID int64,
	actionType string,
	parameters json.RawMessage,
) *ConfigValidationResult {
	// 1. Look up the configuration for this agent.
	ac, err := db.GetActiveActionConfigForAgent(r.Context(), deps.DB, configID, agentID)
	if err != nil {
		log.Printf("[%s] ValidateConfigurationReference: lookup: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to validate configuration"))
		return nil
	}
	if ac == nil {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidConfiguration, "Configuration not found or not active for this agent"))
		return nil
	}

	// 2. Active check is handled by the query (status = 'active'), but we
	// guard defensively in case the query contract changes.
	if ac.Status != "active" {
		RespondError(w, r, http.StatusForbidden, Forbidden(ErrConfigurationDisabled, "Configuration is disabled"))
		return nil
	}

	// 3. Validate action type matches.
	// Wildcard configs (action_type = "*") match any action type.
	if ac.ActionType != db.WildcardActionType && ac.ActionType != actionType {
		resp := BadRequest(ErrConfigActionTypeMismatch, "Action type does not match configuration")
		resp.Error.Details = map[string]any{
			"configuration_action_type": ac.ActionType,
			"request_action_type":       actionType,
		}
		RespondError(w, r, http.StatusBadRequest, resp)
		return nil
	}

	// 4. Validate parameters against configuration constraints.
	// Wildcard configs allow any parameters — skip validation entirely.
	if ac.ActionType != db.WildcardActionType {
		if err := db.ValidateParametersAgainstConfig(ac.Parameters, parameters); err != nil {
			var configErr *db.ConfigValidationError
			if errors.As(err, &configErr) {
				resp := BadRequest(ErrInvalidParameters, "Parameters do not comply with configuration constraints")
				resp.Error.Details = map[string]any{
					"parameter":        configErr.Parameter,
					"constraint_error": configErr.Reason,
				}
				RespondError(w, r, http.StatusBadRequest, resp)
				return nil
			}
			log.Printf("[%s] ValidateConfigurationReference: param validation: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to validate parameters"))
			return nil
		}
	}

	return &ConfigValidationResult{Config: ac}
}
