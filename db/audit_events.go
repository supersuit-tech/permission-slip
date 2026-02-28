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
	AuditEventApprovalRequested AuditEventType = "approval.requested"
	AuditEventApprovalApproved  AuditEventType = "approval.approved"
	AuditEventApprovalDenied    AuditEventType = "approval.denied"
	AuditEventApprovalCancelled AuditEventType = "approval.cancelled"
	AuditEventActionExecuted    AuditEventType = "action.executed"
	AuditEventStandingExecution AuditEventType = "standing_approval.executed"
	AuditEventAgentRegistered   AuditEventType = "agent.registered"
	AuditEventAgentDeactivated  AuditEventType = "agent.deactivated"
)

// validAuditEventTypes is used for input validation.
var validAuditEventTypes = map[AuditEventType]bool{
	AuditEventApprovalRequested: true,
	AuditEventApprovalApproved:  true,
	AuditEventApprovalDenied:    true,
	AuditEventApprovalCancelled: true,
	AuditEventActionExecuted:    true,
	AuditEventStandingExecution: true,
	AuditEventAgentRegistered:   true,
	AuditEventAgentDeactivated:  true,
}

// IsValidAuditEventType checks if the given event type is valid.
func IsValidAuditEventType(t AuditEventType) bool {
	return validAuditEventTypes[t]
}

// AuditEvent represents a single activity event in the audit trail.
type AuditEvent struct {
	ID        int64
	EventType AuditEventType
	Timestamp time.Time
	AgentID   int64
	AgentMeta []byte // raw JSONB (agent metadata snapshot at event time)
	Action    []byte // raw JSONB (approval action details, nullable)
	Outcome   string // "approved", "denied", "cancelled", "auto_executed", "registered", "deactivated", "pending", "expired"
	SourceID  string // unique ID per event source (e.g. approval_id)
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
	AgentID    *int64
	EventTypes []AuditEventType // if non-empty, include only these types
	Outcome    string           // "approved", "denied", "cancelled", "auto_executed", "registered", "deactivated", "pending", "expired"
}

// MaxAuditEventListSize is the hard cap on returned events.
const MaxAuditEventListSize = 100

// DefaultAuditEventLimit is the default page size.
const DefaultAuditEventLimit = 20

// InsertAuditEventParams holds the parameters for inserting an audit event.
type InsertAuditEventParams struct {
	UserID     string
	AgentID    int64
	EventType  AuditEventType
	Outcome    string
	SourceID   string
	SourceType string
	AgentMeta  []byte // raw JSONB
	Action     []byte // raw JSONB, may be nil
}

// InsertAuditEvent writes a new row into the audit_events table.
func InsertAuditEvent(ctx context.Context, db DBTX, p InsertAuditEventParams) error {
	_, err := db.Exec(ctx,
		`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		p.UserID, p.AgentID, string(p.EventType), p.Outcome,
		p.SourceID, p.SourceType, p.AgentMeta, p.Action,
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

// resolvedOutcomeExpr is a SQL CASE expression that resolves "pending" audit
// event outcomes to "expired" when the associated agent's registration has
// timed out. This avoids the need for a background cleanup job.
const resolvedOutcomeExpr = `CASE
		WHEN ae.outcome = 'pending'
		 AND a.status = 'pending'
		 AND a.expires_at IS NOT NULL
		 AND a.expires_at <= now()
		THEN 'expired'
		ELSE ae.outcome
	END`

// ListAuditEvents returns a paginated, chronologically-ordered (newest first)
// activity feed for the given user from the audit_events table.
//
// Pending agent registration outcomes are resolved to "expired" at query time
// by joining the agents table and checking whether the registration TTL has
// elapsed while the agent is still in pending status.
func ListAuditEvents(ctx context.Context, db DBTX, userID string, limit int, cursor *AuditEventCursor, filter *AuditEventFilter) (*AuditEventPage, error) {
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
	}

	limitPlaceholder := b.addArg(fetchLimit)

	query := fmt.Sprintf(
		`SELECT ae.id, ae.event_type, ae.created_at, ae.agent_id, ae.agent_meta, ae.action,
		        %s AS outcome, ae.source_id
		 FROM audit_events ae
		 LEFT JOIN agents a ON ae.agent_id = a.agent_id
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
// "pending" whose agent registration has timed out; "pending" matches stored
// "pending" rows whose registration is still active.
func outcomeFilter(outcome string, b *queryBuilder) string {
	switch outcome {
	case "expired":
		return `(ae.outcome = 'pending' AND a.status = 'pending' AND a.expires_at IS NOT NULL AND a.expires_at <= now())`
	case "pending":
		return `(ae.outcome = 'pending' AND NOT (a.status = 'pending' AND a.expires_at IS NOT NULL AND a.expires_at <= now()))`
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
// Optional parameters:
//   - until: if non-nil, only returns events created before this timestamp
//   - eventTypes: if non-empty, only returns events matching these types
func ExportAuditLogs(ctx context.Context, db DBTX, userID string, since time.Time, until *time.Time, eventTypes []AuditEventType, limit int, cursor *AuditLogExportCursor) (*AuditEventPage, error) {
	if limit <= 0 {
		limit = DefaultAuditLogExportLimit
	}
	if limit > MaxAuditLogExportSize {
		limit = MaxAuditLogExportSize
	}
	fetchLimit := limit + 1

	b := &queryBuilder{}
	b.addArg(userID)            // $1
	sincePh := b.addArg(since)  // $2

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

	if cursor != nil {
		tsPlaceholder := b.addArg(cursor.Timestamp)
		idPlaceholder := b.addArg(cursor.ID)
		where = append(where, fmt.Sprintf("(ae.created_at, ae.id) > (%s, %s)", tsPlaceholder, idPlaceholder))
	}

	limitPlaceholder := b.addArg(fetchLimit)

	query := fmt.Sprintf(
		`SELECT ae.id, ae.event_type, ae.created_at, ae.agent_id, ae.agent_meta, ae.action,
		        %s AS outcome, ae.source_id
		 FROM audit_events ae
		 LEFT JOIN agents a ON ae.agent_id = a.agent_id
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
		if err := rows.Scan(&e.ID, &eventType, &e.Timestamp, &e.AgentID, &e.AgentMeta, &e.Action, &e.Outcome, &e.SourceID); err != nil {
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
