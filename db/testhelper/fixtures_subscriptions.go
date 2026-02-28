package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// InsertSubscription creates a subscription row linking a user to a plan.
// The plan must already exist (the migration seeds "free" and "pay_as_you_go").
func InsertSubscription(t *testing.T, d db.DBTX, userID, planID string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO subscriptions (user_id, plan_id) VALUES ($1, $2)`,
		userID, planID)
}
