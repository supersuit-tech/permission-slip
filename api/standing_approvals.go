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

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/shared"
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
	StartsAt                    *time.Time      `json:"starts_at"`
	ExpiresAt                   *time.Time      `json:"expires_at"`
}

type updateStandingApprovalRequest struct {
	Constraints json.RawMessage `json:"constraints"`
	ExpiresAt   *time.Time      `json:"expires_at"`
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
	"active":  true,
	"expired": true,
	"revoked": true,
	"all":     true,
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
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid status filter; must be one of: active, expired, revoked, all"))
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

		var sourceConfigID *string
		if v := strings.TrimSpace(r.URL.Query().Get("source_action_configuration_id")); v != "" {
			if len(v) > maxActionConfigIDLength {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "source_action_configuration_id exceeds maximum length"))
				return
			}
			sourceConfigID = &v
		}

		page, err := db.ListStandingApprovalsByUser(r.Context(), deps.DB, profile.ID, statusFilter, sourceConfigID, limit, cursor)
		if err != nil {
			log.Printf("[%s] ListStandingApprovals: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
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
		const maxStandingApprovalDuration = 90 * 24 * time.Hour
		if req.ExpiresAt != nil {
			if d := req.ExpiresAt.Sub(startsAt); d > maxStandingApprovalDuration {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Duration exceeds maximum of 90 days"))
				return
			}
		}

		saID, err := generatePrefixedID("sa_", 16)
		if err != nil {
			log.Printf("[%s] CreateStandingApproval: generate ID: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval"))
			return
		}

		var constraintsBytes []byte
		if len(req.Constraints) > 0 {
			s := strings.TrimSpace(string(req.Constraints))
			if s == "{}" || s == "null" {
				// Match-all parameters for this action type (trusted when tied to a source config,
				// e.g. template customize with all-wildcard parameters). Stored as NULL in DB.
				constraintsBytes = nil
			} else {
				var cErr error
				constraintsBytes, cErr = validateStandingApprovalConstraintsForAction(r.Context(), deps.DB, deps.Connectors, req.ActionType, req.Constraints)
				if cErr != nil {
					resp := BadRequest(ErrInvalidConstraints, cErr.Error())
					resp.Error.Details = map[string]any{
						"hint": "Provide a JSON object with at least one non-wildcard constraint, e.g. {\"repo\": \"my-org/my-repo\", \"title\": \"*\"}",
					}
					RespondError(w, r, http.StatusBadRequest, resp)
					return
				}
			}
		} else {
			var cErr error
			constraintsBytes, cErr = validateStandingApprovalConstraintsForAction(r.Context(), deps.DB, deps.Connectors, req.ActionType, req.Constraints)
			if cErr != nil {
				resp := BadRequest(ErrInvalidConstraints, cErr.Error())
				resp.Error.Details = map[string]any{
					"hint": "Provide a JSON object with at least one non-wildcard constraint, e.g. {\"repo\": \"my-org/my-repo\", \"title\": \"*\"}",
				}
				RespondError(w, r, http.StatusBadRequest, resp)
				return
			}
		}

		// Wrap insert in a transaction.
		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] CreateStandingApproval: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		sourceConfigIDPtr, ok := resolveSourceActionConfigForStandingApproval(w, r, tx, profile.ID, req, sourceActionConfigResolveOptions{
			LogLabel:       "CreateStandingApproval",
			AutoCreateName: autoApprovedFromRequestConfigName,
			FailureMessage: "Failed to create standing approval",
		})
		if !ok {
			return
		}

		sa, err := db.CreateStandingApproval(r.Context(), tx, db.CreateStandingApprovalParams{
			StandingApprovalID:          saID,
			AgentID:                     req.AgentID,
			UserID:                      profile.ID,
			ActionType:                  req.ActionType,
			ActionVersion:               req.ActionVersion,
			Constraints:                 constraintsBytes,
			SourceActionConfigurationID: sourceConfigIDPtr,
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
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] CreateStandingApproval: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create standing approval"))
				return
			}
		}

		RespondJSON(w, http.StatusCreated, toStandingApprovalResponse(*sa))
	}
}

const autoApprovedFromRequestConfigName = "Auto-approved from request"

const autoCreatedFromRuleProposalConfigName = "Auto-created from approved rule proposal"

var autoCreatedFromRuleProposalConfigDescription = "Created automatically when approving a standing auto-approve rule proposal"

type sourceActionConfigResolveOptions struct {
	LogLabel              string
	AutoCreateName        string
	AutoCreateDescription *string
	FailureMessage        string
}

