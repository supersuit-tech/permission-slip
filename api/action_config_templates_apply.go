package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/shared"
)

type applyActionConfigTemplateRequest struct {
	AgentID      int64   `json:"agent_id" validate:"gt=0"`
	ApprovalMode *string `json:"approval_mode,omitempty" validate:"omitempty,oneof=auto_approve requires_approval"`
}

type applyActionConfigTemplateResponse struct {
	ActionConfiguration actionConfigResponse      `json:"action_configuration"`
	StandingApproval    *standingApprovalResponse `json:"standing_approval,omitempty"`
}

type standingApprovalTemplateSpec struct {
	DurationDays *int `json:"duration_days"`
}

func handleApplyActionConfigTemplate(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		templateID := strings.TrimSpace(r.PathValue("id"))
		if templateID == "" || len(templateID) > 255 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "template id is required"))
			return
		}

		var req applyActionConfigTemplateRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		tpl, err := db.GetActionConfigTemplateByID(r.Context(), deps.DB, templateID)
		if err != nil {
			log.Printf("[%s] ApplyActionConfigTemplate: get template: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
			return
		}
		if tpl == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrActionConfigTemplateNotFound, "Action configuration template not found"))
			return
		}

		if len(tpl.Name) > shared.ActionConfigNameMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "template name exceeds maximum length"))
			return
		}
		if tpl.Description != nil && len(*tpl.Description) > shared.ActionConfigDescMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "template description exceeds maximum length"))
			return
		}

		params := tpl.Parameters
		if len(params) == 0 {
			params = []byte("{}")
		} else if err := ValidateJSONObject(params); err != nil {
			RespondError(w, r, http.StatusInternalServerError, InternalError("Invalid template parameters"))
			return
		}

		actionType := strings.TrimSpace(tpl.ActionType)
		if actionType != db.WildcardActionType && strings.Contains(actionType, "*") {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid template action_type"))
			return
		}

		if actionType != db.WildcardActionType {
			if err := db.ValidateConfigParameters(params); err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error()))
				return
			}
			exists, err := db.ConnectorActionExists(r.Context(), deps.DB, tpl.ConnectorID, actionType)
			if err != nil {
				log.Printf("[%s] ApplyActionConfigTemplate: connector action: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
				return
			}
			if !exists {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Invalid connector or action type reference"))
				return
			}
		}

		configID, err := generatePrefixedID("ac_", 16)
		if err != nil {
			log.Printf("[%s] ApplyActionConfigTemplate: generate id: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
			return
		}

		var standingBytes []byte
		var spec standingApprovalTemplateSpec
		templateHasStanding := len(tpl.StandingApprovalSpec) > 0 && string(tpl.StandingApprovalSpec) != "null"
		wantStanding := templateHasStanding // default: follow template
		if req.ApprovalMode != nil {
			switch *req.ApprovalMode {
			case "auto_approve":
				wantStanding = true
			case "requires_approval":
				wantStanding = false
			}
		}
		if wantStanding && templateHasStanding {
			if err := json.Unmarshal(tpl.StandingApprovalSpec, &spec); err != nil {
				log.Printf("[%s] ApplyActionConfigTemplate: parse standing spec: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
				return
			}
		}
		if wantStanding {
			standingBytes, err = buildStandingApprovalConstraintsFromTemplate(params)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error()))
				return
			}
		}

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] ApplyActionConfigTemplate: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		enabled, err := db.AgentConnectorEnabled(r.Context(), tx, req.AgentID, profile.ID, tpl.ConnectorID)
		if err != nil {
			log.Printf("[%s] ApplyActionConfigTemplate: agent connector check: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
			return
		}
		if !enabled {
			if _, err := db.EnableAgentConnector(r.Context(), tx, req.AgentID, profile.ID, tpl.ConnectorID); err != nil {
				var acErr *db.AgentConnectorError
				if errors.As(err, &acErr) && acErr.Code == db.AgentConnectorErrAgentNotFound {
					RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
					return
				}
				if errors.As(err, &acErr) && acErr.Code == db.AgentConnectorErrConnectorNotFound {
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Invalid connector reference"))
					return
				}
				log.Printf("[%s] ApplyActionConfigTemplate: enable connector: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
				return
			}
		}

		ac, err := db.CreateActionConfig(r.Context(), tx, db.CreateActionConfigParams{
			ID:          configID,
			AgentID:     req.AgentID,
			UserID:      profile.ID,
			ConnectorID: tpl.ConnectorID,
			ActionType:  actionType,
			Parameters:  params,
			Name:        tpl.Name,
			Description: tpl.Description,
		})
		if err != nil {
			var acErr *db.ActionConfigError
			if errors.As(err, &acErr) {
				switch acErr.Code {
				case db.ActionConfigErrAgentNotFound:
					RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found or not owned by user"))
					return
				case db.ActionConfigErrInvalidRef:
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Invalid connector, action type, or credential reference"))
					return
				}
			}
			log.Printf("[%s] ApplyActionConfigTemplate: create config: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
			return
		}

		var saOut *db.StandingApproval
		if wantStanding {
			if err := db.AcquireStandingApprovalLimitLock(r.Context(), tx, profile.ID); err != nil {
				log.Printf("[%s] ApplyActionConfigTemplate: advisory lock: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
				return
			}
			if checkStandingApprovalLimit(r.Context(), w, r, tx, profile.ID) {
				return
			}

			var expiresAt *time.Time
			startsAt := time.Now().UTC()
			if spec.DurationDays != nil {
				if *spec.DurationDays <= 0 {
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "template standing_approval has invalid duration_days"))
					return
				}
				t := startsAt.Add(time.Duration(*spec.DurationDays) * 24 * time.Hour)
				expiresAt = &t
			}

			saID, err := generatePrefixedID("sa_", 16)
			if err != nil {
				log.Printf("[%s] ApplyActionConfigTemplate: generate sa id: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
				return
			}

			srcID := ac.ID
			sa, err := db.CreateStandingApproval(r.Context(), tx, db.CreateStandingApprovalParams{
				StandingApprovalID:          saID,
				AgentID:                     req.AgentID,
				UserID:                      profile.ID,
				ActionType:                  actionType,
				ActionVersion:               "1",
				Constraints:                 standingBytes,
				SourceActionConfigurationID: &srcID,
				StartsAt:                    startsAt,
				ExpiresAt:                   expiresAt,
			})
			if err != nil {
				var saErr *db.StandingApprovalError
				if errors.As(err, &saErr) && saErr.Code == db.StandingApprovalErrAgentNotFound {
					RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
					return
				}
				log.Printf("[%s] ApplyActionConfigTemplate: create standing approval: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
				return
			}
			saOut = sa
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] ApplyActionConfigTemplate: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
				return
			}
		}

		var linkedSA []db.StandingApproval
		if saOut != nil {
			linkedSA = append(linkedSA, *saOut)
		}
		resp := applyActionConfigTemplateResponse{
			ActionConfiguration: toActionConfigResponseWithLinked(*ac, linkedSA),
		}
		if saOut != nil {
			s := toStandingApprovalResponse(*saOut)
			resp.StandingApproval = &s
		}

		RespondJSON(w, http.StatusCreated, resp)
	}
}

