package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/shared"
)

// Response types for the dashboard standing approval endpoints.

type standingApprovalResponse struct {
	StandingApprovalID          string     `json:"standing_approval_id"`
	AgentID                     int64      `json:"agent_id"`
	UserID                      string     `json:"user_id"`
	ActionType                  string     `json:"action_type"`
	ActionVersion               string     `json:"action_version"`
	Constraints                 any        `json:"constraints"`
	SourceActionConfigurationID *string    `json:"source_action_configuration_id"`
	Status                      string     `json:"status"`
	MaxExecutions               *int       `json:"max_executions"`
	ExecutionCount              int        `json:"execution_count"`
	StartsAt                    time.Time  `json:"starts_at"`
	ExpiresAt                   *time.Time `json:"expires_at"`
	CreatedAt                   time.Time  `json:"created_at"`
	RevokedAt                   *time.Time `json:"revoked_at,omitempty"`
}

type standingApprovalListResponse struct {
	Data       []standingApprovalResponse `json:"data"`
	HasMore    bool                       `json:"has_more"`
	NextCursor *string                    `json:"next_cursor,omitempty"`
}

type revokeStandingApprovalResponse struct {
	StandingApprovalID string    `json:"standing_approval_id"`
	Status             string    `json:"status"`
	RevokedAt          time.Time `json:"revoked_at"`
}

type createStandingApprovalRequest struct {
	AgentID                     int64           `json:"agent_id" validate:"gt=0"`
	ActionType                  string          `json:"action_type" validate:"required"`
	ActionVersion               string          `json:"action_version"`
	Constraints                 json.RawMessage `json:"constraints"`
	SourceActionConfigurationID *string         `json:"source_action_configuration_id"`
	MaxExecutions               *int            `json:"max_executions" validate:"omitempty,gte=1"`
	StartsAt                    *time.Time      `json:"starts_at"`
	ExpiresAt                   *time.Time      `json:"expires_at"`
}

type updateStandingApprovalRequest struct {
	Constraints   json.RawMessage `json:"constraints"`
	MaxExecutions *int            `json:"max_executions" validate:"omitempty,gte=1"`
	ExpiresAt     *time.Time      `json:"expires_at"`
	// ExpiresAtSet is true when the JSON payload explicitly included the "expires_at" key
	// (even if the value was null). This distinguishes "field omitted" (preserve existing)
	// from "field set to null" (clear expiry → until revoked).
	ExpiresAtSet bool `json:"-"`
}

func (r *updateStandingApprovalRequest) UnmarshalJSON(data []byte) error {
	// Check whether "expires_at" key is present in the raw JSON.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	_, r.ExpiresAtSet = raw["expires_at"]

	// Unmarshal into an alias to avoid infinite recursion.
	type alias updateStandingApprovalRequest
	return json.Unmarshal(data, (*alias)(r))
}

type executeStandingApprovalRequest struct {
	Parameters json.RawMessage `json:"parameters"`
}

type executeStandingApprovalResponse struct {
	StandingApprovalID string           `json:"standing_approval_id"`
	ExecutionID        int64            `json:"execution_id"`
	ExecutedAt         time.Time        `json:"executed_at"`
	ActionResult       *json.RawMessage `json:"action_result,omitempty"` // present when a connector action was executed
}

var validStandingApprovalStatusFilters = map[string]bool{
	"active":    true,
	"expired":   true,
	"revoked":   true,
	"exhausted": true,
	"all":       true,
}

var actionVersionPattern = regexp.MustCompile(`^\d+$`)

// maxActionConfigIDLength is the maximum length for source_action_configuration_id.
// Generated IDs are ~35 chars (prefix + 32 hex); 128 is generous headroom.
const maxActionConfigIDLength = 128

func init() {
	RegisterRouteGroup(RegisterStandingApprovalRoutes)
}

// RegisterStandingApprovalRoutes adds standing-approval-related endpoints to the mux.
func RegisterStandingApprovalRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /standing-approvals", requireProfile(handleListStandingApprovals(deps)))
	mux.Handle("POST /standing-approvals/create", requireProfile(handleCreateStandingApproval(deps)))
	mux.Handle("POST /standing-approvals/{standing_approval_id}/revoke", requireProfile(handleRevokeStandingApproval(deps)))
	mux.Handle("POST /standing-approvals/{standing_approval_id}/execute", requireProfile(handleExecuteStandingApproval(deps)))
	mux.Handle("POST /standing-approvals/{standing_approval_id}/update", requireProfile(handleUpdateStandingApproval(deps)))
}

