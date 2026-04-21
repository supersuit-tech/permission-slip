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

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/shared"
)

// ── Request/response types ─────────────────────────────────────────────────

type agentRequestApprovalRequest struct {
	RequestID       string                  `json:"request_id" validate:"required"`
	Action          json.RawMessage         `json:"action" validate:"required"`
	Context         json.RawMessage         `json:"context" validate:"required"`
	ExpiresIn       *int                    `json:"expires_in,omitempty" validate:"omitempty,gte=60,lte=86400"`
	Configuration   *agentApprovalConfigRef `json:"configuration,omitempty"`
	PaymentMethodID string                  `json:"payment_method_id,omitempty"`
	AmountCents     *int                    `json:"amount_cents,omitempty"`
}

type agentApprovalConfigRef struct {
	ConfigurationID string `json:"configuration_id" validate:"required"`
}

type agentRequestApprovalResponse struct {
	// ── Pending fields (status="pending") ──
	// Present only when no standing approval matched and a pending approval
	// was created. The agent should poll GET /approvals/{approval_id}/status.
	ApprovalID  string     `json:"approval_id,omitempty"`
	ApprovalURL string     `json:"approval_url,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`

	// ── Common field ──
	// "pending" = awaiting human approval; "approved" = auto-approved via standing approval.
	Status string `json:"status"`

	// ── Auto-approved fields (status="approved") ──
	// Present only when a standing approval matched and the action was
	// executed immediately. No polling needed — the result is inline.
	Result             *json.RawMessage `json:"result,omitempty"`               // Connector execution output (may be null if connector returns no data).
	StandingApprovalID string           `json:"standing_approval_id,omitempty"` // Which standing approval authorized this execution (useful for audit tracking).
}

type agentCancelApprovalResponse struct {
	ApprovalID  string    `json:"approval_id"`
	Status      string    `json:"status"`
	CancelledAt time.Time `json:"cancelled_at"`
}

// ── Route registration ─────────────────────────────────────────────────────

func init() {
	RegisterRouteGroup(RegisterAgentApprovalRoutes)
}

