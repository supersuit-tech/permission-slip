package db_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestUsagePeriodsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "usage_periods", []string{
		"id", "user_id", "period_start", "period_end",
		"request_count", "sms_count", "breakdown", "created_at",
	})
}

func TestUsagePeriodsUniqueConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	testhelper.RequireUniqueViolation(t, tx,
		"one usage_period per user per period",
		func() error {
			_, err := tx.Exec(ctx,
				`INSERT INTO usage_periods (user_id, period_start, period_end) VALUES ($1, $2, $3)`,
				uid, periodStart, periodEnd)
			return err
		},
		func() error {
			_, err := tx.Exec(ctx,
				`INSERT INTO usage_periods (user_id, period_start, period_end) VALUES ($1, $2, $3)`,
				uid, periodStart, periodEnd)
			return err
		},
	)
}

func TestUsagePeriodsCascadeDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	_, err := tx.Exec(ctx,
		`INSERT INTO usage_periods (user_id, period_start, period_end) VALUES ($1, $2, $3)`,
		uid, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("insert usage_period: %v", err)
	}

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM auth.users WHERE id = '"+uid+"'",
		[]string{"usage_periods"},
		"user_id = '"+uid+"'",
	)
}

func TestUsagePeriodsValidRangeCheck(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	// period_end must be after period_start
	err := testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(ctx,
			`INSERT INTO usage_periods (user_id, period_start, period_end) VALUES ($1, $2, $3)`,
			uid,
			time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), // end before start
		)
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for period_end < period_start")
	}

	// equal should also fail
	same := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	err = testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(ctx,
			`INSERT INTO usage_periods (user_id, period_start, period_end) VALUES ($1, $2, $3)`,
			uid, same, same,
		)
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for period_end == period_start")
	}
}

func TestIncrementRequestCount(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// First increment creates the row
	usage, err := db.IncrementRequestCount(ctx, tx, uid, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("IncrementRequestCount: %v", err)
	}
	if usage.RequestCount != 1 {
		t.Errorf("expected request_count=1, got %d", usage.RequestCount)
	}
	if usage.SMSCount != 0 {
		t.Errorf("expected sms_count=0, got %d", usage.SMSCount)
	}

	// Second increment updates the existing row
	usage, err = db.IncrementRequestCount(ctx, tx, uid, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("IncrementRequestCount: %v", err)
	}
	if usage.RequestCount != 2 {
		t.Errorf("expected request_count=2, got %d", usage.RequestCount)
	}

	// Third increment
	usage, err = db.IncrementRequestCount(ctx, tx, uid, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("IncrementRequestCount: %v", err)
	}
	if usage.RequestCount != 3 {
		t.Errorf("expected request_count=3, got %d", usage.RequestCount)
	}
}

func TestIncrementSMSCount(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	usage, err := db.IncrementSMSCount(ctx, tx, uid, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("IncrementSMSCount: %v", err)
	}
	if usage.SMSCount != 1 {
		t.Errorf("expected sms_count=1, got %d", usage.SMSCount)
	}
	if usage.RequestCount != 0 {
		t.Errorf("expected request_count=0, got %d", usage.RequestCount)
	}

	usage, err = db.IncrementSMSCount(ctx, tx, uid, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("IncrementSMSCount: %v", err)
	}
	if usage.SMSCount != 2 {
		t.Errorf("expected sms_count=2, got %d", usage.SMSCount)
	}
}

func TestIncrementMixed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// Increment requests first
	_, err := db.IncrementRequestCount(ctx, tx, uid, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("IncrementRequestCount: %v", err)
	}

	// Then increment SMS on the same period
	usage, err := db.IncrementSMSCount(ctx, tx, uid, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("IncrementSMSCount: %v", err)
	}
	if usage.RequestCount != 1 {
		t.Errorf("expected request_count=1 after mixed, got %d", usage.RequestCount)
	}
	if usage.SMSCount != 1 {
		t.Errorf("expected sms_count=1 after mixed, got %d", usage.SMSCount)
	}
}

