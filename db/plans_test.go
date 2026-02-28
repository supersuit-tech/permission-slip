package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestPlansSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "plans", []string{
		"id", "name", "max_requests_per_month", "max_agents",
		"max_standing_approvals", "max_credentials",
		"audit_retention_days", "price_per_request_millicents", "created_at",
	})
}

func TestGetPlanByID_Free(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	plan, err := db.GetPlanByID(ctx, tx, "free")
	if err != nil {
		t.Fatalf("GetPlanByID: %v", err)
	}
	if plan == nil {
		t.Fatal("expected free plan to exist")
	}
	if plan.Name != "Free" {
		t.Errorf("expected name %q, got %q", "Free", plan.Name)
	}
	if plan.MaxRequestsPerMonth == nil || *plan.MaxRequestsPerMonth != 1000 {
		t.Errorf("expected max_requests_per_month=1000, got %v", plan.MaxRequestsPerMonth)
	}
	if plan.MaxAgents == nil || *plan.MaxAgents != 3 {
		t.Errorf("expected max_agents=3, got %v", plan.MaxAgents)
	}
	if plan.MaxStandingApprovals == nil || *plan.MaxStandingApprovals != 5 {
		t.Errorf("expected max_standing_approvals=5, got %v", plan.MaxStandingApprovals)
	}
	if plan.MaxCredentials == nil || *plan.MaxCredentials != 5 {
		t.Errorf("expected max_credentials=5, got %v", plan.MaxCredentials)
	}
	if plan.AuditRetentionDays != 7 {
		t.Errorf("expected audit_retention_days=7, got %d", plan.AuditRetentionDays)
	}
	if plan.PricePerRequestMillicents != 0 {
		t.Errorf("expected price=0, got %d", plan.PricePerRequestMillicents)
	}
}

func TestGetPlanByID_PayAsYouGo(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	plan, err := db.GetPlanByID(ctx, tx, "pay_as_you_go")
	if err != nil {
		t.Fatalf("GetPlanByID: %v", err)
	}
	if plan == nil {
		t.Fatal("expected pay_as_you_go plan to exist")
	}
	if plan.Name != "Pay As You Go" {
		t.Errorf("expected name %q, got %q", "Pay As You Go", plan.Name)
	}
	if plan.MaxRequestsPerMonth != nil {
		t.Errorf("expected max_requests_per_month=nil (unlimited), got %v", *plan.MaxRequestsPerMonth)
	}
	if plan.MaxAgents != nil {
		t.Errorf("expected max_agents=nil (unlimited), got %v", *plan.MaxAgents)
	}
	if plan.MaxStandingApprovals != nil {
		t.Errorf("expected max_standing_approvals=nil (unlimited), got %v", *plan.MaxStandingApprovals)
	}
	if plan.MaxCredentials != nil {
		t.Errorf("expected max_credentials=nil (unlimited), got %v", *plan.MaxCredentials)
	}
	if plan.AuditRetentionDays != 90 {
		t.Errorf("expected audit_retention_days=90, got %d", plan.AuditRetentionDays)
	}
	if plan.PricePerRequestMillicents != 500 {
		t.Errorf("expected price=500 millicents, got %d", plan.PricePerRequestMillicents)
	}
}

func TestGetPlanByID_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	plan, err := db.GetPlanByID(ctx, tx, "nonexistent")
	if err != nil {
		t.Fatalf("GetPlanByID: %v", err)
	}
	if plan != nil {
		t.Errorf("expected nil for nonexistent plan, got %+v", plan)
	}
}

func TestListPlans(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	plans, err := db.ListPlans(ctx, tx)
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(plans) < 2 {
		t.Fatalf("expected at least 2 plans, got %d", len(plans))
	}

	ids := make(map[string]bool)
	for _, p := range plans {
		ids[p.ID] = true
	}
	if !ids["free"] {
		t.Error("expected free plan in list")
	}
	if !ids["pay_as_you_go"] {
		t.Error("expected pay_as_you_go plan in list")
	}
}
