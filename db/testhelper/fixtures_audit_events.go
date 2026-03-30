package testhelper

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// SourceTypeForEvent returns the correct source_type for the given event type.
func SourceTypeForEvent(eventType string) string {
	switch {
	case eventType == "agent.registered" || eventType == "agent.deactivated":
		return "agent"
	case eventType == "standing_approval.executed":
		return "standing_approval"
	case eventType == string(db.AuditEventPaymentMethodCharged):
		return db.SourceTypePaymentMethodTx
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

// InsertAuditEventWithConnector inserts an audit event with an explicit connector_id.
func InsertAuditEventWithConnector(t *testing.T, d db.DBTX, userID string, agentID int64, eventType, outcome, sourceID string, connectorID *string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, connector_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, '{"name":"test"}', '{"type":"test.action"}', $7, $8)`,
		userID, agentID, eventType, outcome, sourceID, SourceTypeForEvent(eventType), connectorID, time.Now())
}

// InsertAuditEventWithAction inserts an audit event with explicit agent_meta and action JSONB values.
func InsertAuditEventWithAction(t *testing.T, d db.DBTX, userID string, agentID int64, eventType, outcome, sourceID string, agentMeta, action []byte) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID, agentID, eventType, outcome, sourceID, SourceTypeForEvent(eventType), agentMeta, action)
}

// PaymentChargeFixture holds parameters for inserting a payment_method.charged
// audit event with realistic action JSON. This avoids hand-constructing the
// JSON in every test.
type PaymentChargeFixture struct {
	PaymentMethodID string
	Brand           string
	Last4           string
	ConnectorID     string
	ActionType      string
	AmountCents     int
	Currency        string
	Description     string
}

// InsertPaymentChargedEvent inserts a payment_method.charged audit event with
// properly structured action JSON containing safe payment metadata.
func InsertPaymentChargedEvent(t *testing.T, d db.DBTX, userID string, agentID int64, sourceID string, p PaymentChargeFixture) {
	t.Helper()
	currency := p.Currency
	if currency == "" {
		currency = "usd"
	}
	action, err := json.Marshal(map[string]any{
		"type":              p.ActionType,
		"payment_method_id": p.PaymentMethodID,
		"brand":             p.Brand,
		"last4":             p.Last4,
		"amount_cents":      p.AmountCents,
		"currency":          currency,
		"description":       p.Description,
	})
	if err != nil {
		t.Fatalf("InsertPaymentChargedEvent: marshal action: %v", err)
	}
	connectorID := &p.ConnectorID
	mustExec(t, d,
		`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, connector_id)
		 VALUES ($1, $2, $3, $4, $5, $6, '{"name":"test"}', $7, $8)`,
		userID, agentID, string(db.AuditEventPaymentMethodCharged), db.OutcomeCharged,
		sourceID, db.SourceTypePaymentMethodTx, action, connectorID)
}