func TestGetLatestUsage(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	// No usage yet
	usage, err := db.GetLatestUsage(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetLatestUsage: %v", err)
	}
	if usage != nil {
		t.Errorf("expected nil usage, got %+v", usage)
	}

	// Create usage for Feb
	feb := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	mar := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	_, err = db.IncrementRequestCount(ctx, tx, uid, feb, mar)
	if err != nil {
		t.Fatalf("IncrementRequestCount: %v", err)
	}

	// Create usage for Jan (older)
	jan := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = db.IncrementRequestCount(ctx, tx, uid, jan, feb)
	if err != nil {
		t.Fatalf("IncrementRequestCount: %v", err)
	}

	// GetLatestUsage should return Feb (the most recent)
	usage, err = db.GetLatestUsage(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetLatestUsage: %v", err)
	}
	if usage == nil {
		t.Fatal("expected usage, got nil")
	}
	if !usage.PeriodStart.Equal(feb) {
		t.Errorf("expected period_start=%v, got %v", feb, usage.PeriodStart)
	}
}

func TestGetUsageByPeriod(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// Not found
	usage, err := db.GetUsageByPeriod(ctx, tx, uid, periodStart)
	if err != nil {
		t.Fatalf("GetUsageByPeriod: %v", err)
	}
	if usage != nil {
		t.Errorf("expected nil usage, got %+v", usage)
	}

	// Create and find
	_, err = db.IncrementRequestCount(ctx, tx, uid, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("IncrementRequestCount: %v", err)
	}

	usage, err = db.GetUsageByPeriod(ctx, tx, uid, periodStart)
	if err != nil {
		t.Fatalf("GetUsageByPeriod: %v", err)
	}
	if usage == nil {
		t.Fatal("expected usage, got nil")
	}
	if usage.RequestCount != 1 {
		t.Errorf("expected request_count=1, got %d", usage.RequestCount)
	}
}

