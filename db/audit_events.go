package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// AuditEventType represents the type of an audit event.
type AuditEventType string

const (
	AuditEventApprovalRequested    AuditEventType = "approval.requested"
	AuditEventApprovalApproved     AuditEventType = "approval.approved"
	AuditEventApprovalDenied       AuditEventType = "approval.denied"
	AuditEventApprovalCancelled    AuditEventType = "approval.cancelled"
	AuditEventActionExecuted       AuditEventType = "action.executed"
	AuditEventStandingExecution    AuditEventType = "standing_approval.executed"
	AuditEventStandingUpdated      AuditEventType = "standing_approval.updated"
	AuditEventAgentRegistered      AuditEventType = "agent.registered"
	AuditEventAgentDeactivated     AuditEventType = "agent.deactivated"
	AuditEventPaymentMethodCharged AuditEventType = "payment_method.charged"
)

// validAuditEventTypes is used for input validation.
var validAuditEventTypes = map[AuditEventType]bool{
	AuditEventApprovalRequested:    true,
	AuditEventApprovalApproved:     true,
	AuditEventApprovalDenied:       true,
	AuditEventApprovalCancelled:    true,
	AuditEventActionExecuted:       true,
	AuditEventStandingExecution:    true,
	AuditEventStandingUpdated:      true,
	AuditEventAgentRegistered:      true,
	AuditEventAgentDeactivated:     true,
	AuditEventPaymentMethodCharged: true,
}

// IsValidAuditEventType checks if the given event type is valid.
func IsValidAuditEventType(t AuditEventType) bool {
	return validAuditEventTypes[t]
}

// AuditEvent represents a single activity event in the audit trail.
type AuditEvent struct {
	ID              int64
	EventType       AuditEventType
	Timestamp       time.Time
	AgentID         int64
	AgentMeta       []byte  // raw JSONB (agent metadata snapshot at event time)
	Action          []byte  // raw JSONB (approval action details, nullable)
	Outcome         string  // "approved", "denied", "cancelled", "auto_executed", "registered", "deactivated", "pending", "expired", "charged", "updated"
	SourceID        string  // unique ID per event source (e.g. approval_id)
	SourceType      string  // "approval", "standing_approval", "agent", "payment_method_transaction"
	ConnectorID     *string // which connector handled the action (nullable for lifecycle events)
	ExecutionStatus *string // "success", "failure", "timeout", "skipped" (nullable)
	ExecutionError  *string // failure details (nullable)
}

// AuditEventPage holds a page of audit events plus a flag indicating whether more exist.
type AuditEventPage struct {
	Events  []AuditEvent
	HasMore bool
}

// AuditEventCursor identifies the position of the last item on a page.
// Uses a compound key (created_at, id) to guarantee correct pagination
// even when multiple events share the same timestamp.
type AuditEventCursor struct {
	Timestamp time.Time
	ID        int64
}

// AuditEventFilter holds optional filters for the audit event query.
type AuditEventFilter struct {
	AgentID     *int64
	EventTypes  []AuditEventType // if non-empty, include only these types
	Outcome     string           // "approved", "denied", "cancelled", "auto_executed", "registered", "deactivated", "pending", "expired", "charged", "updated"
	ConnectorID *string          // if non-nil, include only events for this connector
}

// MaxAuditEventListSize is the hard cap on returned events.
const MaxAuditEventListSize = 100

// DefaultAuditEventLimit is the default page size.
const DefaultAuditEventLimit = 20

// ExecutionStatus constants for audit event execution tracking.
const (
	ExecStatusSuccess = "success"
	ExecStatusFailure = "failure"
	ExecStatusTimeout = "timeout"
	ExecStatusSkipped = "skipped"
)

// Payment audit event constants.
const (
	OutcomeCharged            = "charged"
	OutcomeUpdated            = "updated"
	SourceTypePaymentMethodTx = "payment_method_transaction"
)

// InsertAuditEventParams holds the parameters for inserting an audit event.
type InsertAuditEventParams struct {
	UserID          string
	AgentID         int64
	EventType       AuditEventType
	Outcome         string
	SourceID        string
	SourceType      string
	AgentMeta       []byte  // raw JSONB
	Action          []byte  // raw JSONB, may be nil
	ConnectorID     *string // which connector handled the action (nil for lifecycle events)
	ExecutionStatus *string // "success", "failure", "timeout", "skipped" (nil for non-execution events)
	ExecutionError  *string // failure details (nil unless ExecutionStatus is "failure"); exposed in API — never store internal/sensitive details
}

