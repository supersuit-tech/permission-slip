package testhelper

import (
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// SourceTypeForEvent returns the correct source_type for the given event type.
func SourceTypeForEvent(eventType string) string {
	switch {
	case eventType == "agent.registered" || eventType == "agent.deactivated":
		return "agent"
	case eventType == "standing_approval.executed":
		return "standing_approval"
	default:
		return "approval"
	}
}

// InsertAuditEvent inserts a row into the audit_events table for testing.
func InsertAuditEvent(t *testing.T, d db.DBTX, userID string, agentID int64, eventType, outcome, sourceID string) {
	t.Helper()
	InsertAuditEventAt(t, d, userID, agentID, eventType, outcome, sourceID, time.Now())
}

// InsertAuditEventAt inserts an audit event with an explicit created_at timestamp.
func InsertAuditEventAt(t *testing.T, d db.DBTX, userID string, agentID int64, eventType, outcome, sourceID string, createdAt time.Time) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, '{"name":"test"}', '{"type":"test.action"}', $7)`,
		userID, agentID, eventType, outcome, sourceID, SourceTypeForEvent(eventType), createdAt)
}

// InsertAuditEventWithAction inserts an audit event with explicit agent_meta and action JSONB values.
func InsertAuditEventWithAction(t *testing.T, d db.DBTX, userID string, agentID int64, eventType, outcome, sourceID string, agentMeta, action []byte) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID, agentID, eventType, outcome, sourceID, SourceTypeForEvent(eventType), agentMeta, action)
}
