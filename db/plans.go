package db

// Well-known plan IDs (embedded in plans_embedded.json).
const (
	PlanFree       = "free"
	PlanPayAsYouGo = "pay_as_you_go"
	PlanFreePro    = "free_pro"
)

// DefaultPlanID returns the plan assigned to new or unsubscribed users on
// self-hosted deployments (unlimited).
func DefaultPlanID() string {
	return PlanPayAsYouGo
}

// Plan represents resource limits and pricing for a subscription tier.
// Limit fields are nil when the plan allows unlimited usage.
//
// Plan definitions live in plans_embedded.json — no database table needed.
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
