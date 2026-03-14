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

// breakdownUpdateSQL generates the nested jsonb_set expression used to
// atomically increment by_agent, by_connector, and by_action_type counters
// within the breakdown JSONB column. col is the column reference
// (e.g. "usage_periods.breakdown" or just "breakdown"), and agentP,
// connectorP, actionP are the positional parameter numbers for the
// agent key, connector key, and action type key respectively.
func breakdownUpdateSQL(col string, agentP, connectorP, actionP int) string {
	return fmt.Sprintf(`
		jsonb_set(
			jsonb_set(
				jsonb_set(
					COALESCE(%[1]s, '{}'),
					'{by_agent}',
					jsonb_set(
						COALESCE(%[1]s->'by_agent', '{}'),
						ARRAY[$%[2]d::text],
						to_jsonb(COALESCE((%[1]s->'by_agent'->>$%[2]d::text)::int, 0) + 1)
					)
				),
				'{by_connector}',
				CASE WHEN $%[3]d::text = '' THEN COALESCE(%[1]s->'by_connector', '{}')
				ELSE jsonb_set(
					COALESCE(%[1]s->'by_connector', '{}'),
					ARRAY[$%[3]d::text],
					to_jsonb(COALESCE((%[1]s->'by_connector'->>$%[3]d::text)::int, 0) + 1)
				) END
			),
			'{by_action_type}',
			CASE WHEN $%[4]d::text = '' THEN COALESCE(%[1]s->'by_action_type', '{}')
			ELSE jsonb_set(
				COALESCE(%[1]s->'by_action_type', '{}'),
				ARRAY[$%[4]d::text],
				to_jsonb(COALESCE((%[1]s->'by_action_type'->>$%[4]d::text)::int, 0) + 1)
			) END
		)`, col, agentP, connectorP, actionP)
}

// IncrementRequestCountWithBreakdown atomically increments the request count
// and updates the JSONB breakdown for the given user and billing period.
// The breakdown tracks counts by agent, connector, and action type using
// atomic jsonb_set operations.
func IncrementRequestCountWithBreakdown(ctx context.Context, db DBTX, userID string, periodStart, periodEnd time.Time, keys UsageBreakdownKeys) (*UsagePeriod, error) {
	agentKey := strconv.FormatInt(keys.AgentID, 10)

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
		breakdownUpdateSQL("usage_periods.breakdown", 4, 5, 6),
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

	query := fmt.Sprintf(
		`UPDATE usage_periods
		 SET breakdown = %s
		 WHERE user_id = $1 AND period_start = $2`,
		breakdownUpdateSQL("breakdown", 3, 4, 5))

	_, err := d.Exec(ctx, query,
		userID, periodStart, agentKey, keys.ConnectorID, keys.ActionType)
	return err
}

// BillingPeriodBounds returns the start (inclusive) and end (exclusive) of the
// billing month containing the given time. All calculations use UTC.
//
// Free-tier users share calendar-month cycles (1st to 1st) with the full
// monthly allowance, regardless of signup date. This is intentional — see
// docs/adr/010-calendar-month-billing-cycles.md for the rationale.
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