// resolveSourceActionConfigForStandingApproval validates an explicit source config
// or auto-creates/reactivates a backing action configuration when omitted.
// Returns the config ID and true on success; false if an HTTP error was written.
func resolveSourceActionConfigForStandingApproval(
	w http.ResponseWriter,
	r *http.Request,
	tx db.DBTX,
	userID string,
	req createStandingApprovalRequest,
	opts sourceActionConfigResolveOptions,
) (*string, bool) {
	if req.SourceActionConfigurationID != nil {
		sourceConfigID := strings.TrimSpace(*req.SourceActionConfigurationID)
		if sourceConfigID == "" {
			// Empty string is treated as omitted — fall through to auto-create.
		} else {
			if len(sourceConfigID) > maxActionConfigIDLength {
				log.Printf("[%s] %s: source_action_configuration_id too long", TraceID(r.Context()), opts.LogLabel)
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "source_action_configuration_id must be between 1 and 128 characters"))
				return nil, false
			}

			ac, err := db.GetActionConfigByID(r.Context(), tx, sourceConfigID, userID)
			if err != nil {
				log.Printf("[%s] %s: load action config: %v", TraceID(r.Context()), opts.LogLabel, err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError(opts.FailureMessage))
				return nil, false
			}
			if ac == nil {
				log.Printf("[%s] %s: source_action_configuration_id not found: %s", TraceID(r.Context()), opts.LogLabel, sourceConfigID)
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "source_action_configuration_id must reference an existing action configuration"))
				return nil, false
			}
			if ac.AgentID != req.AgentID {
				log.Printf("[%s] %s: action config %s belongs to agent %d, expected %d", TraceID(r.Context()), opts.LogLabel, sourceConfigID, ac.AgentID, req.AgentID)
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action configuration does not belong to the specified agent"))
				return nil, false
			}
			if ac.ActionType != req.ActionType {
				log.Printf("[%s] %s: action config %s action_type mismatch: %q vs %q", TraceID(r.Context()), opts.LogLabel, sourceConfigID, ac.ActionType, req.ActionType)
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action_type must match the referenced action configuration"))
				return nil, false
			}
			return &sourceConfigID, true
		}
	}

	existing, err := db.FindLatestActionConfigForAgentActionType(r.Context(), tx, req.AgentID, userID, req.ActionType)
	if err != nil {
		log.Printf("[%s] %s: find backing action config: %v", TraceID(r.Context()), opts.LogLabel, err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError(opts.FailureMessage))
		return nil, false
	}
	if existing != nil {
		if existing.Status == "active" {
			id := existing.ID
			return &id, true
		}
		if existing.Status == "disabled" {
			active := "active"
			updated, err := db.UpdateActionConfig(r.Context(), tx, db.UpdateActionConfigParams{
				ID:     existing.ID,
				UserID: userID,
				Status: &active,
			})
			if err != nil {
				log.Printf("[%s] %s: reactivate action config: %v", TraceID(r.Context()), opts.LogLabel, err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError(opts.FailureMessage))
				return nil, false
			}
			id := updated.ID
			return &id, true
		}
	}

	connectorIDPtr := connectorIDFromActionType(req.ActionType)
	if connectorIDPtr == nil {
		log.Printf("[%s] %s: action type %q has no connector prefix", TraceID(r.Context()), opts.LogLabel, req.ActionType)
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Cannot auto-create action configuration: action type has no connector prefix"))
		return nil, false
	}
	connectorID := *connectorIDPtr

	exists, err := db.ConnectorActionExists(r.Context(), tx, connectorID, req.ActionType)
	if err != nil {
		log.Printf("[%s] %s: check connector action: %v", TraceID(r.Context()), opts.LogLabel, err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError(opts.FailureMessage))
		return nil, false
	}
	if !exists {
		log.Printf("[%s] %s: action type %q not registered for connector %q", TraceID(r.Context()), opts.LogLabel, req.ActionType, connectorID)
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Cannot auto-create action configuration: action type is not registered for connector"))
		return nil, false
	}

	configID, err := generatePrefixedID("ac_", 16)
	if err != nil {
		log.Printf("[%s] %s: generate action config ID: %v", TraceID(r.Context()), opts.LogLabel, err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError(opts.FailureMessage))
		return nil, false
	}

	ac, err := db.CreateActionConfig(r.Context(), tx, db.CreateActionConfigParams{
		ID:          configID,
		AgentID:     req.AgentID,
		UserID:      userID,
		ConnectorID: connectorID,
		ActionType:  req.ActionType,
		Parameters:  []byte("{}"),
		Name:        opts.AutoCreateName,
		Description: opts.AutoCreateDescription,
	})
	if err != nil {
		var acErr *db.ActionConfigError
		if errors.As(err, &acErr) {
			switch acErr.Code {
			case db.ActionConfigErrAgentNotFound:
				RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
				return nil, false
			case db.ActionConfigErrInvalidRef:
				log.Printf("[%s] %s: invalid connector or action type reference for %q", TraceID(r.Context()), opts.LogLabel, req.ActionType)
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Cannot auto-create action configuration: invalid connector or action type reference"))
				return nil, false
			}
		}
		log.Printf("[%s] %s: create backing action config: %v", TraceID(r.Context()), opts.LogLabel, err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError(opts.FailureMessage))
		return nil, false
	}

	id := ac.ID
	return &id, true
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
			CaptureError(r.Context(), err)
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

		exec, err := db.RecordStandingApprovalExecution(r.Context(), deps.DB, saID, profile.ID, req.Parameters)
		if err != nil {
			if handleStandingApprovalError(w, r, err) {
				return
			}
			log.Printf("[%s] ExecuteStandingApproval: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to execute standing approval"))
			return
		}

		// Defense-in-depth: re-verify that the caller still owns the agent the
		// standing approval references. RecordStandingApprovalExecution already
		// scopes by user_id, but the stored agent_id was bound at creation time
		// — re-checking ownership here protects against edge cases where agent
		// ownership changes after the standing approval is created (and keeps
		// the execution path consistent with every other connector-action entry
		// point, all of which call requireAgentOwnership).
		if !requireAgentOwnership(w, r, deps, exec.AgentID, profile.ID) {
			return
		}

		// Attempt connector execution. If no connector is registered for this
		// action type, the existing behavior (record execution, emit audit event)
		// still works — execution just returns no external result (graceful degradation).
		var connectorInstanceID string
		if exec.ConnectorInstanceID != nil {
			connectorInstanceID = *exec.ConnectorInstanceID
		}
		result, execErr := executeConnectorAction(r.Context(), deps, exec.AgentID, profile.ID, exec.ActionType, req.Parameters, nil, connectorInstanceID)

		// Always emit the audit event with the actual execution result (best-effort).
		emitStandingApprovalAuditEvent(r.Context(), deps.DB, profile.ID, exec.AgentID, saID, exec.ActionType, exec.AgentMeta, execErr)

		if execErr != nil {
			cc := ConnectorContext{ActionType: exec.ActionType, AgentID: exec.AgentID}
			if handleConnectorError(w, r, execErr, cc) {
				return
			}
			log.Printf("[%s] ExecuteStandingApproval: connector execution: %v", TraceID(r.Context()), execErr)
			CaptureConnectorError(r.Context(), execErr, cc)
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

		// Fetch the existing approval to preserve expires_at when the field is omitted.
		existing, err := db.GetStandingApprovalByIDAndUser(r.Context(), deps.DB, saID, profile.ID)
		if err != nil {
			log.Printf("[%s] UpdateStandingApproval: fetch existing: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
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

		constraintsBytes, err := validateStandingApprovalConstraintsForAction(r.Context(), deps.DB, deps.Connectors, existing.ActionType, req.Constraints)
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
			ExpiresAt:          req.ExpiresAt,
		})
		if err != nil {
			if handleStandingApprovalError(w, r, err) {
				return
			}
			log.Printf("[%s] UpdateStandingApproval: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
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
		Outcome:     db.OutcomeUpdated,
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
		if key == db.MetaNamespaceKey {
			var metaObj map[string]json.RawMessage
			if err := json.Unmarshal(v, &metaObj); err != nil {
				return nil, errors.New("$meta constraints must be a JSON object")
			}
			metaMutated := false
			for metaKey, metaVal := range metaObj {
				if string(metaVal) == "null" {
					return nil, errors.New("constraint values must not be null; use \"*\" for a wildcard or omit the key entirely")
				}
				var s string
				if json.Unmarshal(metaVal, &s) == nil {
					if s == "*" {
						continue
					}
					allWildcard = false
					if strings.Contains(s, "*") {
						wrapped, err := json.Marshal(map[string]string{db.PatternKey: s})
						if err != nil {
							return nil, fmt.Errorf("failed to wrap pattern for %q: %w", metaKey, err)
						}
						metaObj[metaKey] = wrapped
						metaMutated = true
					}
				} else {
					allWildcard = false
				}
			}
			if metaMutated {
				if normalized, err := json.Marshal(metaObj); err != nil {
					return nil, fmt.Errorf("failed to normalize $meta constraints: %w", err)
				} else {
					obj[key] = normalized
					mutated = true
				}
			}
			continue
		}
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
