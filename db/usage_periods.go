package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// UsagePeriod represents a row from the usage_periods table.
// Each row tracks billable usage for one user within a calendar-month billing
// period. Counters are incremented atomically via upsert to ensure correctness
// under concurrent access.
//
// Period boundaries use a half-open interval: [PeriodStart, PeriodEnd).
// PeriodStart is the first instant of the month (inclusive), PeriodEnd is the
// first instant of the next month (exclusive). The DB enforces PeriodEnd > PeriodStart.
type UsagePeriod struct {
	ID           int64
	UserID       string
	PeriodStart  time.Time // inclusive: first instant of billing month
	PeriodEnd    time.Time // exclusive: first instant of the next month
	RequestCount int
	SMSCount     int
	Breakdown    []byte // raw JSONB: { "by_agent": {...}, "by_connector": {...}, "by_action_type": {...} }
	CreatedAt    time.Time
}

const usagePeriodColumns = `id, user_id, period_start, period_end, request_count, sms_count, breakdown, created_at`

func scanUsagePeriod(row pgx.Row) (*UsagePeriod, error) {
	var u UsagePeriod
	err := row.Scan(
		&u.ID,
		&u.UserID,
		&u.PeriodStart,
		&u.PeriodEnd,
		&u.RequestCount,
		&u.SMSCount,
		&u.Breakdown,
		&u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetLatestUsage returns the most recent usage record for the user, or nil if
// no record exists. Callers should check period boundaries to determine if the
// returned record belongs to the current billing period.
func GetLatestUsage(ctx context.Context, db DBTX, userID string) (*UsagePeriod, error) {
	u, err := scanUsagePeriod(db.QueryRow(ctx,
		`SELECT `+usagePeriodColumns+`
		 FROM usage_periods
		 WHERE user_id = $1
		 ORDER BY period_start DESC
		 LIMIT 1`,
		userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

// IncrementRequestCount atomically increments the request count for the
// given user and billing period. If no row exists for the period, one is
// created via upsert (INSERT ... ON CONFLICT DO UPDATE).
//
// periodStart and periodEnd define the billing period boundaries.
func IncrementRequestCount(ctx context.Context, db DBTX, userID string, periodStart, periodEnd time.Time) (*UsagePeriod, error) {
	return scanUsagePeriod(db.QueryRow(ctx,
		`INSERT INTO usage_periods (user_id, period_start, period_end, request_count)
		 VALUES ($1, $2, $3, 1)
		 ON CONFLICT (user_id, period_start)
		 DO UPDATE SET request_count = usage_periods.request_count + 1,
		              period_end = EXCLUDED.period_end
		 RETURNING `+usagePeriodColumns,
		userID, periodStart, periodEnd))
}

// UsageBreakdownKeys holds the keys used to update the JSONB breakdown
// when incrementing the request count.
type UsageBreakdownKeys struct {
	AgentID     int64
	ConnectorID string // may be empty if unknown
	ActionType  string // may be empty if unknown
}

// IncrementRequestCountWithBreakdown atomically increments the request count
// and updates the JSONB breakdown for the given user and billing period.
// The breakdown tracks counts by agent, connector, and action type using
// atomic jsonb_set operations.
func IncrementRequestCountWithBreakdown(ctx context.Context, db DBTX, userID string, periodStart, periodEnd time.Time, keys UsageBreakdownKeys) (*UsagePeriod, error) {
	agentKey := strconv.FormatInt(keys.AgentID, 10)

	// Build the breakdown update expression. We use nested jsonb_set calls
	// to atomically increment counters within the JSONB breakdown column.
	// Each path (by_agent.<id>, by_connector.<id>, by_action_type.<type>)
	// is incremented by 1, initialising to 1 if the key doesn't exist.
	breakdownUpdate := `
		jsonb_set(
			jsonb_set(
				jsonb_set(
					COALESCE(usage_periods.breakdown, '{}'),
					'{by_agent}',
					jsonb_set(
						COALESCE(usage_periods.breakdown->'by_agent', '{}'),
						ARRAY[$4::text],
						to_jsonb(COALESCE((usage_periods.breakdown->'by_agent'->>$4::text)::int, 0) + 1)
					)
				),
				'{by_connector}',
				CASE WHEN $5::text = '' THEN COALESCE(usage_periods.breakdown->'by_connector', '{}')
				ELSE jsonb_set(
					COALESCE(usage_periods.breakdown->'by_connector', '{}'),
					ARRAY[$5::text],
					to_jsonb(COALESCE((usage_periods.breakdown->'by_connector'->>$5::text)::int, 0) + 1)
				) END
			),
			'{by_action_type}',
			CASE WHEN $6::text = '' THEN COALESCE(usage_periods.breakdown->'by_action_type', '{}')
			ELSE jsonb_set(
				COALESCE(usage_periods.breakdown->'by_action_type', '{}'),
				ARRAY[$6::text],
				to_jsonb(COALESCE((usage_periods.breakdown->'by_action_type'->>$6::text)::int, 0) + 1)
			) END
		)`

	query := fmt.Sprintf(
		`INSERT INTO usage_periods (user_id, period_start, period_end, request_count, breakdown)
		 VALUES ($1, $2, $3, 1, jsonb_build_object(
			'by_agent', jsonb_build_object($4::text, 1),
			'by_connector', CASE WHEN $5::text = '' THEN '{}'::jsonb ELSE jsonb_build_object($5::text, 1) END,
			'by_action_type', CASE WHEN $6::text = '' THEN '{}'::jsonb ELSE jsonb_build_object($6::text, 1) END
		 ))
		 ON CONFLICT (user_id, period_start)
		 DO UPDATE SET request_count = usage_periods.request_count + 1,
		              period_end = EXCLUDED.period_end,
		              breakdown = %s
		 RETURNING %s`,
		breakdownUpdate,
		usagePeriodColumns,
	)

	return scanUsagePeriod(db.QueryRow(ctx, query,
		userID, periodStart, periodEnd,
		agentKey, keys.ConnectorID, keys.ActionType,
	))
}

// IncrementSMSCount atomically increments the SMS count for the given user
// and billing period. If no row exists for the period, one is created via upsert.
func IncrementSMSCount(ctx context.Context, db DBTX, userID string, periodStart, periodEnd time.Time) (*UsagePeriod, error) {
	return scanUsagePeriod(db.QueryRow(ctx,
		`INSERT INTO usage_periods (user_id, period_start, period_end, sms_count)
		 VALUES ($1, $2, $3, 1)
		 ON CONFLICT (user_id, period_start)
		 DO UPDATE SET sms_count = usage_periods.sms_count + 1,
		              period_end = EXCLUDED.period_end
		 RETURNING `+usagePeriodColumns,
		userID, periodStart, periodEnd))
}

// GetUsageByPeriod returns the usage record for a specific user and period
// start time, or nil if not found.
func GetUsageByPeriod(ctx context.Context, db DBTX, userID string, periodStart time.Time) (*UsagePeriod, error) {
	u, err := scanUsagePeriod(db.QueryRow(ctx,
		`SELECT `+usagePeriodColumns+`
		 FROM usage_periods
		 WHERE user_id = $1 AND period_start = $2`,
		userID, periodStart))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

// UsageBreakdown is the typed representation of the usage_periods.breakdown JSONB
// column. Each map key is an identifier (agent ID, connector ID, or action type)
// and the value is the number of billable requests attributed to that key.
type UsageBreakdown struct {
	ByAgent      map[string]int `json:"by_agent"`
	ByConnector  map[string]int `json:"by_connector"`
	ByActionType map[string]int `json:"by_action_type"`
}

// ParseBreakdown unmarshals the raw JSONB breakdown into a typed struct.
// Returns a zero-value UsageBreakdown (empty maps) if the raw bytes are empty
// or cannot be parsed.
func (u *UsagePeriod) ParseBreakdown() UsageBreakdown {
	var b UsageBreakdown
	if len(u.Breakdown) > 0 {
		_ = json.Unmarshal(u.Breakdown, &b)
	}
	if b.ByAgent == nil {
		b.ByAgent = map[string]int{}
	}
	if b.ByConnector == nil {
		b.ByConnector = map[string]int{}
	}
	if b.ByActionType == nil {
		b.ByActionType = map[string]int{}
	}
	return b
}

// GetCurrentPeriodUsage returns the usage record for the current billing month
// (based on UTC now), or nil if no usage has been recorded yet this month.
func GetCurrentPeriodUsage(ctx context.Context, db DBTX, userID string) (*UsagePeriod, error) {
	periodStart, _ := BillingPeriodBounds(time.Now())
	return GetUsageByPeriod(ctx, db, userID, periodStart)
}

// ReserveRequestQuota atomically increments request_count only if the current
// count is below the given limit. Returns (true, nil) if the reservation
// succeeded, (false, nil) if the quota is exhausted, or (false, err) on
// database errors. This prevents TOCTOU races where concurrent requests could
// each pass a read-then-check gate before any increment is recorded.
func ReserveRequestQuota(ctx context.Context, d DBTX, userID string, periodStart, periodEnd time.Time, limit int) (bool, error) {
	// Use an upsert with a WHERE clause on DO UPDATE that only increments
	// if count < limit. If the row is inserted (new period), count starts
	// at 1 which is always under any limit >= 1. If the row exists but
	// count >= limit, the WHERE prevents the update and 0 rows are returned.
	var id int64
	err := d.QueryRow(ctx,
		`INSERT INTO usage_periods (user_id, period_start, period_end, request_count)
		 VALUES ($1, $2, $3, 1)
		 ON CONFLICT (user_id, period_start)
		 DO UPDATE SET request_count = usage_periods.request_count + 1,
		              period_end = EXCLUDED.period_end
		 WHERE usage_periods.request_count < $4
		 RETURNING id`,
		userID, periodStart, periodEnd, limit).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // quota exhausted
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// UpdateUsageBreakdownOnly updates only the JSONB breakdown for an existing
// usage period row without incrementing request_count. Used when the count
// was already reserved atomically by ReserveRequestQuota.
func UpdateUsageBreakdownOnly(ctx context.Context, d DBTX, userID string, periodStart time.Time, keys UsageBreakdownKeys) error {
	agentKey := strconv.FormatInt(keys.AgentID, 10)

	_, err := d.Exec(ctx,
		`UPDATE usage_periods
		 SET breakdown = jsonb_set(
			jsonb_set(
				jsonb_set(
					COALESCE(breakdown, '{}'),
					'{by_agent}',
					jsonb_set(
						COALESCE(breakdown->'by_agent', '{}'),
						ARRAY[$3::text],
						to_jsonb(COALESCE((breakdown->'by_agent'->>$3::text)::int, 0) + 1)
					)
				),
				'{by_connector}',
				CASE WHEN $4::text = '' THEN COALESCE(breakdown->'by_connector', '{}')
				ELSE jsonb_set(
					COALESCE(breakdown->'by_connector', '{}'),
					ARRAY[$4::text],
					to_jsonb(COALESCE((breakdown->'by_connector'->>$4::text)::int, 0) + 1)
				) END
			),
			'{by_action_type}',
			CASE WHEN $5::text = '' THEN COALESCE(breakdown->'by_action_type', '{}')
			ELSE jsonb_set(
				COALESCE(breakdown->'by_action_type', '{}'),
				ARRAY[$5::text],
				to_jsonb(COALESCE((breakdown->'by_action_type'->>$5::text)::int, 0) + 1)
			) END
		 )
		 WHERE user_id = $1 AND period_start = $2`,
		userID, periodStart, agentKey, keys.ConnectorID, keys.ActionType)
	return err
}

// BillingPeriodBounds returns the start (inclusive) and end (exclusive) of the
// billing month containing the given time. All calculations use UTC.
//
// Example:
//
//	BillingPeriodBounds(2026-02-15T14:30:00Z)
//	→ (2026-02-01T00:00:00Z, 2026-03-01T00:00:00Z)
func BillingPeriodBounds(t time.Time) (start, end time.Time) {
	t = t.UTC()
	start = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, 0)
	return start, end
}

// ── Admin / analytics queries ───────────────────────────────────────────────

// TopUserUsage represents a row from the top-users-by-usage query.
type TopUserUsage struct {
	UserID       string
	RequestCount int
	SMSCount     int
	PeriodStart  time.Time
	PeriodEnd    time.Time
}

// GetTopUsersByUsage returns users with the highest request counts for a given
// billing period, ordered by request_count DESC. limit caps the number of rows
// returned (clamped to 1–100).
func GetTopUsersByUsage(ctx context.Context, db DBTX, periodStart time.Time, limit int) ([]TopUserUsage, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := db.Query(ctx,
		`SELECT user_id, request_count, sms_count, period_start, period_end
		 FROM usage_periods
		 WHERE period_start = $1
		 ORDER BY request_count DESC
		 LIMIT $2`,
		periodStart, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TopUserUsage
	for rows.Next() {
		var u TopUserUsage
		if err := rows.Scan(&u.UserID, &u.RequestCount, &u.SMSCount, &u.PeriodStart, &u.PeriodEnd); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// ConnectorUsage represents aggregate request counts for a single connector
// across all users in a billing period.
type ConnectorUsage struct {
	ConnectorID  string
	RequestCount int
}

// GetUsageByConnector aggregates request counts from the JSONB breakdown
// across all users for a given billing period. Returns connectors ordered
// by request count DESC.
func GetUsageByConnector(ctx context.Context, db DBTX, periodStart time.Time) ([]ConnectorUsage, error) {
	rows, err := db.Query(ctx,
		`SELECT key, SUM(value::int)::int AS total
		 FROM usage_periods,
		      jsonb_each_text(COALESCE(breakdown->'by_connector', '{}'))
		 WHERE period_start = $1
		 GROUP BY key
		 ORDER BY total DESC`,
		periodStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ConnectorUsage
	for rows.Next() {
		var c ConnectorUsage
		if err := rows.Scan(&c.ConnectorID, &c.RequestCount); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// AgentUsage represents request counts for a single agent within a user's
// billing period, extracted from the JSONB breakdown.
type AgentUsage struct {
	AgentID      string
	RequestCount int
}

// GetUsageByAgent extracts per-agent request counts from the JSONB breakdown
// for a specific user and billing period. Returns agents ordered by request
// count DESC.
func GetUsageByAgent(ctx context.Context, db DBTX, userID string, periodStart time.Time) ([]AgentUsage, error) {
	rows, err := db.Query(ctx,
		`SELECT key, value::int AS count
		 FROM usage_periods,
		      jsonb_each_text(COALESCE(breakdown->'by_agent', '{}'))
		 WHERE user_id = $1 AND period_start = $2
		 ORDER BY count DESC`,
		userID, periodStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AgentUsage
	for rows.Next() {
		var a AgentUsage
		if err := rows.Scan(&a.AgentID, &a.RequestCount); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
