package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestGetUsageByConnectorForUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u1_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:8])

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())

	// uid1: 3 via gmail, 2 via stripe
	for i := 0; i < 3; i++ {
		if _, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid1, periodStart, periodEnd, db.UsageBreakdownKeys{
			AgentID: 1, ConnectorID: "gmail", ActionType: "email.send",
		}); err != nil {
			t.Fatalf("increment uid1 gmail: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid1, periodStart, periodEnd, db.UsageBreakdownKeys{
			AgentID: 1, ConnectorID: "stripe", ActionType: "payment.create",
		}); err != nil {
			t.Fatalf("increment uid1 stripe: %v", err)
		}
	}

	// uid2: 5 via gmail — should NOT appear in uid1's results.
	for i := 0; i < 5; i++ {
		if _, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid2, periodStart, periodEnd, db.UsageBreakdownKeys{
			AgentID: 2, ConnectorID: "gmail", ActionType: "email.send",
		}); err != nil {
			t.Fatalf("increment uid2 gmail: %v", err)
		}
	}

	// Query uid1's connector usage only.
	results, err := db.GetUsageByConnectorForUser(ctx, tx, uid1, periodStart)
	if err != nil {
		t.Fatalf("GetUsageByConnectorForUser: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 connectors for uid1, got %d", len(results))
	}
	// gmail should be first (3 for uid1 only), stripe second (2)
	if results[0].ConnectorID != "gmail" {
		t.Errorf("expected first connector to be gmail, got %s", results[0].ConnectorID)
	}
	if results[0].RequestCount != 3 {
		t.Errorf("expected gmail count=3 (uid1 only), got %d", results[0].RequestCount)
	}
	if results[1].ConnectorID != "stripe" {
		t.Errorf("expected second connector to be stripe, got %s", results[1].ConnectorID)
	}
	if results[1].RequestCount != 2 {
		t.Errorf("expected stripe count=2, got %d", results[1].RequestCount)
	}
}

func TestGetUsageByAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())

	// Agent 1: 7 requests, Agent 2: 3 requests
	for i := 0; i < 7; i++ {
		if _, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid, periodStart, periodEnd, db.UsageBreakdownKeys{
			AgentID: 1, ConnectorID: "gmail", ActionType: "email.send",
		}); err != nil {
			t.Fatalf("increment agent 1: %v", err)
		}
	}
	for i := 0; i < 3; i++ {
		if _, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid, periodStart, periodEnd, db.UsageBreakdownKeys{
			AgentID: 2, ConnectorID: "stripe", ActionType: "payment.create",
		}); err != nil {
			t.Fatalf("increment agent 2: %v", err)
		}
	}

	results, err := db.GetUsageByAgent(ctx, tx, uid, periodStart)
	if err != nil {
		t.Fatalf("GetUsageByAgent: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(results))
	}
	// Agent 1 should be first (7), agent 2 second (3)
	if results[0].AgentID != "1" {
		t.Errorf("expected first agent to be 1, got %s", results[0].AgentID)
	}
	if results[0].RequestCount != 7 {
		t.Errorf("expected agent 1 count=7, got %d", results[0].RequestCount)
	}
	if results[1].AgentID != "2" {
		t.Errorf("expected second agent to be 2, got %s", results[1].AgentID)
	}
	if results[1].RequestCount != 3 {
		t.Errorf("expected agent 2 count=3, got %d", results[1].RequestCount)
	}
}

func TestGetUsageByAgent_NoData(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	periodStart, _ := db.BillingPeriodBounds(time.Now())

	results, err := db.GetUsageByAgent(ctx, tx, uid, periodStart)
	if err != nil {
		t.Fatalf("GetUsageByAgent: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
