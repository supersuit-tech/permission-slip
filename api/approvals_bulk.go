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

	"github.com/supersuit-tech/permission-slip/db"
)

func RegisterApprovalBulkRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /approval-groups/{group_id}", requireProfile(handleGetApprovalBulkGroup(deps)))
	mux.Handle("POST /approval-groups/{group_id}/decide", requireProfile(handleDecideApprovalBulkGroup(deps)))
}

func init() {
	RegisterRouteGroup(func(mux *http.ServeMux, deps *Deps) {
		RegisterApprovalBulkRoutes(mux, deps)
	})
}

type bulkGroupResponse struct {
	BulkGroupID string                   `json:"bulk_group_id"`
	AgentID     int64                    `json:"agent_id"`
	ActionType  string                   `json:"action_type"`
	ItemCount   int                      `json:"item_count"`
	Status      string                   `json:"status"`
	ExpiresAt   time.Time                `json:"expires_at"`
	CreatedAt   time.Time                `json:"created_at"`
	Items       []approvalDetailResponse `json:"items"`
}

type bulkDecisionRequest struct {
	Decisions []bulkDecisionItem `json:"decisions" validate:"required,min=1,max=50,dive"`
}

type bulkDecisionItem struct {
	ApprovalID string `json:"approval_id" validate:"required"`
	Decision   string `json:"decision" validate:"required,oneof=approve deny"`
}

type bulkDecisionResult struct {
	ApprovalID       string           `json:"approval_id"`
	Status           string           `json:"status"`
	ConfirmationCode string           `json:"confirmation_code,omitempty"`
	ExecutionStatus  *string          `json:"execution_status,omitempty"`
	ExecutionResult  *json.RawMessage `json:"execution_result,omitempty"`
	Error            *bulkItemError   `json:"error,omitempty"`
}

type bulkDecisionResponse struct {
	BulkGroupID string               `json:"bulk_group_id"`
	Results     []bulkDecisionResult `json:"results"`
}

func handleGetApprovalBulkGroup(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		groupID := strings.TrimSpace(r.PathValue("group_id"))
		if groupID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "group_id is required"))
			return
		}

		group, err := db.GetApprovalBulkGroupByIDAndApprover(r.Context(), deps.DB, groupID, profile.ID)
		if err != nil {
			log.Printf("[%s] GetBulkGroup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get bulk group"))
			return
		}
		if group == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Bulk group not found"))
			return
		}

		approvals, err := db.ListApprovalsByBulkGroupID(r.Context(), deps.DB, groupID)
		if err != nil {
			log.Printf("[%s] GetBulkGroup list: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get bulk group"))
			return
		}

		items := make([]approvalDetailResponse, len(approvals))
		for i, a := range approvals {
			items[i] = toApprovalDetailResponse(a)
		}

		RespondJSON(w, http.StatusOK, bulkGroupResponse{
			BulkGroupID: group.BulkGroupID,
			AgentID:     group.AgentID,
			ActionType:  group.ActionType,
			ItemCount:   group.ItemCount,
			Status:      db.BulkGroupAggregateStatus(approvals, time.Now().UTC()),
			ExpiresAt:   group.ExpiresAt,
			CreatedAt:   group.CreatedAt,
			Items:       items,
		})
	}
}

