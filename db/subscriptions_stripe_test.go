package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestGetSubscriptionByStripeCustomerID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Before setting Stripe ID, lookup should return nil.
	sub, err := db.GetSubscriptionByStripeCustomerID(ctx, tx, "cus_test123")
	if err != nil {
		t.Fatalf("GetSubscriptionByStripeCustomerID: %v", err)
	}
	if sub != nil {
		t.Fatal("expected nil before setting stripe_customer_id")
	}

	// Set the Stripe customer ID.
	customerID := "cus_test123"
	_, err = db.UpdateSubscriptionStripe(ctx, tx, uid, &customerID, nil)
	if err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}

	// Now lookup should succeed.
	sub, err = db.GetSubscriptionByStripeCustomerID(ctx, tx, "cus_test123")
	if err != nil {
		t.Fatalf("GetSubscriptionByStripeCustomerID: %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscription, got nil")
	}
	if sub.UserID != uid {
		t.Errorf("expected user_id=%s, got %s", uid, sub.UserID)
	}
}

func TestGetSubscriptionByStripeSubscriptionID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Set both Stripe IDs.
	customerID := "cus_test456"
	subscriptionID := "sub_test789"
	_, err := db.UpdateSubscriptionStripe(ctx, tx, uid, &customerID, &subscriptionID)
	if err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}

	// Lookup by subscription ID.
	sub, err := db.GetSubscriptionByStripeSubscriptionID(ctx, tx, "sub_test789")
	if err != nil {
		t.Fatalf("GetSubscriptionByStripeSubscriptionID: %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscription, got nil")
	}
	if sub.UserID != uid {
		t.Errorf("expected user_id=%s, got %s", uid, sub.UserID)
	}
	if sub.StripeCustomerID == nil || *sub.StripeCustomerID != customerID {
		t.Errorf("expected stripe_customer_id=%s, got %v", customerID, sub.StripeCustomerID)
	}

	// Lookup with non-existent ID should return nil.
	sub, err = db.GetSubscriptionByStripeSubscriptionID(ctx, tx, "sub_nonexistent")
	if err != nil {
		t.Fatalf("GetSubscriptionByStripeSubscriptionID: %v", err)
	}
	if sub != nil {
		t.Fatal("expected nil for non-existent subscription ID")
	}
}

func TestUpdateSubscriptionStripe_SetsIDs(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Set customer ID only (subscription ID nil).
	customerID := "cus_partial"
	sub, err := db.UpdateSubscriptionStripe(ctx, tx, uid, &customerID, nil)
	if err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}
	if sub.StripeCustomerID == nil || *sub.StripeCustomerID != customerID {
		t.Errorf("expected stripe_customer_id=%s, got %v", customerID, sub.StripeCustomerID)
	}
	if sub.StripeSubscriptionID != nil {
		t.Errorf("expected stripe_subscription_id=nil, got %v", sub.StripeSubscriptionID)
	}

	// Now set both.
	subID := "sub_full"
	sub, err = db.UpdateSubscriptionStripe(ctx, tx, uid, &customerID, &subID)
	if err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}
	if sub.StripeSubscriptionID == nil || *sub.StripeSubscriptionID != subID {
		t.Errorf("expected stripe_subscription_id=%s, got %v", subID, sub.StripeSubscriptionID)
	}
}
