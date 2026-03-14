package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestSubscriptionsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "subscriptions", []string{
		"id", "user_id", "plan_id", "status",
		"stripe_customer_id", "stripe_subscription_id",
		"current_period_start", "current_period_end",
		"downgraded_at", "created_at", "updated_at",
	})
}

func TestSubscriptionsStatusCheck(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	testhelper.RequireCheckValues(t, tx,
		"subscription status",
		[]string{"active", "past_due", "cancelled"},
		"invalid_status",
		func(value string, index int) error {
			u := testhelper.GenerateUID(t)
			testhelper.InsertUser(t, tx, u, "chk_"+u[:8])
			_, err := tx.Exec(ctx,
				`INSERT INTO subscriptions (user_id, plan_id, status) VALUES ($1, 'free', $2)`,
				u, value)
			return err
		},
	)
}

func TestSubscriptionsUniqueUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	testhelper.RequireUniqueViolation(t, tx,
		"one subscription per user",
		func() error {
			_, err := tx.Exec(ctx,
				`INSERT INTO subscriptions (user_id, plan_id) VALUES ($1, 'free')`, uid)
			return err
		},
		func() error {
			_, err := tx.Exec(ctx,
				`INSERT INTO subscriptions (user_id, plan_id) VALUES ($1, 'free')`, uid)
			return err
		},
	)
}

func TestSubscriptionsCascadeDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertSubscription(t, tx, uid, "free")

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM auth.users WHERE id = '"+uid+"'",
		[]string{"subscriptions"},
		"user_id = '"+uid+"'",
	)
}

func TestCreateSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	sub, err := db.CreateSubscription(ctx, tx, uid, db.PlanFree)
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}
	if sub.UserID != uid {
		t.Errorf("expected user_id=%s, got %s", uid, sub.UserID)
	}
	if sub.PlanID != db.PlanFree {
		t.Errorf("expected plan_id=%s, got %s", db.PlanFree, sub.PlanID)
	}
	if sub.Status != db.SubscriptionStatusActive {
		t.Errorf("expected status=active, got %s", sub.Status)
	}
	if sub.StripeCustomerID != nil {
		t.Errorf("expected nil stripe_customer_id, got %v", *sub.StripeCustomerID)
	}
	if sub.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestGetSubscriptionByUserID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	// No subscription yet
	sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub != nil {
		t.Errorf("expected nil subscription, got %+v", sub)
	}

	// Create one
	_, err = db.CreateSubscription(ctx, tx, uid, "free")
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	sub, err = db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscription, got nil")
	}
	if sub.PlanID != "free" {
		t.Errorf("expected plan_id=free, got %s", sub.PlanID)
	}
}

func TestUpdateSubscriptionPlan(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	_, err := db.CreateSubscription(ctx, tx, uid, "free")
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	updated, err := db.UpdateSubscriptionPlan(ctx, tx, uid, "pay_as_you_go")
	if err != nil {
		t.Fatalf("UpdateSubscriptionPlan: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated subscription, got nil")
	}
	if updated.PlanID != "pay_as_you_go" {
		t.Errorf("expected plan_id=pay_as_you_go, got %s", updated.PlanID)
	}
}

func TestUpdateSubscriptionStatus(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	_, err := db.CreateSubscription(ctx, tx, uid, "free")
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	updated, err := db.UpdateSubscriptionStatus(ctx, tx, uid, db.SubscriptionStatusPastDue)
	if err != nil {
		t.Fatalf("UpdateSubscriptionStatus: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated subscription, got nil")
	}
	if updated.Status != db.SubscriptionStatusPastDue {
		t.Errorf("expected status=past_due, got %s", updated.Status)
	}

	// Verify Go-level validation rejects invalid status
	_, err = db.UpdateSubscriptionStatus(ctx, tx, uid, "bogus")
	if err == nil {
		t.Error("expected error for invalid status, got nil")
	}
}

func TestUpdateSubscriptionStripe(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	_, err := db.CreateSubscription(ctx, tx, uid, "free")
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	custID := "cus_test123"
	subID := "sub_test456"
	updated, err := db.UpdateSubscriptionStripe(ctx, tx, uid, &custID, &subID)
	if err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated subscription, got nil")
	}
	if updated.StripeCustomerID == nil || *updated.StripeCustomerID != custID {
		t.Errorf("expected stripe_customer_id=%s, got %v", custID, updated.StripeCustomerID)
	}
	if updated.StripeSubscriptionID == nil || *updated.StripeSubscriptionID != subID {
		t.Errorf("expected stripe_subscription_id=%s, got %v", subID, updated.StripeSubscriptionID)
	}
}

