package db_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

func TestGetPlan_Free(t *testing.T) {
	t.Parallel()
	plan := db.GetPlan("free")
	if plan == nil {
		t.Fatal("expected free plan to exist")
	}
	if plan.Name != "Free" {
		t.Errorf("expected name %q, got %q", "Free", plan.Name)
	}
	if plan.MaxRequestsPerMonth == nil || *plan.MaxRequestsPerMonth != 250 {
		t.Errorf("expected max_requests_per_month=250, got %v", plan.MaxRequestsPerMonth)
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

func TestGetPlan_PayAsYouGo(t *testing.T) {
	t.Parallel()
	plan := db.GetPlan("pay_as_you_go")
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

func TestGetPlan_NotFound(t *testing.T) {
	t.Parallel()
	plan := db.GetPlan("nonexistent")
	if plan != nil {
		t.Errorf("expected nil for nonexistent plan, got %+v", plan)
	}
}

func TestDefaultPlanID(t *testing.T) {
	t.Parallel()
	if got := db.DefaultPlanID(false); got != db.PlanPayAsYouGo {
		t.Errorf("DefaultPlanID(false) = %q, want %q", got, db.PlanPayAsYouGo)
	}
	if got := db.DefaultPlanID(true); got != db.PlanFree {
		t.Errorf("DefaultPlanID(true) = %q, want %q", got, db.PlanFree)
	}
}