// InsertAuditEvent writes a new row into the audit_events table.
func InsertAuditEvent(ctx context.Context, db DBTX, p InsertAuditEventParams) error {
	_, err := db.Exec(ctx,
		`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, connector_id, execution_status, execution_error)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		p.UserID, p.AgentID, string(p.EventType), p.Outcome,
		p.SourceID, p.SourceType, p.AgentMeta, p.Action,
		p.ConnectorID, p.ExecutionStatus, p.ExecutionError,
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

// resolvedOutcomeExpr is a SQL CASE expression that resolves "pending" audit
// event outcomes to "expired" when the associated entity has timed out:
//   - Agent registrations: checks agents.expires_at via the LEFT JOIN on agents.
//   - Approval requests: checks approvals.expires_at via the LEFT JOIN on approvals.
//
// This avoids the need for a background cleanup job.
const resolvedOutcomeExpr = `CASE
		WHEN ae.outcome = 'pending'
		 AND a.status = 'pending'
		 AND a.expires_at IS NOT NULL
		 AND a.expires_at <= now()
		THEN 'expired'
		WHEN ae.outcome = 'pending'
		 AND ae.source_type = 'approval'
		 AND appr.status = 'pending'
		 AND appr.expires_at <= now()
		THEN 'expired'
		ELSE ae.outcome
	END`

// ListAuditEvents returns a paginated, chronologically-ordered (newest first)
// activity feed for the given user from the audit_events table.
//
// retentionDays controls the retention window: if > 0, only events within the
// last N days are returned. Pass 0 to disable retention filtering.
//
// Pending agent registration outcomes are resolved to "expired" at query time
// by joining the agents table and checking whether the registration TTL has
// elapsed while the agent is still in pending status.
func ListAuditEvents(ctx context.Context, db DBTX, userID string, limit int, cursor *AuditEventCursor, filter *AuditEventFilter, retentionDays int) (*AuditEventPage, error) {
	if limit <= 0 {
		limit = DefaultAuditEventLimit
	}
	if limit > MaxAuditEventListSize {
		limit = MaxAuditEventListSize
	}
	fetchLimit := limit + 1

	b := &queryBuilder{}
	b.addArg(userID) // $1

	where := []string{"ae.user_id = $1"}

	if retentionDays > 0 {
		where = append(where, "ae.created_at >= now() - make_interval(days => "+b.addArg(retentionDays)+")")
	}

	if cursor != nil {
		tsPlaceholder := b.addArg(cursor.Timestamp)
		idPlaceholder := b.addArg(cursor.ID)
		where = append(where, fmt.Sprintf("(ae.created_at, ae.id) < (%s, %s)", tsPlaceholder, idPlaceholder))
	}

	if filter != nil {
		if filter.AgentID != nil {
			where = append(where, "ae.agent_id = "+b.addArg(*filter.AgentID))
		}
		if len(filter.EventTypes) > 0 {
			placeholders := make([]string, len(filter.EventTypes))
			for i, et := range filter.EventTypes {
				placeholders[i] = b.addArg(string(et))
			}
			where = append(where, "ae.event_type IN ("+strings.Join(placeholders, ", ")+")")
		}
		if filter.Outcome != "" {
			where = append(where, outcomeFilter(filter.Outcome, b))
		}
		if filter.ConnectorID != nil {
			where = append(where, "ae.connector_id = "+b.addArg(*filter.ConnectorID))
		}
	}

	// Suppress approval.requested events that have a corresponding resolution
	// event (approved/denied/cancelled) for the same source_id. This keeps the
	// activity feed clean by showing only the final state of each approval.
	// Skip the dedup when the caller explicitly filters by approval.requested —
	// an explicit filter signals intent to see all matching events (e.g. audits).
	requestedTypeExplicit := false
	if filter != nil {
		for _, et := range filter.EventTypes {
			if et == AuditEventApprovalRequested {
				requestedTypeExplicit = true
				break
			}
		}
	}
	if !requestedTypeExplicit {
		where = append(where, `NOT (
			ae.event_type = 'approval.requested'
			AND ae.source_id IS NOT NULL
			AND EXISTS (
				SELECT 1 FROM audit_events ae2
				WHERE ae2.source_id = ae.source_id
				  AND ae2.user_id = ae.user_id
				  AND ae2.event_type IN ('approval.approved', 'approval.denied', 'approval.cancelled')
			)
		)`)
	}

	limitPlaceholder := b.addArg(fetchLimit)

	query := fmt.Sprintf(
		`SELECT ae.id, ae.event_type, ae.created_at, ae.agent_id, ae.agent_meta, ae.action,
		        %s AS outcome, ae.source_id, ae.source_type, ae.connector_id, ae.execution_status, ae.execution_error
		 FROM audit_events ae
		 LEFT JOIN agents a ON ae.agent_id = a.agent_id
		 LEFT JOIN approvals appr ON ae.source_id = appr.approval_id AND ae.source_type = 'approval'
		 WHERE %s
		 ORDER BY ae.created_at DESC, ae.id DESC
		 LIMIT %s`,
		resolvedOutcomeExpr,
		strings.Join(where, " AND "),
		limitPlaceholder,
	)

	rows, err := db.Query(ctx, query, b.args...)
	if err != nil {
		return nil, fmt.Errorf("audit events query: %w", err)
	}

	events, err := scanAuditEvents(rows)
	if err != nil {
		return nil, err
	}

	return paginateEvents(events, limit), nil
}

// outcomeFilter returns a WHERE clause fragment that correctly handles the
// "expired" and "pending" virtual outcomes. "expired" matches rows stored as
// "pending" whose agent registration or approval request has timed out;
// "pending" matches stored "pending" rows that are still active.
func outcomeFilter(outcome string, b *queryBuilder) string {
	switch outcome {
	case "expired":
		// Agent registration expired OR approval request expired.
		return `(ae.outcome = 'pending' AND (
			(a.status = 'pending' AND a.expires_at IS NOT NULL AND a.expires_at <= now())
			OR (ae.source_type = 'approval' AND appr.status = 'pending' AND appr.expires_at <= now())
		))`
	case "pending":
		// Still active: neither agent registration nor approval has expired.
		return `(ae.outcome = 'pending'
			AND NOT (a.status = 'pending' AND a.expires_at IS NOT NULL AND a.expires_at <= now())
			AND NOT (ae.source_type = 'approval' AND appr.status = 'pending' AND appr.expires_at <= now())
		)`
	default:
		return "ae.outcome = " + b.addArg(outcome)
	}
}

// MaxAuditLogExportSize is the hard cap on rows per page for the export
// endpoint. Higher than the activity feed limit because compliance/SIEM
// consumers typically process large batches.
const MaxAuditLogExportSize = 1000

// DefaultAuditLogExportLimit is the default page size for the export endpoint
// when the caller does not specify a limit.
const DefaultAuditLogExportLimit = 100

// AuditLogExportCursor identifies the position for keyset pagination in
// chronological (oldest-first) order. Uses the same compound key approach
// as AuditEventCursor but paginates forward (ASC) instead of backward (DESC).
type AuditLogExportCursor struct {
	Timestamp time.Time
	ID        int64
}

// ExportAuditLogs returns audit events for the given user created at or after
// the `since` timestamp, in chronological order (oldest first). This is
// designed for compliance export use cases and supports cursor-based pagination
// via a compound (created_at, id) key.
//
// retentionDays controls the retention window: if > 0, the effective `since`
// is clamped to at least now()-retentionDays, even if the caller passes an
// earlier value. Pass 0 to disable retention filtering.
//
// Optional parameters:
//   - until: if non-nil, only returns events created before this timestamp
//   - eventTypes: if non-empty, only returns events matching these types
//   - connectorID: if non-nil, only returns events for the specified connector
func ExportAuditLogs(ctx context.Context, db DBTX, userID string, since time.Time, until *time.Time, eventTypes []AuditEventType, connectorID *string, limit int, cursor *AuditLogExportCursor, retentionDays int) (*AuditEventPage, error) {
	if limit <= 0 {
		limit = DefaultAuditLogExportLimit
	}
	if limit > MaxAuditLogExportSize {
		limit = MaxAuditLogExportSize
	}
	fetchLimit := limit + 1

	// Clamp `since` to the retention window when enforcement is active.
	// Use UTC + fixed 24h days to match the SQL `make_interval(days => N)`
	// semantics used in the list endpoint and purge jobs.
	if retentionDays > 0 {
		retentionFloor := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
		if since.Before(retentionFloor) {
			since = retentionFloor
		}
	}

	b := &queryBuilder{}
	b.addArg(userID)           // $1
	sincePh := b.addArg(since) // $2

	where := []string{"ae.user_id = $1", fmt.Sprintf("ae.created_at >= %s", sincePh)}

	if until != nil {
		untilPh := b.addArg(*until)
		where = append(where, fmt.Sprintf("ae.created_at < %s", untilPh))
	}

	if len(eventTypes) > 0 {
		placeholders := make([]string, len(eventTypes))
		for i, et := range eventTypes {
			placeholders[i] = b.addArg(string(et))
		}
		where = append(where, "ae.event_type IN ("+strings.Join(placeholders, ", ")+")")
	}

	if connectorID != nil {
		where = append(where, "ae.connector_id = "+b.addArg(*connectorID))
	}

	if cursor != nil {
		tsPlaceholder := b.addArg(cursor.Timestamp)
		idPlaceholder := b.addArg(cursor.ID)
		where = append(where, fmt.Sprintf("(ae.created_at, ae.id) > (%s, %s)", tsPlaceholder, idPlaceholder))
	}

	limitPlaceholder := b.addArg(fetchLimit)

	query := fmt.Sprintf(
		`SELECT ae.id, ae.event_type, ae.created_at, ae.agent_id, ae.agent_meta, ae.action,
		        %s AS outcome, ae.source_id, ae.source_type, ae.connector_id, ae.execution_status, ae.execution_error
		 FROM audit_events ae
		 LEFT JOIN agents a ON ae.agent_id = a.agent_id
		 LEFT JOIN approvals appr ON ae.source_id = appr.approval_id AND ae.source_type = 'approval'
		 WHERE %s
		 ORDER BY ae.created_at ASC, ae.id ASC
		 LIMIT %s`,
		resolvedOutcomeExpr,
		strings.Join(where, " AND "),
		limitPlaceholder,
	)

	rows, err := db.Query(ctx, query, b.args...)
	if err != nil {
		return nil, fmt.Errorf("export audit logs query: %w", err)
	}

	events, err := scanAuditEvents(rows)
	if err != nil {
		return nil, err
	}

	return paginateEvents(events, limit), nil
}

// scanAuditEvents reads all rows from the query result into a slice of AuditEvent.
// Both ListAuditEvents and ExportAuditLogs use identical SELECT columns, so this
// centralises the scan logic to avoid divergence.
func scanAuditEvents(rows pgx.Rows) ([]AuditEvent, error) {
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		var eventType string
		if err := rows.Scan(
			&e.ID, &eventType, &e.Timestamp, &e.AgentID, &e.AgentMeta, &e.Action,
			&e.Outcome, &e.SourceID, &e.SourceType, &e.ConnectorID, &e.ExecutionStatus, &e.ExecutionError,
		); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		e.EventType = AuditEventType(eventType)
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit event rows: %w", err)
	}
	return events, nil
}

// paginateEvents applies the limit+1 pagination pattern: if more rows were
// fetched than the requested limit, trim the slice and report has_more=true.
func paginateEvents(events []AuditEvent, limit int) *AuditEventPage {
	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}
	return &AuditEventPage{Events: events, HasMore: hasMore}
}

// queryBuilder tracks positional placeholder arguments ($1, $2, …) for
// constructing parameterised SQL queries. All user-supplied values MUST be
// added via addArg to prevent SQL injection.
type queryBuilder struct {
	args []any
}

// addArg appends a value to the argument list and returns the corresponding
// positional placeholder (e.g. "$3").
func (qb *queryBuilder) addArg(v any) string {
	qb.args = append(qb.args, v)
	return fmt.Sprintf("$%d", len(qb.args))
}
