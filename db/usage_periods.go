package db

import (
	"context"
	"errors"
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
	CreatedAt    time.Time
}

const usagePeriodColumns = `id, user_id, period_start, period_end, request_count, sms_count, created_at`

func scanUsagePeriod(row pgx.Row) (*UsagePeriod, error) {
	var u UsagePeriod
	err := row.Scan(
		&u.ID,
		&u.UserID,
		&u.PeriodStart,
		&u.PeriodEnd,
		&u.RequestCount,
		&u.SMSCount,
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
