package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// auditEventResponse is the JSON representation of a single audit event
// for the activity feed (GET /audit-events). Includes source_id and source_type
// so the frontend can link events back to their source approval for detail views.
type auditEventResponse struct {
	EventType       string    `json:"event_type"`
	Timestamp       time.Time `json:"timestamp"`
	AgentID         int64     `json:"agent_id"`
	AgentMeta       any       `json:"agent_metadata,omitempty"`
	Action          any       `json:"action,omitempty"`
	Outcome         string    `json:"outcome"`
	SourceID        string    `json:"source_id,omitempty"`
	SourceType      string    `json:"source_type,omitempty"`
	ConnectorID     *string   `json:"connector_id,omitempty"`
	ExecutionStatus *string   `json:"execution_status,omitempty"`
	ExecutionError  *string   `json:"execution_error,omitempty"`
}

// auditLogExportEventResponse is the JSON representation of a single audit
// event for the export endpoint (GET /audit-logs). Includes id and source_id
// (always present, never omitted) for SIEM deduplication and event correlation.
type auditLogExportEventResponse struct {
	ID              int64     `json:"id"`
	EventType       string    `json:"event_type"`
	Timestamp       time.Time `json:"timestamp"`
	AgentID         int64     `json:"agent_id"`
	AgentMeta       any       `json:"agent_metadata,omitempty"`
	Action          any       `json:"action,omitempty"`
	Outcome         string    `json:"outcome"`
	SourceID        string    `json:"source_id"`
	ConnectorID     *string   `json:"connector_id,omitempty"`
	ExecutionStatus *string   `json:"execution_status,omitempty"`
	ExecutionError  *string   `json:"execution_error,omitempty"`
}

// retentionMeta is included in audit event responses so the frontend knows
// the user's effective retention window and can display contextual UI (e.g.
// "Showing events from the last 7 days" or an upgrade prompt).
type retentionMeta struct {
	Days              int        `json:"days"`
	GracePeriodEndsAt *time.Time `json:"grace_period_ends_at,omitempty"`
}

// auditEventListResponse is the paginated JSON response for GET /audit-events.
type auditEventListResponse struct {
	Data       []auditEventResponse `json:"data"`
	HasMore    bool                 `json:"has_more"`
	NextCursor *string              `json:"next_cursor,omitempty"`
	Retention  *retentionMeta       `json:"retention,omitempty"`
}

var validOutcomeValues = []string{
	"approved", "auto_executed", "cancelled", "charged", "deactivated", "denied", "expired", "pending", "registered",
}

var validOutcomeFilters = func() map[string]bool {
	m := make(map[string]bool, len(validOutcomeValues))
	for _, v := range validOutcomeValues {
		m[v] = true
	}
	return m
}()

// auditLogExportResponse is the JSON response for GET /audit-logs.
type auditLogExportResponse struct {
	Data           []auditLogExportEventResponse `json:"data"`
	HasMore        bool                          `json:"has_more"`
	NextCursor     *string                       `json:"next_cursor,omitempty"`
	Retention      *retentionMeta                `json:"retention,omitempty"`
	EffectiveSince *time.Time                    `json:"effective_since,omitempty"`
}

func init() {
	RegisterRouteGroup(RegisterAuditEventRoutes)
}

// RegisterAuditEventRoutes adds audit event endpoints to the mux.
func RegisterAuditEventRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /audit-events", requireProfile(handleListAuditEvents(deps)))
	mux.Handle("GET /audit-logs", requireProfile(handleExportAuditLogs(deps)))
}

// defaultFreeRetentionDays is the retention window used when a user has no
// subscription. Matches the free plan's audit_retention_days.
const defaultFreeRetentionDays = 7

// retentionInfo holds the resolved retention policy for a user, including
// the effective retention window and any active grace period.
type retentionInfo struct {
	Days              int
	GracePeriodEndsAt *time.Time
}

// retentionDays returns the number of retention days, or 0 if r is nil (billing disabled).
func (r *retentionInfo) retentionDays() int {
	if r == nil {
		return 0
	}
	return r.Days
}

// toMeta converts the retention info to the API response metadata, or nil if
// retention is not enforced (billing disabled).
func (r *retentionInfo) toMeta() *retentionMeta {
	if r == nil {
		return nil
	}
	return &retentionMeta{
		Days:              r.Days,
		GracePeriodEndsAt: r.GracePeriodEndsAt,
	}
}

// effectiveRetention resolves the full audit log retention policy for the user.
// Returns nil when billing is disabled (no retention filtering).
func effectiveRetention(ctx context.Context, deps *Deps, userID string) (*retentionInfo, error) {
	if !deps.BillingEnabled {
		return nil, nil
	}
	sp, err := db.GetSubscriptionWithPlan(ctx, deps.DB, userID)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return &retentionInfo{Days: defaultFreeRetentionDays}, nil
	}
	return &retentionInfo{
		Days:              sp.EffectiveRetentionDays(),
		GracePeriodEndsAt: sp.GracePeriodEndsAt(),
	}, nil
}

