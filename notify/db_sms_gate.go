package notify

import (
	"context"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// DBSMSGate implements SMSGate by checking the user's subscription plan
// in the database. Free-tier users are blocked; paid-tier users are allowed
// and have their SMS usage tracked.
type DBSMSGate struct {
	DB db.DBTX
}

// CanSendSMS returns true if the user is on a paid plan (not "free").
// Users without a subscription are treated as not allowed.
func (g *DBSMSGate) CanSendSMS(ctx context.Context, userID string) (bool, error) {
	sub, err := db.GetSubscriptionByUserID(ctx, g.DB, userID)
	if err != nil {
		return false, err
	}
	if sub == nil {
		return false, nil
	}
	return sub.PlanID != db.PlanFree, nil
}

// RecordSMSSent increments the sms_count for the user's current billing period.
func (g *DBSMSGate) RecordSMSSent(ctx context.Context, userID string) error {
	now := time.Now()
	periodStart, periodEnd := db.BillingPeriodBounds(now)
	_, err := db.IncrementSMSCount(ctx, g.DB, userID, periodStart, periodEnd)
	return err
}
