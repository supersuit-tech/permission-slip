package api

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/shared"
)

// ── Request/response types ─────────────────────────────────────────────────

// executeActionRequest is a combined struct used for dual-auth routing.
// If Token is non-empty, the token path is used. Otherwise, the standing
// approval path is used.
type executeActionRequest struct {
	// Token path fields
	Token      string          `json:"token"`
	ActionID   string          `json:"action_id"`
	Parameters json.RawMessage `json:"parameters"`

	// Standing approval path fields
	RequestID       string      `json:"request_id"`
	ConfigurationID string      `json:"configuration_id"`
	Action          *actionBody `json:"action"`
}

type actionBody struct {
	Type       string          `json:"type"`
	Version    string          `json:"version"`
	Parameters json.RawMessage `json:"parameters"`
}

type executeActionTokenResponse struct {
	Status     string           `json:"status"`
	ActionID   string           `json:"action_id"`
	ExecutedAt time.Time        `json:"executed_at"`
	Result     *json.RawMessage `json:"result,omitempty"`
}

type executeActionStandingResponse struct {
	Result              *json.RawMessage `json:"result"`
	StandingApprovalID  string           `json:"standing_approval_id"`
	ExecutionsRemaining *int             `json:"executions_remaining"`
}

// ── Route registration ─────────────────────────────────────────────────────

// RegisterActionExecuteRoutes adds the action execution endpoint to the mux.
func RegisterActionExecuteRoutes(mux *http.ServeMux, deps *Deps) {
	requireAgent := RequireAgentSignature(deps)
	mux.Handle("POST /actions/execute", requireAgent(handleExecuteAction(deps)))
}

// ── POST /actions/execute (dual-auth router) ───────────────────────────────

func handleExecuteAction(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())

		var req executeActionRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		if req.Token != "" {
			handleTokenPath(w, r, deps, agent, &req)
		} else {
			handleStandingApprovalPath(w, r, deps, agent, &req)
		}
	}
}

// ── Token-based execution path ─────────────────────────────────────────────

func handleTokenPath(w http.ResponseWriter, r *http.Request, deps *Deps, agent *db.Agent, req *executeActionRequest) {
	if req.ActionID == "" {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Missing required field: action_id"))
		return
	}
	if len(req.Parameters) > shared.MaxParametersBytes {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "parameters exceeds maximum size"))
		return
	}
	if err := ValidateJSONObject(req.Parameters); err != nil {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "parameters must be a JSON object"))
		return
	}

	// ── Validate the action token ─────────────────────────────────

	if deps.ActionTokenSigningKey == nil {
		log.Printf("[%s] ExecuteActionToken: action token signing key not configured", TraceID(r.Context()))
		RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Action token verification not available"))
		return
	}

	claims, err := parseAndValidateActionToken(req.Token, &deps.ActionTokenSigningKey.PublicKey)
	if err != nil {
		log.Printf("[%s] ExecuteActionToken: token validation: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Token is invalid or has expired"))
		return
	}

	// Verify the token's subject matches the authenticated agent.
	tokenAgentID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || tokenAgentID != agent.AgentID {
		RespondError(w, r, http.StatusForbidden, Forbidden(ErrInvalidToken, "Token does not belong to this agent"))
		return
	}

	// Verify scope matches the requested action_id.
	if claims.Scope != req.ActionID {
		resp := Forbidden(ErrInsufficientScope, "Token does not have the required scope for this action")
		resp.Error.Details = map[string]any{
			"token_scope":      claims.Scope,
			"requested_action": req.ActionID,
		}
		RespondError(w, r, http.StatusForbidden, resp)
		return
	}

	// Verify parameters hash matches the token's params_hash claim.
	if err := VerifyParamsHash(req.Parameters, claims.ParamsHash); err != nil {
		log.Printf("[%s] ExecuteActionToken: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusForbidden, Forbidden(ErrInvalidParameters, "Request parameters do not match the approved parameters"))
		return
	}

	// ── Cross-reference approval row's token_jti ──────────────────

	approval, err := db.GetApprovalByIDAndAgent(r.Context(), deps.DB, claims.ApprovalID, agent.AgentID)
	if err != nil {
		log.Printf("[%s] ExecuteActionToken: approval lookup: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to validate token"))
		return
	}
	if approval == nil {
		RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Approval not found"))
		return
	}
	if approval.TokenJTI == nil || *approval.TokenJTI != claims.ID {
		RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Token does not match the approval"))
		return
	}

	// ── Consume token (replay prevention) ─────────────────────────

	if err := db.ConsumeToken(r.Context(), deps.DB, claims.ID); err != nil {
		var ctErr *db.ConsumedTokenError
		if errors.As(err, &ctErr) && ctErr.Code == db.ConsumedTokenErrAlreadyConsumed {
			resp := Forbidden(ErrTokenAlreadyUsed, "Token has already been consumed")
			resp.Error.Details = map[string]any{
				"jti": claims.ID,
			}
			RespondError(w, r, http.StatusForbidden, resp)
			return
		}
		log.Printf("[%s] ExecuteActionToken: consume token: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to consume token"))
		return
	}

	// ── Execute the action via connector ──────────────────────────

	result, execErr := executeConnectorAction(r.Context(), deps, approval.ApproverID, req.ActionID, req.Parameters)

	// Always emit the audit event with the actual execution result,
	// regardless of success or failure.
	emitActionExecutedAuditEvent(r.Context(), deps.DB, approval.ApproverID, agent.AgentID, claims.ApprovalID, req.ActionID, agent.Metadata, execErr)

	if execErr != nil {
		if handleConnectorError(w, r, execErr) {
			return
		}
		log.Printf("[%s] ExecuteActionToken: connector execution: %v", TraceID(r.Context()), execErr)
		CaptureError(r.Context(), execErr)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to execute action"))
		return
	}

	var actionResultPtr *json.RawMessage
	if result != nil {
		actionResultPtr = &result.Data
	}

	RespondJSON(w, http.StatusOK, executeActionTokenResponse{
		Status:     "success",
		ActionID:   req.ActionID,
		ExecutedAt: time.Now().UTC(),
		Result:     actionResultPtr,
	})
}

