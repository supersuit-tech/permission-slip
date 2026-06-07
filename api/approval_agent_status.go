package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// approvalStateFlags describes whether an approval status is terminal and
// whether an agent should retry the request.
func approvalStateFlags(status string) (terminal, retryable bool) {
	switch status {
	case "pending":
		return false, false
	case "approved":
		return true, false
	case "denied", "cancelled", "expired":
		return true, false
	default:
		return true, false
	}
}

func respondAgentApprovalStatus(w http.ResponseWriter, r *http.Request, appr *db.Approval) {
	status := resolvedApprovalStatus(*appr)

	switch status {
	case "denied":
		resp := Conflict(ErrApprovalDenied, "Approval was denied by the approver")
		resp.Error.Details = terminalApprovalErrorDetails(appr, status)
		RespondError(w, r, http.StatusConflict, resp)
		return
	case "expired":
		resp := Gone(ErrApprovalExpired, "Approval has expired")
		resp.Error.Details = terminalApprovalErrorDetails(appr, status)
		RespondError(w, r, http.StatusGone, resp)
		return
	case "cancelled":
		resp := Conflict(ErrApprovalCancelled, "Approval was cancelled")
		resp.Error.Details = terminalApprovalErrorDetails(appr, status)
		RespondError(w, r, http.StatusConflict, resp)
		return
	}

	terminal, retryable := approvalStateFlags(status)
	resp := buildAgentApprovalStatusResponse(appr, status, terminal, retryable)
	RespondJSON(w, http.StatusOK, resp)
}

func respondRecentlyDeniedApproval(w http.ResponseWriter, r *http.Request, appr *db.Approval) {
	resp := Conflict(ErrApprovalRecentlyDenied, "This action was recently denied; do not retry without user intervention")
	resp.Error.Details = terminalApprovalErrorDetails(appr, "denied")
	RespondError(w, r, http.StatusConflict, resp)
}

func terminalApprovalErrorDetails(appr *db.Approval, status string) map[string]any {
	details := map[string]any{
		"approval_id": appr.ApprovalID,
		"status":      status,
		"terminal":    true,
		"retryable":   false,
	}
	if appr.DenialReason != nil && *appr.DenialReason != "" {
		details["reason"] = *appr.DenialReason
	}
	if appr.DeniedAt != nil {
		details["denied_at"] = appr.DeniedAt.UTC().Format(time.RFC3339Nano)
	}
	if appr.CancelledAt != nil {
		details["cancelled_at"] = appr.CancelledAt.UTC().Format(time.RFC3339Nano)
	}
	if !appr.ExpiresAt.IsZero() {
		details["expires_at"] = appr.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	return details
}

func buildAgentApprovalStatusResponse(appr *db.Approval, status string, terminal, retryable bool) agentApprovalStatusResponse {
	resp := agentApprovalStatusResponse{
		ApprovalID: appr.ApprovalID,
		Status:     status,
		Terminal:   terminal,
		Retryable:  retryable,
		ExpiresAt:  appr.ExpiresAt,
		CreatedAt:  appr.CreatedAt,
	}
	if appr.DenialReason != nil && *appr.DenialReason != "" {
		resp.Reason = appr.DenialReason
	}
	if appr.ExecutionStatus != nil {
		resp.ExecutionStatus = appr.ExecutionStatus
	}
	if len(appr.ExecutionResult) > 0 {
		raw := json.RawMessage(appr.ExecutionResult)
		resp.ExecutionResult = &raw
	}
	return resp
}
