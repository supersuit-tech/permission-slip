package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// tryStandingApprovalAutoApprove checks if an active standing approval matches
// the given agent, action type, and parameters. If a match is found, the action
// is executed immediately and the result is written to w. Returns true if the
// response was written (either success or error), false if no standing approval
// matched and the caller should fall through to the pending approval flow.
func tryStandingApprovalAutoApprove(w http.ResponseWriter, r *http.Request, deps *Deps, agent *db.Agent, actionType string, params json.RawMessage, requestID string) bool {
	approvals, err := db.FindActiveStandingApprovalsForAgent(r.Context(), deps.DB, agent.AgentID, actionType)
	if err != nil {
		log.Printf("[%s] AutoApprove: find standing approvals: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), err)
		// Don't fail the request — fall through to the pending approval flow.
		return false
	}
	if len(approvals) == 0 {
		return false
	}

	// Find first standing approval whose constraints match.
	var sa *db.StandingApproval
	for _, candidate := range approvals {
		if len(candidate.Constraints) == 0 {
			sa = candidate
			break
		}
		if err := db.ValidateParametersAgainstConfig(candidate.Constraints, params); err != nil {
			var configErr *db.ConfigValidationError
			if errors.As(err, &configErr) {
				continue
			}
			// Corrupt constraint JSON — log and skip this candidate.
			log.Printf("[%s] AutoApprove: constraint validation for %s: %v",
				TraceID(r.Context()), candidate.StandingApprovalID, err)
			CaptureError(r.Context(), err)
			continue
		}
		sa = candidate
		break
	}

	if sa == nil {
		// Standing approvals exist but none matched — fall through to pending.
		return false
	}

	// Record the execution against the standing approval.
	exec, err := db.RecordStandingApprovalExecutionByAgent(r.Context(), deps.DB, sa.StandingApprovalID, agent.AgentID, requestID, params)
	if err != nil {
		var saErr *db.StandingApprovalError
		if errors.As(err, &saErr) {
			switch saErr.Code {
			case db.StandingApprovalErrNotActive:
				// Race: approval became inactive between find and record.
				// Fall through to the pending approval flow.
				return false
			case db.StandingApprovalErrDuplicateRequest:
				RespondError(w, r, http.StatusConflict, Conflict(ErrDuplicateRequestID, "A request with this request_id has already been submitted"))
				return true
			}
		}
		log.Printf("[%s] AutoApprove: record execution: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), err)
		// Fall through to the pending approval flow rather than failing.
		return false
	}

	// Execute the action via connector.
	result, execErr := executeConnectorAction(r.Context(), deps, exec.AgentID, exec.UserID, actionType, params, &paymentParams{})

	// Emit audit event (best-effort).
	emitStandingApprovalAuditEvent(r.Context(), deps.DB, exec.UserID, exec.AgentID, sa.StandingApprovalID, exec.ActionType, exec.AgentMeta, execErr)

	if execErr != nil {
		cc := ConnectorContext{ActionType: actionType, AgentID: agent.AgentID}
		if handleConnectorError(w, r, execErr, cc) {
			return true
		}
		log.Printf("[%s] AutoApprove: connector execution: %v", TraceID(r.Context()), execErr)
		CaptureConnectorError(r.Context(), execErr, cc)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to execute connector action"))
		return true
	}

	// Notify user of standing approval execution (fire-and-forget).
	NotifyStandingApprovalExecution(r.Context(), deps, exec, agent, actionType, params)

	var actionResultPtr *json.RawMessage
	if result != nil {
		actionResultPtr = &result.Data
	}

	var executionsRemaining *int
	if exec.MaxExecutions != nil {
		remaining := *exec.MaxExecutions - exec.ExecutionCount
		if remaining < 0 {
			remaining = 0
		}
		executionsRemaining = &remaining
	}

	// Update agent's last_active_at (best-effort).
	if err := db.TouchAgentLastActive(r.Context(), deps.DB, agent.AgentID); err != nil {
		log.Printf("auto_approve: failed to update last_active_at for agent %d: %v", agent.AgentID, err)
	}

	RespondJSON(w, http.StatusOK, agentRequestApprovalResponse{
		Status:              "approved",
		Result:              actionResultPtr,
		StandingApprovalID:  sa.StandingApprovalID,
		ExecutionsRemaining: executionsRemaining,
	})
	return true
}
