// Package config provides application configuration loaded from embedded files.
// Plan definitions live in plans.json — the single source of truth for all
// plan limits and pricing. No database table needed.
package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
)

//go:embed plans.json
var plansFS embed.FS

// Plan defines resource limits and pricing for a subscription tier.
// Limit fields are nil when the plan allows unlimited usage.
type Plan struct {
	ID                        string `json:"-"`
	Name                      string `json:"name"`
	MaxRequestsPerMonth       *int   `json:"max_requests_per_month"`
	MaxAgents                 *int   `json:"max_agents"`
	MaxStandingApprovals      *int   `json:"max_standing_approvals"`
	MaxCredentials            *int   `json:"max_credentials"`
	AuditRetentionDays        int    `json:"audit_retention_days"`
	PricePerRequestMillicents int    `json:"price_per_request_millicents"`
}

// Well-known plan IDs.
const (
	PlanFree       = "free"
	PlanPayAsYouGo = "pay_as_you_go"
)

// DefaultPlanID returns the plan to assign new or unsubscribed users.
// When billing is enabled, users start on the free plan. When disabled
// (self-hosted), users get the unlimited pay_as_you_go plan so that
// enforcement code sees no limits without needing billing-specific guards.
func DefaultPlanID(billingEnabled bool) string {
	if billingEnabled {
		return PlanFree
	}
	return PlanPayAsYouGo
}

// validPlanID restricts plan IDs to alphanumeric characters and underscores.
// This is validated at load time to prevent SQL injection when plan IDs are
// interpolated into CASE expressions (e.g. in data_retention.go).
var validPlanID = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

var (
	plans    map[string]*Plan
	planList []*Plan
	loadOnce sync.Once
	loadErr  error
)

func ensureLoaded() error {
	loadOnce.Do(func() {
		data, err := plansFS.ReadFile("plans.json")
		if err != nil {
			loadErr = fmt.Errorf("read embedded plans.json: %w", err)
			return
		}
		var raw map[string]*Plan
		if err := json.Unmarshal(data, &raw); err != nil {
			loadErr = fmt.Errorf("parse plans.json: %w", err)
			return
		}
		plans = make(map[string]*Plan, len(raw))
		planList = make([]*Plan, 0, len(raw))
		for id, p := range raw {
			if !validPlanID.MatchString(id) {
				loadErr = fmt.Errorf("invalid plan ID %q: must be alphanumeric/underscore", id)
				return
			}
			p.ID = id
			plans[id] = p
			planList = append(planList, p)
		}
	})
	return loadErr
}

// GetPlan returns the plan with the given ID, or nil if not found.
func GetPlan(id string) *Plan {
	if err := ensureLoaded(); err != nil {
		return nil
	}
	return plans[id]
}

// MustGetPlan returns the plan with the given ID, panicking if not found.
// Use only during startup or in contexts where a missing plan is a fatal error.
func MustGetPlan(id string) *Plan {
	p := GetPlan(id)
	if p == nil {
		panic(fmt.Sprintf("plan %q not found in config/plans.json", id))
	}
	return p
}

// AllPlans returns all configured plans.
func AllPlans() []*Plan {
	if err := ensureLoaded(); err != nil {
		return nil
	}
	out := make([]*Plan, len(planList))
	copy(out, planList)
	return out
}
