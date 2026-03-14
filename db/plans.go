package db

import (
	"github.com/supersuit-tech/permission-slip-web/config"
)

// Plan ID constants — re-exported from config for backward compatibility.
const (
	PlanFree       = config.PlanFree
	PlanPayAsYouGo = config.PlanPayAsYouGo
)

// DefaultPlanID returns the plan to assign new or unsubscribed users.
// Re-exported from config for backward compatibility.
func DefaultPlanID(billingEnabled bool) string {
	return config.DefaultPlanID(billingEnabled)
}

// Plan represents resource limits and pricing for a subscription tier.
// Limit fields are nil when the plan allows unlimited usage.
//
// Plan definitions live in config/plans.json — no database table needed.
type Plan struct {
	ID                        string
	Name                      string
	MaxRequestsPerMonth       *int // nil = unlimited
	MaxAgents                 *int // nil = unlimited
	MaxStandingApprovals      *int // nil = unlimited (active only)
	MaxCredentials            *int // nil = unlimited
	AuditRetentionDays        int
	PricePerRequestMillicents int // 1 millicent = 1/1000 cent; $0.005 = 500 millicents
}

// PlanFromConfig converts a config.Plan to a db.Plan.
func PlanFromConfig(cp *config.Plan) *Plan {
	if cp == nil {
		return nil
	}
	return &Plan{
		ID:                        cp.ID,
		Name:                      cp.Name,
		MaxRequestsPerMonth:       cp.MaxRequestsPerMonth,
		MaxAgents:                 cp.MaxAgents,
		MaxStandingApprovals:      cp.MaxStandingApprovals,
		MaxCredentials:            cp.MaxCredentials,
		AuditRetentionDays:        cp.AuditRetentionDays,
		PricePerRequestMillicents: cp.PricePerRequestMillicents,
	}
}

// GetPlan returns the plan with the given ID from config, or nil if not found.
// This replaces the old GetPlanByID that queried the database.
func GetPlan(id string) *Plan {
	return PlanFromConfig(config.GetPlan(id))
}