func handleListStandingApprovals(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		statusFilter := r.URL.Query().Get("status")
		if statusFilter == "" {
			statusFilter = "active"
		}
		if !validStandingApprovalStatusFilters[statusFilter] {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid status filter; must be one of: active, expired, revoked, exhausted, all"))
			return
		}

		limit, ok := parsePaginationLimit(w, r)
		if !ok {
			return
		}

		// Parse cursor: "<RFC3339Nano>,<standing_approval_id>".
		var cursor *db.StandingApprovalCursor
		if v := r.URL.Query().Get("after"); v != "" {
			c, err := parseStandingApprovalCursor(v)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid pagination cursor"))
				return
			}
			cursor = c
		}

		page, err := db.ListStandingApprovalsByUser(r.Context(), deps.DB, profile.ID, statusFilter, limit, cursor)
		if err != nil {
			log.Printf("[%s] ListStandingApprovals: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list standing approvals"))
			return
		}

		data := make([]standingApprovalResponse, len(page.Approvals))
		for i, sa := range page.Approvals {
			data[i] = toStandingApprovalResponse(sa)
		}

		resp := standingApprovalListResponse{
			Data:    data,
			HasMore: page.HasMore,
		}
		if page.HasMore && len(page.Approvals) > 0 {
			c := encodeStandingApprovalCursor(page.Approvals[len(page.Approvals)-1])
			resp.NextCursor = &c
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

func handleCreateStandingApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req createStandingApprovalRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		// Trim before validation so the required tag rejects whitespace-only strings.
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

		startsAt := time.Now().UTC()
		if req.StartsAt != nil {
			startsAt = *req.StartsAt
		}

		if req.ExpiresAt != nil && req.ExpiresAt.Before(startsAt) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "expires_at must be after starts_at"))
			return
		}

		saID, err := generatePrefixedID("sa_", 16)
		if err != nil {
			log.Printf("[%s] CreateStandingApproval: generate ID: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval"))
			return
		}

		if req.SourceActionConfigurationID != nil && (len(*req.SourceActionConfigurationID) == 0 || len(*req.SourceActionConfigurationID) > maxActionConfigIDLength) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "source_action_configuration_id must be between 1 and 128 characters"))
			return
		}

		constraintsBytes, err := validateStandingApprovalConstraints(req.Constraints)
		if err != nil {
			resp := BadRequest(ErrInvalidConstraints, err.Error())
			resp.Error.Details = map[string]any{
				"hint": "Provide a JSON object with at least one non-wildcard constraint, e.g. {\"repo\": \"my-org/my-repo\", \"title\": \"*\"}",
			}
			RespondError(w, r, http.StatusBadRequest, resp)
			return
		}

		// Wrap limit check + insert in a transaction with an advisory lock
		// to prevent TOCTOU races where concurrent requests could both pass
		// the limit check and exceed the plan cap.
		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] CreateStandingApproval: begin tx: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		if err := db.AcquireStandingApprovalLimitLock(r.Context(), tx, profile.ID); err != nil {
			log.Printf("[%s] CreateStandingApproval: advisory lock: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval"))
			return
		}

		if checkStandingApprovalLimit(r.Context(), w, r, tx, profile.ID) {
			return
		}

		sa, err := db.CreateStandingApproval(r.Context(), tx, db.CreateStandingApprovalParams{
			StandingApprovalID:          saID,
			AgentID:                     req.AgentID,
			UserID:                      profile.ID,
			ActionType:                  req.ActionType,
			ActionVersion:               req.ActionVersion,
			Constraints:                 constraintsBytes,
			SourceActionConfigurationID: req.SourceActionConfigurationID,
			MaxExecutions:               req.MaxExecutions,
			StartsAt:                    startsAt,
			ExpiresAt:                   req.ExpiresAt,
		})
		if err != nil {
			var saErr *db.StandingApprovalError
			if errors.As(err, &saErr) && saErr.Code == db.StandingApprovalErrAgentNotFound {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
				return
			}
			log.Printf("[%s] CreateStandingApproval: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] CreateStandingApproval: commit: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval"))
				return
			}
		}

		RespondJSON(w, http.StatusCreated, toStandingApprovalResponse(*sa))
	}
}

func handleRevokeStandingApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		saID := r.PathValue("standing_approval_id")

		if strings.TrimSpace(saID) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "standing_approval_id is required"))
			return
		}

		sa, err := db.RevokeStandingApproval(r.Context(), deps.DB, saID, profile.ID)
		if err != nil {
			if handleStandingApprovalError(w, r, err) {
				return
			}
			log.Printf("[%s] RevokeStandingApproval: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to revoke standing approval"))
			return
		}

		RespondJSON(w, http.StatusOK, revokeStandingApprovalResponse{
			StandingApprovalID: sa.StandingApprovalID,
			Status:             sa.Status,
			RevokedAt:          *sa.RevokedAt,
		})
	}
}

// handleStandingApprovalError maps db.StandingApprovalError to the appropriate HTTP response.
// Returns true if the error was handled, false if the caller should handle it.
func handleStandingApprovalError(w http.ResponseWriter, r *http.Request, err error) bool {
	var saErr *db.StandingApprovalError
	if !errors.As(err, &saErr) {
		return false
	}
	switch saErr.Code {
	case db.StandingApprovalErrNotFound:
		RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Standing approval not found"))
	case db.StandingApprovalErrAlreadyRevoked:
		resp := Conflict(ErrApprovalAlreadyResolved, "Standing approval already revoked")
		resp.Error.Details = map[string]any{"status": saErr.Status}
		RespondError(w, r, http.StatusConflict, resp)
	case db.StandingApprovalErrNotActive:
		resp := Gone(ErrStandingExpired, "Standing approval is no longer active")
		if saErr.Status != "" {
			resp.Error.Details = map[string]any{"status": saErr.Status}
		}
		RespondError(w, r, http.StatusGone, resp)
	case db.StandingApprovalErrMaxExecutionsTooLow:
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "max_executions cannot be less than the current execution count"))
	default:
		return false
	}
	return true
}

func handleExecuteStandingApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		saID := r.PathValue("standing_approval_id")

		if strings.TrimSpace(saID) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "standing_approval_id is required"))
			return
		}

		var req executeStandingApprovalRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		// Validate parameters size and type.
		if len(req.Parameters) > shared.MaxParametersBytes {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "parameters exceeds maximum size"))
			return
		}
		if err := ValidateJSONObject(req.Parameters); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "parameters must be a JSON object"))
			return
		}

		// Check monthly request quota before executing.
		var blocked bool
		r, blocked = checkRequestQuota(r.Context(), w, r, deps.DB, profile.ID)
		if blocked {
			return
		}

		exec, err := db.RecordStandingApprovalExecution(r.Context(), deps.DB, saID, profile.ID, req.Parameters)
		if err != nil {
			if handleStandingApprovalError(w, r, err) {
				return
			}
			log.Printf("[%s] ExecuteStandingApproval: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to execute standing approval"))
			return
		}

		// Attempt connector execution. If no connector is registered for this
		// action type, the existing behavior (record execution, emit audit event)
		// still works — execution just returns no external result (graceful degradation).
		result, execErr := executeConnectorAction(r.Context(), deps, exec.AgentID, profile.ID, exec.ActionType, req.Parameters, nil)

		// Always emit the audit event with the actual execution result (best-effort).
		emitStandingApprovalAuditEvent(r.Context(), deps.DB, profile.ID, exec.AgentID, saID, exec.ActionType, exec.AgentMeta, execErr)

		if execErr != nil {
			if handleConnectorError(w, r, execErr) {
				return
			}
			log.Printf("[%s] ExecuteStandingApproval: connector execution: %v", TraceID(r.Context()), execErr)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to execute connector action"))
			return
		}

		var actionResultPtr *json.RawMessage
		if result != nil {
			actionResultPtr = &result.Data
		}

		RespondJSON(w, http.StatusOK, executeStandingApprovalResponse{
			StandingApprovalID: saID,
			ExecutionID:        exec.ExecutionID,
			ExecutedAt:         exec.ExecutedAt,
			ActionResult:       actionResultPtr,
		})
	}
}

func handleUpdateStandingApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		saID := r.PathValue("standing_approval_id")

		if strings.TrimSpace(saID) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "standing_approval_id is required"))
			return
		}

		var req updateStandingApprovalRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		if !ValidateRequest(w, r, &req) {
			return
		}

		if len(req.Constraints) > shared.MaxConstraintsBytes {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "constraints exceeds maximum size"))
			return
		}

		if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now().UTC()) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "expires_at must be in the future"))
			return
		}

		// Fetch the existing approval to validate max_executions against current
		// execution_count and to preserve expires_at when the field is omitted.
		existing, err := db.GetStandingApprovalByIDAndUser(r.Context(), deps.DB, saID, profile.ID)
		if err != nil {
			log.Printf("[%s] UpdateStandingApproval: fetch existing: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update standing approval"))
			return
		}
		if existing == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Standing approval not found"))
			return
		}
		if existing.Status != "active" {
			errCode := db.StandingApprovalErrNotActive
			if existing.Status == "revoked" {
				errCode = db.StandingApprovalErrAlreadyRevoked
			}
			handleStandingApprovalError(w, r, &db.StandingApprovalError{Code: errCode, Status: existing.Status})
			return
		}

		// When the client omits "expires_at" from the request body, preserve the
		// existing value. Only clear the expiry when the client explicitly sends null.
		if !req.ExpiresAtSet {
			req.ExpiresAt = existing.ExpiresAt
		}

		// Prevent setting max_executions below the current execution count, which
		// would leave the approval active in the dashboard but unreachable by agents.
		if req.MaxExecutions != nil && *req.MaxExecutions < existing.ExecutionCount {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "max_executions cannot be less than the current execution count"))
			return
		}

		constraintsBytes, err := validateStandingApprovalConstraints(req.Constraints)
		if err != nil {
			resp := BadRequest(ErrInvalidConstraints, err.Error())
			resp.Error.Details = map[string]any{
				"hint": "Provide a JSON object with at least one non-wildcard constraint, e.g. {\"repo\": \"my-org/my-repo\", \"title\": \"*\"}",
			}
			RespondError(w, r, http.StatusBadRequest, resp)
			return
		}

		sa, err := db.UpdateStandingApproval(r.Context(), deps.DB, db.UpdateStandingApprovalParams{
			StandingApprovalID: saID,
			UserID:             profile.ID,
			Constraints:        constraintsBytes,
			MaxExecutions:      req.MaxExecutions,
			ExpiresAt:          req.ExpiresAt,
		})
		if err != nil {
			if handleStandingApprovalError(w, r, err) {
				return
			}
			log.Printf("[%s] UpdateStandingApproval: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update standing approval"))
			return
		}

		emitStandingApprovalUpdateAuditEvent(r.Context(), deps.DB, profile.ID, sa.AgentID, saID, sa.ActionType)
		RespondJSON(w, http.StatusOK, toStandingApprovalResponse(*sa))
	}
}

// emitStandingApprovalUpdateAuditEvent writes a standing_approval.updated audit event.
func emitStandingApprovalUpdateAuditEvent(ctx context.Context, d db.DBTX, userID string, agentID int64, saID, actionType string) {
	actionJSON, _ := json.Marshal(map[string]string{"type": actionType})
	emitAuditEventWithUsage(ctx, d, db.InsertAuditEventParams{
		UserID:      userID,
		AgentID:     agentID,
		EventType:   db.AuditEventStandingUpdated,
		Outcome:     "updated",
		SourceID:    saID,
		SourceType:  "standing_approval",
		Action:      actionJSON,
		ConnectorID: connectorIDFromActionType(actionType),
	}, false)
}