func TestUpdateSubscriptionPeriod(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	sub, err := db.CreateSubscription(ctx, tx, uid, "free")
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	newStart := sub.CurrentPeriodEnd
	newEnd := newStart.AddDate(0, 1, 0)
	updated, err := db.UpdateSubscriptionPeriod(ctx, tx, uid, newStart, newEnd)
	if err != nil {
		t.Fatalf("UpdateSubscriptionPeriod: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated subscription, got nil")
	}
	if !updated.CurrentPeriodStart.Equal(newStart) {
		t.Errorf("expected period_start=%v, got %v", newStart, updated.CurrentPeriodStart)
	}
	if !updated.CurrentPeriodEnd.Equal(newEnd) {
		t.Errorf("expected period_end=%v, got %v", newEnd, updated.CurrentPeriodEnd)
	}
}

func TestGetSubscriptionWithPlan(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	ctx := context.Background()

	// No subscription yet
	sp, err := db.GetSubscriptionWithPlan(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionWithPlan: %v", err)
	}
	if sp != nil {
		t.Errorf("expected nil, got %+v", sp)
	}

	// Create a free subscription
	_, err = db.CreateSubscription(ctx, tx, uid, db.PlanFree)
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	sp, err = db.GetSubscriptionWithPlan(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionWithPlan: %v", err)
	}
	if sp == nil {
		t.Fatal("expected subscription with plan, got nil")
	}
	if sp.PlanID != db.PlanFree {
		t.Errorf("expected plan_id=%s, got %s", db.PlanFree, sp.PlanID)
	}
	if sp.Plan.Name != "Free" {
		t.Errorf("expected plan name=%q, got %q", "Free", sp.Plan.Name)
	}
	if sp.Plan.MaxAgents == nil || *sp.Plan.MaxAgents != 3 {
		t.Errorf("expected plan max_agents=3, got %v", sp.Plan.MaxAgents)
	}
	if sp.Plan.AuditRetentionDays != 7 {
		t.Errorf("expected plan audit_retention_days=7, got %d", sp.Plan.AuditRetentionDays)
	}
}

func TestUpdateSubscriptionPlan_DowngradeTracking(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("SetsDowngradedAtOnDowngrade", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		_, err := db.CreateSubscription(ctx, tx, uid, db.PlanPayAsYouGo)
		if err != nil {
			t.Fatalf("CreateSubscription: %v", err)
		}

		// Downgrade from paid to free → should set downgraded_at.
		updated, err := db.UpdateSubscriptionPlan(ctx, tx, uid, db.PlanFree)
		if err != nil {
			t.Fatalf("UpdateSubscriptionPlan: %v", err)
		}
		if updated.DowngradedAt == nil {
			t.Fatal("expected downgraded_at to be set after downgrade, got nil")
		}
		if updated.PlanID != db.PlanFree {
			t.Errorf("expected plan_id=%s, got %s", db.PlanFree, updated.PlanID)
		}
	})

	t.Run("ClearsDowngradedAtOnUpgrade", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		_, err := db.CreateSubscription(ctx, tx, uid, db.PlanPayAsYouGo)
		if err != nil {
			t.Fatalf("CreateSubscription: %v", err)
		}

		// Downgrade first.
		_, err = db.UpdateSubscriptionPlan(ctx, tx, uid, db.PlanFree)
		if err != nil {
			t.Fatalf("downgrade: %v", err)
		}

		// Upgrade back → should clear downgraded_at.
		upgraded, err := db.UpdateSubscriptionPlan(ctx, tx, uid, db.PlanPayAsYouGo)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		if upgraded.DowngradedAt != nil {
			t.Errorf("expected downgraded_at=nil after upgrade, got %v", *upgraded.DowngradedAt)
		}
	})

	t.Run("DowngradedAtNilOnSamePlanUpdate", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		_, err := db.CreateSubscription(ctx, tx, uid, db.PlanFree)
		if err != nil {
			t.Fatalf("CreateSubscription: %v", err)
		}

		// Update to the same plan (free → free) → downgraded_at stays nil.
		updated, err := db.UpdateSubscriptionPlan(ctx, tx, uid, db.PlanFree)
		if err != nil {
			t.Fatalf("UpdateSubscriptionPlan: %v", err)
		}
		if updated.DowngradedAt != nil {
			t.Errorf("expected downgraded_at=nil for same-plan update, got %v", *updated.DowngradedAt)
		}
	})
}

