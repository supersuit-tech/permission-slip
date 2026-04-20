package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
)

type agentConnectorInstanceResponse struct {
	ConnectorInstanceID string    `json:"connector_instance_id"`
	AgentID             int64     `json:"agent_id"`
	ConnectorID         string    `json:"connector_id"`
	Label               string    `json:"label"`
	IsDefault           bool      `json:"is_default"`
	EnabledAt           time.Time `json:"enabled_at"`
}

type agentConnectorInstanceListResponse struct {
	Data []agentConnectorInstanceResponse `json:"data"`
}

type createAgentConnectorInstanceRequest struct {
	Label string `json:"label"`
}

type patchAgentConnectorInstanceRequest struct {
	Label     *string `json:"label,omitempty"`
	IsDefault *bool   `json:"is_default,omitempty"`
}

func init() {
	RegisterRouteGroup(RegisterAgentConnectorInstanceRoutes)
}

// RegisterAgentConnectorInstanceRoutes registers multi-instance connector endpoints.
func RegisterAgentConnectorInstanceRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	base := "/agents/{agent_id}/connectors/{connector_id}/instances"
	mux.Handle("GET "+base, requireProfile(handleListAgentConnectorInstances(deps)))
	mux.Handle("POST "+base, requireProfile(handleCreateAgentConnectorInstance(deps)))

	inst := base + "/{instance_id}"
	mux.Handle("GET "+inst, requireProfile(handleGetAgentConnectorInstance(deps)))
	mux.Handle("PATCH "+inst, requireProfile(handlePatchAgentConnectorInstance(deps)))
	mux.Handle("DELETE "+inst, requireProfile(handleDeleteAgentConnectorInstance(deps)))

	mux.Handle("PUT "+inst+"/credential", requireProfile(handleAssignAgentConnectorInstanceCredential(deps)))
	mux.Handle("DELETE "+inst+"/credential", requireProfile(handleRemoveAgentConnectorInstanceCredential(deps)))
	mux.Handle("GET "+inst+"/credential", requireProfile(handleGetAgentConnectorInstanceCredential(deps)))

	mux.Handle("GET "+inst+"/channels", requireProfile(handleListAgentConnectorInstanceChannels(deps)))
	mux.Handle("GET "+inst+"/calendars", requireProfile(handleListAgentConnectorInstanceCalendars(deps)))
	mux.Handle("GET "+inst+"/users", requireProfile(handleListAgentConnectorInstanceUsers(deps)))
}

func toAgentConnectorInstanceResponse(inst db.AgentConnectorInstance) agentConnectorInstanceResponse {
	return agentConnectorInstanceResponse{
		ConnectorInstanceID: inst.ConnectorInstanceID,
		AgentID:             inst.AgentID,
		ConnectorID:         inst.ConnectorID,
		Label:               inst.Label,
		IsDefault:           inst.IsDefault,
		EnabledAt:           inst.EnabledAt,
	}
}

func handleListAgentConnectorInstances(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID
		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		instances, err := db.ListAgentConnectorInstances(r.Context(), deps.DB, agentID, userID, connectorID)
		if err != nil {
			log.Printf("[%s] ListAgentConnectorInstances: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list connector instances"))
			return
		}

		data := make([]agentConnectorInstanceResponse, len(instances))
		for i, inst := range instances {
			data[i] = toAgentConnectorInstanceResponse(inst)
		}
		RespondJSON(w, http.StatusOK, agentConnectorInstanceListResponse{Data: data})
	}
}

func handleCreateAgentConnectorInstance(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID
		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		var req createAgentConnectorInstanceRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		inst, err := db.CreateAgentConnectorInstance(r.Context(), deps.DB, db.CreateAgentConnectorInstanceParams{
			AgentID:     agentID,
			ApproverID:  userID,
			ConnectorID: connectorID,
			Label:       req.Label,
		})
		if err != nil {
			if errors.Is(err, db.ErrAgentConnectorInstanceLabelRequired) || errors.Is(err, db.ErrAgentConnectorInstanceLabelTooLong) {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "label must be 1–256 characters"))
				return
			}
			var acErr *db.AgentConnectorError
			if errors.As(err, &acErr) {
				switch acErr.Code {
				case db.AgentConnectorErrAgentNotFound:
					RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
					return
				case db.AgentConnectorErrConnectorNotFound:
					RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "Connector not found"))
					return
				case db.AgentConnectorErrConnectorNotEnabled:
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Enable this connector for the agent before adding another instance"))
					return
				}
			}
			var instErr *db.AgentConnectorInstanceError
			if errors.As(err, &instErr) && instErr.Code == db.AgentConnectorInstanceErrDuplicateLabel {
				RespondError(w, r, http.StatusConflict, Conflict(ErrConstraintViolation, "An instance with this label already exists"))
				return
			}
			log.Printf("[%s] CreateAgentConnectorInstance: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create connector instance"))
			return
		}

		RespondJSON(w, http.StatusCreated, toAgentConnectorInstanceResponse(*inst))
	}
}