// emitStandingApprovalAuditEvent writes a standing_approval.executed audit event.
// Billable: standing approval executions count toward the user's monthly request quota.
//
// execErr should be the error from connector execution, or nil on success.
// The execution_status and execution_error fields are derived from execErr.
func emitStandingApprovalAuditEvent(ctx context.Context, d db.DBTX, userID string, agentID int64, saID, actionType string, agentMeta []byte, execErr error) {
	actionJSON, _ := json.Marshal(map[string]string{"type": actionType})
	execStatus, execErrMsg := resolveExecResult(execErr)

	emitAuditEventWithUsage(ctx, d, db.InsertAuditEventParams{
		UserID:          userID,
		AgentID:         agentID,
		EventType:       db.AuditEventStandingExecution,
		Outcome:         "auto_executed",
		SourceID:        saID,
		SourceType:      "standing_approval",
		AgentMeta:       agentMeta,
		Action:          actionJSON,
		ConnectorID:     connectorIDFromActionType(actionType),
		ExecutionStatus: &execStatus,
		ExecutionError:  execErrMsg,
	}, true)
}

func toStandingApprovalResponse(sa db.StandingApproval) standingApprovalResponse {
	resp := standingApprovalResponse{
		StandingApprovalID:          sa.StandingApprovalID,
		AgentID:                     sa.AgentID,
		UserID:                      sa.UserID,
		ActionType:                  sa.ActionType,
		ActionVersion:               sa.ActionVersion,
		SourceActionConfigurationID: sa.SourceActionConfigurationID,
		Status:                      sa.Status,
		MaxExecutions:               sa.MaxExecutions,
		ExecutionCount:              sa.ExecutionCount,
		StartsAt:                    sa.StartsAt,
		ExpiresAt:                   sa.ExpiresAt,
		CreatedAt:                   sa.CreatedAt,
		RevokedAt:                   sa.RevokedAt,
	}

	if len(sa.Constraints) > 0 {
		var constraints any
		if err := json.Unmarshal(sa.Constraints, &constraints); err != nil {
			log.Printf("warning: failed to unmarshal standing approval %s constraints: %v", sa.StandingApprovalID, err)
		} else {
			resp.Constraints = constraints
		}
	}

	return resp
}

// validateStandingApprovalConstraints is the single validation point for standing
// approval constraints. It checks type, presence, and content. Returns the
// normalized bytes to store, or an error describing why the constraints are invalid.
//
// Rules:
//   - non-object JSON (array, string, number) → rejected
//   - null, empty, or {} → rejected (constraints are required)
//   - all values are "*" → rejected (at least one must be Fixed or Pattern)
//   - bare strings containing "*" (except the wildcard "*") → auto-wrapped as {"$pattern": "<value>"}
//   - valid otherwise → returns the normalized bytes
func validateStandingApprovalConstraints(raw json.RawMessage) ([]byte, error) {
	// Null or absent.
	if len(raw) == 0 || string(raw) == "null" {
		return nil, errors.New("constraints are required for standing approvals")
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, errors.New("constraints must be a JSON object")
	}

	if len(obj) == 0 {
		return nil, errors.New("constraints are required for standing approvals")
	}

	// Check that at least one constraint value is not a wildcard ("*").
	// Null values are rejected outright — use "*" for a wildcard or omit the key.
	// Bare strings containing "*" (except the wildcard "*") are auto-wrapped as patterns.
	allWildcard := true
	mutated := false
	for key, v := range obj {
		if string(v) == "null" {
			return nil, errors.New("constraint values must not be null; use \"*\" for a wildcard or omit the key entirely")
		}
		var s string
		if json.Unmarshal(v, &s) == nil {
			if s == "*" {
				continue // bare wildcard — stays as-is
			}
			allWildcard = false
			// Auto-wrap bare strings containing "*" as $pattern.
			// Only plain strings are wrapped; objects (e.g. already-wrapped
			// {"$pattern": "..."}) are left unchanged since json.Unmarshal
			// into a string fails for non-string JSON values.
			// Note: other glob metacharacters (?, [...]) are NOT auto-wrapped;
			// users who need them must use {"$pattern": "..."} explicitly.
			if strings.Contains(s, "*") {
				wrapped, err := json.Marshal(map[string]string{db.PatternKey: s})
				if err != nil {
					return nil, fmt.Errorf("failed to wrap pattern for %q: %w", key, err)
				}
				obj[key] = wrapped
				mutated = true
			}
		} else {
			allWildcard = false
		}
	}
	if allWildcard {
		return nil, errors.New("at least one constraint must be a non-wildcard value")
	}

	if mutated {
		normalized, err := json.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize constraints: %w", err)
		}
		return normalized, nil
	}

	return raw, nil
}
