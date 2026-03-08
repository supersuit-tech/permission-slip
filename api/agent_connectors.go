package api

import (
	"errors"
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
