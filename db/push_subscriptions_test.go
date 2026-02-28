package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestPushSubscriptionsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "push_subscriptions", []string{
		"id", "user_id", "endpoint", "p256dh", "auth", "created_at",
	})
}

func TestPushSubscriptionsCascadeDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	_, err := db.UpsertPushSubscription(ctx, tx, uid, "https://push.example.com/sub1", "p256dh_key", "auth_key")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM auth.users WHERE id = '"+uid+"'",
		[]string{"push_subscriptions"},
		"user_id = '"+uid+"'",
	)
}

func TestUpsertPushSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	sub, err := db.UpsertPushSubscription(ctx, tx, uid, "https://push.example.com/sub1", "p256dh_key", "auth_key")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if sub.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if sub.Endpoint != "https://push.example.com/sub1" {
		t.Errorf("expected endpoint, got %q", sub.Endpoint)
	}
	if sub.P256dh != "p256dh_key" {
		t.Errorf("expected p256dh, got %q", sub.P256dh)
	}
	if sub.Auth != "auth_key" {
		t.Errorf("expected auth, got %q", sub.Auth)
	}
}

func TestUpsertPushSubscription_Update(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// Insert
	sub1, err := db.UpsertPushSubscription(ctx, tx, uid, "https://push.example.com/sub1", "old_p256dh", "old_auth")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Upsert with same endpoint should update keys
	sub2, err := db.UpsertPushSubscription(ctx, tx, uid, "https://push.example.com/sub1", "new_p256dh", "new_auth")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	// ID may differ due to RETURNING behavior, but there should be only one row
	_ = sub1
	if sub2.P256dh != "new_p256dh" {
		t.Errorf("expected updated p256dh, got %q", sub2.P256dh)
	}
	if sub2.Auth != "new_auth" {
		t.Errorf("expected updated auth, got %q", sub2.Auth)
	}

	// Should have only 1 subscription
	subs, err := db.ListPushSubscriptionsByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("expected 1 subscription after upsert, got %d", len(subs))
	}
}

func TestListPushSubscriptionsByUserID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// Empty initially
	subs, err := db.ListPushSubscriptionsByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 subscriptions, got %d", len(subs))
	}

	// Insert two
	_, err = db.UpsertPushSubscription(ctx, tx, uid, "https://push.example.com/sub1", "p1", "a1")
	if err != nil {
		t.Fatalf("upsert 1: %v", err)
	}
	_, err = db.UpsertPushSubscription(ctx, tx, uid, "https://push.example.com/sub2", "p2", "a2")
	if err != nil {
		t.Fatalf("upsert 2: %v", err)
	}

	subs, err = db.ListPushSubscriptionsByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(subs))
	}
}

func TestDeletePushSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	sub, err := db.UpsertPushSubscription(ctx, tx, uid, "https://push.example.com/sub1", "p1", "a1")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	deleted, err := db.DeletePushSubscription(ctx, tx, uid, sub.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}

	// Should be gone
	subs, err := db.ListPushSubscriptionsByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 subscriptions after delete, got %d", len(subs))
	}
}

func TestDeletePushSubscription_WrongUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u_"+uid2[:8])

	ctx := context.Background()

	sub, err := db.UpsertPushSubscription(ctx, tx, uid1, "https://push.example.com/sub1", "p1", "a1")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// user2 shouldn't be able to delete user1's subscription
	deleted, err := db.DeletePushSubscription(ctx, tx, uid2, sub.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted {
		t.Error("expected deleted=false for wrong user")
	}
}

func TestDeletePushSubscriptionByEndpoint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	_, err := db.UpsertPushSubscription(ctx, tx, uid, "https://push.example.com/expired", "p1", "a1")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	err = db.DeletePushSubscriptionByEndpoint(ctx, tx, "https://push.example.com/expired")
	if err != nil {
		t.Fatalf("delete by endpoint: %v", err)
	}

	subs, err := db.ListPushSubscriptionsByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 subscriptions after endpoint delete, got %d", len(subs))
	}
}
