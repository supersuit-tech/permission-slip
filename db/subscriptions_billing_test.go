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
		t.Errorf("expected at least 2 backfilled, got %d", count)
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
		t.Errorf("expected at least 1 backfilled, got %d", count)
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

func TestEnsureAllUsersSubscribed_SkipsExisting(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	// User with existing subscription should not be modified.
	uid1 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u_"+uid1[:8])
	testhelper.InsertSubscription(t, tx, uid1, db.PlanFree)

	// User without subscription should get one.
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u_"+uid2[:8])

	count, err := db.EnsureAllUsersSubscribed(ctx, tx, false)
	if err != nil {
		t.Fatalf("EnsureAllUsersSubscribed: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 backfilled (skipping existing), got %d", count)
	}

	// Existing subscription should be unchanged.
	sub1, err := db.GetSubscriptionByUserID(ctx, tx, uid1)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID(%s): %v", uid1, err)
	}
	if sub1.PlanID != db.PlanFree {
		t.Errorf("existing subscription should remain free, got %s", sub1.PlanID)
	}

	// New subscription should be pay_as_you_go (billing disabled).
	sub2, err := db.GetSubscriptionByUserID(ctx, tx, uid2)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID(%s): %v", uid2, err)
	}
	if sub2 == nil {
		t.Fatal("expected subscription for uid2, got nil")
	}
	if sub2.PlanID != db.PlanPayAsYouGo {
		t.Errorf("expected plan_id=%s, got %s", db.PlanPayAsYouGo, sub2.PlanID)
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

	// Second call should find no unsubscribed users.
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
