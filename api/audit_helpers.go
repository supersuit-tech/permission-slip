package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// ── Shared audit utilities ──────────────────────────────────────────────────
//
// These helpers are used by multiple audit event emission functions across the
// api package. They are centralised here to avoid duplication and keep the
// audit infrastructure in one place.

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

	keys := db.UsageBreakdownKeys{
		AgentID:     p.AgentID,
		ConnectorID: connectorID,
		ActionType:  actionType,
	}

	if IsQuotaReserved(ctx) {
		// Count was already atomically incremented by checkRequestQuota.
		// Only update the breakdown without re-incrementing.
		if err := db.UpdateUsageBreakdownOnly(ctx, d, p.UserID, periodStart, keys); err != nil {
			log.Printf("audit: failed to update usage breakdown for %s event: %v", p.EventType, err)
		}
	} else {
		if _, err := db.IncrementRequestCountWithBreakdown(ctx, d, p.UserID, periodStart, periodEnd, keys); err != nil {
			log.Printf("audit: failed to increment usage for %s event: %v", p.EventType, err)
		}
	}
}

// connectorIDFromActionType extracts the connector ID from an action type string.
// Action types follow the convention "connector_id.action_name" (e.g. "github.create_issue").
// Returns nil if the action type is malformed (no dot, empty prefix, or empty string).
//
// Examples:
//
//	"github.create_issue" → &"github"
//	"slack.send_message"  → &"slack"
//	"malformed_type"      → nil
//	".missing_prefix"     → nil
//	""                    → nil
func connectorIDFromActionType(actionType string) *string {
	if parts := strings.SplitN(actionType, ".", 2); len(parts) == 2 && parts[0] != "" {
		return &parts[0]
	}
	return nil
}

// actionTypeFromJSON extracts the "type" field from an action JSON blob.
// Returns "" if the type cannot be extracted.
func actionTypeFromJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var obj struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &obj) != nil {
		return ""
	}
	return obj.Type
}

// resolveExecResult maps a connector execution error to the appropriate
// execution_status and execution_error values for the audit event. Returns
// ("success", nil) on success, ("timeout", &message) for deadline exceeded,
// or ("failure", &message) for all other errors.
//
// The error message is truncated to avoid storing excessively large strings;
// execution_error is exposed in the API so internal details must not leak.
func resolveExecResult(execErr error) (status string, errMsg *string) {
	if execErr == nil {
		return db.ExecStatusSuccess, nil
	}
	msg := execErr.Error()
	if len(msg) > 512 {
		msg = msg[:512]
	}
	if errors.Is(execErr, context.DeadlineExceeded) {
		return db.ExecStatusTimeout, &msg
	}
	return db.ExecStatusFailure, &msg
}

// redactActionToType extracts only the "type" field from an action JSON blob,
// discarding parameters and other user-provided data. Returns {"type":"…"} or
// nil if the type cannot be extracted.
func redactActionToType(raw []byte) []byte {
	if len(raw) == 0 {
		return nil
	}
	var obj struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &obj) != nil || obj.Type == "" {
		return nil
	}
	redacted, _ := json.Marshal(map[string]string{"type": obj.Type})
	return redacted
}
