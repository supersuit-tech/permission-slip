package api

import (
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	googleconnector "github.com/supersuit-tech/permission-slip-web/connectors/google"
	"github.com/supersuit-tech/permission-slip-web/connectors/microsoft"
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
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}

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
			if handleConnectorError(w, r, err, ConnectorContext{ConnectorID: connectorID}) {
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

		var cals []connectors.UserCalendar
		switch c := conn.(type) {
		case *googleconnector.GoogleConnector:
			cals, err = c.ListUserCalendars(r.Context(), creds)
		case *microsoft.MicrosoftConnector:
			cals, err = c.ListUserCalendars(r.Context(), creds)
		default:
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "This connector does not support calendar listing"))
			return
		}

		if err != nil {
			if handleConnectorError(w, r, err, ConnectorContext{ConnectorID: connectorID}) {
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
