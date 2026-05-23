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
	MaxExecutions               *int            `json:"max_executions,omitempty" validate:"omitempty,gt=0"`
	ExpiresInSeconds            *int            `json:"expires_in_seconds,omitempty" validate:"omitempty,gt=0"`
	SourceActionConfigurationID *string         `json:"source_action_configuration_id"`
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

		constraintsBytes, err := validateStandingApprovalConstraints(req.Constraints)
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

		requestID, err := generatePrefixedID("sar_", 16)
		if err != nil {
			log.Printf("[%s] CreateStandingApprovalRequest: generate ID: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval request"))
			return
		}

		sar, err := db.InsertStandingApprovalRequest(r.Context(), deps.DB, db.InsertStandingApprovalRequestParams{
			RequestID:                   requestID,
			AgentID:                     agent.AgentID,
			UserID:                      agent.ApproverID,
			ActionType:                  req.ActionType,
			ActionVersion:               req.ActionVersion,
			Constraints:                 constraintsBytes,
			MaxExecutions:               req.MaxExecutions,
			ExpiresInSeconds:            req.ExpiresInSeconds,
			SourceActionConfigurationID: req.SourceActionConfigurationID,
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