func handleDecideApprovalBulkGroup(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		groupID := strings.TrimSpace(r.PathValue("group_id"))
		if groupID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "group_id is required"))
			return
		}

		group, err := db.GetApprovalBulkGroupByIDAndApprover(r.Context(), deps.DB, groupID, profile.ID)
		if err != nil || group == nil {
			if err != nil {
				log.Printf("[%s] DecideBulkGroup lookup: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to decide bulk group"))
				return
			}
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Bulk group not found"))
			return
		}

		var req bulkDecisionRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		groupApprovals, err := db.ListApprovalsByBulkGroupID(r.Context(), deps.DB, groupID)
		if err != nil {
			log.Printf("[%s] DecideBulkGroup list: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to decide bulk group"))
			return
		}

		byID := make(map[string]db.Approval, len(groupApprovals))
		for _, a := range groupApprovals {
			byID[a.ApprovalID] = a
		}

		results := make([]bulkDecisionResult, 0, len(req.Decisions))
		for _, d := range req.Decisions {
			appr, ok := byID[d.ApprovalID]
			if !ok {
				results = append(results, bulkDecisionResult{
					ApprovalID: d.ApprovalID,
					Status:     "skipped",
					Error: &bulkItemError{
						Code:    "approval_not_in_group",
						Message: "Approval is not part of this bulk group",
					},
				})
				continue
			}

			if resolvedApprovalStatus(appr) != "pending" {
				results = append(results, bulkDecisionResult{
					ApprovalID: d.ApprovalID,
					Status:     "skipped",
					Error: &bulkItemError{
						Code:    "approval_already_resolved",
						Message: fmt.Sprintf("Approval is already %s", resolvedApprovalStatus(appr)),
					},
				})
				continue
			}

			switch d.Decision {
			case "approve":
				result := approveBulkItem(r, deps, profile.ID, d.ApprovalID)
				results = append(results, result)
			case "deny":
				result := denyBulkItem(r, deps, profile.ID, d.ApprovalID)
				results = append(results, result)
			}
		}

		notifyBulkApprovalChange(deps, profile.ID, groupID)

		RespondJSON(w, http.StatusOK, bulkDecisionResponse{
			BulkGroupID: groupID,
			Results:     results,
		})
	}
}

func approveBulkItem(r *http.Request, deps *Deps, userID, approvalID string) bulkDecisionResult {
	appr, agentMeta, err := db.ApproveApproval(r.Context(), deps.DB, approvalID, userID)
	if err != nil {
		var apprErr *db.ApprovalError
		if errors.As(err, &apprErr) {
			return bulkDecisionResult{
				ApprovalID: approvalID,
				Status:     "skipped",
				Error: &bulkItemError{
					Code:    apprErr.Code,
					Message: "Could not approve item",
				},
			}
		}
		return bulkDecisionResult{
			ApprovalID: approvalID,
			Status:     "skipped",
			Error:      &bulkItemError{Code: "internal_error", Message: "Failed to approve item"},
		}
	}

	confirmCode, err := generateConfirmationCodePlaintext()
	if err != nil {
		log.Printf("[%s] bulk approve confirm code: %v", TraceID(r.Context()), err)
		confirmCode = ""
	}

	execCtx, cancel := contextWithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	execStatus, execResultJSON := executeApprovalAction(execCtx, deps, userID, appr)

	emitApprovalAuditEvent(r.Context(), deps.DB, userID, appr, agentMeta)
	notifyApprovalChange(deps, userID, "approval_resolved", appr.ApprovalID)
	notifyApprovalExecuted(deps, userID, appr.ApprovalID, execStatus)

	result := bulkDecisionResult{
		ApprovalID:       appr.ApprovalID,
		Status:           "approved",
		ConfirmationCode: confirmCode,
		ExecutionStatus:  &execStatus,
	}
	if execResultJSON != nil {
		raw := json.RawMessage(execResultJSON)
		result.ExecutionResult = &raw
	}
	return result
}

func denyBulkItem(r *http.Request, deps *Deps, userID, approvalID string) bulkDecisionResult {
	appr, agentMeta, err := db.DenyApproval(r.Context(), deps.DB, approvalID, userID, "")
	if err != nil {
		return bulkDecisionResult{
			ApprovalID: approvalID,
			Status:     "skipped",
			Error:      &bulkItemError{Code: "deny_failed", Message: "Failed to deny item"},
		}
	}

	emitApprovalAuditEvent(r.Context(), deps.DB, userID, appr, agentMeta)
	notifyApprovalChange(deps, userID, "approval_resolved", appr.ApprovalID)

	return bulkDecisionResult{
		ApprovalID: appr.ApprovalID,
		Status:     "denied",
	}
}

func contextWithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), d)
}
