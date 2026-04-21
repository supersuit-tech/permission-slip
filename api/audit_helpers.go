package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
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
		CaptureError(ctx, err)
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
			CaptureError(ctx, err)
		}
	} else {
		if _, err := db.IncrementRequestCountWithBreakdown(ctx, d, p.UserID, periodStart, periodEnd, keys); err != nil {
			log.Printf("audit: failed to increment usage for %s event: %v", p.EventType, err)
			CaptureError(ctx, err)
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

// PaymentChargeParams holds the parameters for recording a payment method charge
// and emitting the corresponding audit event.
type PaymentChargeParams struct {
	UserID          string
	AgentID         int64
	AgentMeta       []byte // raw JSONB snapshot of agent metadata
	PaymentMethodID string // opaque payment method ID (not card details)
	Brand           string // card brand (e.g. "visa", "mastercard")
	Last4           string // last 4 digits of the card
	ConnectorID     string // which connector triggered the charge
	ActionType      string // action that triggered the charge (e.g. "expedia.create_booking")
	AmountCents     int    // charge amount in cents
	Currency        string // currency code (e.g. "usd")
	Description     string // human-readable description
	ApprovalID      string // approval ID if applicable (may be empty)
}

// RecordPaymentMethodUsage creates a payment method transaction record and emits
// a payment_method.charged audit event. The audit event includes only safe metadata
// (payment method ID, brand, last4, amount, currency) — never raw card details.
//
// Both the transaction and audit event are best-effort: errors are logged but
// not propagated so they don't block the request path.
func RecordPaymentMethodUsage(ctx context.Context, d db.DBTX, p PaymentChargeParams) {
	// Sanitise inputs to prevent accidental logging of sensitive data.
	last4 := sanitiseLast4(p.Last4)
	if p.AmountCents < 0 {
		log.Printf("audit: refusing to record negative amount_cents=%d for payment method %s", p.AmountCents, p.PaymentMethodID)
		CaptureError(ctx, fmt.Errorf("refusing to record negative amount_cents=%d for payment method %s", p.AmountCents, p.PaymentMethodID))
		return
	}

	// Record the payment method transaction.
	pmTx, err := db.CreatePaymentMethodTransaction(ctx, d, &db.PaymentMethodTransaction{
		PaymentMethodID: p.PaymentMethodID,
		UserID:          p.UserID,
		ConnectorID:     p.ConnectorID,
		ActionType:      p.ActionType,
		AmountCents:     p.AmountCents,
		Description:     p.Description,
	})
	if err != nil {
		log.Printf("audit: failed to record payment method transaction: %v", err)
		CaptureError(ctx, err)
		return
	}

	actionPayload := buildPaymentActionJSON(p, last4)

	connectorID := connectorIDFromActionType(p.ActionType)
	if connectorID == nil && p.ConnectorID != "" {
		connectorID = &p.ConnectorID
	}

	sourceID := pmTx.ID
	if p.ApprovalID != "" {
		sourceID = p.ApprovalID
	}

	emitAuditEventWithUsage(ctx, d, db.InsertAuditEventParams{
		UserID:      p.UserID,
		AgentID:     p.AgentID,
		EventType:   db.AuditEventPaymentMethodCharged,
		Outcome:     db.OutcomeCharged,
		SourceID:    sourceID,
		SourceType:  db.SourceTypePaymentMethodTx,
		AgentMeta:   p.AgentMeta,
		Action:      actionPayload,
		ConnectorID: connectorID,
	}, false) // not billable — the triggering action already counted
}

// sanitiseLast4 ensures the last4 value is at most 4 characters to prevent
// accidental logging of full card numbers. If the input is longer than 4
// characters, only the last 4 are retained.
func sanitiseLast4(raw string) string {
	if len(raw) > 4 {
		return raw[len(raw)-4:]
	}
	return raw
}

// buildPaymentActionJSON constructs the action JSON for a payment_method.charged
// audit event containing only safe metadata.
func buildPaymentActionJSON(p PaymentChargeParams, last4 string) []byte {
	currency := p.Currency
	if currency == "" {
		currency = "usd"
	}
	actionData := map[string]any{
		"type":              p.ActionType,
		"payment_method_id": p.PaymentMethodID,
		"brand":             p.Brand,
		"last4":             last4,
		"amount_cents":      p.AmountCents,
		"currency":          currency,
	}
	if p.Description != "" {
		actionData["description"] = p.Description
	}
	payload, _ := json.Marshal(actionData)
	return payload
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

// redactActionToTypeWithConnectorInstance is like redactActionToType but preserves
// frozen multi-instance routing fields for audit visibility (no parameters).
func redactActionToTypeWithConnectorInstance(raw []byte) []byte {
	if len(raw) == 0 {
		return nil
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return redactActionToType(raw)
	}
	typeRaw, ok := obj["type"]
	if !ok {
		return redactActionToType(raw)
	}
	var actionType string
	if json.Unmarshal(typeRaw, &actionType) != nil || actionType == "" {
		return redactActionToType(raw)
	}
	out := map[string]any{"type": actionType}
	if v, ok := obj["_connector_instance_id"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil && s != "" {
			out["_connector_instance_id"] = s
		}
	}
	if v, ok := obj["_connector_instance_display"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil && s != "" {
			out["_connector_instance_display"] = s
		}
	}
	if v, ok := obj["_connector_instance_label"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil && s != "" {
			out["_connector_instance_label"] = s
		}
	}
	redacted, _ := json.Marshal(out)
	return redacted
}
