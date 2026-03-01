package testhelper

// assertions_usage.go — reusable test assertions for billing usage counters.
//
// These helpers verify that the usage_periods table reflects the expected
// request_count and JSONB breakdown after billable API operations.

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// RequireUsageCount asserts that the current billing period's request_count
// for the given user equals want. Returns the usage period for further
// inspection (e.g., breakdown checks). Passing want=0 succeeds if no usage
// row exists or if the row has request_count=0.
func RequireUsageCount(t *testing.T, d db.DBTX, userID string, want int) *db.UsagePeriod {
	t.Helper()
	usage, err := db.GetCurrentPeriodUsage(context.Background(), d, userID)
	if err != nil {
		t.Fatalf("RequireUsageCount: GetCurrentPeriodUsage(%s): %v", userID, err)
	}
	if want == 0 {
		if usage != nil && usage.RequestCount != 0 {
			t.Errorf("RequireUsageCount: expected request_count=0 for user %s, got %d", userID, usage.RequestCount)
		}
		return usage
	}
	if usage == nil {
		t.Fatalf("RequireUsageCount: expected usage row with request_count=%d for user %s, got nil", want, userID)
	}
	if usage.RequestCount != want {
		t.Errorf("RequireUsageCount: expected request_count=%d for user %s, got %d", want, userID, usage.RequestCount)
	}
	return usage
}

// RequireUsageBreakdown asserts specific values in the usage period's JSONB
// breakdown. Pass nil for any map you don't want to check. Only the keys
// present in the expected maps are verified — extra keys in the actual
// breakdown are ignored.
func RequireUsageBreakdown(t *testing.T, usage *db.UsagePeriod, wantByAgent map[string]int, wantByConnector map[string]int, wantByActionType map[string]int) {
	t.Helper()
	if usage == nil {
		t.Fatal("RequireUsageBreakdown: usage is nil")
	}
	b := usage.ParseBreakdown()

	if wantByAgent != nil {
		for k, v := range wantByAgent {
			if b.ByAgent[k] != v {
				t.Errorf("RequireUsageBreakdown: by_agent[%s]: expected %d, got %d", k, v, b.ByAgent[k])
			}
		}
	}
	if wantByConnector != nil {
		for k, v := range wantByConnector {
			if b.ByConnector[k] != v {
				t.Errorf("RequireUsageBreakdown: by_connector[%s]: expected %d, got %d", k, v, b.ByConnector[k])
			}
		}
	}
	if wantByActionType != nil {
		for k, v := range wantByActionType {
			if b.ByActionType[k] != v {
				t.Errorf("RequireUsageBreakdown: by_action_type[%s]: expected %d, got %d", k, v, b.ByActionType[k])
			}
		}
	}
}
