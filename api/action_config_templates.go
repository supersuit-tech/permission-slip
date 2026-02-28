package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// --- Response types ---

type actionConfigTemplateResponse struct {
	ID          string  `json:"id"`
	ConnectorID string  `json:"connector_id"`
	ActionType  string  `json:"action_type"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Parameters  any     `json:"parameters"`
	CreatedAt   time.Time `json:"created_at"`
}

type actionConfigTemplateListResponse struct {
	Data []actionConfigTemplateResponse `json:"data"`
}

// --- Routes ---

// RegisterActionConfigTemplateRoutes adds action configuration template endpoints to the mux.
func RegisterActionConfigTemplateRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /action-config-templates", requireProfile(handleListActionConfigTemplates(deps)))
}

// --- Handlers ---

func handleListActionConfigTemplates(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		connectorID := r.URL.Query().Get("connector_id")
		if connectorID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id query parameter is required"))
			return
		}
		if len(connectorID) > 128 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id exceeds maximum length"))
			return
		}

		templates, err := db.ListTemplatesByConnector(r.Context(), deps.DB, connectorID)
		if err != nil {
			log.Printf("[%s] ListActionConfigTemplates: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list action configuration templates"))
			return
		}

		data := make([]actionConfigTemplateResponse, len(templates))
		for i, t := range templates {
			data[i] = toActionConfigTemplateResponse(t)
		}

		RespondJSON(w, http.StatusOK, actionConfigTemplateListResponse{Data: data})
	}
}

// --- Helpers ---

func toActionConfigTemplateResponse(t db.ActionConfigTemplate) actionConfigTemplateResponse {
	resp := actionConfigTemplateResponse{
		ID:          t.ID,
		ConnectorID: t.ConnectorID,
		ActionType:  t.ActionType,
		Name:        t.Name,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
	}
	if len(t.Parameters) > 0 {
		var params any
		if err := json.Unmarshal(t.Parameters, &params); err != nil {
			log.Printf("warning: failed to unmarshal template %s parameters: %v", t.ID, err)
		} else {
			resp.Parameters = params
		}
	}
	if resp.Parameters == nil {
		resp.Parameters = map[string]any{}
	}
	return resp
}