// TestIncrementRequestCountConcurrent verifies that concurrent upserts from
// multiple goroutines don't lose increments. This uses the shared pool (not a
// single transaction) so each goroutine gets its own connection/transaction.
func TestIncrementRequestCountConcurrent(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupPool(t)
	ctx := context.Background()

	// Create user directly via pool (not in a test transaction).
	uid := testhelper.GenerateUID(t)
	if _, err := pool.Exec(ctx,
		`INSERT INTO auth.users (id) VALUES ($1)`, uid); err != nil {
		t.Fatalf("insert auth.users: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO profiles (id, username) VALUES ($1, $2)`, uid, "conc_"+uid[:8]); err != nil {
		t.Fatalf("insert profiles: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM usage_periods WHERE user_id = $1`, uid)  //nolint:errcheck
		pool.Exec(context.Background(), `DELETE FROM profiles WHERE id = $1`, uid)             //nolint:errcheck
		pool.Exec(context.Background(), `DELETE FROM auth.users WHERE id = $1`, uid)           //nolint:errcheck
	})

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	const goroutines = 20
	const incrementsPerGoroutine = 10
	var wg sync.WaitGroup
	errs := make(chan error, goroutines*incrementsPerGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				_, err := db.IncrementRequestCount(ctx, pool, uid, periodStart, periodEnd)
				if err != nil {
					errs <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent IncrementRequestCount failed: %v", err)
	}

	// Verify the total count
	usage, err := db.GetUsageByPeriod(ctx, pool, uid, periodStart)
	if err != nil {
		t.Fatalf("GetUsageByPeriod: %v", err)
	}
	if usage == nil {
		t.Fatal("expected usage row, got nil")
	}
	want := goroutines * incrementsPerGoroutine
	if usage.RequestCount != want {
		t.Errorf("expected request_count=%d after concurrent increments, got %d", want, usage.RequestCount)
	}
}

// TestReserveRequestQuota verifies basic reservation behavior: succeeds when
// under limit, fails when at limit.
func TestReserveRequestQuota(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	// Use current billing period to match SetUsageCount (which uses time.Now()).
	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())

	// Reserve with no existing row — should succeed (creates row with count=1).
	ok, err := db.ReserveRequestQuota(ctx, tx, uid, periodStart, periodEnd, 5)
	if err != nil {
		t.Fatalf("ReserveRequestQuota: %v", err)
	}
	if !ok {
		t.Fatal("expected reservation to succeed on first call")
	}

	// Verify count is 1.
	usage, err := db.GetUsageByPeriod(ctx, tx, uid, periodStart)
	if err != nil {
		t.Fatalf("GetUsageByPeriod: %v", err)
	}
	if usage.RequestCount != 1 {
		t.Errorf("expected request_count=1, got %d", usage.RequestCount)
	}

	// Set count to exactly the limit.
	testhelper.SetUsageCount(t, tx, uid, 5)

	// Reserve at limit — should fail.
	ok, err = db.ReserveRequestQuota(ctx, tx, uid, periodStart, periodEnd, 5)
	if err != nil {
		t.Fatalf("ReserveRequestQuota: %v", err)
	}
	if ok {
		t.Fatal("expected reservation to fail when at limit")
	}

	// Count should still be 5 (not incremented).
	usage, err = db.GetUsageByPeriod(ctx, tx, uid, periodStart)
	if err != nil {
		t.Fatalf("GetUsageByPeriod: %v", err)
	}
	if usage.RequestCount != 5 {
		t.Errorf("expected request_count=5 (unchanged), got %d", usage.RequestCount)
	}
}

// TestReserveRequestQuotaConcurrent verifies that concurrent reservation
// attempts at the boundary (limit-1) result in exactly one success. This
// proves the atomic conditional increment prevents TOCTOU over-counting.
func TestReserveRequestQuotaConcurrent(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupPool(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	if _, err := pool.Exec(ctx,
		`INSERT INTO auth.users (id) VALUES ($1)`, uid); err != nil {
		t.Fatalf("insert auth.users: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO profiles (id, username) VALUES ($1, $2)`, uid, "rq_"+uid[:8]); err != nil {
		t.Fatalf("insert profiles: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		pool.Exec(bg, `DELETE FROM usage_periods WHERE user_id = $1`, uid) //nolint:errcheck
		pool.Exec(bg, `DELETE FROM profiles WHERE id = $1`, uid)           //nolint:errcheck
		pool.Exec(bg, `DELETE FROM auth.users WHERE id = $1`, uid)         //nolint:errcheck
	})

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	const limit = 5

	// Set count to limit-1 so exactly one more reservation can succeed.
	testhelper.SetUsageCount(t, pool, uid, limit-1)

	const goroutines = 20
	var wg sync.WaitGroup
	results := make(chan bool, goroutines)
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := db.ReserveRequestQuota(ctx, pool, uid, periodStart, periodEnd, limit)
			if err != nil {
				errs <- err
				return
			}
			results <- ok
		}()
	}

	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent ReserveRequestQuota failed: %v", err)
	}

	successCount := 0
	for ok := range results {
		if ok {
			successCount++
		}
	}

	if successCount != 1 {
		t.Errorf("expected exactly 1 successful reservation out of %d, got %d", goroutines, successCount)
	}

	// Final count should be exactly the limit.
	usage, err := db.GetUsageByPeriod(ctx, pool, uid, periodStart)
	if err != nil {
		t.Fatalf("GetUsageByPeriod: %v", err)
	}
	if usage == nil {
		t.Fatal("expected usage row, got nil")
	}
	if usage.RequestCount != limit {
		t.Errorf("expected request_count=%d, got %d", limit, usage.RequestCount)
	}
}

