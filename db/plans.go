package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Plan ID constants match the seeded rows in the plans table.
// These are referenced by subscriptions and used for plan-gated logic.
const (
	PlanFree       = "free"          // 1k req/mo, limited resources, 7-day audit
	PlanPayAsYouGo = "pay_as_you_go" // unlimited, 90-day audit, per-request billing
)

// Plan represents a row from the plans table. Plans define resource limits
// and pricing tiers. Limit fields are nil when the plan allows unlimited usage.
type Plan struct {
	ID                       string
	Name                     string
	MaxRequestsPerMonth      *int // nil = unlimited
	MaxAgents                *int // nil = unlimited
	MaxStandingApprovals     *int // nil = unlimited (active only)
	MaxCredentials           *int // nil = unlimited
	AuditRetentionDays       int
	PricePerRequestMillicents int // 1 millicent = 1/1000 cent; $0.005 = 500 millicents
	CreatedAt                time.Time
}

const planColumns = `id, name, max_requests_per_month, max_agents, max_standing_approvals, max_credentials, audit_retention_days, price_per_request_millicents, created_at`

func scanPlan(row pgx.Row) (*Plan, error) {
	var p Plan
	err := row.Scan(
		&p.ID,
		&p.Name,
		&p.MaxRequestsPerMonth,
		&p.MaxAgents,
		&p.MaxStandingApprovals,
		&p.MaxCredentials,
		&p.AuditRetentionDays,
		&p.PricePerRequestMillicents,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPlanByID returns the plan with the given ID, or nil if not found.
func GetPlanByID(ctx context.Context, db DBTX, id string) (*Plan, error) {
	p, err := scanPlan(db.QueryRow(ctx,
		"SELECT "+planColumns+" FROM plans WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

// ListPlans returns all available plans ordered by creation time.
func ListPlans(ctx context.Context, db DBTX) ([]Plan, error) {
	rows, err := db.Query(ctx,
		"SELECT "+planColumns+" FROM plans ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var p Plan
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.MaxRequestsPerMonth,
			&p.MaxAgents,
			&p.MaxStandingApprovals,
			&p.MaxCredentials,
			&p.AuditRetentionDays,
			&p.PricePerRequestMillicents,
			&p.CreatedAt,
		); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}
