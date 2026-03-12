package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/shared"
)

// ── Request/response types ─────────────────────────────────────────────────

// executeActionRequest holds the fields for standing-approval execution.
type executeActionRequest struct {
	RequestID       string      `json:"request_id"`
	ConfigurationID string      `json:"configuration_id"`
	Action          *actionBody `json:"action"`
	PaymentMethodID string      `json:"payment_method_id,omitempty"`
	AmountCents     *int        `json:"amount_cents,omitempty"`
}

type actionBody struct {
	Type       string          `json:"type"`
	Version    string          `json:"version"`
	Parameters json.RawMessage `json:"parameters"`
}

type executeActionStandingResponse struct {
	Result              *json.RawMessage `json:"result"`
	StandingApprovalID  string           `json:"standing_approval_id"`
	ExecutionsRemaining *int             `json:"executions_remaining"`
}

// ── Route registration ─────────────────────────────────────────────────────

func init() {
	RegisterRouteGroup(RegisterActionExecuteRoutes)
}

// RegisterActionExecuteRoutes adds the action execution endpoint to the mux.
func RegisterActionExecuteRoutes(mux *http.ServeMux, deps *Deps) {
	requireAgent := RequireAgentSignature(deps)
	mux.Handle("POST /actions/execute", requireAgent(handleExecuteAction(deps)))
}

// ── POST /actions/execute ───────────────────────────────────────────────────

func handleExecuteAction(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())

		var req executeActionRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		handleStandingApprovalPath(w, r, deps, agent, &req)
	}
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

	// ── Normalize parameter aliases ──────────────────────────────

	if deps.Connectors != nil {
		if action, ok := deps.Connectors.GetAction(req.Action.Type); ok {
			if aliaser, ok := action.(connectors.ParameterAliaser); ok {
				if aliases := aliaser.ParameterAliases(); len(aliases) > 0 {
					params = connectors.NormalizeParameters(aliases, params)
				}
			}
		}
	}

	// ── Normalize nested parameter structures ────────────────────

	if deps.Connectors != nil {
		if action, ok := deps.Connectors.GetAction(req.Action.Type); ok {
			if normalizer, ok := action.(connectors.Normalizer); ok {
				params = normalizer.Normalize(params)
			}
		}
	}

	// ── Check monthly request quota ─────────────────────────────

	var blocked bool
	r, blocked = checkRequestQuota(r.Context(), w, r, deps.DB, agent.ApproverID)
	if blocked {
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

	result, execErr := executeConnectorAction(r.Context(), deps, exec.UserID, req.Action.Type, params, &paymentParams{
		PaymentMethodID: req.PaymentMethodID,
		AmountCents:     req.AmountCents,
	})

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

