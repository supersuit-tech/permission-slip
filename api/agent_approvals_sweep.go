package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

type agentApprovalSweepItemResponse struct {
	ApprovalID   string     `json:"approval_id"`
	Status       string     `json:"status"`
	ExpiresAt    time.Time  `json:"expires_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	DenialReason *string    `json:"denial_reason,omitempty"`
}

type agentApprovalSweepResponse struct {
	Pending  []agentApprovalSweepItemResponse `json:"pending"`
	Resolved []agentApprovalSweepItemResponse `json:"resolved"`
}

func init() {
	RegisterRouteGroup(RegisterAgentApprovalSweepRoutes)
}

func RegisterAgentApprovalSweepRoutes(mux *http.ServeMux, deps *Deps) {
	requireAgent := RequireAgentSignature(deps)
	mux.Handle("GET /agent/approvals", requireAgent(handleAgentApprovalSweep(deps)))
}

func handleAgentApprovalSweep(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())

		resolvedSince := time.Now().UTC().Add(-24 * time.Hour)
		if raw := strings.TrimSpace(r.URL.Query().Get("resolved_since")); raw != "" {
			parsed, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "resolved_since must be RFC3339"))
				return
			}
			resolvedSince = parsed.UTC()
		}

		items, err := db.ListAgentApprovalsForSweep(r.Context(), deps.DB, agent.AgentID, resolvedSince)
		if err != nil {
			log.Printf("[%s] AgentApprovalSweep: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list approvals"))
			return
		}

		resp := agentApprovalSweepResponse{
			Pending:  []agentApprovalSweepItemResponse{},
			Resolved: []agentApprovalSweepItemResponse{},
		}
		for _, item := range items {
			row := agentApprovalSweepItemResponse{
				ApprovalID:   item.ApprovalID,
				Status:       item.Status,
				ExpiresAt:    item.ExpiresAt,
				ResolvedAt:   item.ResolvedAt,
				DenialReason: item.DenialReason,
			}
			if item.Status == "pending" {
				resp.Pending = append(resp.Pending, row)
			} else {
				resp.Resolved = append(resp.Resolved, row)
			}
		}
		RespondJSON(w, http.StatusOK, resp)
	}
}

// agentHasWebhookConfigured reports whether the agent has a push wake URL set.
func agentHasWebhookConfigured(ctx context.Context, deps *Deps, agentID int64) bool {
	ok, err := db.AgentHasWebhook(ctx, deps.DB, agentID)
	return err == nil && ok
}
