package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// --- Response types ---

type agentConnectorSummaryResponse struct {
	connectorSummaryResponse
	EnabledAt time.Time `json:"enabled_at"`
}

type agentConnectorListResponse struct {
	Data []agentConnectorSummaryResponse `json:"data"`
}

type agentConnectorResponse struct {
	AgentID     int64     `json:"agent_id"`
	ConnectorID string    `json:"connector_id"`
	EnabledAt   time.Time `json:"enabled_at"`
}

type disableAgentConnectorResponse struct {
	AgentID                  int64     `json:"agent_id"`
	ConnectorID              string    `json:"connector_id"`
	DisabledAt               time.Time `json:"disabled_at"`
	RevokedStandingApprovals int64     `json:"revoked_standing_approvals"`
}

func init() {
	RegisterRouteGroup(RegisterAgentConnectorRoutes)
}

// RegisterAgentConnectorRoutes adds agent connector endpoints to the mux.
func RegisterAgentConnectorRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /agents/{agent_id}/connectors", requireProfile(handleListAgentConnectors(deps)))
	mux.Handle("PUT /agents/{agent_id}/connectors/{connector_id}", requireProfile(handleEnableAgentConnector(deps)))
	mux.Handle("DELETE /agents/{agent_id}/connectors/{connector_id}", requireProfile(handleDisableAgentConnector(deps)))
	mux.Handle("PUT /agents/{agent_id}/connectors/{connector_id}/credential", requireProfile(handleAssignAgentConnectorCredential(deps)))
	mux.Handle("DELETE /agents/{agent_id}/connectors/{connector_id}/credential", requireProfile(handleRemoveAgentConnectorCredential(deps)))
	mux.Handle("GET /agents/{agent_id}/connectors/{connector_id}/credential", requireProfile(handleGetAgentConnectorCredential(deps)))
}

func handleListAgentConnectors(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		connectors, err := db.ListAgentConnectors(r.Context(), deps.DB, agentID, userID)
		if err != nil {
			log.Printf("[%s] ListAgentConnectors: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list agent connectors"))
			return
		}

		data := make([]agentConnectorSummaryResponse, len(connectors))
		for i, ac := range connectors {
			data[i] = agentConnectorSummaryResponse{
				connectorSummaryResponse: toConnectorSummaryResponse(ac.ConnectorSummary),
				EnabledAt:                ac.EnabledAt,
			}
		}

		RespondJSON(w, http.StatusOK, agentConnectorListResponse{Data: data})
	}
}

func handleEnableAgentConnector(deps *Deps) http.HandlerFunc {
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

		row, err := db.EnableAgentConnector(r.Context(), deps.DB, agentID, userID, connectorID)
		if err != nil {
			var acErr *db.AgentConnectorError
			if errors.As(err, &acErr) {
				switch acErr.Code {
				case db.AgentConnectorErrAgentNotFound:
					RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
					return
				case db.AgentConnectorErrConnectorNotFound:
					RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "Connector not found"))
					return
				}
			}
			log.Printf("[%s] EnableAgentConnector: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to enable connector"))
			return
		}

		RespondJSON(w, http.StatusOK, agentConnectorResponse{
			AgentID:     row.AgentID,
			ConnectorID: row.ConnectorID,
			EnabledAt:   row.EnabledAt,
		})
	}
}

func handleDisableAgentConnector(deps *Deps) http.HandlerFunc {
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

		// Delete is scoped by approver_id: returns nil for both "agent not found"
		// and "connector not enabled" to avoid leaking agent existence.
		result, err := db.DisableAgentConnector(r.Context(), deps.DB, agentID, userID, connectorID)
		if err != nil {
			log.Printf("[%s] DisableAgentConnector: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to disable connector"))
			return
		}
		if result == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "Connector not enabled for this agent"))
			return
		}

		RespondJSON(w, http.StatusOK, disableAgentConnectorResponse{
			AgentID:                  result.AgentID,
			ConnectorID:              result.ConnectorID,
			DisabledAt:               result.DisabledAt,
			RevokedStandingApprovals: result.RevokedStandingApprovals,
		})
	}
}

// --- Credential binding ---

type assignCredentialRequest struct {
	CredentialID      *string `json:"credential_id,omitempty"`
	OAuthConnectionID *string `json:"oauth_connection_id,omitempty"`
}

type agentConnectorCredentialResponse struct {
	AgentID           int64   `json:"agent_id"`
	ConnectorID       string  `json:"connector_id"`
	CredentialID      *string `json:"credential_id,omitempty"`
	OAuthConnectionID *string `json:"oauth_connection_id,omitempty"`
}

func handleAssignAgentConnectorCredential(deps *Deps) http.HandlerFunc {
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

		var req assignCredentialRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		// Exactly one of credential_id or oauth_connection_id must be provided.
		if (req.CredentialID == nil) == (req.OAuthConnectionID == nil) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "provide exactly one of credential_id or oauth_connection_id"))
			return
		}

		// Validate ownership and service match for static credentials.
		if req.CredentialID != nil {
			cred, err := db.GetCredentialByID(r.Context(), deps.DB, *req.CredentialID)
			if err != nil {
				log.Printf("[%s] AssignCredential: credential check: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify credential"))
				return
			}
			if cred == nil || cred.UserID != userID {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Credential not found"))
				return
			}
			if cred.Service != connectorID {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
					fmt.Sprintf("credential service %q does not match connector %q", cred.Service, connectorID)))
				return
			}
		}
		if req.OAuthConnectionID != nil {
			conn, err := db.GetOAuthConnectionByID(r.Context(), deps.DB, *req.OAuthConnectionID)
			if err != nil {
				log.Printf("[%s] AssignCredential: oauth check: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify OAuth connection"))
				return
			}
			if conn == nil || conn.UserID != userID {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "OAuth connection not found"))
				return
			}
			if conn.Provider != connectorID {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
					fmt.Sprintf("OAuth connection provider %q does not match connector %q", conn.Provider, connectorID)))
				return
			}
		}

		bindingID, err := generatePrefixedID("acc_", 16)
		if err != nil {
			log.Printf("[%s] AssignCredential: generate ID: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to assign credential"))
			return
		}

		binding, err := db.UpsertAgentConnectorCredential(r.Context(), deps.DB, db.UpsertAgentConnectorCredentialParams{
			ID:                bindingID,
			AgentID:           agentID,
			ConnectorID:       connectorID,
			ApproverID:        userID,
			CredentialID:      req.CredentialID,
			OAuthConnectionID: req.OAuthConnectionID,
		})
		if err != nil {
			log.Printf("[%s] AssignCredential: upsert: %v", TraceID(r.Context()), err)
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

func handleRemoveAgentConnectorCredential(deps *Deps) http.HandlerFunc {
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

		deleted, err := db.DeleteAgentConnectorCredential(r.Context(), deps.DB, agentID, userID, connectorID)
		if err != nil {
			log.Printf("[%s] RemoveCredential: %v", TraceID(r.Context()), err)
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

func handleGetAgentConnectorCredential(deps *Deps) http.HandlerFunc {
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

		binding, err := db.GetAgentConnectorCredential(r.Context(), deps.DB, agentID, connectorID)
		if err != nil {
			log.Printf("[%s] GetCredentialBinding: %v", TraceID(r.Context()), err)
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