// handleListAuditEvents returns a paginated activity feed for the authenticated
// user. Supports filtering by agent_id, event_type, and outcome, with
// cursor-based pagination via the "after" query parameter.
func handleListAuditEvents(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		// Parse limit (default 20, max 100).
		limit := db.DefaultAuditEventLimit
		if v := r.URL.Query().Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "limit must be a positive integer"))
				return
			}
			if n > db.MaxAuditEventListSize {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "limit must be between 1 and 100"))
				return
			}
			limit = n
		}

		// Parse cursor (compound: "RFC3339Nano,id").
		var cursor *db.AuditEventCursor
		if v := r.URL.Query().Get("after"); v != "" {
			ts, id, err := parseTimestampIDCursor(v)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid pagination cursor"))
				return
			}
			cursor = &db.AuditEventCursor{Timestamp: ts, ID: id}
		}

		// Parse filters.
		var filter *db.AuditEventFilter

		agentIDStr := r.URL.Query().Get("agent_id")
		eventTypesStr := r.URL.Query().Get("event_type")
		outcomeStr := r.URL.Query().Get("outcome")
		connectorIDStr := r.URL.Query().Get("connector_id")

		if agentIDStr != "" || eventTypesStr != "" || outcomeStr != "" || connectorIDStr != "" {
			filter = &db.AuditEventFilter{}

			if agentIDStr != "" {
				id, err := strconv.ParseInt(agentIDStr, 10, 64)
				if err != nil || id <= 0 {
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "agent_id must be a positive integer"))
					return
				}
				filter.AgentID = &id
			}

			if eventTypesStr != "" {
				types, err := parseEventTypeFilter(eventTypesStr)
				if err != nil {
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error()))
					return
				}
				filter.EventTypes = types
			}

			if outcomeStr != "" {
				if !validOutcomeFilters[outcomeStr] {
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
						"invalid outcome filter: "+outcomeStr+". Allowed values: "+strings.Join(validOutcomeValues, ", ")))
					return
				}
				filter.Outcome = outcomeStr
			}

			if connectorIDStr != "" {
				if len(connectorIDStr) > 128 {
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id exceeds maximum length"))
					return
				}
				filter.ConnectorID = &connectorIDStr
			}
		}

		retention, err := effectiveRetention(r.Context(), deps, userID)
		if err != nil {
			log.Printf("[%s] effectiveRetention: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list audit events"))
			return
		}

		page, err := db.ListAuditEvents(r.Context(), deps.DB, userID, limit, cursor, filter, retention.retentionDays())
		if err != nil {
			log.Printf("[%s] ListAuditEvents: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list audit events"))
			return
		}

		data := make([]auditEventResponse, len(page.Events))
		for i, e := range page.Events {
			data[i] = toAuditEventResponse(e)
		}

		resp := auditEventListResponse{
			Data:      data,
			HasMore:   page.HasMore,
			Retention: retention.toMeta(),
		}
		if page.HasMore && len(page.Events) > 0 {
			last := page.Events[len(page.Events)-1]
			c := last.Timestamp.UTC().Format(time.RFC3339Nano) + "," + strconv.FormatInt(last.ID, 10)
			resp.NextCursor = &c
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// handleExportAuditLogs returns a chronologically-ordered (oldest first) export
// of audit events for the authenticated user, filtered by a required `since`
// timestamp. Designed for compliance/SIEM export use cases.
//
// Key differences from the activity feed (handleListAuditEvents):
//   - Sort order: ASC (oldest first) vs DESC (newest first)
//   - Page size:  up to 1000 vs 100, for efficient bulk export
//   - Required:   `since` timestamp to bound the query
//   - Optional:   `until` for bounded time windows (e.g. monthly reports)
//   - Fields:     includes `id` and `source_id` for SIEM deduplication
func handleExportAuditLogs(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		// Parse required `since` parameter (RFC3339, with optional fractional seconds).
		sinceStr := r.URL.Query().Get("since")
		if sinceStr == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "since query parameter is required (RFC3339 timestamp)"))
			return
		}
		since, err := time.Parse(time.RFC3339Nano, sinceStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "since must be a valid RFC3339 timestamp"))
			return
		}

		// Parse optional `until` parameter (RFC3339, with optional fractional seconds).
		var until *time.Time
		if v := r.URL.Query().Get("until"); v != "" {
			t, err := time.Parse(time.RFC3339Nano, v)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "until must be a valid RFC3339 timestamp"))
				return
			}
			if !t.After(since) {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "until must be after since"))
				return
			}
			until = &t
		}

		// Parse optional limit (default 100, max 1000).
		limit := db.DefaultAuditLogExportLimit
		if v := r.URL.Query().Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "limit must be a positive integer"))
				return
			}
			if n > db.MaxAuditLogExportSize {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "limit must be between 1 and 1000"))
				return
			}
			limit = n
		}

		// Parse optional event_type filter (comma-separated).
		var eventTypes []db.AuditEventType
		if v := r.URL.Query().Get("event_type"); v != "" {
			var err error
			eventTypes, err = parseEventTypeFilter(v)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error()))
				return
			}
		}

		// Parse optional connector_id filter.
		var connectorID *string
		if v := r.URL.Query().Get("connector_id"); v != "" {
			if len(v) > 128 {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "connector_id exceeds maximum length"))
				return
			}
			connectorID = &v
		}

		// Parse optional cursor (compound: "RFC3339Nano,id").
		var cursor *db.AuditLogExportCursor
		if v := r.URL.Query().Get("after"); v != "" {
			ts, id, err := parseTimestampIDCursor(v)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid pagination cursor"))
				return
			}
			cursor = &db.AuditLogExportCursor{Timestamp: ts, ID: id}
		}

		retention, err := effectiveRetention(r.Context(), deps, userID)
		if err != nil {
			log.Printf("[%s] effectiveRetention: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to export audit logs"))
			return
		}

		// Track whether `since` gets clamped so we can tell the client.
		originalSince := since
		days := retention.retentionDays()
		page, err := db.ExportAuditLogs(r.Context(), deps.DB, userID, since, until, eventTypes, connectorID, limit, cursor, days)
		if err != nil {
			log.Printf("[%s] ExportAuditLogs: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to export audit logs"))
			return
		}

		data := make([]auditLogExportEventResponse, len(page.Events))
		for i, e := range page.Events {
			data[i] = toExportAuditEventResponse(e)
		}

		resp := auditLogExportResponse{
			Data:      data,
			HasMore:   page.HasMore,
			Retention: retention.toMeta(),
		}
		if days > 0 {
			// If the requested `since` was before the retention window,
			// tell the client what effective start date was actually used.
			retentionFloor := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
			if originalSince.Before(retentionFloor) {
				resp.EffectiveSince = &retentionFloor
			}
		}
		if page.HasMore && len(page.Events) > 0 {
			last := page.Events[len(page.Events)-1]
			c := last.Timestamp.UTC().Format(time.RFC3339Nano) + "," + strconv.FormatInt(last.ID, 10)
			resp.NextCursor = &c
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// toAuditEventResponse converts a db.AuditEvent to its JSON representation
// for the activity feed endpoint.
func toAuditEventResponse(e db.AuditEvent) auditEventResponse {
	return auditEventResponse{
		EventType:       string(e.EventType),
		Timestamp:       e.Timestamp,
		AgentID:         e.AgentID,
		AgentMeta:       unmarshalJSONB(e.AgentMeta),
		Action:          unmarshalJSONB(e.Action),
		Outcome:         e.Outcome,
		SourceID:        e.SourceID,
		SourceType:      e.SourceType,
		ConnectorID:     e.ConnectorID,
		ExecutionStatus: e.ExecutionStatus,
		ExecutionError:  e.ExecutionError,
	}
}

// toExportAuditEventResponse converts a db.AuditEvent to its JSON representation
// for the export endpoint. Includes ID and SourceID for SIEM deduplication
// and event correlation. These fields are never omitted (no `omitempty`),
// matching the OpenAPI schema's `required` declaration.
func toExportAuditEventResponse(e db.AuditEvent) auditLogExportEventResponse {
	return auditLogExportEventResponse{
		ID:              e.ID,
		EventType:       string(e.EventType),
		Timestamp:       e.Timestamp,
		AgentID:         e.AgentID,
		AgentMeta:       unmarshalJSONB(e.AgentMeta),
		Action:          unmarshalJSONB(e.Action),
		Outcome:         e.Outcome,
		SourceID:        e.SourceID,
		ConnectorID:     e.ConnectorID,
		ExecutionStatus: e.ExecutionStatus,
		ExecutionError:  e.ExecutionError,
	}
}

// unmarshalJSONB attempts to unmarshal raw JSONB bytes into an any value.
// Returns nil for empty or malformed input. Malformed JSON is silently
// dropped because the JSONB is always written by our own InsertAuditEvent
// path — corruption would indicate a deeper data integrity issue.
func unmarshalJSONB(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err == nil {
		return v
	}
	return nil
}

// parseTimestampIDCursor parses a compound pagination cursor of the form
// "RFC3339Nano,int64". Both the activity feed and export endpoints use this
// format to guarantee stable keyset pagination across events that share a
// timestamp.
func parseTimestampIDCursor(raw string) (time.Time, int64, error) {
	comma := strings.IndexByte(raw, ',')
	if comma < 0 || comma == len(raw)-1 {
		return time.Time{}, 0, fmt.Errorf("missing separator")
	}
	t, err := time.Parse(time.RFC3339Nano, raw[:comma])
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid timestamp: %w", err)
	}
	id, err := strconv.ParseInt(raw[comma+1:], 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid id: %w", err)
	}
	return t, id, nil
}

// parseEventTypeFilter splits a comma-separated event_type query parameter
// and validates each type against the known set. Returns an error for the
// first unrecognised type.
func parseEventTypeFilter(raw string) ([]db.AuditEventType, error) {
	parts := strings.Split(raw, ",")
	types := make([]db.AuditEventType, 0, len(parts))
	for _, s := range parts {
		et := db.AuditEventType(strings.TrimSpace(s))
		if !db.IsValidAuditEventType(et) {
			return nil, fmt.Errorf("invalid event_type: %s", string(et))
		}
		types = append(types, et)
	}
	return types, nil
}