// TestUpdateUsageBreakdownOnly verifies that breakdown updates don't increment
// request_count (used when quota was already atomically reserved).
func TestUpdateUsageBreakdownOnly(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// Create a usage row with count=5.
	_, err := tx.Exec(ctx,
		`INSERT INTO usage_periods (user_id, period_start, period_end, request_count) VALUES ($1, $2, $3, $4)`,
		uid, periodStart, periodEnd, 5)
	if err != nil {
		t.Fatalf("insert usage row: %v", err)
	}

	// Update breakdown without incrementing count.
	err = db.UpdateUsageBreakdownOnly(ctx, tx, uid, periodStart, db.UsageBreakdownKeys{
		AgentID:     42,
		ConnectorID: "github",
		ActionType:  "github.create_issue",
	})
	if err != nil {
		t.Fatalf("UpdateUsageBreakdownOnly: %v", err)
	}

	// Count should still be 5.
	usage, err := db.GetUsageByPeriod(ctx, tx, uid, periodStart)
	if err != nil {
		t.Fatalf("GetUsageByPeriod: %v", err)
	}
	if usage.RequestCount != 5 {
		t.Errorf("expected request_count=5 (unchanged), got %d", usage.RequestCount)
	}

	// Breakdown should have the agent entry.
	b := usage.ParseBreakdown()
	if b.ByAgent["42"] != 1 {
		t.Errorf("expected by_agent[42]=1, got %d", b.ByAgent["42"])
	}
	if b.ByConnector["github"] != 1 {
		t.Errorf("expected by_connector[github]=1, got %d", b.ByConnector["github"])
	}
}

func TestIncrementRequestCountWithBreakdown(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// First increment creates the row with breakdown
	usage, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid, periodStart, periodEnd, db.UsageBreakdownKeys{
		AgentID:     42,
		ConnectorID: "github",
		ActionType:  "github.create_issue",
	})
	if err != nil {
		t.Fatalf("IncrementRequestCountWithBreakdown: %v", err)
	}
	if usage.RequestCount != 1 {
		t.Errorf("expected request_count=1, got %d", usage.RequestCount)
	}
	if usage.Breakdown == nil {
		t.Fatal("expected non-nil breakdown")
	}

	// Second increment for same agent/connector
	usage, err = db.IncrementRequestCountWithBreakdown(ctx, tx, uid, periodStart, periodEnd, db.UsageBreakdownKeys{
		AgentID:     42,
		ConnectorID: "github",
		ActionType:  "github.create_issue",
	})
	if err != nil {
		t.Fatalf("IncrementRequestCountWithBreakdown: %v", err)
	}
	if usage.RequestCount != 2 {
		t.Errorf("expected request_count=2, got %d", usage.RequestCount)
	}

	// Third increment with different agent/connector
	usage, err = db.IncrementRequestCountWithBreakdown(ctx, tx, uid, periodStart, periodEnd, db.UsageBreakdownKeys{
		AgentID:     99,
		ConnectorID: "slack",
		ActionType:  "slack.send_message",
	})
	if err != nil {
		t.Fatalf("IncrementRequestCountWithBreakdown: %v", err)
	}
	if usage.RequestCount != 3 {
		t.Errorf("expected request_count=3, got %d", usage.RequestCount)
	}

	// Verify breakdown JSONB is populated
	var breakdown map[string]map[string]int
	if err := json.Unmarshal(usage.Breakdown, &breakdown); err != nil {
		t.Fatalf("failed to unmarshal breakdown: %v", err)
	}
	if breakdown["by_agent"]["42"] != 2 {
		t.Errorf("expected by_agent[42]=2, got %d", breakdown["by_agent"]["42"])
	}
	if breakdown["by_agent"]["99"] != 1 {
		t.Errorf("expected by_agent[99]=1, got %d", breakdown["by_agent"]["99"])
	}
	if breakdown["by_connector"]["github"] != 2 {
		t.Errorf("expected by_connector[github]=2, got %d", breakdown["by_connector"]["github"])
	}
	if breakdown["by_connector"]["slack"] != 1 {
		t.Errorf("expected by_connector[slack]=1, got %d", breakdown["by_connector"]["slack"])
	}
}

