package db_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestListAuditEvents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 0 {
			t.Errorf("expected 0 events, got %d", len(page.Events))
		}
		if page.HasMore {
			t.Error("expected has_more=false")
		}
	})

	t.Run("ApprovalEvents", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(page.Events) != 2 {
			t.Fatalf("expected 2 approval events, got %d", len(page.Events))
		}

		for _, e := range page.Events {
			if e.EventType != db.AuditEventApprovalApproved && e.EventType != db.AuditEventApprovalDenied {
				t.Errorf("unexpected event type: %s", e.EventType)
			}
		}
	})

	t.Run("AgentRegisteredEvent", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "agent.registered", "registered", fmt.Sprintf("ar:%d", agentID))

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found := false
		for _, e := range page.Events {
			if e.EventType == db.AuditEventAgentRegistered && e.AgentID == agentID {
				found = true
				if e.Outcome != "registered" {
					t.Errorf("expected outcome 'registered', got %q", e.Outcome)
				}
			}
		}
		if !found {
			t.Error("expected agent.registered event, not found")
		}
	})

	t.Run("AgentDeactivatedEvent", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "agent.deactivated", "deactivated", fmt.Sprintf("ad:%d", agentID))

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found := false
		for _, e := range page.Events {
			if e.EventType == db.AuditEventAgentDeactivated && e.AgentID == agentID {
				found = true
				if e.Outcome != "deactivated" {
					t.Errorf("expected outcome 'deactivated', got %q", e.Outcome)
				}
			}
		}
		if !found {
			t.Error("expected agent.deactivated event, not found")
		}
	})

	t.Run("StandingApprovalExecution", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Audit events only store the action type — parameters are redacted.
		actionTypeOnly := []byte(`{"type":"send_email"}`)

		testhelper.InsertAuditEventWithAction(t, tx, uid, agentID, "standing_approval.executed", "auto_executed", testhelper.GenerateID(t, "sae_"), []byte(`{"name":"test"}`), actionTypeOnly)
		testhelper.InsertAuditEventWithAction(t, tx, uid, agentID, "standing_approval.executed", "auto_executed", testhelper.GenerateID(t, "sae_"), []byte(`{"name":"test"}`), actionTypeOnly)
		testhelper.InsertAuditEventWithAction(t, tx, uid, agentID, "standing_approval.executed", "auto_executed", testhelper.GenerateID(t, "sae_"), []byte(`{"name":"test"}`), actionTypeOnly)

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		executionCount := 0
		for _, e := range page.Events {
			if e.EventType == db.AuditEventStandingExecution {
				executionCount++
				if e.Outcome != "auto_executed" {
					t.Errorf("expected outcome 'auto_executed', got %q", e.Outcome)
				}

				// Verify action contains only the type key (no parameters).
				var action map[string]json.RawMessage
				if err := json.Unmarshal(e.Action, &action); err != nil {
					t.Fatalf("failed to parse action JSON: %v", err)
				}
				if _, ok := action["type"]; !ok {
					t.Error("action missing 'type' key")
				}
				if _, ok := action["parameters"]; ok {
					t.Error("action should not contain 'parameters' key — sensitive data must be redacted")
				}
			}
		}
		if executionCount != 3 {
			t.Errorf("expected 3 standing_approval.executed events, got %d", executionCount)
		}
	})

	t.Run("FilterByAgentID", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		agent1 := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
		agent2 := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

		testhelper.InsertAuditEvent(t, tx, uid, agent1, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
		testhelper.InsertAuditEvent(t, tx, uid, agent2, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))

		filter := &db.AuditEventFilter{AgentID: &agent1}
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event for agent1, got %d", len(page.Events))
		}
		for _, e := range page.Events {
			if e.AgentID != agent1 {
				t.Errorf("expected only agent %d events, got agent %d", agent1, e.AgentID)
			}
		}
	})

	t.Run("FilterByEventType", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))

		filter := &db.AuditEventFilter{
			EventTypes: []db.AuditEventType{db.AuditEventApprovalApproved},
		}
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}
		for _, e := range page.Events {
			if e.EventType != db.AuditEventApprovalApproved {
				t.Errorf("expected only approval.approved events, got %s", e.EventType)
			}
		}
	})

	t.Run("FilterByOutcome", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))

		filter := &db.AuditEventFilter{Outcome: "denied"}
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}
		for _, e := range page.Events {
			if e.Outcome != "denied" {
				t.Errorf("expected only denied outcome, got %q", e.Outcome)
			}
		}
	})

	t.Run("FilterByConnectorID", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		github := "github"
		slack := "slack"
		testhelper.InsertAuditEventWithConnector(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), &github)
		testhelper.InsertAuditEventWithConnector(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), &slack)
		testhelper.InsertAuditEventWithConnector(t, tx, uid, agentID, "agent.registered", "registered", testhelper.GenerateID(t, "ar_"), nil)

		filter := &db.AuditEventFilter{ConnectorID: &github}
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}
		if page.Events[0].ConnectorID == nil || *page.Events[0].ConnectorID != "github" {
			t.Errorf("expected connector_id=github, got %v", page.Events[0].ConnectorID)
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Insert 5 events with distinct timestamps for stable ordering
		base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		for i := 0; i < 5; i++ {
			ts := base.Add(time.Duration(i) * time.Minute)
			testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), ts)
		}

		// First page
		page1, err := db.ListAuditEvents(ctx, tx, uid, 2, nil, nil, 0)
		if err != nil {
			t.Fatalf("page 1: %v", err)
		}
		if len(page1.Events) != 2 {
			t.Fatalf("page 1: expected 2 events, got %d", len(page1.Events))
		}
		if !page1.HasMore {
			t.Error("page 1: expected has_more=true")
		}

		// Second page
		last := page1.Events[len(page1.Events)-1]
		cursor := &db.AuditEventCursor{Timestamp: last.Timestamp, ID: last.ID}
		page2, err := db.ListAuditEvents(ctx, tx, uid, 2, cursor, nil, 0)
		if err != nil {
			t.Fatalf("page 2: %v", err)
		}
		if len(page2.Events) != 2 {
			t.Fatalf("page 2: expected 2 events, got %d", len(page2.Events))
		}
		if !page2.HasMore {
			t.Error("page 2: expected has_more=true")
		}

		// Third page - 1 remaining
		last2 := page2.Events[len(page2.Events)-1]
		cursor2 := &db.AuditEventCursor{Timestamp: last2.Timestamp, ID: last2.ID}
		page3, err := db.ListAuditEvents(ctx, tx, uid, 2, cursor2, nil, 0)
		if err != nil {
			t.Fatalf("page 3: %v", err)
		}
		if len(page3.Events) != 1 {
			t.Fatalf("page 3: expected 1 event, got %d", len(page3.Events))
		}
		if page3.HasMore {
			t.Error("page 3: expected has_more=false")
		}
	})

	t.Run("LimitDefaults", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		// limit=0 should default to 20
		page, err := db.ListAuditEvents(ctx, tx, uid, 0, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page == nil {
			t.Fatal("expected non-nil page")
		}
	})

	t.Run("LimitClampedTo100", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		// limit=200 should be clamped to 100
		page, err := db.ListAuditEvents(ctx, tx, uid, 200, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page == nil {
			t.Fatal("expected non-nil page")
		}
	})

	t.Run("UserIsolation", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid1 := testhelper.GenerateUID(t)
		agent1 := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])
		testhelper.InsertAuditEvent(t, tx, uid1, agent1, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))

		uid2 := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

		// uid2 should see no events from uid1
		page, err := db.ListAuditEvents(ctx, tx, uid2, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 0 {
			t.Errorf("expected 0 events for different user, got %d", len(page.Events))
		}
	})

	t.Run("OrderDescending", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		for i := 0; i < 3; i++ {
			ts := base.Add(time.Duration(i) * time.Hour)
			testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), ts)
		}

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i := 1; i < len(page.Events); i++ {
			if page.Events[i].Timestamp.After(page.Events[i-1].Timestamp) {
				t.Errorf("events not in descending order: %v > %v",
					page.Events[i].Timestamp, page.Events[i-1].Timestamp)
			}
		}
	})

	t.Run("DeduplicatesResolvedApproval", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Same source_id for both events — simulates a requested→approved lifecycle.
		sourceID := testhelper.GenerateID(t, "appr_")
		base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.requested", "pending", sourceID, base)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", sourceID, base.Add(time.Minute))

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event (deduped), got %d", len(page.Events))
		}
		if page.Events[0].EventType != db.AuditEventApprovalApproved {
			t.Errorf("expected approval.approved, got %s", page.Events[0].EventType)
		}
	})

	t.Run("ShowsUnresolvedPendingApproval", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Only a requested event, no resolution — should still appear.
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.requested", "pending", testhelper.GenerateID(t, "appr_"))

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 pending event, got %d", len(page.Events))
		}
		if page.Events[0].EventType != db.AuditEventApprovalRequested {
			t.Errorf("expected approval.requested, got %s", page.Events[0].EventType)
		}
	})

	t.Run("DeduplicationSkippedWhenFilteringByRequested", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Same source_id: requested + approved.
		sourceID := testhelper.GenerateID(t, "appr_")
		base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.requested", "pending", sourceID, base)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", sourceID, base.Add(time.Minute))

		// Explicit filter for approval.requested should bypass dedup and return all.
		filter := &db.AuditEventFilter{
			EventTypes: []db.AuditEventType{db.AuditEventApprovalRequested},
		}
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 approval.requested event (dedup bypassed), got %d", len(page.Events))
		}
		if page.Events[0].EventType != db.AuditEventApprovalRequested {
			t.Errorf("expected approval.requested, got %s", page.Events[0].EventType)
		}
	})

	t.Run("DeduplicatesAcrossResolutionTypes", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

		// Approval that was denied.
		denied := testhelper.GenerateID(t, "appr_")
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.requested", "pending", denied, base)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.denied", "denied", denied, base.Add(time.Minute))

		// Approval that was cancelled.
		cancelled := testhelper.GenerateID(t, "appr_")
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.requested", "pending", cancelled, base.Add(2*time.Minute))
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.cancelled", "cancelled", cancelled, base.Add(3*time.Minute))

		// Approval still pending (no resolution).
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.requested", "pending", testhelper.GenerateID(t, "appr_"), base.Add(4*time.Minute))

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should see: pending request + cancelled + denied = 3 (not 5).
		if len(page.Events) != 3 {
			t.Fatalf("expected 3 events (2 resolved + 1 pending), got %d", len(page.Events))
		}

		types := map[db.AuditEventType]int{}
		for _, e := range page.Events {
			types[e.EventType]++
		}
		if types[db.AuditEventApprovalRequested] != 1 {
			t.Errorf("expected 1 approval.requested (unresolved), got %d", types[db.AuditEventApprovalRequested])
		}
		if types[db.AuditEventApprovalDenied] != 1 {
			t.Errorf("expected 1 approval.denied, got %d", types[db.AuditEventApprovalDenied])
		}
		if types[db.AuditEventApprovalCancelled] != 1 {
			t.Errorf("expected 1 approval.cancelled, got %d", types[db.AuditEventApprovalCancelled])
		}
	})

	t.Run("MixedEventTypes", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "agent.registered", "registered", fmt.Sprintf("ar:%d", agentID))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "standing_approval.executed", "auto_executed", testhelper.GenerateID(t, "sa_"))

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		eventTypes := map[db.AuditEventType]int{}
		for _, e := range page.Events {
			eventTypes[e.EventType]++
		}

		if eventTypes[db.AuditEventApprovalApproved] != 1 {
			t.Errorf("expected 1 approval.approved, got %d", eventTypes[db.AuditEventApprovalApproved])
		}
		if eventTypes[db.AuditEventApprovalDenied] != 1 {
			t.Errorf("expected 1 approval.denied, got %d", eventTypes[db.AuditEventApprovalDenied])
		}
		if eventTypes[db.AuditEventAgentRegistered] != 1 {
			t.Errorf("expected 1 agent.registered, got %d", eventTypes[db.AuditEventAgentRegistered])
		}
		if eventTypes[db.AuditEventStandingExecution] != 1 {
			t.Errorf("expected 1 standing_approval.executed, got %d", eventTypes[db.AuditEventStandingExecution])
		}
	})
}

func TestPaymentMethodChargedAuditEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("InsertAndList", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		actionJSON := []byte(`{"type":"expedia.create_booking","payment_method_id":"pm_test123","brand":"visa","last4":"4242","amount_cents":15000,"currency":"usd"}`)
		connectorID := "expedia"

		err := db.InsertAuditEvent(ctx, tx, db.InsertAuditEventParams{
			UserID:      uid,
			AgentID:     agentID,
			EventType:   db.AuditEventPaymentMethodCharged,
			Outcome:     db.OutcomeCharged,
			SourceID:    testhelper.GenerateID(t, "pmtx_"),
			SourceType:  db.SourceTypePaymentMethodTx,
			AgentMeta:   []byte(`{"name":"travel-agent"}`),
			Action:      actionJSON,
			ConnectorID: &connectorID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("list error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}
		e := page.Events[0]
		if e.EventType != db.AuditEventPaymentMethodCharged {
			t.Errorf("expected payment_method.charged, got %s", e.EventType)
		}
		if e.Outcome != db.OutcomeCharged {
			t.Errorf("expected outcome charged, got %q", e.Outcome)
		}
		if e.ConnectorID == nil || *e.ConnectorID != "expedia" {
			t.Errorf("expected connector_id=expedia, got %v", e.ConnectorID)
		}
	})

	t.Run("ActionContainsSafeMetadataOnly", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		actionJSON := []byte(`{"type":"expedia.create_booking","payment_method_id":"pm_abc","brand":"mastercard","last4":"5678","amount_cents":9900,"currency":"usd"}`)
		connectorID := "expedia"

		err := db.InsertAuditEvent(ctx, tx, db.InsertAuditEventParams{
			UserID:      uid,
			AgentID:     agentID,
			EventType:   db.AuditEventPaymentMethodCharged,
			Outcome:     db.OutcomeCharged,
			SourceID:    testhelper.GenerateID(t, "pmtx_"),
			SourceType:  db.SourceTypePaymentMethodTx,
			AgentMeta:   []byte(`{"name":"test"}`),
			Action:      actionJSON,
			ConnectorID: &connectorID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("list error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}

		var action map[string]json.RawMessage
		if err := json.Unmarshal(page.Events[0].Action, &action); err != nil {
			t.Fatalf("failed to parse action JSON: %v", err)
		}

		// Verify safe fields are present.
		requiredFields := []string{"type", "payment_method_id", "brand", "last4", "amount_cents", "currency"}
		for _, field := range requiredFields {
			if _, ok := action[field]; !ok {
				t.Errorf("action missing required field %q", field)
			}
		}

		// Verify no sensitive fields are present.
		sensitiveFields := []string{"card_number", "cvv", "expiry", "full_number"}
		for _, field := range sensitiveFields {
			if _, ok := action[field]; ok {
				t.Errorf("action must not contain sensitive field %q", field)
			}
		}
	})

	t.Run("FilterByEventType", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		connectorID := "expedia"
		// Insert a payment event and an approval event.
		testhelper.InsertAuditEventWithConnector(t, tx, uid, agentID,
			string(db.AuditEventPaymentMethodCharged), db.OutcomeCharged,
			testhelper.GenerateID(t, "pmtx_"), &connectorID)
		testhelper.InsertAuditEvent(t, tx, uid, agentID,
			"approval.approved", "approved",
			testhelper.GenerateID(t, "appr_"))

		filter := &db.AuditEventFilter{
			EventTypes: []db.AuditEventType{db.AuditEventPaymentMethodCharged},
		}
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 payment event, got %d", len(page.Events))
		}
		if page.Events[0].EventType != db.AuditEventPaymentMethodCharged {
			t.Errorf("expected payment_method.charged, got %s", page.Events[0].EventType)
		}
	})

	t.Run("FilterByChargedOutcome", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		connectorID := "expedia"
		testhelper.InsertAuditEventWithConnector(t, tx, uid, agentID,
			string(db.AuditEventPaymentMethodCharged), db.OutcomeCharged,
			testhelper.GenerateID(t, "pmtx_"), &connectorID)
		testhelper.InsertAuditEvent(t, tx, uid, agentID,
			"approval.approved", "approved",
			testhelper.GenerateID(t, "appr_"))

		filter := &db.AuditEventFilter{Outcome: db.OutcomeCharged}
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 charged event, got %d", len(page.Events))
		}
		if page.Events[0].Outcome != db.OutcomeCharged {
			t.Errorf("expected outcome charged, got %q", page.Events[0].Outcome)
		}
	})

	t.Run("ExportIncludesPaymentEvents", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		connectorID := "expedia"
		testhelper.InsertAuditEventWithConnector(t, tx, uid, agentID,
			string(db.AuditEventPaymentMethodCharged), db.OutcomeCharged,
			testhelper.GenerateID(t, "pmtx_"), &connectorID)

		since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		eventTypes := []db.AuditEventType{db.AuditEventPaymentMethodCharged}
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, eventTypes, nil, 100, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 payment event in export, got %d", len(page.Events))
		}
		if page.Events[0].EventType != db.AuditEventPaymentMethodCharged {
			t.Errorf("expected payment_method.charged, got %s", page.Events[0].EventType)
		}
		if page.Events[0].ID == 0 {
			t.Error("export events should include ID")
		}
		if page.Events[0].SourceID == "" {
			t.Error("export events should include source_id")
		}
	})
}

