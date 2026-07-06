package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

type standingApprovalTemplateResponse struct {
	ID           string    `json:"id"`
	ConnectorID  string    `json:"connector_id"`
	ActionType   string    `json:"action_type"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	Constraints  any       `json:"constraints"`
	DurationDays *int      `json:"duration_days,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type standingApprovalTemplateListResponse struct {
	Data []standingApprovalTemplateResponse `json:"data"`
}

func init() {
	RegisterRouteGroup(RegisterStandingApprovalTemplateRoutes)
}

func RegisterStandingApprovalTemplateRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /standing-approval-templates", requireProfile(handleListStandingApprovalTemplates(deps)))
	mux.Handle("POST /standing-approval-templates/bulk-apply", requireProfile(handleBulkApplyStandingApprovalTemplates(deps)))
	mux.Handle("POST /standing-approval-templates/{id}/apply", requireProfile(handleApplyStandingApprovalTemplate(deps)))
}

func handleListStandingApprovalTemplates(deps *Deps) http.HandlerFunc {
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

		templates, err := db.ListStandingApprovalTemplatesByConnector(r.Context(), deps.DB, connectorID)
		if err != nil {
			log.Printf("[%s] ListStandingApprovalTemplates: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list standing approval templates"))
			return
		}

		data := make([]standingApprovalTemplateResponse, len(templates))
		for i, t := range templates {
			data[i] = toStandingApprovalTemplateResponse(t)
		}

		RespondJSON(w, http.StatusOK, standingApprovalTemplateListResponse{Data: data})
	}
}

func toStandingApprovalTemplateResponse(t db.StandingApprovalTemplate) standingApprovalTemplateResponse {
	resp := standingApprovalTemplateResponse{
		ID:           t.ID,
		ConnectorID:  t.ConnectorID,
		ActionType:   t.ActionType,
		Name:         t.Name,
		Description:  t.Description,
		DurationDays: t.DurationDays,
		CreatedAt:    t.CreatedAt,
	}
	if len(t.Constraints) > 0 {
		var constraints any
		if err := json.Unmarshal(t.Constraints, &constraints); err != nil {
			log.Printf("warning: failed to unmarshal template %s constraints: %v", t.ID, err)
		} else {
			resp.Constraints = constraints
		}
	}
	if resp.Constraints == nil {
		resp.Constraints = map[string]any{}
	}
	return resp
}
