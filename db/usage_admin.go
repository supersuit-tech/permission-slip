package db

import (
	"context"
	"time"
)

// ── Admin / analytics queries ───────────────────────────────────────────────

// ConnectorUsage represents per-connector request counts for a single user
// in a billing period, enriched with the connector name.
type ConnectorUsage struct {
	ConnectorID  string
	Name         string // from connectors table; empty if connector was deleted
	RequestCount int
}

// GetUsageByConnectorForUser extracts per-connector request counts from the
// JSONB breakdown for a specific user and billing period. Returns connectors
// ordered by request count DESC, enriched with connector names.
func GetUsageByConnectorForUser(ctx context.Context, db DBTX, userID string, periodStart time.Time) ([]ConnectorUsage, error) {
	rows, err := db.Query(ctx,
		`SELECT b.key, COALESCE(c.name, ''), b.value::int AS count
		 FROM usage_periods u,
		      jsonb_each_text(COALESCE(u.breakdown->'by_connector', '{}')) b
		 LEFT JOIN connectors c ON c.id = b.key
		 WHERE u.user_id = $1 AND u.period_start = $2
		 ORDER BY count DESC`,
		userID, periodStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ConnectorUsage
	for rows.Next() {
		var c ConnectorUsage
		if err := rows.Scan(&c.ConnectorID, &c.Name, &c.RequestCount); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// AgentUsage represents request counts for a single agent within a user's
// billing period, extracted from the JSONB breakdown and enriched with agent name.
type AgentUsage struct {
	AgentID      string
	Name         string // from agents.metadata->>'agent_name'; empty if not set
	RequestCount int
}

// GetUsageByAgent extracts per-agent request counts from the JSONB breakdown
// for a specific user and billing period. Returns agents ordered by request
// count DESC, enriched with agent names from metadata.
//
// The JOIN to agents is scoped to agents owned by the same user
// (approver_id = user_id) to prevent leaking agent names across users.
func GetUsageByAgent(ctx context.Context, db DBTX, userID string, periodStart time.Time) ([]AgentUsage, error) {
	rows, err := db.Query(ctx,
		`SELECT b.key,
		        COALESCE(a.metadata->>'agent_name', ''),
		        b.value::int AS count
		 FROM usage_periods u,
		      jsonb_each_text(COALESCE(u.breakdown->'by_agent', '{}')) b
		 LEFT JOIN agents a
		   ON a.agent_id = b.key::bigint
		   AND a.approver_id = $1
		 WHERE u.user_id = $1 AND u.period_start = $2
		 ORDER BY count DESC`,
		userID, periodStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AgentUsage
	for rows.Next() {
		var a AgentUsage
		if err := rows.Scan(&a.AgentID, &a.Name, &a.RequestCount); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
