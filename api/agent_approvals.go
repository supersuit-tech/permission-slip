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
	ApprovalID           string    `json:"approval_id"`
	ApprovalURL          string    `json:"approval_url"`
	Status               string    `json:"status"`
	ExpiresAt            time.Time `json:"expires_at"`
	VerificationRequired bool      `json:"verification_required"`
	CreatedAt            time.Time `json:"created_at"`
}

type agentVerifyApprovalRequest struct {
	ConfirmationCode string `json:"confirmation_code" validate:"required"`
}

type agentVerifyApprovalResponse struct {
	Status     string                  `json:"status"`
	ApprovedAt time.Time               `json:"approved_at"`
	Token      *agentApprovalTokenResp `json:"token"`
}

type agentApprovalTokenResp struct {
	AccessToken  string    `json:"access_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
	ScopeVersion string    `json:"scope_version"`
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
	mux.Handle("POST /approvals/{approval_id}/verify", requireAgent(handleAgentVerifyApproval(deps)))
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
			ApprovalID:           approval.ApprovalID,
			ApprovalURL:          approvalURL,
			Status:               approval.Status,
			ExpiresAt:            approval.ExpiresAt,
			VerificationRequired: true,
			CreatedAt:            approval.CreatedAt,
		})
	}
}

// ── POST /approvals/{approval_id}/verify ───────────────────────────────────

func handleAgentVerifyApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())
		approvalID := r.PathValue("approval_id")

		if strings.TrimSpace(approvalID) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "approval_id is required"))
			return
		}

		var req agentVerifyApprovalRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		normalized := normalizeConfirmationCode(req.ConfirmationCode)
		if len(normalized) != shared.ConfirmationCodeLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid confirmation code format"))
			return
		}

		codeHash := hashCodeHex(normalized, deps.InviteHMACKey)

		// Wrap code verification + token minting in a transaction so that
		// if token minting fails, the confirmation code is not consumed.
		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] AgentVerifyApproval: begin tx: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify approval"))
			return
		}
		if owned {
			defer func() { _ = db.RollbackTx(r.Context(), tx) }()
		}

		appr, err := db.VerifyApprovalConfirmationCode(r.Context(), tx, approvalID, agent.AgentID, codeHash)
		if err != nil {
			// On invalid code, commit the transaction to persist the attempt
			// increment so the 5-attempt brute-force lockout stays effective.
			// Other error paths (not found, expired, locked) didn't modify
			// any rows, so the deferred rollback is harmless for those.
			var apprErr *db.ApprovalError
			if owned && errors.As(err, &apprErr) && apprErr.Code == db.ApprovalErrInvalidCode {
				if commitErr := db.CommitTx(r.Context(), tx); commitErr != nil {
					log.Printf("[%s] AgentVerifyApproval: commit attempt increment: %v", TraceID(r.Context()), commitErr)
				}
			}
			if handleAgentApprovalError(w, r, err) {
				return
			}
			log.Printf("[%s] AgentVerifyApproval: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify approval"))
			return
		}

		// Check signing key after verification so that verification errors
		// (not found, expired, wrong code) are returned with the correct
		// status code instead of a generic 503.
		if deps.ActionTokenSigningKey == nil {
			log.Printf("[%s] AgentVerifyApproval: action token signing key not configured", TraceID(r.Context()))
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Action token signing not available"))
			return
		}

		// Look up the approver's username for the JWT "approver" claim.
		// approver_id is an FK, so a nil profile indicates data inconsistency.
		approverProfile, err := db.GetProfileByUserID(r.Context(), tx, appr.ApproverID)
		if err != nil {
			log.Printf("[%s] AgentVerifyApproval: profile lookup: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to mint action token"))
			return
		}
		if approverProfile == nil {
			log.Printf("[%s] AgentVerifyApproval: no profile for approver_id=%s", TraceID(r.Context()), appr.ApproverID)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to mint action token"))
			return
		}
		approverUsername := approverProfile.Username

		// Generate a unique JTI for single-use enforcement.
		jti, err := generatePrefixedID("tok_", 16)
		if err != nil {
			log.Printf("[%s] AgentVerifyApproval: generate JTI: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to mint action token"))
			return
		}

		claims, err := buildActionTokenClaims(agent.AgentID, approverUsername, appr.ApprovalID, appr.Action, appr.ExpiresAt, jti)
		if err != nil {
			log.Printf("[%s] AgentVerifyApproval: build claims: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to mint action token"))
			return
		}

		accessToken, err := MintActionToken(deps.ActionTokenSigningKey, deps.ActionTokenKeyID, claims)
		if err != nil {
			log.Printf("[%s] AgentVerifyApproval: sign token: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to mint action token"))
			return
		}

		// Persist the JTI for single-use tracking. This is now within the
		// transaction, so failure here rolls back the code consumption too.
		if err := db.SetApprovalTokenJTI(r.Context(), tx, appr.ApprovalID, jti); err != nil {
			log.Printf("[%s] AgentVerifyApproval: set token_jti: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to mint action token"))
			return
		}

		// Everything succeeded — commit the transaction.
		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] AgentVerifyApproval: commit: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify approval"))
				return
			}
		}

		approvedAt := time.Now().UTC()
		if appr.ApprovedAt != nil {
			approvedAt = *appr.ApprovedAt
		}

		RespondJSON(w, http.StatusOK, agentVerifyApprovalResponse{
			Status:     appr.Status,
			ApprovedAt: approvedAt,
			Token: &agentApprovalTokenResp{
				AccessToken:  accessToken,
				ExpiresAt:    claims.ExpiresAt.Time,
				Scope:        claims.Scope,
				ScopeVersion: claims.ScopeVersion,
			},
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
	case db.ApprovalErrInvalidCode:
		RespondError(w, r, http.StatusUnprocessableEntity, unprocessableEntity(ErrInvalidCode, "Incorrect confirmation code"))
	case db.ApprovalErrVerificationLocked:
		RespondError(w, r, http.StatusForbidden, Forbidden(ErrVerificationLocked, "Too many failed verification attempts"))
	case db.ApprovalErrNotYetApproved:
		resp := Conflict(ErrApprovalAlreadyResolved, "Approval has not been approved yet")
		resp.Error.Details = map[string]any{"status": "pending"}
		RespondError(w, r, http.StatusConflict, resp)
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
// Also increments the usage meter for billing (approval requests are billable events).
func emitApprovalRequestAuditEvent(ctx context.Context, d db.DBTX, userID string, appr *db.Approval, agentMeta []byte) {
	actionType := actionTypeFromJSON(appr.Action)

	if err := db.InsertAuditEvent(ctx, d, db.InsertAuditEventParams{
		UserID:      userID,
		AgentID:     appr.AgentID,
		EventType:   db.AuditEventApprovalRequested,
		Outcome:     "pending",
		SourceID:    appr.ApprovalID,
		SourceType:  "approval",
		AgentMeta:   agentMeta,
		Action:      redactActionToType(appr.Action),
		ConnectorID: connectorIDFromActionType(actionType),
	}); err != nil {
		log.Printf("audit: failed to insert approval request audit event: %v", err)
	}

	// Increment usage meter (best-effort, billing).
	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	connectorID := ""
	if cid := connectorIDFromActionType(actionType); cid != nil {
		connectorID = *cid
	}
	if _, err := db.IncrementRequestCountWithBreakdown(ctx, d, userID, periodStart, periodEnd, db.UsageBreakdownKeys{
		AgentID:     appr.AgentID,
		ConnectorID: connectorID,
		ActionType:  actionType,
	}); err != nil {
		log.Printf("audit: failed to increment usage count for approval request: %v", err)
	}
}