func TestEffectiveRetentionDays(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("UsesCurrentPlanWithNoDowngrade", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		_, err := db.CreateSubscription(ctx, tx, uid, db.PlanFree)
		if err != nil {
			t.Fatalf("CreateSubscription: %v", err)
		}

		sp, err := db.GetSubscriptionWithPlan(ctx, tx, uid)
		if err != nil {
			t.Fatalf("GetSubscriptionWithPlan: %v", err)
		}

		if got := sp.EffectiveRetentionDays(); got != 7 {
			t.Errorf("expected 7-day retention for free plan, got %d", got)
		}
	})

	t.Run("UsesPaidRetentionDuringGracePeriod", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		_, err := db.CreateSubscription(ctx, tx, uid, db.PlanPayAsYouGo)
		if err != nil {
			t.Fatalf("CreateSubscription: %v", err)
		}

		// Downgrade to free → triggers grace period.
		_, err = db.UpdateSubscriptionPlan(ctx, tx, uid, db.PlanFree)
		if err != nil {
			t.Fatalf("downgrade: %v", err)
		}

		sp, err := db.GetSubscriptionWithPlan(ctx, tx, uid)
		if err != nil {
			t.Fatalf("GetSubscriptionWithPlan: %v", err)
		}

		// During grace period, should use the paid plan's 90-day retention.
		if got := sp.EffectiveRetentionDays(); got != db.PaidPlanRetentionDays() {
			t.Errorf("expected %d-day retention during grace period, got %d",
				db.PaidPlanRetentionDays(), got)
		}
	})

	t.Run("UsesCurrentPlanAfterGracePeriodExpires", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		_, err := db.CreateSubscription(ctx, tx, uid, db.PlanPayAsYouGo)
		if err != nil {
			t.Fatalf("CreateSubscription: %v", err)
		}

		// Downgrade to free.
		_, err = db.UpdateSubscriptionPlan(ctx, tx, uid, db.PlanFree)
		if err != nil {
			t.Fatalf("downgrade: %v", err)
		}

		// Simulate grace period having expired (set downgraded_at to 10 days ago).
		testhelper.MustExec(t, tx,
			`UPDATE subscriptions SET downgraded_at = now() - interval '10 days' WHERE user_id = $1`,
			uid)

		sp, err := db.GetSubscriptionWithPlan(ctx, tx, uid)
		if err != nil {
			t.Fatalf("GetSubscriptionWithPlan: %v", err)
		}

		// After grace period, should use the free plan's 7-day retention.
		if got := sp.EffectiveRetentionDays(); got != 7 {
			t.Errorf("expected 7-day retention after grace period expired, got %d", got)
		}
	})

	t.Run("GracePeriodEndsAtReturnedDuringGracePeriod", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		_, err := db.CreateSubscription(ctx, tx, uid, db.PlanPayAsYouGo)
		if err != nil {
			t.Fatalf("CreateSubscription: %v", err)
		}

		// Downgrade to free → triggers grace period.
		_, err = db.UpdateSubscriptionPlan(ctx, tx, uid, db.PlanFree)
		if err != nil {
			t.Fatalf("downgrade: %v", err)
		}

		sp, err := db.GetSubscriptionWithPlan(ctx, tx, uid)
		if err != nil {
			t.Fatalf("GetSubscriptionWithPlan: %v", err)
		}

		endsAt := sp.GracePeriodEndsAt()
		if endsAt == nil {
			t.Fatal("expected non-nil GracePeriodEndsAt during active grace period")
		}
	})

	t.Run("GracePeriodEndsAtNilWhenNoDowngrade", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		_, err := db.CreateSubscription(ctx, tx, uid, db.PlanFree)
		if err != nil {
			t.Fatalf("CreateSubscription: %v", err)
		}

		sp, err := db.GetSubscriptionWithPlan(ctx, tx, uid)
		if err != nil {
			t.Fatalf("GetSubscriptionWithPlan: %v", err)
		}

		if sp.GracePeriodEndsAt() != nil {
			t.Error("expected nil GracePeriodEndsAt when no downgrade")
		}
	})
}
