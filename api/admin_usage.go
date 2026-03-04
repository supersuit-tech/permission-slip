package api

import (
	"log"
	"net/http"
	"strconv"
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

type topUsersResponse struct {
	PeriodStart time.Time      `json:"period_start"`
	PeriodEnd   time.Time      `json:"period_end"`
	Users       []topUserEntry `json:"users"`
}

type topUserEntry struct {
	UserID       string  `json:"user_id"`
	Username     string  `json:"username"`
	Email        *string `json:"email,omitempty"`
	RequestCount int     `json:"request_count"`
	SMSCount     int     `json:"sms_count"`
}

type connectorUsageResponse struct {
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

// RegisterAdminUsageRoutes adds admin usage analytics endpoints to the mux.
// All endpoints require session auth via RequireProfile.
func RegisterAdminUsageRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /admin/usage", requireProfile(handleAdminGetUsage(deps)))
	mux.Handle("GET /admin/usage/top-users", requireProfile(handleAdminTopUsers(deps)))
	mux.Handle("GET /admin/usage/by-connector", requireProfile(handleAdminUsageByConnector(deps)))
	mux.Handle("GET /admin/usage/by-agent", requireProfile(handleAdminUsageByAgent(deps)))
}

// ── GET /admin/usage ────────────────────────────────────────────────────────

// handleAdminGetUsage returns usage metrics for a specific user and billing period.
// Query params:
//   - user_id (optional): defaults to the authenticated user
//   - period (optional): billing period in YYYY-MM, YYYY-MM-DD, or RFC3339 format; defaults to current
func handleAdminGetUsage(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = profile.ID
		}

		periodStart, ok := parsePeriodParam(w, r)
		if !ok {
			return
		}

		var usage *db.UsagePeriod
		var err error
		if periodStart.IsZero() {
			usage, err = db.GetCurrentPeriodUsage(r.Context(), deps.DB, userID)
			periodStart, _ = db.BillingPeriodBounds(time.Now())
		} else {
			usage, err = db.GetUsageByPeriod(r.Context(), deps.DB, userID, periodStart)
		}
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

// ── GET /admin/usage/top-users ──────────────────────────────────────────────

// handleAdminTopUsers returns the users with the highest request counts
// for a billing period. Useful for abuse detection.
// Query params:
//   - period (optional): billing period; defaults to current billing period
//   - limit (optional): max users to return (1–100, default 10)
func handleAdminTopUsers(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		periodStart, ok := parsePeriodParam(w, r)
		if !ok {
			return
		}
		if periodStart.IsZero() {
			periodStart, _ = db.BillingPeriodBounds(time.Now())
		}

		limit := 10
		if l := r.URL.Query().Get("limit"); l != "" {
			parsed, err := strconv.Atoi(l)
			if err != nil || parsed < 1 {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "limit must be a positive integer"))
				return
			}
			limit = parsed
		}

		users, err := db.GetTopUsersByUsage(r.Context(), deps.DB, periodStart, limit)
		if err != nil {
			log.Printf("[%s] AdminTopUsers: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch top users"))
			return
		}

		_, periodEnd := db.BillingPeriodBounds(periodStart)

		entries := make([]topUserEntry, len(users))
		for i, u := range users {
			entries[i] = topUserEntry{
				UserID:       u.UserID,
				Username:     u.Username,
				Email:        u.Email,
				RequestCount: u.RequestCount,
				SMSCount:     u.SMSCount,
			}
		}

		RespondJSON(w, http.StatusOK, topUsersResponse{
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Users:       entries,
		})
	}
}

// ── GET /admin/usage/by-connector ──────────────────────────────────────────

// handleAdminUsageByConnector returns aggregate request counts per connector
// across all users for a billing period.
// Query params:
//   - period (optional): billing period; defaults to current billing period
func handleAdminUsageByConnector(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		periodStart, ok := parsePeriodParam(w, r)
		if !ok {
			return
		}
		if periodStart.IsZero() {
			periodStart, _ = db.BillingPeriodBounds(time.Now())
		}

		connectors, err := db.GetUsageByConnector(r.Context(), deps.DB, periodStart)
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
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Connectors:  entries,
		})
	}
}

// ── GET /admin/usage/by-agent ──────────────────────────────────────────────

// handleAdminUsageByAgent returns per-agent request counts for a specific user
// and billing period.
// Query params:
//   - user_id (optional): defaults to the authenticated user
//   - period (optional): billing period; defaults to current billing period
func handleAdminUsageByAgent(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = profile.ID
		}

		periodStart, ok := parsePeriodParam(w, r)
		if !ok {
			return
		}
		if periodStart.IsZero() {
			periodStart, _ = db.BillingPeriodBounds(time.Now())
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

// parsePeriodParam extracts and validates the "period" query parameter.
// Accepts three formats for convenience:
//   - "2026-03"              → YYYY-MM (month shorthand)
//   - "2026-03-01"           → YYYY-MM-DD (date)
//   - "2026-03-01T00:00:00Z" → RFC3339 (full timestamp)
//
// Returns the parsed time and true on success, or writes a 400 error and
// returns zero time and false on failure.
func parsePeriodParam(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	ps := r.URL.Query().Get("period")
	if ps == "" {
		return time.Time{}, true
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