// buildStandingApprovalConstraintsFromTemplate turns template parameter JSON into
// standing-approval constraint JSON (non-wildcard values required by validateStandingApprovalConstraints).
//
// When every parameter value is a bare wildcard ("*") or the object is empty, validation fails
// because standing approvals require at least one non-wildcard constraint. In that case we return
// nil bytes: empty constraints mean "match all" in tryStandingApprovalAutoApprove (see
// api/standing_approval_match.go). A synthetic _scope key would never appear in execution
// parameters and would prevent matching.
func buildStandingApprovalConstraintsFromTemplate(templateParams []byte) ([]byte, error) {
	validated, err := validateStandingApprovalConstraints(json.RawMessage(templateParams))
	if err == nil {
		return validated, nil
	}
	var obj map[string]json.RawMessage
	if jsonErr := json.Unmarshal(templateParams, &obj); jsonErr != nil {
		return nil, fmt.Errorf("template parameters must be a JSON object")
	}
	if obj == nil {
		obj = map[string]json.RawMessage{}
	}
	allBareWildcard := true
	for _, v := range obj {
		var s string
		if json.Unmarshal(v, &s) != nil || s != "*" {
			allBareWildcard = false
			break
		}
	}
	if allBareWildcard {
		return nil, nil
	}
	return nil, fmt.Errorf("standing approval constraints could not be derived from template parameters: %w", err)
}
