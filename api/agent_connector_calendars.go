package api

import (
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
)

type calendarListResponse struct {
	Data []connectors.CalendarListItem `json:"data"`
}

func init() {
	RegisterRouteGroup(RegisterAgentConnectorCalendarRoutes)
}

// RegisterAgentConnectorCalendarRoutes registers session-authenticated endpoints
// that proxy calendar metadata for action configuration UIs.
func RegisterAgentConnectorCalendarRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /agents/{agent_id}/connectors/{connector_id}/calendars", requireProfile(handleListAgentConnectorCalendars(deps)))
}

func handleListAgentConnectorCalendars(deps *Deps) http.HandlerFunc {
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
			log.Printf("[%s] ListAgentConnectorCalendars required creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to look up connector credentials"))
			return
		}

		defaultInst, err := db.GetDefaultAgentConnectorInstance(r.Context(), deps.DB, agentID, userID, connectorID)
		if err != nil {
			log.Printf("[%s] ListAgentConnectorCalendars default instance: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to resolve connector instance"))
			return
		}
		if defaultInst == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "Connector not enabled for this agent"))
			return
		}

		creds, err := resolveCredentialsForConnectorInstance(r.Context(), deps, agentID, userID, listAction, connectorID, defaultInst.ConnectorInstanceID, reqCreds)
		if err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListAgentConnectorCalendars resolve creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to resolve credentials"))
			return
		}

		if err := conn.ValidateCredentials(r.Context(), creds); err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListAgentConnectorCalendars validate creds: %v", TraceID(r.Context()), err)
			CaptureConnectorError(r.Context(), err, connErrCtx)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Credential validation failed"))
			return
		}

		items, err := lister.ListCalendars(r.Context(), creds)
		if err != nil {
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] ListCalendars: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list calendars"))
			return
		}

		RespondJSON(w, http.StatusOK, calendarListResponse{Data: items})
	}
}
