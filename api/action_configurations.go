package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/shared"
)

// --- Request / response types ---

type createActionConfigRequest struct {
	AgentID      int64            `json:"agent_id" validate:"gt=0"`
	ConnectorID  string           `json:"connector_id" validate:"required"`
	ActionType   string           `json:"action_type" validate:"required"`
	CredentialID *string          `json:"credential_id,omitempty"`
	Parameters   json.RawMessage  `json:"parameters"`
	Name         string           `json:"name" validate:"required"`
	Description  *string          `json:"description,omitempty"`
}

type updateActionConfigRequest struct {
	CredentialID *string          `json:"credential_id,omitempty"`
	Parameters   json.RawMessage  `json:"parameters,omitempty"`
	Status       *string          `json:"status,omitempty" validate:"omitempty,oneof=active disabled"`
	Name         *string          `json:"name,omitempty"`
	Description  *string          `json:"description,omitempty"`
}

type actionConfigResponse struct {
	ID           string    `json:"id"`
	AgentID      int64     `json:"agent_id"`
	ConnectorID  string    `json:"connector_id"`
	ActionType   string    `json:"action_type"`
	CredentialID *string   `json:"credential_id,omitempty"`
	Parameters   any       `json:"parameters"`
	Status       string    `json:"status"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type actionConfigListResponse struct {
	Data []actionConfigResponse `json:"data"`
}

type deleteActionConfigResponse struct {
	ID        string    `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

// --- Routes ---

func init() {
	RegisterRouteGroup(RegisterActionConfigRoutes)
}

// RegisterActionConfigRoutes adds action configuration endpoints to the mux.
func RegisterActionConfigRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("POST /action-configurations", requireProfile(handleCreateActionConfig(deps)))
	mux.Handle("GET /action-configurations", requireProfile(handleListActionConfigs(deps)))
	mux.Handle("GET /action-configurations/{config_id}", requireProfile(handleGetActionConfig(deps)))
	mux.Handle("PUT /action-configurations/{config_id}", requireProfile(handleUpdateActionConfig(deps)))
	mux.Handle("DELETE /action-configurations/{config_id}", requireProfile(handleDeleteActionConfig(deps)))
}

// --- Handlers ---

func handleCreateActionConfig(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req createActionConfigRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		if len(req.Name) > shared.ActionConfigNameMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "name exceeds maximum length"))
			return
		}
		if req.Description != nil && len(*req.Description) > shared.ActionConfigDescMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "description exceeds maximum length"))
			return
		}

		// Reject action types that contain "*" but are not exactly "*".
		if req.ActionType != db.WildcardActionType && strings.Contains(req.ActionType, "*") {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "action_type must be '*' for wildcard or a specific action type without '*'"))
			return
		}

		// Wildcard configs cover all actions with all parameters agent-controlled.
		// Force empty parameters and skip parameter validation.
		var params json.RawMessage
		if req.ActionType == db.WildcardActionType {
			params = []byte("{}")
		} else {
			// Validate parameters is a JSON object (if provided).
			params = req.Parameters
			if len(params) == 0 {
				params = []byte("{}")
			} else if err := ValidateJSONObject(params); err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "parameters must be a JSON object"))
				return
			}

			// Reject malformed $pattern wrappers (e.g. without any "*").
			if err := db.ValidateConfigParameters(params); err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error()))
				return
			}

			// Validate that the action type exists for this connector (replaces
			// the dropped composite FK).
			exists, err := db.ConnectorActionExists(r.Context(), deps.DB, req.ConnectorID, req.ActionType)
			if err != nil {
				log.Printf("[%s] CreateActionConfig: check connector action: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create action configuration"))
				return
			}
			if !exists {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Invalid connector or action type reference"))
				return
			}
		}

		if !verifyCredentialOwnership(w, r, deps, req.CredentialID, profile.ID) {
			return
		}

		configID, err := generatePrefixedID("ac_", 16)
		if err != nil {
			log.Printf("[%s] CreateActionConfig: generate ID: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create action configuration"))
			return
		}

		ac, err := db.CreateActionConfig(r.Context(), deps.DB, db.CreateActionConfigParams{
			ID:           configID,
			AgentID:      req.AgentID,
			UserID:       profile.ID,
			ConnectorID:  req.ConnectorID,
			ActionType:   req.ActionType,
			CredentialID: req.CredentialID,
			Parameters:   params,
			Name:         req.Name,
			Description:  req.Description,
		})
		if err != nil {
			var acErr *db.ActionConfigError
			if errors.As(err, &acErr) {
				switch acErr.Code {
				case db.ActionConfigErrAgentNotFound:
					RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found or not owned by user"))
					return
				case db.ActionConfigErrInvalidRef:
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Invalid connector, action type, or credential reference"))
					return
				}
			}
			log.Printf("[%s] CreateActionConfig: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create action configuration"))
			return
		}

		RespondJSON(w, http.StatusCreated, toActionConfigResponse(*ac))
	}
}

func handleListActionConfigs(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		agentIDStr := r.URL.Query().Get("agent_id")
		if agentIDStr == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "agent_id query parameter is required"))
			return
		}
		agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
		if err != nil || agentID <= 0 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "agent_id must be a positive integer"))
			return
		}

		configs, err := db.ListActionConfigsByAgent(r.Context(), deps.DB, agentID, profile.ID)
		if err != nil {
			log.Printf("[%s] ListActionConfigs: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list action configurations"))
			return
		}

		data := make([]actionConfigResponse, len(configs))
		for i, ac := range configs {
			data[i] = toActionConfigResponse(ac)
		}

		RespondJSON(w, http.StatusOK, actionConfigListResponse{Data: data})
	}
}

