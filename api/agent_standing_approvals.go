package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// ── Response types ───────────────────────────────────────────────────────────

type agentStandingApprovalListResponse struct {
	StandingApprovals []standingApprovalResponse `json:"standing_approvals"`
	HasMore           bool                       `json:"has_more"`
	NextCursor        *string                    `json:"next_cursor,omitempty"`
}

// ── Route registration ──────────────────────────────────────────────────────

func init() {
	RegisterRouteGroup(RegisterAgentStandingApprovalRoutes)
}

// RegisterAgentStandingApprovalRoutes adds agent-authenticated standing approval endpoints.
func RegisterAgentStandingApprovalRoutes(mux *http.ServeMux, deps *Deps) {
	mux.HandleFunc("GET /agents/{agent_id}/standing-approvals", handleAgentListStandingApprovals(deps))
}

// ── GET /agents/{agent_id}/standing-approvals ────────────────────────────────

func handleAgentListStandingApprovals(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		// Authenticate via signature and validate agent_id match.
		agent, _, ok := requireAgentSignature(w, r, deps, agentID, nil)
		if !ok {
			return
		}

		// Only registered agents can list standing approvals.
		if agent.Status != "registered" {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
			return
		}

		limit, ok := parsePaginationLimit(w, r)
		if !ok {
			return
		}

		// Parse cursor.
		var cursor *db.StandingApprovalCursor
		if v := r.URL.Query().Get("after"); v != "" {
			c, err := parseStandingApprovalCursor(v)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid pagination cursor"))
				return
			}
			cursor = c
		}

		page, err := db.ListStandingApprovalsByAgent(r.Context(), deps.DB, agentID, limit, cursor)
		if err != nil {
			log.Printf("[%s] AgentListStandingApprovals: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list standing approvals"))
			return
		}

		data := make([]standingApprovalResponse, len(page.Approvals))
		for i, sa := range page.Approvals {
			data[i] = toStandingApprovalResponse(sa)
		}

		resp := agentStandingApprovalListResponse{
			StandingApprovals: data,
			HasMore:           page.HasMore,
		}
		if page.HasMore && len(page.Approvals) > 0 {
			last := page.Approvals[len(page.Approvals)-1]
			c := encodeStandingApprovalCursor(last)
			resp.NextCursor = &c
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// ── Cursor helpers ──────────────────────────────────────────────────────────

func encodeStandingApprovalCursor(sa db.StandingApproval) string {
	return sa.CreatedAt.UTC().Format(time.RFC3339Nano) + "," + sa.StandingApprovalID
}

func parseStandingApprovalCursor(raw string) (*db.StandingApprovalCursor, error) {
	comma := strings.IndexByte(raw, ',')
	if comma < 0 {
		return nil, fmt.Errorf("missing comma separator")
	}
	t, err := time.Parse(time.RFC3339Nano, raw[:comma])
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}
	saID := raw[comma+1:]
	if saID == "" {
		return nil, fmt.Errorf("empty standing_approval_id")
	}
	return &db.StandingApprovalCursor{CreatedAt: t, StandingApprovalID: saID}, nil
}