func TestIncrementRequestCountWithBreakdownEmptyConnector(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// Increment with empty connector_id and action_type (e.g. agent lifecycle events)
	usage, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid, periodStart, periodEnd, db.UsageBreakdownKeys{
		AgentID:     42,
		ConnectorID: "",
		ActionType:  "",
	})
	if err != nil {
		t.Fatalf("IncrementRequestCountWithBreakdown: %v", err)
	}
	if usage.RequestCount != 1 {
		t.Errorf("expected request_count=1, got %d", usage.RequestCount)
	}
}

func TestBillingPeriodBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      time.Time
		wantStart  time.Time
		wantEnd    time.Time
	}{
		{
			name:      "middle of month",
			input:     time.Date(2026, 2, 15, 14, 30, 0, 0, time.UTC),
			wantStart: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "first of month",
			input:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			wantStart: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "last day of year",
			input:     time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
			wantStart: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			start, end := db.BillingPeriodBounds(tt.input)
			if !start.Equal(tt.wantStart) {
				t.Errorf("start: got %v, want %v", start, tt.wantStart)
			}
			if !end.Equal(tt.wantEnd) {
				t.Errorf("end: got %v, want %v", end, tt.wantEnd)
			}
		})
	}
}

func TestParseBreakdown(t *testing.T) {
	t.Parallel()

	t.Run("nil breakdown", func(t *testing.T) {
		t.Parallel()
		u := &db.UsagePeriod{Breakdown: nil}
		b := u.ParseBreakdown()
		if b.ByAgent == nil || b.ByConnector == nil || b.ByActionType == nil {
			t.Fatal("expected initialized maps, got nil")
		}
		if len(b.ByAgent) != 0 || len(b.ByConnector) != 0 || len(b.ByActionType) != 0 {
			t.Error("expected empty maps for nil breakdown")
		}
	})

	t.Run("empty object", func(t *testing.T) {
		t.Parallel()
		u := &db.UsagePeriod{Breakdown: []byte(`{}`)}
		b := u.ParseBreakdown()
		if len(b.ByAgent) != 0 || len(b.ByConnector) != 0 || len(b.ByActionType) != 0 {
			t.Error("expected empty maps for empty object")
		}
	})

	t.Run("partial keys", func(t *testing.T) {
		t.Parallel()
		u := &db.UsagePeriod{Breakdown: []byte(`{"by_agent":{"42":5}}`)}
		b := u.ParseBreakdown()
		if b.ByAgent["42"] != 5 {
			t.Errorf("ByAgent[42] = %d, want 5", b.ByAgent["42"])
		}
		if b.ByConnector == nil || b.ByActionType == nil {
			t.Fatal("expected initialized maps for missing keys")
		}
	})

	t.Run("full breakdown", func(t *testing.T) {
		t.Parallel()
		raw := `{"by_agent":{"1":10,"2":3},"by_connector":{"github":8,"slack":5},"by_action_type":{"github.create_issue":6,"slack.send_message":7}}`
		u := &db.UsagePeriod{Breakdown: []byte(raw)}
		b := u.ParseBreakdown()
		if b.ByAgent["1"] != 10 || b.ByAgent["2"] != 3 {
			t.Errorf("ByAgent = %v", b.ByAgent)
		}
		if b.ByConnector["github"] != 8 || b.ByConnector["slack"] != 5 {
			t.Errorf("ByConnector = %v", b.ByConnector)
		}
		if b.ByActionType["github.create_issue"] != 6 || b.ByActionType["slack.send_message"] != 7 {
			t.Errorf("ByActionType = %v", b.ByActionType)
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		t.Parallel()
		u := &db.UsagePeriod{Breakdown: []byte(`not json`)}
		b := u.ParseBreakdown()
		// Should degrade gracefully with empty initialized maps.
		if b.ByAgent == nil || b.ByConnector == nil || b.ByActionType == nil {
			t.Fatal("expected initialized maps, got nil")
		}
	})
}