func parsePathInstanceID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("instance_id")
	if id == "" {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "instance_id is required"))
		return "", false
	}
	if _, err := uuid.Parse(id); err != nil {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "instance_id must be a valid UUID"))
		return "", false
	}
	return id, true
}

// requireAgentConnectorInstance loads the instance or writes 404/500 and returns nil, false on failure.
func requireAgentConnectorInstance(w http.ResponseWriter, r *http.Request, deps *Deps, agentID int64, userID, connectorID, instanceID string) (*db.AgentConnectorInstance, bool) {
	inst, err := db.GetAgentConnectorInstance(r.Context(), deps.DB, agentID, userID, connectorID, instanceID)
	if err != nil {
		log.Printf("[%s] GetAgentConnectorInstance: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify connector instance"))
		return nil, false
	}
	if inst == nil {
		RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorInstanceNotFound, "Connector instance not found"))
		return nil, false
	}
	return inst, true
}

func handleGetAgentConnectorInstance(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID
		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		instanceID, ok := parsePathInstanceID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		inst, instOK := requireAgentConnectorInstance(w, r, deps, agentID, userID, connectorID, instanceID)
		if !instOK {
			return
		}
		RespondJSON(w, http.StatusOK, toAgentConnectorInstanceResponse(*inst))
	}
}

func handlePatchAgentConnectorInstance(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID
		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		instanceID, ok := parsePathInstanceID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		var req patchAgentConnectorInstanceRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if req.Label == nil && req.IsDefault == nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Provide label and/or is_default"))
			return
		}
		if req.IsDefault != nil && !*req.IsDefault {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "is_default may only be set to true"))
			return
		}

		ctx := r.Context()
		var inst *db.AgentConnectorInstance

		if req.Label != nil {
			var err error
			inst, err = db.RenameAgentConnectorInstance(ctx, deps.DB, agentID, userID, connectorID, instanceID, *req.Label)
			if err != nil {
				if errors.Is(err, db.ErrAgentConnectorInstanceLabelRequired) || errors.Is(err, db.ErrAgentConnectorInstanceLabelTooLong) {
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "label must be 1–256 characters"))
					return
				}
				var instErr *db.AgentConnectorInstanceError
				if errors.As(err, &instErr) && instErr.Code == db.AgentConnectorInstanceErrDuplicateLabel {
					RespondError(w, r, http.StatusConflict, Conflict(ErrConstraintViolation, "An instance with this label already exists"))
					return
				}
				log.Printf("[%s] RenameAgentConnectorInstance: %v", TraceID(ctx), err)
				CaptureError(ctx, err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to rename connector instance"))
				return
			}
			if inst == nil {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorInstanceNotFound, "Connector instance not found"))
				return
			}
		}

		if req.IsDefault != nil && *req.IsDefault {
			def, err := db.SetDefaultAgentConnectorInstance(ctx, deps.DB, agentID, userID, connectorID, instanceID)
			if err != nil {
				log.Printf("[%s] SetDefaultAgentConnectorInstance: %v", TraceID(ctx), err)
				CaptureError(ctx, err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update default connector instance"))
				return
			}
			if def == nil {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorInstanceNotFound, "Connector instance not found"))
				return
			}
			inst = def
		}

		if inst == nil {
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update connector instance"))
			return
		}
		RespondJSON(w, http.StatusOK, toAgentConnectorInstanceResponse(*inst))
	}
}

