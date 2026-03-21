package config

import (
	"testing"
)

func TestGetPlan_Free(t *testing.T) {
	p := GetPlan(PlanFree)
	if p == nil {
		t.Fatal("free plan not found")
	}
	if p.ID != "free" {
		t.Errorf("expected ID 'free', got %q", p.ID)
	}
	if p.Name != "Free" {
		t.Errorf("expected Name 'Free', got %q", p.Name)
	}
	if p.MaxRequestsPerMonth == nil || *p.MaxRequestsPerMonth != 1000 {
		t.Errorf("expected MaxRequestsPerMonth=1000, got %v", p.MaxRequestsPerMonth)
	}
	if p.MaxAgents == nil || *p.MaxAgents != 3 {
		t.Errorf("expected MaxAgents=3, got %v", p.MaxAgents)
	}
	if p.AuditRetentionDays != 7 {
		t.Errorf("expected AuditRetentionDays=7, got %d", p.AuditRetentionDays)
	}
}

func TestGetPlan_PayAsYouGo(t *testing.T) {
	p := GetPlan(PlanPayAsYouGo)
	if p == nil {
		t.Fatal("pay_as_you_go plan not found")
	}
	if p.MaxRequestsPerMonth != nil {
		t.Errorf("expected MaxRequestsPerMonth=nil (unlimited), got %v", *p.MaxRequestsPerMonth)
	}
	if p.MaxAgents != nil {
		t.Errorf("expected MaxAgents=nil (unlimited), got %v", *p.MaxAgents)
	}
	if p.AuditRetentionDays != 90 {
		t.Errorf("expected AuditRetentionDays=90, got %d", p.AuditRetentionDays)
	}
	if p.PricePerRequestMillicents != 500 {
		t.Errorf("expected PricePerRequestMillicents=500, got %d", p.PricePerRequestMillicents)
	}
}

func TestGetPlan_NotFound(t *testing.T) {
	p := GetPlan("nonexistent")
	if p != nil {
		t.Errorf("expected nil for nonexistent plan, got %+v", p)
	}
}

func TestAllPlans(t *testing.T) {
	all := AllPlans()
	if len(all) != 3 {
		t.Fatalf("expected 3 plans, got %d", len(all))
	}
	ids := make(map[string]bool)
	for _, p := range all {
		ids[p.ID] = true
	}
	if !ids["free"] || !ids["pay_as_you_go"] || !ids["free_pro"] {
		t.Errorf("expected free, pay_as_you_go, and free_pro plans, got %v", ids)
	}
}

func TestPlanIDsAreAlphanumeric(t *testing.T) {
	for _, p := range AllPlans() {
		if !validPlanID.MatchString(p.ID) {
			t.Errorf("plan ID %q contains invalid characters", p.ID)
		}
	}
}

func TestDefaultPlanID(t *testing.T) {
	if id := DefaultPlanID(true); id != PlanFree {
		t.Errorf("billing enabled: expected %q, got %q", PlanFree, id)
	}
	if id := DefaultPlanID(false); id != PlanPayAsYouGo {
		t.Errorf("billing disabled: expected %q, got %q", PlanPayAsYouGo, id)
	}
}
