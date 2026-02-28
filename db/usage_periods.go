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
