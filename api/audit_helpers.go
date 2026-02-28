package api

import (
	"context"
	"log"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// emitAuditEventWithUsage inserts an audit event and optionally increments the
// usage meter for billing. Both operations are best-effort: errors are logged
// but never propagated to the caller (audit and metering should not block the
// request path).
//
// Set billable to true for events that count toward the user's monthly request
// quota (approval.requested and standing_approval.executed).
func emitAuditEventWithUsage(ctx context.Context, d db.DBTX, p db.InsertAuditEventParams, billable bool) {
	if err := db.InsertAuditEvent(ctx, d, p); err != nil {
		log.Printf("audit: failed to insert %s event: %v", p.EventType, err)
	}

	if !billable {
		return
	}

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	connectorID := ""
	if p.ConnectorID != nil {
		connectorID = *p.ConnectorID
	}
	actionType := ""
	if p.Action != nil {
		actionType = actionTypeFromJSON(p.Action)
	}

	if _, err := db.IncrementRequestCountWithBreakdown(ctx, d, p.UserID, periodStart, periodEnd, db.UsageBreakdownKeys{
		AgentID:     p.AgentID,
		ConnectorID: connectorID,
		ActionType:  actionType,
	}); err != nil {
		log.Printf("audit: failed to increment usage for %s event: %v", p.EventType, err)
	}
}
