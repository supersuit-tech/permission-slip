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

	"github.com/getsentry/sentry-go"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/shared"
)

// Response types for the dashboard approval endpoints.

type approvalResponse struct {
	ApprovalID      string     `json:"approval_id"`
	AgentID         int64      `json:"agent_id"`
	Action          any        `json:"action"`
	Context         any        `json:"context"`
	Status          string     `json:"status"`
	ResourceDetails any        `json:"resource_details,omitempty"`
	ExpiresAt       time.Time  `json:"expires_at"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	DeniedAt        *time.Time `json:"denied_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type approvalListResponse struct {
	Data       []approvalResponse `json:"data"`
	HasMore    bool               `json:"has_more"`
	NextCursor *string            `json:"next_cursor,omitempty"`
}

type approveResponse struct {
	ApprovalID       string           `json:"approval_id"`
	Status           string           `json:"status"`
	ApprovedAt       time.Time        `json:"approved_at"`
	ConfirmationCode string           `json:"confirmation_code"`
	ExecutionStatus  *string          `json:"execution_status,omitempty"`
	ExecutionResult  *json.RawMessage `json:"execution_result,omitempty"`
}

type denyResponse struct {
	ApprovalID string    `json:"approval_id"`
	Status     string    `json:"status"`
	DeniedAt   time.Time `json:"denied_at"`
}

// approvalDetailResponse includes execution details for the activity feed
// detail panel. Extends the base approvalResponse with execution fields.
type approvalDetailResponse struct {
	ApprovalID      string     `json:"approval_id"`
	AgentID         int64      `json:"agent_id"`
	Action          any        `json:"action"`
	Context         any        `json:"context"`
	Status          string     `json:"status"`
	ExecutionStatus *string    `json:"execution_status,omitempty"`
	ExecutionResult any        `json:"execution_result,omitempty"`
	ExecutedAt      *time.Time `json:"executed_at,omitempty"`
	ResourceDetails any        `json:"resource_details,omitempty"`
	ExpiresAt       time.Time  `json:"expires_at"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	DeniedAt        *time.Time `json:"denied_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func init() {
	RegisterRouteGroup(RegisterApprovalRoutes)
}

// RegisterApprovalRoutes adds approval-related endpoints to the mux.
func RegisterApprovalRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /approvals", requireProfile(handleListApprovals(deps)))
	mux.Handle("GET /approvals/{approval_id}", requireProfile(handleGetApproval(deps)))
	mux.Handle("POST /approvals/{approval_id}/approve", requireProfile(handleApproveApproval(deps)))
	mux.Handle("POST /approvals/{approval_id}/deny", requireProfile(handleDenyApproval(deps)))
}

var validStatusFilters = map[string]bool{
	"pending":   true,
	"approved":  true,
	"denied":    true,
	"cancelled": true,
	"all":       true,
}

func handleListApprovals(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		statusFilter := r.URL.Query().Get("status")
		if statusFilter == "" {
			statusFilter = "pending"
		}
		if !validStatusFilters[statusFilter] {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid status filter; must be one of: pending, approved, denied, cancelled, all"))
			return
		}

		limit, ok := parsePaginationLimit(w, r)
		if !ok {
			return
		}

		// Parse cursor: "<RFC3339Nano>,<approval_id>".
		var cursor *db.ApprovalCursor
		if v := r.URL.Query().Get("after"); v != "" {
			c, err := parseApprovalCursor(v)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid pagination cursor"))
				return
			}
			cursor = c
		}

		page, err := db.ListApprovalsByApproverPaginated(r.Context(), deps.DB, profile.ID, statusFilter, limit, cursor)
		if err != nil {
			log.Printf("[%s] ListApprovals: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list approvals"))
			return
		}

		data := make([]approvalResponse, len(page.Approvals))
		for i, a := range page.Approvals {
			data[i] = toApprovalResponse(a)
		}

		resp := approvalListResponse{
			Data:    data,
			HasMore: page.HasMore,
		}
		if page.HasMore && len(page.Approvals) > 0 {
			c := encodeApprovalCursor(page.Approvals[len(page.Approvals)-1])
			resp.NextCursor = &c
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// handleGetApproval returns a single approval by ID for the authenticated user.
// Used by the activity feed detail panel to show full action parameters,
// context, and execution results for historical approvals.
func handleGetApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		approvalID := r.PathValue("approval_id")

		if strings.TrimSpace(approvalID) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "approval_id is required"))
			return
		}

		appr, err := db.GetApprovalByIDAndApprover(r.Context(), deps.DB, approvalID, profile.ID)
		if err != nil {
			log.Printf("[%s] GetApprovalByIDAndApprover: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get approval"))
			return
		}
		if appr == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Approval not found"))
			return
		}

		resp := toApprovalDetailResponse(*appr)
		RespondJSON(w, http.StatusOK, resp)
	}
}

func handleApproveApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		approvalID := r.PathValue("approval_id")

		if strings.TrimSpace(approvalID) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "approval_id is required"))
			return
		}

		appr, agentMeta, err := db.ApproveApproval(r.Context(), deps.DB, approvalID, profile.ID)
		if err != nil {
			if handleApprovalError(w, r, err) {
				return
			}
			log.Printf("[%s] ApproveApproval: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to approve approval"))
			return
		}

		// Generate an ephemeral confirmation code for the approver to share
		// with the agent as a visual receipt of the approval.
		confirmCode, err := generateConfirmationCodePlaintext()
		if err != nil {
			log.Printf("[%s] generateConfirmationCodePlaintext: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to generate confirmation code"))
			return
		}

		// Execute the action via the connector now that the approval is granted.
		// Detach from the request context so a client disconnect doesn't abort
		// the connector call or DB update mid-flight.
		execCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 30*time.Second)
		defer cancel()
		execStatus, execResultJSON := executeApprovalAction(execCtx, deps, profile.ID, appr)

		emitApprovalAuditEvent(r.Context(), deps.DB, profile.ID, appr, agentMeta)

		// Notify any connected SSE clients (e.g. other browser tabs).
		// Keep approval_resolved for backward compatibility (frontend listens for it).
		notifyApprovalChange(deps, profile.ID, "approval_resolved", appr.ApprovalID)
		notifyApprovalExecuted(deps, profile.ID, appr.ApprovalID, execStatus)

		resp := approveResponse{
			ApprovalID:       appr.ApprovalID,
			Status:           appr.Status,
			ApprovedAt:       *appr.ApprovedAt,
			ConfirmationCode: confirmCode,
			ExecutionStatus:  &execStatus,
		}
		if execResultJSON != nil {
			raw := json.RawMessage(execResultJSON)
			resp.ExecutionResult = &raw
		}
		RespondJSON(w, http.StatusOK, resp)
	}
}

