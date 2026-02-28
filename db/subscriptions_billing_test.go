package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestEnsureAllUsersSubscribed_BillingDisabled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	// Create users without subscriptions.
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u_"+uid2[:8])

	// Billing disabled → users should get pay_as_you_go.
	count, err := db.EnsureAllUsersSubscribed(ctx, tx, false)
	if err != nil {
		t.Fatalf("EnsureAllUsersSubscribed: %v", err)
	}
	if count < 2 {
		t.Errorf("expected at least 2 affected, got %d", count)
	}

	// Verify both users got pay_as_you_go.
	for _, uid := range []string{uid1, uid2} {
		sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
		if err != nil {
			t.Fatalf("GetSubscriptionByUserID(%s): %v", uid, err)
		}
		if sub == nil {
			t.Fatalf("expected subscription for %s, got nil", uid)
		}
		if sub.PlanID != db.PlanPayAsYouGo {
			t.Errorf("expected plan_id=%s for %s, got %s", db.PlanPayAsYouGo, uid, sub.PlanID)
		}
	}
}

func TestEnsureAllUsersSubscribed_BillingEnabled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Billing enabled → user should get free plan.
	count, err := db.EnsureAllUsersSubscribed(ctx, tx, true)
	if err != nil {
		t.Fatalf("EnsureAllUsersSubscribed: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 affected, got %d", count)
	}

	sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscription, got nil")
	}
	if sub.PlanID != db.PlanFree {
		t.Errorf("expected plan_id=%s, got %s", db.PlanFree, sub.PlanID)
	}
}

func TestEnsureAllUsersSubscribed_UpgradesFreeWhenBillingDisabled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	// Simulate the migration backfill: user already has a "free" subscription.
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Billing disabled → "free" subscriptions should be upgraded to pay_as_you_go.
	count, err := db.EnsureAllUsersSubscribed(ctx, tx, false)
	if err != nil {
		t.Fatalf("EnsureAllUsersSubscribed: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 upgraded, got %d", count)
	}

	sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscription, got nil")
	}
	if sub.PlanID != db.PlanPayAsYouGo {
		t.Errorf("existing free subscription should be upgraded to %s, got %s", db.PlanPayAsYouGo, sub.PlanID)
	}
}

func TestEnsureAllUsersSubscribed_PreservesPayAsYouGo(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	// User already on pay_as_you_go should not be touched.
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Billing enabled → should NOT downgrade pay_as_you_go to free.
	_, err := db.EnsureAllUsersSubscribed(ctx, tx, true)
	if err != nil {
		t.Fatalf("EnsureAllUsersSubscribed: %v", err)
	}

	sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub.PlanID != db.PlanPayAsYouGo {
		t.Errorf("expected pay_as_you_go to be preserved, got %s", sub.PlanID)
	}
}

func TestEnsureAllUsersSubscribed_Idempotent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// First call subscribes unsubscribed users.
	_, err := db.EnsureAllUsersSubscribed(ctx, tx, false)
	if err != nil {
		t.Fatalf("first EnsureAllUsersSubscribed: %v", err)
	}

	// Second call should find nothing to do.
	count, err := db.EnsureAllUsersSubscribed(ctx, tx, false)
	if err != nil {
		t.Fatalf("second EnsureAllUsersSubscribed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 on second run (idempotent), got %d", count)
	}

	// Verify our user still has the right plan.
	sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscription, got nil")
	}
	if sub.PlanID != db.PlanPayAsYouGo {
		t.Errorf("expected plan_id=%s, got %s", db.PlanPayAsYouGo, sub.PlanID)
	}
}
