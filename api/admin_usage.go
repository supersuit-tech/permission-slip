package api

import (
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// ── Response types ──────────────────────────────────────────────────────────

type adminUsageResponse struct {
	UserID      string             `json:"user_id"`
	PeriodStart time.Time          `json:"period_start"`
	PeriodEnd   time.Time          `json:"period_end"`
	Requests    int                `json:"request_count"`
	SMS         int                `json:"sms_count"`
	Breakdown   *usageBreakdownDTO `json:"breakdown,omitempty"`
}

type connectorUsageResponse struct {
	UserID      string                `json:"user_id"`
	PeriodStart time.Time             `json:"period_start"`
	PeriodEnd   time.Time             `json:"period_end"`
	Connectors  []connectorUsageEntry `json:"connectors"`
}

type connectorUsageEntry struct {
	ConnectorID  string `json:"connector_id"`
	Name         string `json:"name,omitempty"`
	RequestCount int    `json:"request_count"`
}

type agentUsageResponse struct {
	UserID      string            `json:"user_id"`
	PeriodStart time.Time         `json:"period_start"`
	PeriodEnd   time.Time         `json:"period_end"`
	Agents      []agentUsageEntry `json:"agents"`
}

type agentUsageEntry struct {
	AgentID      string `json:"agent_id"`
	Name         string `json:"name,omitempty"`
	RequestCount int    `json:"request_count"`
}

// ── Route registration ──────────────────────────────────────────────────────

// RegisterAdminUsageRoutes adds usage analytics endpoints to the mux.
// All endpoints require session auth via RequireProfile and are scoped
// to the authenticated user's own data.
//
// NOTE: Cross-user endpoints (top-users leaderboard, platform-wide
// connector aggregation) are intentionally omitted. They require an
// admin role system that does not yet exist. Adding them behind
// RequireProfile alone would create a cross-tenant data leak.
func RegisterAdminUsageRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /admin/usage", requireProfile(handleAdminGetUsage(deps)))
	mux.Handle("GET /admin/usage/by-connector", requireProfile(handleAdminUsageByConnector(deps)))
	mux.Handle("GET /admin/usage/by-agent", requireProfile(handleAdminUsageByAgent(deps)))
}

// ── GET /admin/usage ────────────────────────────────────────────────────────

// handleAdminGetUsage returns the authenticated user's usage metrics for a
// billing period. Includes request/SMS totals and an optional breakdown by
// agent, connector, and action type.
//
// Query params:
//   - period (optional): billing period in YYYY-MM, YYYY-MM-DD, or RFC3339 format; defaults to current
func handleAdminGetUsage(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		userID := profile.ID

		periodStart, ok := resolvePeriodParam(w, r)
		if !ok {
			return
		}

		usage, err := db.GetUsageByPeriod(r.Context(), deps.DB, userID, periodStart)
		if err != nil {
			log.Printf("[%s] AdminGetUsage: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch usage data"))
			return
		}

		_, periodEnd := db.BillingPeriodBounds(periodStart)

		resp := adminUsageResponse{
			UserID:      userID,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
		}
		if usage != nil {
			resp.Requests = usage.RequestCount
			resp.SMS = usage.SMSCount

			b := usage.ParseBreakdown()
			if len(b.ByAgent) > 0 || len(b.ByConnector) > 0 || len(b.ByActionType) > 0 {
				resp.Breakdown = &usageBreakdownDTO{
					ByAgent:      b.ByAgent,
					ByConnector:  b.ByConnector,
					ByActionType: b.ByActionType,
				}
			}
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// ── GET /admin/usage/by-connector ──────────────────────────────────────────

// handleAdminUsageByConnector returns the authenticated user's per-connector
// request counts for a billing period, ordered by count descending.
//
// Query params:
//   - period (optional): billing period; defaults to current billing period
func handleAdminUsageByConnector(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		userID := profile.ID

		periodStart, ok := resolvePeriodParam(w, r)
		if !ok {
			return
		}

		connectors, err := db.GetUsageByConnectorForUser(r.Context(), deps.DB, userID, periodStart)
		if err != nil {
			log.Printf("[%s] AdminUsageByConnector: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch connector usage"))
			return
		}

		_, periodEnd := db.BillingPeriodBounds(periodStart)

		entries := make([]connectorUsageEntry, len(connectors))
		for i, c := range connectors {
			entries[i] = connectorUsageEntry{
				ConnectorID:  c.ConnectorID,
				Name:         c.Name,
				RequestCount: c.RequestCount,
			}
		}

		RespondJSON(w, http.StatusOK, connectorUsageResponse{
			UserID:      userID,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Connectors:  entries,
		})
	}
}

// ── GET /admin/usage/by-agent ──────────────────────────────────────────────

// handleAdminUsageByAgent returns the authenticated user's per-agent request
// counts for a billing period, ordered by count descending.
//
// Query params:
//   - period (optional): billing period; defaults to current billing period
func handleAdminUsageByAgent(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		userID := profile.ID

		periodStart, ok := resolvePeriodParam(w, r)
		if !ok {
			return
		}

		agents, err := db.GetUsageByAgent(r.Context(), deps.DB, userID, periodStart)
		if err != nil {
			log.Printf("[%s] AdminUsageByAgent: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch agent usage"))
			return
		}

		_, periodEnd := db.BillingPeriodBounds(periodStart)

		entries := make([]agentUsageEntry, len(agents))
		for i, a := range agents {
			entries[i] = agentUsageEntry{
				AgentID:      a.AgentID,
				Name:         a.Name,
				RequestCount: a.RequestCount,
			}
		}

		RespondJSON(w, http.StatusOK, agentUsageResponse{
			UserID:      userID,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Agents:      entries,
		})
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// resolvePeriodParam parses the "period" query parameter and returns the
// billing period start time. If the parameter is omitted or empty, defaults
// to the current billing period. On parse failure, writes a 400 error and
// returns zero time with ok=false.
//
// Accepted formats: YYYY-MM, YYYY-MM-DD, or full RFC3339.
func resolvePeriodParam(w http.ResponseWriter, r *http.Request) (periodStart time.Time, ok bool) {
	ps := r.URL.Query().Get("period")
	if ps == "" {
		start, _ := db.BillingPeriodBounds(time.Now())
		return start, true
	}

	// Try formats from shortest to longest.
	for _, layout := range []string{"2006-01", "2006-01-02", time.RFC3339} {
		if parsed, err := time.Parse(layout, ps); err == nil {
			return parsed.UTC(), true
		}
	}

	RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
		"period must be YYYY-MM, YYYY-MM-DD, or a full ISO 8601 date-time"))
	return time.Time{}, false
}
