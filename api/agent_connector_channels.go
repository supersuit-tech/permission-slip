package api

import (
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/db"
)

type channelListResponse struct {
	Data []connectors.ChannelListItem `json:"data"`
}

func init() {
	RegisterRouteGroup(RegisterAgentConnectorChannelRoutes)
}

// RegisterAgentConnectorChannelRoutes registers session-authenticated endpoints
// that proxy channel metadata for action configuration UIs (e.g. Slack).
func RegisterAgentConnectorChannelRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /agents/{agent_id}/connectors/{connector_id}/channels", requireProfile(handleListAgentConnectorChannels(deps)))
}

func handleListAgentConnectorChannels(deps *Deps) http.HandlerFunc {
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

		if !requireAgentOwnership(w, r, deps, agentID, userID) {
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
			log.Printf("[%s] ListAgentConnectorChannels required creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to look up connector credentials"))
			return
		}

		creds, err := resolveCredentialsWithFallback(r.Context(), deps, agentID, userID, listAction, connectorID, reqCreds)
		if err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListAgentConnectorChannels resolve creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to resolve credentials"))
			return
		}

		if err := conn.ValidateCredentials(r.Context(), creds); err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListAgentConnectorChannels validate creds: %v", TraceID(r.Context()), err)
			CaptureConnectorError(r.Context(), err, connErrCtx)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Credential validation failed"))
			return
		}

		items, err := lister.ListChannels(r.Context(), creds, userEmail)
		if err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListChannels: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list channels"))
			return
		}

		RespondJSON(w, http.StatusOK, channelListResponse{Data: items})
	}
}
