package notify_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/notify"
)

func TestDBSMSGate_FreeTier_Blocked(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "free_user")
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	gate := &notify.DBSMSGate{DB: tx}
	allowed, err := gate.CanSendSMS(context.Background(), uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected CanSendSMS=false for free tier user")
	}
}

func TestDBSMSGate_PaidTier_Allowed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "paid_user")
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	gate := &notify.DBSMSGate{DB: tx}
	allowed, err := gate.CanSendSMS(context.Background(), uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected CanSendSMS=true for paid tier user")
	}
}

func TestDBSMSGate_NoSubscription_Blocked(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "no_sub_user")
	// No subscription inserted.

	gate := &notify.DBSMSGate{DB: tx}
	allowed, err := gate.CanSendSMS(context.Background(), uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected CanSendSMS=false for user without subscription")
	}
}

func TestDBSMSGate_RecordSMSSent_IncrementsCount(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "sms_user")
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	gate := &notify.DBSMSGate{DB: tx}

	// Record two SMS sends.
	if err := gate.RecordSMSSent(context.Background(), uid); err != nil {
		t.Fatalf("first RecordSMSSent: %v", err)
	}
	if err := gate.RecordSMSSent(context.Background(), uid); err != nil {
		t.Fatalf("second RecordSMSSent: %v", err)
	}

	// Verify sms_count = 2.
	usage, err := db.GetCurrentPeriodUsage(context.Background(), tx, uid)
	if err != nil {
		t.Fatalf("GetCurrentPeriodUsage: %v", err)
	}
	if usage == nil {
		t.Fatal("expected usage row, got nil")
	}
	if usage.SMSCount != 2 {
		t.Errorf("expected sms_count=2, got %d", usage.SMSCount)
	}
}