func handleDeleteAgentConnectorInstance(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID
		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		instanceID, ok := parsePathInstanceID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		err := db.DeleteAgentConnectorInstance(r.Context(), deps.DB, agentID, userID, connectorID, instanceID)
		if err != nil {
			var instErr *db.AgentConnectorInstanceError
			if errors.As(err, &instErr) {
				switch instErr.Code {
				case db.AgentConnectorInstanceErrNotFound:
					RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorInstanceNotFound, "Connector instance not found"))
					return
				case db.AgentConnectorInstanceErrCannotDeleteDefault:
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Cannot delete the default connector instance"))
					return
				}
			}
			log.Printf("[%s] DeleteAgentConnectorInstance: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete connector instance"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAssignAgentConnectorInstanceCredential(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID
		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		instanceID, ok := parsePathInstanceID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		if _, ok := requireAgentConnectorInstance(w, r, deps, agentID, userID, connectorID, instanceID); !ok {
			return
		}

		var req assignCredentialRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !validateAssignCredentialRequest(w, r, deps, userID, connectorID, &req) {
			return
		}

		bindingID, err := generatePrefixedID("acc_", 16)
		if err != nil {
			log.Printf("[%s] AssignInstanceCredential: generate ID: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to assign credential"))
			return
		}

		binding, err := db.UpsertAgentConnectorCredentialByInstance(r.Context(), deps.DB, db.UpsertAgentConnectorCredentialByInstanceParams{
			ID:                  bindingID,
			AgentID:             agentID,
			ConnectorID:         connectorID,
			ConnectorInstanceID: instanceID,
			ApproverID:          userID,
			CredentialID:        req.CredentialID,
			OAuthConnectionID:   req.OAuthConnectionID,
		})
		if err != nil {
			log.Printf("[%s] AssignInstanceCredential: upsert: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to assign credential"))
			return
		}

		RespondJSON(w, http.StatusOK, agentConnectorCredentialResponse{
			AgentID:           binding.AgentID,
			ConnectorID:       binding.ConnectorID,
			CredentialID:      binding.CredentialID,
			OAuthConnectionID: binding.OAuthConnectionID,
		})
	}
}

func handleRemoveAgentConnectorInstanceCredential(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID
		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		instanceID, ok := parsePathInstanceID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		if _, ok := requireAgentConnectorInstance(w, r, deps, agentID, userID, connectorID, instanceID); !ok {
			return
		}

		deleted, err := db.DeleteAgentConnectorCredentialByInstance(r.Context(), deps.DB, agentID, userID, connectorID, instanceID)
		if err != nil {
			log.Printf("[%s] RemoveInstanceCredential: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to remove credential binding"))
			return
		}
		if !deleted {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrCredentialNotFound, "No credential binding found"))
			return
		}
		RespondJSON(w, http.StatusOK, map[string]string{"status": "removed"})
	}
}

func handleGetAgentConnectorInstanceCredential(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID
		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		instanceID, ok := parsePathInstanceID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		if _, ok := requireAgentConnectorInstance(w, r, deps, agentID, userID, connectorID, instanceID); !ok {
			return
		}

		binding, err := db.GetAgentConnectorCredentialByInstance(r.Context(), deps.DB, agentID, connectorID, instanceID)
		if err != nil {
			log.Printf("[%s] GetInstanceCredential: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get credential binding"))
			return
		}
		if binding == nil {
			RespondJSON(w, http.StatusOK, agentConnectorCredentialResponse{
				AgentID:     agentID,
				ConnectorID: connectorID,
			})
			return
		}
		RespondJSON(w, http.StatusOK, agentConnectorCredentialResponse{
			AgentID:           binding.AgentID,
			ConnectorID:       binding.ConnectorID,
			CredentialID:      binding.CredentialID,
			OAuthConnectionID: binding.OAuthConnectionID,
		})
	}
}

func handleListAgentConnectorInstanceChannels(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID
		userEmail := UserEmail(r.Context())

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		instanceID, ok := parsePathInstanceID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		if _, ok := requireAgentConnectorInstance(w, r, deps, agentID, userID, connectorID, instanceID); !ok {
			return
		}

		if deps.Connectors == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Connectors are not available"))
			return
		}

		conn, ok := deps.Connectors.Get(connectorID)
		if !ok {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "Connector not found"))
			return
		}

		lister, ok := conn.(connectors.ChannelLister)
		if !ok {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "This connector does not support channel listing"))
			return
		}

		listAction := lister.ChannelListCredentialActionType()
		connErrCtx := ConnectorContext{
			ConnectorID: connectorID,
			ActionType:  listAction,
			AgentID:     agentID,
		}
		reqCreds, err := db.GetRequiredCredentialsByActionType(r.Context(), deps.DB, listAction)
		if err != nil {
			log.Printf("[%s] ListAgentConnectorInstanceChannels required creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to look up connector credentials"))
			return
		}

		creds, err := resolveCredentialsForConnectorInstance(r.Context(), deps, agentID, userID, listAction, connectorID, instanceID, reqCreds)
		if err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListAgentConnectorInstanceChannels resolve creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to resolve credentials"))
			return
		}

		if err := conn.ValidateCredentials(r.Context(), creds); err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListAgentConnectorInstanceChannels validate creds: %v", TraceID(r.Context()), err)
			CaptureConnectorError(r.Context(), err, connErrCtx)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Credential validation failed"))
			return
		}

		items, err := lister.ListChannels(r.Context(), creds, userEmail)
		if err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListChannels (instance): %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list channels"))
			return
		}

		RespondJSON(w, http.StatusOK, channelListResponse{Data: items})
	}
}