// ── Standing approval execution path ───────────────────────────────────────

func handleStandingApprovalPath(w http.ResponseWriter, r *http.Request, deps *Deps, agent *db.Agent, req *executeActionRequest) {
	// Validate required fields for standing approval mode.
	if req.Action == nil {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Missing required field: action"))
		return
	}
	req.Action.Type = strings.TrimSpace(req.Action.Type)
	if req.Action.Type == "" {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Missing required field: action.type"))
		return
	}
	if len(req.Action.Type) > shared.ActionTypeMaxLength {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action.type exceeds maximum length"))
		return
	}
	req.RequestID = strings.TrimSpace(req.RequestID)
	if req.RequestID == "" {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Missing required field: request_id"))
		return
	}
	if len(req.RequestID) > 255 {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "request_id exceeds maximum length"))
		return
	}
	if req.Action.Version != "" && len(req.Action.Version) > shared.ActionVersionMaxLength {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action.version exceeds maximum length"))
		return
	}

	params := req.Action.Parameters
	if len(params) > shared.MaxParametersBytes {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "parameters exceeds maximum size"))
		return
	}
	if err := ValidateJSONObject(params); err != nil {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "parameters must be a JSON object"))
		return
	}

	// ── Check monthly request quota ─────────────────────────────

	if checkRequestQuota(r.Context(), w, r, deps.DB, agent.ApproverID) {
		return
	}

	// ── Find matching standing approval ──────────────────────────

	sa, err := db.FindActiveStandingApprovalForAgent(r.Context(), deps.DB, agent.AgentID, req.Action.Type)
	if err != nil {
		log.Printf("[%s] ExecuteActionStanding: find standing approval: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to find standing approval"))
		return
	}
	if sa == nil {
		resp := NotFound(ErrNoMatchingStanding, "No active standing approval matches this agent, action type, and parameters")
		resp.Error.Details = map[string]any{
			"action_type": req.Action.Type,
			"hint":        "Use POST /v1/approvals/request for one-off approval",
		}
		RespondError(w, r, http.StatusNotFound, resp)
		return
	}

	// ── Validate configuration reference (optional) ──────────────

	if req.ConfigurationID != "" {
		if ValidateConfigurationReference(w, r, deps, req.ConfigurationID, agent.AgentID, req.Action.Type, params) == nil {
			return // error already written
		}
	}

	// ── Validate parameters against standing approval constraints ─

	if len(sa.Constraints) > 0 {
		if err := db.ValidateParametersAgainstConfig(sa.Constraints, params); err != nil {
			var configErr *db.ConfigValidationError
			if errors.As(err, &configErr) {
				resp := Forbidden(ErrConstraintViolation, "Request parameters violate standing approval constraints")
				resp.Error.Details = map[string]any{
					"standing_approval_id": sa.StandingApprovalID,
					"violated_constraint":  configErr.Parameter,
					"constraint_error":     configErr.Reason,
				}
				RespondError(w, r, http.StatusForbidden, resp)
				return
			}
			log.Printf("[%s] ExecuteActionStanding: constraint validation: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to validate parameters against constraints"))
			return
		}
	}

	// ── Record execution ─────────────────────────────────────────

	exec, err := db.RecordStandingApprovalExecutionByAgent(r.Context(), deps.DB, sa.StandingApprovalID, agent.AgentID, req.RequestID, params)
	if err != nil {
		var saErr *db.StandingApprovalError
		if errors.As(err, &saErr) {
			switch saErr.Code {
			case db.StandingApprovalErrNotActive:
				// The standing approval became inactive between find and record
				// (race condition: expired, exhausted, or revoked).
				resp := Gone(ErrStandingExpired, "Standing approval is no longer active")
				resp.Error.Details = map[string]any{
					"standing_approval_id": sa.StandingApprovalID,
				}
				RespondError(w, r, http.StatusGone, resp)
				return
			case db.StandingApprovalErrDuplicateRequest:
				RespondError(w, r, http.StatusConflict, Conflict(ErrDuplicateRequestID, "A request with this request_id has already been executed"))
				return
			}
		}
		log.Printf("[%s] ExecuteActionStanding: record execution: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to record execution"))
		return
	}

	// ── Execute the action via connector ─────────────────────────

	result, execErr := executeConnectorAction(r.Context(), deps, exec.UserID, req.Action.Type, params)

	// Always emit the audit event with the actual execution result,
	// regardless of success or failure (best-effort).
	emitStandingApprovalAuditEvent(r.Context(), deps.DB, exec.UserID, exec.AgentID, sa.StandingApprovalID, exec.ActionType, exec.AgentMeta, execErr)

	if execErr != nil {
		if handleConnectorError(w, r, execErr) {
			return
		}
		log.Printf("[%s] ExecuteActionStanding: connector execution: %v", TraceID(r.Context()), execErr)
		CaptureError(r.Context(), execErr)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to execute connector action"))
		return
	}

	var actionResultPtr *json.RawMessage
	if result != nil {
		actionResultPtr = &result.Data
	}

	// ── Compute executions remaining ─────────────────────────────

	var executionsRemaining *int
	if exec.MaxExecutions != nil {
		remaining := *exec.MaxExecutions - exec.ExecutionCount
		if remaining < 0 {
			remaining = 0
		}
		executionsRemaining = &remaining
	}

	RespondJSON(w, http.StatusOK, executeActionStandingResponse{
		Result:              actionResultPtr,
		StandingApprovalID:  sa.StandingApprovalID,
		ExecutionsRemaining: executionsRemaining,
	})
}

