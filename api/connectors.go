package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// --- Response types ---

type connectorSummaryResponse struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         *string  `json:"description,omitempty"`
	Actions             []string `json:"actions"`
	RequiredCredentials []string `json:"required_credentials"`
}

type connectorListResponse struct {
	Data []connectorSummaryResponse `json:"data"`
}

type connectorActionResponse struct {
	ActionType       string `json:"action_type"`
	Name             string `json:"name"`
	Description      *string `json:"description,omitempty"`
	RiskLevel        *string `json:"risk_level,omitempty"`
	ParametersSchema any    `json:"parameters_schema,omitempty"`
}

type requiredCredentialResponse struct {
	Service         string   `json:"service"`
	AuthType        string   `json:"auth_type"`
	InstructionsURL *string  `json:"instructions_url,omitempty"`
	OAuthProvider   *string  `json:"oauth_provider,omitempty"`
	OAuthScopes     []string `json:"oauth_scopes,omitempty"`
}

type connectorDetailResponse struct {
	ID                  string                       `json:"id"`
	Name                string                       `json:"name"`
	Description         *string                      `json:"description,omitempty"`
	Actions             []connectorActionResponse    `json:"actions"`
	RequiredCredentials []requiredCredentialResponse  `json:"required_credentials"`
}

// RegisterConnectorRoutes adds connector-related endpoints to the mux.
// These are public endpoints (no auth required).
func RegisterConnectorRoutes(mux *http.ServeMux, deps *Deps) {
	mux.Handle("GET /connectors", handleListConnectors(deps))
	mux.Handle("GET /connectors/{connector_id}", handleGetConnector(deps))
}

func handleListConnectors(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		connectors, err := db.ListConnectors(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] ListConnectors: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list connectors"))
			return
		}

		data := make([]connectorSummaryResponse, len(connectors))
		for i, c := range connectors {
			data[i] = toConnectorSummaryResponse(c)
		}

		RespondJSON(w, http.StatusOK, connectorListResponse{Data: data})
	}
}

func handleGetConnector(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		connectorID := r.PathValue("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id is required"))
			return
		}

		connector, err := db.GetConnectorByID(r.Context(), deps.DB, connectorID)
		if err != nil {
			log.Printf("[%s] GetConnector: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get connector"))
			return
		}
		if connector == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConnectorNotFound, "Connector not found"))
			return
		}

		RespondJSON(w, http.StatusOK, toConnectorDetailResponse(r.Context(), *connector))
	}
}

func toConnectorSummaryResponse(c db.ConnectorSummary) connectorSummaryResponse {
	return connectorSummaryResponse{
		ID:                  c.ID,
		Name:                c.Name,
		Description:         c.Description,
		Actions:             c.Actions,
		RequiredCredentials: c.RequiredCredentials,
	}
}

func toConnectorDetailResponse(ctx context.Context, c db.ConnectorDetail) connectorDetailResponse {
	actions := make([]connectorActionResponse, len(c.Actions))
	for i, a := range c.Actions {
		resp := connectorActionResponse{
			ActionType:  a.ActionType,
			Name:        a.Name,
			Description: a.Description,
			RiskLevel:   a.RiskLevel,
		}
		if len(a.ParametersSchema) > 0 {
			var schema any
			if err := json.Unmarshal(a.ParametersSchema, &schema); err != nil {
				log.Printf("[%s] warning: failed to unmarshal connector %s action %s parameters_schema: %v", TraceID(ctx), c.ID, a.ActionType, err)
			} else {
				resp.ParametersSchema = schema
			}
		}
		actions[i] = resp
	}

	creds := make([]requiredCredentialResponse, len(c.RequiredCredentials))
	for i, rc := range c.RequiredCredentials {
		creds[i] = requiredCredentialResponse{
			Service:         rc.Service,
			AuthType:        rc.AuthType,
			InstructionsURL: rc.InstructionsURL,
			OAuthProvider:   rc.OAuthProvider,
			OAuthScopes:     rc.OAuthScopes,
		}
	}

	return connectorDetailResponse{
		ID:                  c.ID,
		Name:                c.Name,
		Description:         c.Description,
		Actions:             actions,
		RequiredCredentials: creds,
	}
}