func handleListAgentConnectorInstanceCalendars(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		instanceID, ok := parsePathInstanceID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		if _, ok := requireAgentConnectorInstance(w, r, deps, agentID, userID, connectorID, instanceID); !ok {
			return
		}

		if deps.Connectors == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Connectors are not available"))
			return
		}

		conn, ok := deps.Connectors.Get(connectorID)
		if !ok {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "Connector not found"))
			return
		}

		lister, ok := conn.(connectors.CalendarLister)
		if !ok {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "This connector does not support calendar listing"))
			return
		}

		listAction := lister.CalendarListCredentialActionType()
		connErrCtx := ConnectorContext{
			ConnectorID: connectorID,
			ActionType:  listAction,
			AgentID:     agentID,
		}
		reqCreds, err := db.GetRequiredCredentialsByActionType(r.Context(), deps.DB, listAction)
		if err != nil {
			log.Printf("[%s] ListAgentConnectorInstanceCalendars required creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to look up connector credentials"))
			return
		}

		creds, err := resolveCredentialsForConnectorInstance(r.Context(), deps, agentID, userID, listAction, connectorID, instanceID, reqCreds)
		if err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListAgentConnectorInstanceCalendars resolve creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to resolve credentials"))
			return
		}

		if err := conn.ValidateCredentials(r.Context(), creds); err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListAgentConnectorInstanceCalendars validate creds: %v", TraceID(r.Context()), err)
			CaptureConnectorError(r.Context(), err, connErrCtx)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Credential validation failed"))
			return
		}

		items, err := lister.ListCalendars(r.Context(), creds)
		if err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListCalendars (instance): %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list calendars"))
			return
		}

		RespondJSON(w, http.StatusOK, calendarListResponse{Data: items})
	}
}

func handleListAgentConnectorInstanceUsers(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}
		instanceID, ok := parsePathInstanceID(w, r)
		if !ok {
			return
		}
		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		if _, ok := requireAgentConnectorInstance(w, r, deps, agentID, userID, connectorID, instanceID); !ok {
			return
		}

		if deps.Connectors == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Connectors are not available"))
			return
		}

		conn, ok := deps.Connectors.Get(connectorID)
		if !ok {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "Connector not found"))
			return
		}

		lister, ok := conn.(connectors.UserLister)
		if !ok {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "This connector does not support user listing"))
			return
		}

		listAction := lister.UserListCredentialActionType()
		connErrCtx := ConnectorContext{
			ConnectorID: connectorID,
			ActionType:  listAction,
			AgentID:     agentID,
		}
		reqCreds, err := db.GetRequiredCredentialsByActionType(r.Context(), deps.DB, listAction)
		if err != nil {
			log.Printf("[%s] ListAgentConnectorInstanceUsers required creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to look up connector credentials"))
			return
		}

		creds, err := resolveCredentialsForConnectorInstance(r.Context(), deps, agentID, userID, listAction, connectorID, instanceID, reqCreds)
		if err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListAgentConnectorInstanceUsers resolve creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to resolve credentials"))
			return
		}

		if err := conn.ValidateCredentials(r.Context(), creds); err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListAgentConnectorInstanceUsers validate creds: %v", TraceID(r.Context()), err)
			CaptureConnectorError(r.Context(), err, connErrCtx)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Credential validation failed"))
			return
		}

		items, err := lister.ListUsers(r.Context(), creds)
		if err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListUsers (instance): %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list users"))
			return
		}

		RespondJSON(w, http.StatusOK, userListResponse{Data: items})
	}
}