// executeApprovalAction runs the connector action for an approved one-off
// approval and persists the execution result. Returns the execution status
// ("success" or "error") and the result JSON. Errors are logged but never
// fail the approve response — the approval itself is already committed.
func executeApprovalAction(ctx context.Context, deps *Deps, userID string, appr *db.Approval) (string, json.RawMessage) {
	actionType := actionTypeFromJSON(appr.Action)

	// Extract parameters from the action JSON.
	var actionObj struct {
		Parameters json.RawMessage `json:"parameters"`
	}
	if len(appr.Action) > 0 {
		_ = json.Unmarshal(appr.Action, &actionObj)
	}

	result, execErr := executeConnectorAction(ctx, deps, appr.AgentID, userID, actionType, actionObj.Parameters, nil)

	var execStatus string
	var resultJSON json.RawMessage

	if execErr != nil {
		execStatus = "error"
		errMsg := execErr.Error()
		if len(errMsg) > 512 {
			errMsg = errMsg[:512]
		}
		resultJSON, _ = json.Marshal(map[string]string{"error": errMsg})
		log.Printf("[%s] executeApprovalAction: connector error for approval %s: %v", TraceID(ctx), appr.ApprovalID, execErr)
		CaptureConnectorError(ctx, execErr, ConnectorContext{ActionType: actionType, AgentID: appr.AgentID})
	} else if result == nil {
		// No connector registered for this action type — nothing was executed.
		execStatus = "error"
		resultJSON, _ = json.Marshal(map[string]string{"error": "no connector registered for action type"})
		log.Printf("[%s] executeApprovalAction: no connector for approval %s action type %q", TraceID(ctx), appr.ApprovalID, actionType)
		CaptureError(ctx, fmt.Errorf("no connector registered for approval %s action type %q", appr.ApprovalID, actionType))
	} else {
		execStatus = "success"
		resultJSON = result.Data
	}

	// Persist execution result on the approval row (best-effort).
	if err := db.UpdateApprovalExecution(ctx, deps.DB, appr.ApprovalID, execStatus, resultJSON); err != nil {
		log.Printf("[%s] executeApprovalAction: failed to store execution result for approval %s: %v", TraceID(ctx), appr.ApprovalID, err)
		CaptureError(ctx, err)
	}

	return execStatus, resultJSON
}

func handleDenyApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		approvalID := r.PathValue("approval_id")

		if strings.TrimSpace(approvalID) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "approval_id is required"))
			return
		}

		appr, agentMeta, err := db.DenyApproval(r.Context(), deps.DB, approvalID, profile.ID)
		if err != nil {
			if handleApprovalError(w, r, err) {
				return
			}
			log.Printf("[%s] DenyApproval: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to deny approval"))
			return
		}

		emitApprovalAuditEvent(r.Context(), deps.DB, profile.ID, appr, agentMeta)

		// Notify any connected SSE clients (e.g. other browser tabs).
		notifyApprovalChange(deps, profile.ID, "approval_resolved", appr.ApprovalID)

		RespondJSON(w, http.StatusOK, denyResponse{
			ApprovalID: appr.ApprovalID,
			Status:     appr.Status,
			DeniedAt:   *appr.DeniedAt,
		})
	}
}