func handleGetActionConfig(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		configID := r.PathValue("config_id")

		if configID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "config_id is required"))
			return
		}

		ac, err := db.GetActionConfigByID(r.Context(), deps.DB, configID, profile.ID)
		if err != nil {
			log.Printf("[%s] GetActionConfig: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get action configuration"))
			return
		}
		if ac == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrActionConfigNotFound, "Action configuration not found"))
			return
		}

		RespondJSON(w, http.StatusOK, toActionConfigResponse(*ac))
	}
}

func handleUpdateActionConfig(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		configID := r.PathValue("config_id")

		if configID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "config_id is required"))
			return
		}

		var req updateActionConfigRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		// Validate at least one field is provided.
		if req.CredentialID == nil && req.Parameters == nil && req.Status == nil && req.Name == nil && req.Description == nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "at least one field must be provided for update"))
			return
		}

		if req.Name != nil && *req.Name == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "name cannot be empty"))
			return
		}
		if req.Name != nil && len(*req.Name) > shared.ActionConfigNameMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "name exceeds maximum length"))
			return
		}
		if req.Description != nil && len(*req.Description) > shared.ActionConfigDescMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "description exceeds maximum length"))
			return
		}

		// Wildcard configs (action_type = "*") do not allow parameter changes —
		// their parameters must remain {}. Look up the existing config to check.
		var params []byte
		if req.Parameters != nil {
			existing, err := db.GetActionConfigByID(r.Context(), deps.DB, configID, profile.ID)
			if err != nil {
				log.Printf("[%s] UpdateActionConfig: lookup for wildcard check: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update action configuration"))
				return
			}
			if existing == nil {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrActionConfigNotFound, "Action configuration not found"))
				return
			}
			if existing.ActionType == db.WildcardActionType {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "cannot modify parameters on a wildcard (enable-all) configuration"))
				return
			}

			if err := ValidateJSONObject(req.Parameters); err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "parameters must be a JSON object"))
				return
			}
			// Reject malformed $pattern wrappers (e.g. without any "*").
			if err := db.ValidateConfigParameters(req.Parameters); err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error()))
				return
			}
			params = req.Parameters
		}

		if !verifyCredentialOwnership(w, r, deps, req.CredentialID, profile.ID) {
			return
		}

		ac, err := db.UpdateActionConfig(r.Context(), deps.DB, db.UpdateActionConfigParams{
			ID:           configID,
			UserID:       profile.ID,
			CredentialID: req.CredentialID,
			Parameters:   params,
			Status:       req.Status,
			Name:         req.Name,
			Description:  req.Description,
		})
		if err != nil {
			var acErr *db.ActionConfigError
			if errors.As(err, &acErr) {
				switch acErr.Code {
				case db.ActionConfigErrNotFound:
					RespondError(w, r, http.StatusNotFound, NotFound(ErrActionConfigNotFound, "Action configuration not found"))
					return
				case db.ActionConfigErrInvalidRef:
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Invalid credential reference"))
					return
				}
			}
			log.Printf("[%s] UpdateActionConfig: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update action configuration"))
			return
		}

		RespondJSON(w, http.StatusOK, toActionConfigResponse(*ac))
	}
}

func handleDeleteActionConfig(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		configID := r.PathValue("config_id")

		if configID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "config_id is required"))
			return
		}

		ac, err := db.DeleteActionConfig(r.Context(), deps.DB, configID, profile.ID)
		if err != nil {
			log.Printf("[%s] DeleteActionConfig: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete action configuration"))
			return
		}
		if ac == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrActionConfigNotFound, "Action configuration not found"))
			return
		}

		RespondJSON(w, http.StatusOK, deleteActionConfigResponse{
			ID:        configID,
			DeletedAt: ac.UpdatedAt,
		})
	}
}

// --- Helpers ---

// verifyCredentialOwnership checks that the credential belongs to the user and
// writes an error response if it doesn't. Returns true if the caller should
// continue (credential is valid or nil), false if an error response was sent.
func verifyCredentialOwnership(w http.ResponseWriter, r *http.Request, deps *Deps, credentialID *string, userID string) bool {
	if credentialID == nil {
		return true
	}
	owns, err := db.CredentialBelongsToUser(r.Context(), deps.DB, *credentialID, userID)
	if err != nil {
		log.Printf("[%s] credential ownership check: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify credential"))
		return false
	}
	if !owns {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Credential not found"))
		return false
	}
	return true
}

func toActionConfigResponse(ac db.ActionConfiguration) actionConfigResponse {
	resp := actionConfigResponse{
		ID:           ac.ID,
		AgentID:      ac.AgentID,
		ConnectorID:  ac.ConnectorID,
		ActionType:   ac.ActionType,
		CredentialID: ac.CredentialID,
		Status:       ac.Status,
		Name:         ac.Name,
		Description:  ac.Description,
		CreatedAt:    ac.CreatedAt,
		UpdatedAt:    ac.UpdatedAt,
	}
	// Parse parameters into a generic map for clean JSON output.
	if len(ac.Parameters) > 0 {
		var params any
		if err := json.Unmarshal(ac.Parameters, &params); err != nil {
			log.Printf("warning: failed to unmarshal action configuration %s parameters: %v", ac.ID, err)
		} else {
			resp.Parameters = params
		}
	}
	if resp.Parameters == nil {
		resp.Parameters = map[string]any{}
	}
	return resp
}
