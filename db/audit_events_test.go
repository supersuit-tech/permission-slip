package db_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestListAuditEvents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter)
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
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter)
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
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter)
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
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, filter)
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
		page1, err := db.ListAuditEvents(ctx, tx, uid, 2, nil, nil)
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
		page2, err := db.ListAuditEvents(ctx, tx, uid, 2, cursor, nil)
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
		page3, err := db.ListAuditEvents(ctx, tx, uid, 2, cursor2, nil)
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
		page, err := db.ListAuditEvents(ctx, tx, uid, 0, nil, nil)
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
		page, err := db.ListAuditEvents(ctx, tx, uid, 200, nil, nil)
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
		page, err := db.ListAuditEvents(ctx, tx, uid2, 20, nil, nil)
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

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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

	t.Run("MixedEventTypes", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "agent.registered", "registered", fmt.Sprintf("ar:%d", agentID))
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "standing_approval.executed", "auto_executed", testhelper.GenerateID(t, "sa_"))

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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

func TestExportAuditLogs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 100, nil)
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
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 100, nil)
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
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 100, nil)
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
		page1, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 2, nil)
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
		page2, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 2, cursor)
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
		page3, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 2, cursor2)
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
		page, err := db.ExportAuditLogs(ctx, tx, uid2, since, nil, nil, nil, 100, nil)
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
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, nil, 0, nil)
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
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, &until, nil, nil, 100, nil)
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
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, eventTypes, nil, 100, nil)
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
		page, err := db.ExportAuditLogs(ctx, tx, uid, since, nil, nil, &slack, 100, nil)
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
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
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