// RegisterAgentApprovalRoutes adds agent-authenticated approval endpoints.
func RegisterAgentApprovalRoutes(mux *http.ServeMux, deps *Deps) {
	requireAgent := RequireAgentSignature(deps)
	mux.Handle("POST /approvals/request", requireAgent(handleAgentRequestApproval(deps)))
	mux.Handle("GET /approvals/{approval_id}/status", requireAgent(handleAgentApprovalStatus(deps)))
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

		// Validate action type and normalize parameters before storage.
		// Rejects unregistered action types immediately so agents get a clear
		// error instead of approvals that silently fail at execution time.
		// Then rewrites flat aliases (ParameterAliaser) and applies nested
		// normalization (Normalizer). Must run before ValidateConfigurationReference
		// so constraints are evaluated against canonical keys.
		if deps.Connectors != nil {
			action, conn, ok := deps.Connectors.GetActionWithConnector(actionType)
			if !ok {
				errResp := BadRequest(ErrUnsupportedActionType, fmt.Sprintf("unknown action type %q", actionType))
				errResp.Error.Details = map[string]any{"action_type": actionType}
				RespondError(w, r, http.StatusBadRequest, errResp)
				return
			}
			if aliaser, ok := action.(connectors.ParameterAliaser); ok {
				if aliases := aliaser.ParameterAliases(); len(aliases) > 0 {
					if rawParams, hasParams := actionObj["parameters"]; hasParams {
						normalized := connectors.NormalizeParameters(aliases, rawParams)
						actionObj["parameters"] = normalized
						if updated, err := json.Marshal(actionObj); err == nil {
							req.Action = updated
						}
					}
				}
			}
			if normalizer, ok := action.(connectors.Normalizer); ok {
				if rawParams, hasParams := actionObj["parameters"]; hasParams {
					actionObj["parameters"] = normalizer.Normalize(rawParams)
					if updated, err := json.Marshal(actionObj); err == nil {
						req.Action = updated
					}
				}
			}
			// Validate parameters early so the agent gets an immediate error
			// instead of the approval failing at execution time.
			// If "parameters" is absent entirely, skip — schema validation
			// (validateActionParameters below) catches missing required fields.
			//
			// Priority: action-level RequestValidator first, then fall back to
			// connector-level ParamValidator. This allows per-action overrides
			// (e.g., Slack channel ID format checks) while providing generic
			// coverage via the connector's dispatch table.
			if rawParams, hasParams := actionObj["parameters"]; hasParams {
				var validationErr error
				if rv, ok := action.(connectors.RequestValidator); ok {
					validationErr = rv.ValidateRequest(json.RawMessage(rawParams))
				} else if pv, ok := conn.(connectors.ParamValidator); ok {
					validationErr = pv.ValidateParams(actionType, json.RawMessage(rawParams))
				}
				if validationErr != nil {
					var ve *connectors.ValidationError
					if errors.As(validationErr, &ve) {
						RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, ve.Message))
					} else {
						RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid action parameters"))
					}
					return
				}
			}
		}

		// Multi-instance routing: resolve optional parameters.connector_instance, freeze UUID+label on action, strip from parameters.
		connectorInstanceID, err := applyConnectorInstanceToAction(r.Context(), deps.DB, agent, actionType, actionObj)
		if err != nil {
			var ciErr *connectorInstanceResolutionError
			if errors.As(err, &ciErr) {
				RespondError(w, r, ciErr.status, ciErr.resp)
				return
			}
			log.Printf("[%s] applyConnectorInstanceToAction: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to process action"))
			return
		}
		if updated, mErr := json.Marshal(actionObj); mErr == nil {
			req.Action = updated
		}

		// Validate action parameters against the connector's parameters_schema.
		// Runs after alias normalization so canonical keys are checked.
		// Fail-open: skipped if the action has no schema.
		actionParams := json.RawMessage(actionObj["parameters"])
		if !validateActionParameters(w, r, deps.DB, actionType, actionParams) {
			return
		}

		// Optional: validate configuration reference — sees canonical keys after normalization.
		if req.Configuration != nil {
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

		// ── Check for matching standing approval (auto-approve) ─────
		//
		// Before creating a pending approval, check if an active standing
		// approval matches this agent + action type + parameters. If so,
		// execute immediately and return the result — no human review needed.
		pp := &paymentParams{PaymentMethodID: req.PaymentMethodID, AmountCents: req.AmountCents}
		if handled := tryStandingApprovalAutoApprove(w, r, deps, agent, actionType, actionParams, req.RequestID, pp, connectorInstanceID); handled {
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
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create approval"))
			return
		}

		// Look up the approver profile (the agent's owner).
		approverProfile, err := db.GetProfileByUserID(r.Context(), deps.DB, agent.ApproverID)
		if err != nil {
			log.Printf("[%s] AgentRequestApproval: profile lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create approval"))
			return
		}
		if approverProfile == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApproverNotFound, "Approver profile not found"))
			return
		}

		// Best-effort: resolve human-readable resource details for the action.
		var resourceDetails []byte
		if deps.Connectors != nil {
			if cid := strings.SplitN(actionType, ".", 2); len(cid) == 2 {
				if conn, ok := deps.Connectors.Get(cid[0]); ok {
					if resolver, ok := conn.(connectors.ResourceDetailResolver); ok {
						resolveCtx, resolveCancel := context.WithTimeout(r.Context(), 5*time.Second)
						details, resolveErr := resolver.ResolveResourceDetails(resolveCtx, actionType, actionParams, resolveCredentialsForResolver(resolveCtx, deps, agent.AgentID, agent.ApproverID, actionType, cid[0], connectorInstanceID))
						resolveCancel()
						if resolveErr != nil {
							log.Printf("[%s] ResolveResourceDetails: %v", TraceID(r.Context()), resolveErr)
							CaptureError(r.Context(), resolveErr)
						} else if details != nil {
							if encoded, encErr := json.Marshal(details); encErr == nil {
								resourceDetails = encoded
							}
						}
					}
				}
			}
		}

		contextForStore := mergeEmailThreadFromResourceDetailsIntoContext(req.Context, resourceDetails)

		approval, err := db.InsertApproval(r.Context(), deps.DB, db.InsertApprovalParams{
			ApprovalID:      approvalID,
			AgentID:         agent.AgentID,
			ApproverID:      agent.ApproverID,
			Action:          req.Action,
			Context:         contextForStore,
			ResourceDetails: resourceDetails,
			ExpiresAt:       expiresAt,
		}, req.RequestID)
		if err != nil {
			var apprErr *db.ApprovalError
			if errors.As(err, &apprErr) && apprErr.Code == db.ApprovalErrDuplicateRequest {
				RespondError(w, r, http.StatusConflict, Conflict(ErrDuplicateRequestID, "A request with this request_id has already been submitted"))
				return
			}
			log.Printf("[%s] AgentRequestApproval: insert: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create approval"))
			return
		}

		// Update agent's last_active_at (best-effort).
		if err := db.TouchAgentLastActive(r.Context(), deps.DB, agent.AgentID); err != nil {
			log.Printf("[%s] AgentRequestApproval: failed to update last_active_at for agent %d: %v", TraceID(r.Context()), agent.AgentID, err)
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
			ExpiresAt:   &approval.ExpiresAt,
			CreatedAt:   &approval.CreatedAt,
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
			CaptureError(r.Context(), err)
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

// ── GET /approvals/{approval_id}/status ─────────────────────────────────────

type agentApprovalStatusResponse struct {
	ApprovalID      string           `json:"approval_id"`
	Status          string           `json:"status"`
	ExecutionStatus *string          `json:"execution_status,omitempty"`
	ExecutionResult *json.RawMessage `json:"execution_result,omitempty"`
	ExpiresAt       time.Time        `json:"expires_at"`
	CreatedAt       time.Time        `json:"created_at"`
}

func handleAgentApprovalStatus(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())
		approvalID := r.PathValue("approval_id")

		if strings.TrimSpace(approvalID) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "approval_id is required"))
			return
		}

		appr, err := db.GetApprovalByIDAndAgent(r.Context(), deps.DB, approvalID, agent.AgentID)
		if err != nil {
			log.Printf("[%s] AgentApprovalStatus: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get approval status"))
			return
		}
		if appr == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Approval not found"))
			return
		}

		resp := agentApprovalStatusResponse{
			ApprovalID: appr.ApprovalID,
			Status:     resolvedApprovalStatus(*appr),
			ExpiresAt:  appr.ExpiresAt,
			CreatedAt:  appr.CreatedAt,
		}
		if appr.ExecutionStatus != nil {
			resp.ExecutionStatus = appr.ExecutionStatus
		}
		if len(appr.ExecutionResult) > 0 {
			raw := json.RawMessage(appr.ExecutionResult)
			resp.ExecutionResult = &raw
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// ── Error handling ─────────────────────────────────────────────────────────

// handleAgentApprovalError is an alias for the shared handleApprovalError
// in approvals.go. Both dashboard and agent endpoints use the same mapping.
var handleAgentApprovalError = handleApprovalError

// resolveCredentialsForResolver is a best-effort credential resolver for the
// ResourceDetailResolver call. It mirrors the credential resolution logic from
// connector execution but returns empty credentials on any failure (since
// resource detail resolution is non-fatal).
func resolveCredentialsForResolver(ctx context.Context, deps *Deps, agentID int64, userID, actionType, connectorID, connectorInstanceID string) connectors.Credentials {
	reqCreds, err := db.GetRequiredCredentialsByActionType(ctx, deps.DB, actionType)
	if err != nil || len(reqCreds) == 0 {
		return connectors.NewCredentials(nil)
	}
	creds, err := resolveCredentialsWithFallback(ctx, deps, agentID, userID, actionType, connectorID, connectorInstanceID, reqCreds)
	if err != nil {
		return connectors.NewCredentials(nil)
	}
	return creds
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
		Action:      redactActionToTypeWithConnectorInstance(appr.Action),
		ConnectorID: connectorIDFromActionType(actionTypeFromJSON(appr.Action)),
	}, true)
}