// handleApprovalError maps db.ApprovalError to the appropriate HTTP response.
// Returns true if the error was handled, false if the caller should handle it.
func handleApprovalError(w http.ResponseWriter, r *http.Request, err error) bool {
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

func encodeApprovalCursor(a db.Approval) string {
	return a.CreatedAt.UTC().Format(time.RFC3339Nano) + "," + a.ApprovalID
}

func parseApprovalCursor(raw string) (*db.ApprovalCursor, error) {
	ts, approvalID, ok := strings.Cut(raw, ",")
	if !ok {
		return nil, fmt.Errorf("missing comma separator")
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}
	if approvalID == "" {
		return nil, fmt.Errorf("empty approval_id")
	}
	return &db.ApprovalCursor{CreatedAt: t, ApprovalID: approvalID}, nil
}

func toApprovalResponse(a db.Approval) approvalResponse {
	resp := approvalResponse{
		ApprovalID:  a.ApprovalID,
		AgentID:     a.AgentID,
		Status:      a.Status,
		ExpiresAt:   a.ExpiresAt,
		ApprovedAt:  a.ApprovedAt,
		DeniedAt:    a.DeniedAt,
		CancelledAt: a.CancelledAt,
		CreatedAt:   a.CreatedAt,
	}

	// Unmarshal JSONB fields into generic types for JSON output.
	if len(a.Action) > 0 {
		var action any
		if err := json.Unmarshal(a.Action, &action); err == nil {
			resp.Action = action
		} else {
			log.Printf("WARNING: corrupt action JSONB for approval %s: %v", a.ApprovalID, err)
			sentry.CurrentHub().CaptureException(fmt.Errorf("corrupt action JSONB for approval %s: %w", a.ApprovalID, err))
		}
	}
	if len(a.Context) > 0 {
		var ctx any
		if err := json.Unmarshal(a.Context, &ctx); err == nil {
			resp.Context = ctx
		} else {
			log.Printf("WARNING: corrupt context JSONB for approval %s: %v", a.ApprovalID, err)
			sentry.CurrentHub().CaptureException(fmt.Errorf("corrupt context JSONB for approval %s: %w", a.ApprovalID, err))
		}
	}
	if len(a.ResourceDetails) > 0 {
		var details any
		if err := json.Unmarshal(a.ResourceDetails, &details); err == nil {
			resp.ResourceDetails = details
		}
	}

	return resp
}

// toApprovalDetailResponse converts a db.Approval to the detail response
// that includes execution fields for the activity feed detail panel.
func toApprovalDetailResponse(a db.Approval) approvalDetailResponse {
	resp := approvalDetailResponse{
		ApprovalID:      a.ApprovalID,
		AgentID:         a.AgentID,
		Status:          a.Status,
		ExecutionStatus: a.ExecutionStatus,
		ExecutedAt:      a.ExecutedAt,
		ExpiresAt:       a.ExpiresAt,
		ApprovedAt:      a.ApprovedAt,
		DeniedAt:        a.DeniedAt,
		CancelledAt:     a.CancelledAt,
		CreatedAt:       a.CreatedAt,
	}

	if len(a.Action) > 0 {
		var action any
		if err := json.Unmarshal(a.Action, &action); err == nil {
			resp.Action = action
		}
	}
	if len(a.Context) > 0 {
		var ctx any
		if err := json.Unmarshal(a.Context, &ctx); err == nil {
			resp.Context = ctx
		}
	}
	if len(a.ExecutionResult) > 0 {
		var result any
		if err := json.Unmarshal(a.ExecutionResult, &result); err == nil {
			resp.ExecutionResult = result
		}
	}
	if len(a.ResourceDetails) > 0 {
		var details any
		if err := json.Unmarshal(a.ResourceDetails, &details); err == nil {
			resp.ResourceDetails = details
		}
	}

	return resp
}

// emitApprovalAuditEvent writes an audit event for an approval resolution
// (approve/deny/cancel). Errors are logged but not propagated — the audit
// trail is best-effort and must not block the primary operation.
//
// Only the action type is persisted — parameters and other user-provided
// fields are redacted to avoid storing sensitive data in the audit trail.
func emitApprovalAuditEvent(ctx context.Context, d db.DBTX, userID string, appr *db.Approval, agentMeta []byte) {
	var eventType db.AuditEventType
	switch appr.Status {
	case "approved":
		eventType = db.AuditEventApprovalApproved
	case "denied":
		eventType = db.AuditEventApprovalDenied
	case "cancelled":
		eventType = db.AuditEventApprovalCancelled
	default:
		return
	}

	emitAuditEventWithUsage(ctx, d, db.InsertAuditEventParams{
		UserID:      userID,
		AgentID:     appr.AgentID,
		EventType:   eventType,
		Outcome:     appr.Status,
		SourceID:    appr.ApprovalID,
		SourceType:  "approval",
		AgentMeta:   agentMeta,
		Action:      redactActionToType(appr.Action),
		ConnectorID: connectorIDFromActionType(actionTypeFromJSON(appr.Action)),
	}, false)
}

// generateConfirmationCodePlaintext produces a 6-character confirmation code
// formatted as "XXX-XXX". No hash is computed — the plaintext is stored
// directly. Used by the agent registration flow where codes are short-lived
// and displayed on the dashboard.
func generateConfirmationCodePlaintext() (string, error) {
	raw, err := generateRandomCode(shared.ConfirmationCodeLength)
	if err != nil {
		return "", err
	}
	return raw[:3] + "-" + raw[3:], nil
}