// ── Audit event emission ───────────────────────────────────────────────────

// emitActionExecutedAuditEvent writes an action.executed audit event for
// one-off token-based execution. Not billable because the approval request
// was already counted when it was created.
//
// execErr should be the error from connector execution, or nil on success.
// The execution_status and execution_error fields are derived from execErr.
func emitActionExecutedAuditEvent(ctx context.Context, d db.DBTX, userID string, agentID int64, approvalID, actionType string, agentMeta []byte, execErr error) {
	actionJSON, _ := json.Marshal(map[string]string{"type": actionType})
	execStatus, execErrMsg := resolveExecResult(execErr)

	emitAuditEventWithUsage(ctx, d, db.InsertAuditEventParams{
		UserID:          userID,
		AgentID:         agentID,
		EventType:       db.AuditEventActionExecuted,
		Outcome:         "auto_executed",
		SourceID:        approvalID,
		SourceType:      "approval",
		AgentMeta:       agentMeta,
		Action:          actionJSON,
		ConnectorID:     connectorIDFromActionType(actionType),
		ExecutionStatus: &execStatus,
		ExecutionError:  execErrMsg,
	}, false)
}

// ── JWT token parsing ──────────────────────────────────────────────────────

// parseAndValidateActionToken parses an ES256 JWT action token and validates
// its signature, expiration, and audience.
func parseAndValidateActionToken(tokenStr string, pubKey *ecdsa.PublicKey) (*ActionTokenClaims, error) {
	claims := &ActionTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return pubKey, nil
		},
		jwt.WithAudience(actionTokenAudience),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"ES256"}),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}
