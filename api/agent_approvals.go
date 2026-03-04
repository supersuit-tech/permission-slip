package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/shared"
)

// ── Request/response types ─────────────────────────────────────────────────

type agentRequestApprovalRequest struct {
	RequestID     string          `json:"request_id" validate:"required"`
	Action        json.RawMessage `json:"action" validate:"required"`
	Context       json.RawMessage `json:"context" validate:"required"`
	ExpiresIn     *int            `json:"expires_in,omitempty" validate:"omitempty,gte=60,lte=86400"`
	Configuration *agentApprovalConfigRef `json:"configuration,omitempty"`
}

type agentApprovalConfigRef struct {
	ConfigurationID string `json:"configuration_id" validate:"required"`
}

type agentRequestApprovalResponse struct {
	ApprovalID  string    `json:"approval_id"`
	ApprovalURL string    `json:"approval_url"`
	Status      string    `json:"status"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type agentCancelApprovalResponse struct {
	ApprovalID  string    `json:"approval_id"`
	Status      string    `json:"status"`
	CancelledAt time.Time `json:"cancelled_at"`
}

// ── Route registration ─────────────────────────────────────────────────────

// RegisterAgentApprovalRoutes adds agent-authenticated approval endpoints.
func RegisterAgentApprovalRoutes(mux *http.ServeMux, deps *Deps) {
	requireAgent := RequireAgentSignature(deps)
	mux.Handle("POST /approvals/request", requireAgent(handleAgentRequestApproval(deps)))
	mux.Handle("POST /approvals/{approval_id}/cancel", requireAgent(handleAgentCancelApproval(deps)))
}

// ── POST /approvals/request ────────────────────────────────────────────────

func handleAgentRequestApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())

		var req agentRequestApprovalRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		req.RequestID = strings.TrimSpace(req.RequestID)
		if !ValidateRequest(w, r, &req) {
			return
		}

		if len(req.RequestID) > 255 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "request_id exceeds maximum length"))
			return
		}

		// Validate action and context as non-null JSON objects.
		// ValidateJSONObject treats null as absent (for optional fields), so
		// we reject it explicitly here since both fields are required.
		if isRawJSONNull(req.Action) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action must be a JSON object"))
			return
		}
		if err := ValidateJSONObject(req.Action); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action must be a JSON object"))
			return
		}
		if isRawJSONNull(req.Context) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "context must be a JSON object"))
			return
		}
		if err := ValidateJSONObject(req.Context); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "context must be a JSON object"))
			return
		}

		// Validate action/context sizes.
		if len(req.Action) > 65536 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action exceeds maximum size"))
			return
		}
		if len(req.Context) > 65536 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "context exceeds maximum size"))
			return
		}

		// Validate action has a "type" field.
		var actionObj map[string]json.RawMessage
		if err := json.Unmarshal(req.Action, &actionObj); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action must be a JSON object"))
			return
		}
		typeRaw, hasType := actionObj["type"]
		if !hasType {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action.type is required"))
			return
		}
		var actionType string
		if err := json.Unmarshal(typeRaw, &actionType); err != nil || strings.TrimSpace(actionType) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action.type must be a non-empty string"))
			return
		}
		if len(actionType) > shared.ActionTypeMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action.type exceeds maximum length"))
			return
		}

		// Optional: validate configuration reference.
		if req.Configuration != nil {
			// ValidateConfigurationReference expects action.parameters, not
			// the full action object. Extract it from the already-parsed map.
			actionParams := json.RawMessage(actionObj["parameters"])
			result := ValidateConfigurationReference(w, r, deps, req.Configuration.ConfigurationID, agent.AgentID, actionType, actionParams)
			if result == nil {
				return // error already written
			}
		}

		// Check monthly request quota before creating the approval.
		var blocked bool
		r, blocked = checkRequestQuota(r.Context(), w, r, deps.DB, agent.ApproverID)
		if blocked {
			return
		}

		// Compute expiration.
		expiresAt := time.Now().UTC().Add(db.DefaultApprovalTTL)
		if req.ExpiresIn != nil {
			expiresAt = time.Now().UTC().Add(time.Duration(*req.ExpiresIn) * time.Second)
		}

		// Generate approval ID.
		approvalID, err := generatePrefixedID("appr_", 16)
		if err != nil {
			log.Printf("[%s] AgentRequestApproval: generate ID: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create approval"))
			return
		}

		// Look up the approver profile (the agent's owner).
		approverProfile, err := db.GetProfileByUserID(r.Context(), deps.DB, agent.ApproverID)
		if err != nil {
			log.Printf("[%s] AgentRequestApproval: profile lookup: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create approval"))
			return
		}
		if approverProfile == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApproverNotFound, "Approver profile not found"))
			return
		}

		approval, err := db.InsertApproval(r.Context(), deps.DB, db.InsertApprovalParams{
			ApprovalID: approvalID,
			AgentID:    agent.AgentID,
			ApproverID: agent.ApproverID,
			Action:     req.Action,
			Context:    req.Context,
			ExpiresAt:  expiresAt,
		}, req.RequestID)
		if err != nil {
			var apprErr *db.ApprovalError
			if errors.As(err, &apprErr) && apprErr.Code == db.ApprovalErrDuplicateRequest {
				RespondError(w, r, http.StatusConflict, Conflict(ErrDuplicateRequestID, "A request with this request_id has already been submitted"))
				return
			}
			log.Printf("[%s] AgentRequestApproval: insert: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create approval"))
			return
		}

		// Update agent's last_active_at (best-effort).
		if err := db.TouchAgentLastActive(r.Context(), deps.DB, agent.AgentID); err != nil {
			log.Printf("agent_approvals: failed to update last_active_at for agent %d: %v", agent.AgentID, err)
		}

		// Fire notification to approver (best-effort, async).
		NotifyApprovalRequest(r.Context(), deps, approval, agent, approverProfile)

		// Notify any connected SSE clients for this approver.
		notifyApprovalChange(deps, agent.ApproverID, "approval_created", approval.ApprovalID)

		// Emit audit event for request creation (best-effort).
		emitApprovalRequestAuditEvent(r.Context(), deps.DB, agent.ApproverID, approval, agent.Metadata)

		approvalURL := fmt.Sprintf("%s/approve/%s", deps.BaseURL, approval.ApprovalID)

		RespondJSON(w, http.StatusOK, agentRequestApprovalResponse{
			ApprovalID:  approval.ApprovalID,
			ApprovalURL: approvalURL,
			Status:      approval.Status,
			ExpiresAt:   approval.ExpiresAt,
			CreatedAt:   approval.CreatedAt,
		})
	}
}

// ── POST /approvals/{approval_id}/cancel ───────────────────────────────────

func handleAgentCancelApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())
		approvalID := r.PathValue("approval_id")

		if strings.TrimSpace(approvalID) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "approval_id is required"))
			return
		}

		appr, err := db.CancelApproval(r.Context(), deps.DB, approvalID, agent.AgentID)
		if err != nil {
			if handleAgentApprovalError(w, r, err) {
				return
			}
			log.Printf("[%s] AgentCancelApproval: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to cancel approval"))
			return
		}

		// Emit audit event for cancellation (best-effort).
		emitApprovalAuditEvent(r.Context(), deps.DB, agent.ApproverID, appr, agent.Metadata)

		// Notify any connected SSE clients for this approver.
		notifyApprovalChange(deps, agent.ApproverID, "approval_cancelled", appr.ApprovalID)

		RespondJSON(w, http.StatusOK, agentCancelApprovalResponse{
			ApprovalID:  appr.ApprovalID,
			Status:      appr.Status,
			CancelledAt: *appr.CancelledAt,
		})
	}
}

// ── Error handling ─────────────────────────────────────────────────────────

// handleAgentApprovalError maps db.ApprovalError to the appropriate HTTP
// response for agent-facing endpoints. Returns true if the error was handled.
func handleAgentApprovalError(w http.ResponseWriter, r *http.Request, err error) bool {
	var apprErr *db.ApprovalError
	if !errors.As(err, &apprErr) {
		return false
	}
	switch apprErr.Code {
	case db.ApprovalErrNotFound:
		RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Approval not found"))
	case db.ApprovalErrAlreadyResolved:
		resp := Conflict(ErrApprovalAlreadyResolved, "Approval already resolved")
		if apprErr.Status != "" {
			resp.Error.Details = map[string]any{"status": apprErr.Status}
		}
		RespondError(w, r, http.StatusConflict, resp)
	case db.ApprovalErrExpired:
		RespondError(w, r, http.StatusGone, Gone(ErrApprovalExpired, "Approval has expired"))
	default:
		return false
	}
	return true
}

// unprocessableEntity returns a 422 ErrorResponse.
func unprocessableEntity(code ErrorCode, message string) ErrorResponse {
	return ErrorResponse{Error: Error{Code: code, Message: message, Retryable: false}}
}

// emitApprovalRequestAuditEvent writes an audit event for a new approval request.
// Only the action type is persisted — parameters are redacted.
// Billable: approval requests count toward the user's monthly request quota.
func emitApprovalRequestAuditEvent(ctx context.Context, d db.DBTX, userID string, appr *db.Approval, agentMeta []byte) {
	emitAuditEventWithUsage(ctx, d, db.InsertAuditEventParams{
		UserID:      userID,
		AgentID:     appr.AgentID,
		EventType:   db.AuditEventApprovalRequested,
		Outcome:     "pending",
		SourceID:    appr.ApprovalID,
		SourceType:  "approval",
		AgentMeta:   agentMeta,
		Action:      redactActionToType(appr.Action),
		ConnectorID: connectorIDFromActionType(actionTypeFromJSON(appr.Action)),
	}, true)
}
