package api

import (
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type userCalendarResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsPrimary   bool   `json:"is_primary"`
}

type agentConnectorCalendarsResponse struct {
	Data []userCalendarResponse `json:"data"`
}

func init() {
	RegisterRouteGroup(RegisterAgentConnectorCalendarRoutes)
}

// RegisterAgentConnectorCalendarRoutes adds calendar listing for agent connector UI.
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

		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		connectorID := r.PathValue("connector_id")

		// External provider calls below are bounded by the global per-IP rate limit
		// middleware on the API router; no separate calendar-specific limiter.

		if deps.Vault == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Credential vault is not configured"))
			return
		}
		if deps.Connectors == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Connectors are not available"))
			return
		}

		creds, err := resolveAgentConnectorBoundCredentials(r.Context(), deps, agentID, userID, connectorID)
		if err != nil {
			if handleConnectorError(w, r, err, ConnectorContext{AgentID: agentID}) {
				return
			}
			log.Printf("[%s] ListAgentConnectorCalendars resolve creds: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to resolve credentials"))
			return
		}

		conn, ok := deps.Connectors.Get(connectorID)
		if !ok {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "Connector not found"))
			return
		}

		lister, ok := conn.(connectors.CalendarLister)
		if !ok {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "This connector does not support calendar listing"))
			return
		}

		var cals []connectors.UserCalendar
		cals, err = lister.ListUserCalendars(r.Context(), creds)

		if err != nil {
			if handleConnectorError(w, r, err, ConnectorContext{AgentID: agentID}) {
				return
			}
			log.Printf("[%s] ListUserCalendars: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list calendars"))
			return
		}

		data := make([]userCalendarResponse, len(cals))
		for i, cal := range cals {
			data[i] = userCalendarResponse{
				ID:          cal.ID,
				Name:        cal.Name,
				Description: cal.Description,
				IsPrimary:   cal.IsPrimary,
			}
		}

		RespondJSON(w, http.StatusOK, agentConnectorCalendarsResponse{Data: data})
	}
}
