package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/shared"
)

type agentStandingApprovalRequestBody struct {
	ActionType                  string          `json:"action_type" validate:"required"`
	ActionVersion               string          `json:"action_version"`
	Constraints                 json.RawMessage `json:"constraints" validate:"required"`
	SourceActionConfigurationID *string         `json:"source_action_configuration_id"`
	ConnectorInstance           *string         `json:"connector_instance"`
}

type agentStandingApprovalRequestResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
}

func init() {
	RegisterRouteGroup(RegisterAgentStandingApprovalRequestRoutes)
}

// RegisterAgentStandingApprovalRequestRoutes adds agent-signed standing approval proposal endpoint.
func RegisterAgentStandingApprovalRequestRoutes(mux *http.ServeMux, deps *Deps) {
	requireAgent := RequireAgentSignature(deps)
	mux.Handle("POST /standing-approvals/request", requireAgent(handleAgentCreateStandingApprovalRequest(deps)))
}

func handleAgentCreateStandingApprovalRequest(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())

		var req agentStandingApprovalRequestBody
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		req.ActionType = strings.TrimSpace(req.ActionType)
		if !ValidateRequest(w, r, &req) {
			return
		}

		if len(req.ActionType) > shared.ActionTypeMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action_type exceeds maximum length"))
			return
		}
		if len(req.ActionVersion) > shared.ActionVersionMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action_version exceeds maximum length"))
			return
		}
		if len(req.Constraints) > shared.MaxConstraintsBytes {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "constraints exceeds maximum size"))
			return
		}

		if req.ActionVersion == "" {
			req.ActionVersion = "1"
		} else if !actionVersionPattern.MatchString(req.ActionVersion) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action_version must contain only digits"))
			return
		}

		constraintsBytes, err := validateStandingApprovalConstraintsForAction(r.Context(), deps.DB, deps.Connectors, req.ActionType, req.Constraints)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidConstraints, err.Error()))
			return
		}

		if req.SourceActionConfigurationID != nil {
			id := strings.TrimSpace(*req.SourceActionConfigurationID)
			if id == "" {
				req.SourceActionConfigurationID = nil
			} else {
				req.SourceActionConfigurationID = &id
				if len(id) > maxActionConfigIDLength {
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "source_action_configuration_id exceeds maximum length"))
					return
				}
			}
		}

		var connectorInstanceSelector string
		if req.ConnectorInstance != nil {
			connectorInstanceSelector = strings.TrimSpace(*req.ConnectorInstance)
			if connectorInstanceSelector == "" {
				req.ConnectorInstance = nil
			}
		}

		requestID, err := generatePrefixedID("sar_", 16)
		if err != nil {
			log.Printf("[%s] CreateStandingApprovalRequest: generate ID: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval request"))
			return
		}

		display := resolveStandingApprovalRequestDisplay(
			r.Context(), deps.DB, agent.AgentID, agent.ApproverID,
			req.ActionType, req.SourceActionConfigurationID,
		)
		var connectorName, connectorInstanceID, connectorInstanceDisplay *string
		if display.ConnectorName != "" {
			connectorName = &display.ConnectorName
		}
		if display.ConnectorInstanceDisplay != "" {
			connectorInstanceDisplay = &display.ConnectorInstanceDisplay
		}

		if connectorInstanceSelector != "" {
			resolved, err := resolveConnectorInstanceForStandingApprovalRequest(
				r.Context(), deps.DB, agent.AgentID, agent.ApproverID,
				req.ActionType, connectorInstanceSelector,
			)
			if err != nil {
				if respondConnectorInstanceResolutionError(w, r, err) {
					return
				}
				log.Printf("[%s] resolveConnectorInstanceForStandingApprovalRequest: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval request"))
				return
			}
			if resolved != nil {
				connectorInstanceID = resolved.ConnectorInstanceID
				if resolved.ConnectorInstanceDisplay != nil {
					connectorInstanceDisplay = resolved.ConnectorInstanceDisplay
				}
			}

			if req.SourceActionConfigurationID != nil {
				ac, err := db.GetActionConfigByID(r.Context(), deps.DB, *req.SourceActionConfigurationID, agent.ApproverID)
				if err != nil {
					log.Printf("[%s] GetActionConfigByID for instance conflict check: %v", TraceID(r.Context()), err)
					CaptureError(r.Context(), err)
					RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval request"))
					return
				}
				if ac == nil || ac.AgentID != agent.AgentID {
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "source_action_configuration_id not found"))
					return
				}
				if connectorInstanceID != nil {
					if err := validateStandingApprovalRequestInstanceAgainstPinnedConfig(
						r.Context(), deps.DB, agent.AgentID, agent.ApproverID, ac, *connectorInstanceID,
					); err != nil {
						if respondConnectorInstanceResolutionError(w, r, err) {
							return
						}
						log.Printf("[%s] validateStandingApprovalRequestInstanceAgainstPinnedConfig: %v", TraceID(r.Context()), err)
						CaptureError(r.Context(), err)
						RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval request"))
						return
					}
				}
			}
		}

		sar, err := db.InsertStandingApprovalRequest(r.Context(), deps.DB, db.InsertStandingApprovalRequestParams{
			RequestID:                   requestID,
			AgentID:                     agent.AgentID,
			UserID:                      agent.ApproverID,
			ActionType:                  req.ActionType,
			ActionVersion:               req.ActionVersion,
			Constraints:                 constraintsBytes,
			SourceActionConfigurationID: req.SourceActionConfigurationID,
			ConnectorName:               connectorName,
			ConnectorInstanceID:         connectorInstanceID,
			ConnectorInstanceDisplay:    connectorInstanceDisplay,
		})
		if err != nil {
			log.Printf("[%s] InsertStandingApprovalRequest: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval request"))
			return
		}

		approver, err := db.GetProfileByUserID(r.Context(), deps.DB, agent.ApproverID)
		if err != nil {
			log.Printf("[%s] GetProfileByUserID for notify: %v", TraceID(r.Context()), err)
		} else if approver != nil {
			NotifyStandingApprovalRequest(r.Context(), deps, sar, agent, approver)
		}

		RespondJSON(w, http.StatusOK, agentStandingApprovalRequestResponse{
			RequestID: sar.RequestID,
			Status:    sar.Status,
		})
	}
}