func TestExportAuditLogs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 100, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 0 {
			t.Errorf("expected 0 events, got %d", len(page.Events))
		}
		if page.HasMore {
			t.Error("expected has_more=false")
		}
	})

	t.Run("FiltersBySince", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		old := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		recent := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), old)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"), recent)

		since := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 100, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event after since filter, got %d", len(page.Events))
		}
		if page.Events[0].EventType != db.AuditEventApprovalDenied {
			t.Errorf("expected approval.denied, got %s", page.Events[0].EventType)
		}
	})

	t.Run("ChronologicalOrder", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		for i := 0; i < 5; i++ {
			ts := base.Add(time.Duration(i) * time.Minute)
			testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), ts)
		}

		since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 100, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 5 {
			t.Fatalf("expected 5 events, got %d", len(page.Events))
		}
		for i := 1; i < len(page.Events); i++ {
			if page.Events[i].Timestamp.Before(page.Events[i-1].Timestamp) {
				t.Errorf("events not in chronological order: [%d]=%v before [%d]=%v",
					i, page.Events[i].Timestamp, i-1, page.Events[i-1].Timestamp)
			}
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		for i := 0; i < 5; i++ {
			ts := base.Add(time.Duration(i) * time.Minute)
			testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), ts)
		}

		since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

		// Page 1
		page1, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 2, nil, 0)
		if err != nil {
			t.Fatalf("page 1: %v", err)
		}
		if len(page1.Events) != 2 {
			t.Fatalf("page 1: expected 2, got %d", len(page1.Events))
		}
		if !page1.HasMore {
			t.Error("page 1: expected has_more=true")
		}

		// Page 2
		last := page1.Events[len(page1.Events)-1]
		cursor := &db.AuditLogExportCursor{Timestamp: last.Timestamp, ID: last.ID}
		page2, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 2, cursor, 0)
		if err != nil {
			t.Fatalf("page 2: %v", err)
		}
		if len(page2.Events) != 2 {
			t.Fatalf("page 2: expected 2, got %d", len(page2.Events))
		}
		if !page2.HasMore {
			t.Error("page 2: expected has_more=true")
		}

		// Verify page2 is after page1 using compound (timestamp, id) ordering.
		lastEventPage1 := page1.Events[len(page1.Events)-1]
		firstEventPage2 := page2.Events[0]
		if firstEventPage2.Timestamp.Before(lastEventPage1.Timestamp) ||
			(firstEventPage2.Timestamp.Equal(lastEventPage1.Timestamp) && firstEventPage2.ID <= lastEventPage1.ID) {
			t.Error("page 2 events should be after page 1 events")
		}

		// Page 3
		last2 := page2.Events[len(page2.Events)-1]
		cursor2 := &db.AuditLogExportCursor{Timestamp: last2.Timestamp, ID: last2.ID}
		page3, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 2, cursor2, 0)
		if err != nil {
			t.Fatalf("page 3: %v", err)
		}
		if len(page3.Events) != 1 {
			t.Fatalf("page 3: expected 1, got %d", len(page3.Events))
		}
		if page3.HasMore {
			t.Error("page 3: expected has_more=false")
		}
	})

	t.Run("UserIsolation", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid1 := testhelper.GenerateUID(t)
		agent1 := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])
		testhelper.InsertAuditEvent(t, tx, uid1, agent1, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))

		uid2 := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

		since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		page, err := db.ExportAuditLogs(ctx, tx, uid2, since, nil, nil, nil, 100, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 0 {
			t.Errorf("expected 0 events for different user, got %d", len(page.Events))
		}
	})

	t.Run("LimitDefaults", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 0, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page == nil {
			t.Fatal("expected non-nil page")
		}
	})

	t.Run("UntilFilter", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		jan := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
		feb := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
		mar := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), jan)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"), feb)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.cancelled", "cancelled", testhelper.GenerateID(t, "appr_"), mar)

		since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		until := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, &until, nil, nil, 100, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 2 {
			t.Fatalf("expected 2 events within [jan, mar), got %d", len(page.Events))
		}
	})

	t.Run("EventTypeFilter", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "agent.registered", "registered", fmt.Sprintf("ar:%d", agentID))

		since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		eventTypes := []db.AuditEventType{db.AuditEventApprovalApproved, db.AuditEventApprovalDenied}
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, eventTypes, nil, 100, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 2 {
			t.Fatalf("expected 2 approval events, got %d", len(page.Events))
		}
		for _, e := range page.Events {
			if e.EventType != db.AuditEventApprovalApproved && e.EventType != db.AuditEventApprovalDenied {
				t.Errorf("unexpected event type: %s", e.EventType)
			}
		}
	})

	t.Run("ConnectorIDFilter", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		github := "github"
		slack := "slack"
		testhelper.InsertAuditEventWithConnector(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), &github)
		testhelper.InsertAuditEventWithConnector(t, tx, uid, agentID, "standing_approval.executed", "auto_executed", testhelper.GenerateID(t, "sae_"), &slack)
		testhelper.InsertAuditEventWithConnector(t, tx, uid, agentID, "agent.registered", "registered", fmt.Sprintf("ar:%d", agentID), nil)

		since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, &slack, 100, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}
		if page.Events[0].ConnectorID == nil || *page.Events[0].ConnectorID != "slack" {
			t.Errorf("expected connector_id=slack, got %v", page.Events[0].ConnectorID)
		}
	})
}

func TestInsertAuditEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("BasicInsert", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		err := db.InsertAuditEvent(ctx, tx, db.InsertAuditEventParams{
			UserID:     uid,
			AgentID:    agentID,
			EventType:  db.AuditEventApprovalApproved,
			Outcome:    "approved",
			SourceID:   "test-approval-1",
			SourceType: "approval",
			AgentMeta:  []byte(`{"name":"test agent"}`),
			Action:     []byte(`{"type":"test.action"}`),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify it shows up in list
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("list error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}
		e := page.Events[0]
		if e.EventType != db.AuditEventApprovalApproved {
			t.Errorf("expected approval.approved, got %s", e.EventType)
		}
		if e.Outcome != "approved" {
			t.Errorf("expected outcome approved, got %q", e.Outcome)
		}
		if e.SourceID != "test-approval-1" {
			t.Errorf("expected source_id test-approval-1, got %q", e.SourceID)
		}
		if e.ID == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("StandingApprovalUpdated", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		testhelper.InsertConnector(t, tx, "github")
		saID := testhelper.GenerateID(t, "sa_")
		connectorID := "github"
		actionJSON := []byte(`{"type":"github.create_issue"}`)

		err := db.InsertAuditEvent(ctx, tx, db.InsertAuditEventParams{
			UserID:      uid,
			AgentID:     agentID,
			EventType:   db.AuditEventStandingUpdated,
			Outcome:     db.OutcomeUpdated,
			SourceID:    saID,
			SourceType:  "standing_approval",
			Action:      actionJSON,
			ConnectorID: &connectorID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("list error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}
		e := page.Events[0]
		if e.EventType != db.AuditEventStandingUpdated {
			t.Errorf("expected standing_approval.updated, got %s", e.EventType)
		}
		if e.Outcome != db.OutcomeUpdated {
			t.Errorf("expected outcome updated, got %q", e.Outcome)
		}
		if e.SourceID != saID {
			t.Errorf("expected source_id %q, got %q", saID, e.SourceID)
		}
	})

	t.Run("NilAction", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		err := db.InsertAuditEvent(ctx, tx, db.InsertAuditEventParams{
			UserID:     uid,
			AgentID:    agentID,
			EventType:  db.AuditEventAgentRegistered,
			Outcome:    "registered",
			SourceID:   fmt.Sprintf("ar:%d", agentID),
			SourceType: "agent",
			AgentMeta:  []byte(`{"name":"test"}`),
			Action:     nil,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("list error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}
		if page.Events[0].Action != nil {
			t.Errorf("expected nil action, got %s", string(page.Events[0].Action))
		}
	})

	t.Run("WithConnectorAndExecutionStatus", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Insert a connector so the FK is satisfied.
		testhelper.InsertConnector(t, tx, "github")

		connectorID := "github"
		execStatus := db.ExecStatusSuccess

		err := db.InsertAuditEvent(ctx, tx, db.InsertAuditEventParams{
			UserID:          uid,
			AgentID:         agentID,
			EventType:       db.AuditEventActionExecuted,
			Outcome:         "auto_executed",
			SourceID:        "test-approval-2",
			SourceType:      "approval",
			AgentMeta:       []byte(`{"name":"test"}`),
			Action:          []byte(`{"type":"github.create_issue"}`),
			ConnectorID:     &connectorID,
			ExecutionStatus: &execStatus,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("list error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}
		e := page.Events[0]
		if e.ConnectorID == nil || *e.ConnectorID != "github" {
			t.Errorf("expected connector_id=github, got %v", e.ConnectorID)
		}
		if e.ExecutionStatus == nil || *e.ExecutionStatus != db.ExecStatusSuccess {
			t.Errorf("expected execution_status=success, got %v", e.ExecutionStatus)
		}
		if e.ExecutionError != nil {
			t.Errorf("expected nil execution_error, got %v", e.ExecutionError)
		}
	})

	t.Run("WithExecutionFailure", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.InsertConnector(t, tx, "slack")

		connectorID := "slack"
		execStatus := db.ExecStatusFailure
		execError := "API rate limit exceeded"

		err := db.InsertAuditEvent(ctx, tx, db.InsertAuditEventParams{
			UserID:          uid,
			AgentID:         agentID,
			EventType:       db.AuditEventStandingExecution,
			Outcome:         "auto_executed",
			SourceID:        "test-sa-1",
			SourceType:      "standing_approval",
			AgentMeta:       []byte(`{"name":"test"}`),
			Action:          []byte(`{"type":"slack.send_message"}`),
			ConnectorID:     &connectorID,
			ExecutionStatus: &execStatus,
			ExecutionError:  &execError,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("list error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}
		e := page.Events[0]
		if e.ExecutionStatus == nil || *e.ExecutionStatus != db.ExecStatusFailure {
			t.Errorf("expected execution_status=failure, got %v", e.ExecutionStatus)
		}
		if e.ExecutionError == nil || *e.ExecutionError != "API rate limit exceeded" {
			t.Errorf("expected execution_error='API rate limit exceeded', got %v", e.ExecutionError)
		}
	})
}

func TestListAuditEvents_RetentionFiltering(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ExcludesEventsOutsideRetentionWindow", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Insert an event dated 10 days ago — outside 7-day retention.
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved",
			testhelper.GenerateID(t, "appr_"), time.Now().AddDate(0, 0, -10))

		// Insert a recent event — inside 7-day retention.
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied",
			testhelper.GenerateID(t, "appr_"))

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event within retention window, got %d", len(page.Events))
		}
		if page.Events[0].EventType != db.AuditEventApprovalDenied {
			t.Errorf("expected recent event, got %s", page.Events[0].EventType)
		}
	})

	t.Run("IncludesAllEventsWithinRetentionWindow", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Insert an event dated 30 days ago — inside 90-day retention.
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved",
			testhelper.GenerateID(t, "appr_"), time.Now().AddDate(0, 0, -30))

		// Insert a recent event.
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied",
			testhelper.GenerateID(t, "appr_"))

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 90)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 2 {
			t.Fatalf("expected 2 events within 90-day retention, got %d", len(page.Events))
		}
	})

	t.Run("ZeroRetentionDaysDisablesFiltering", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Insert an event dated 200 days ago.
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved",
			testhelper.GenerateID(t, "appr_"), time.Now().AddDate(0, 0, -200))

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied",
			testhelper.GenerateID(t, "appr_"))

		// retentionDays=0 means no filtering.
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 2 {
			t.Fatalf("expected 2 events with no retention filter, got %d", len(page.Events))
		}
	})
}

func TestExportAuditLogs_RetentionFiltering(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ClampsSinceToRetentionWindow", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Insert event 10 days ago — outside 7-day retention.
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved",
			testhelper.GenerateID(t, "appr_"), time.Now().AddDate(0, 0, -10))

		// Insert recent event — inside 7-day retention.
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied",
			testhelper.GenerateID(t, "appr_"))

		// Even though since=2020, the 7-day retention clamps the effective window.
		since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 100, nil, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event within retention window, got %d", len(page.Events))
		}
	})

	t.Run("ZeroRetentionDaysDisablesFiltering", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Insert event 200 days ago.
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved",
			testhelper.GenerateID(t, "appr_"), time.Now().AddDate(0, 0, -200))

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied",
			testhelper.GenerateID(t, "appr_"))

		since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 100, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Events) != 2 {
			t.Fatalf("expected 2 events with no retention filter, got %d", len(page.Events))
		}
	})
}
