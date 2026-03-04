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
		"id", "user_id", "channel", "endpoint", "p256dh", "auth", "expo_token", "created_at",
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
	if sub.Channel != db.PushChannelWebPush {
		t.Errorf("expected channel %q, got %q", db.PushChannelWebPush, sub.Channel)
	}
	if sub.Endpoint == nil || *sub.Endpoint != "https://push.example.com/sub1" {
		t.Errorf("expected endpoint, got %v", sub.Endpoint)
	}
	if sub.P256dh == nil || *sub.P256dh != "p256dh_key" {
		t.Errorf("expected p256dh, got %v", sub.P256dh)
	}
	if sub.Auth == nil || *sub.Auth != "auth_key" {
		t.Errorf("expected auth, got %v", sub.Auth)
	}
	if sub.ExpoToken != nil {
		t.Errorf("expected nil expo_token for web-push, got %v", sub.ExpoToken)
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
	if sub2.P256dh == nil || *sub2.P256dh != "new_p256dh" {
		t.Errorf("expected updated p256dh, got %v", sub2.P256dh)
	}
	if sub2.Auth == nil || *sub2.Auth != "new_auth" {
		t.Errorf("expected updated auth, got %v", sub2.Auth)
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

// --- Expo push token tests ---

func TestUpsertExpoPushToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	sub, err := db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[test123]")
	if err != nil {
		t.Fatalf("upsert expo token: %v", err)
	}
	if sub.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if sub.Channel != db.PushChannelMobilePush {
		t.Errorf("expected channel %q, got %q", db.PushChannelMobilePush, sub.Channel)
	}
	if sub.ExpoToken == nil || *sub.ExpoToken != "ExponentPushToken[test123]" {
		t.Errorf("expected expo_token, got %v", sub.ExpoToken)
	}
	if sub.Endpoint != nil {
		t.Errorf("expected nil endpoint for mobile-push, got %v", sub.Endpoint)
	}
	if sub.P256dh != nil {
		t.Errorf("expected nil p256dh for mobile-push, got %v", sub.P256dh)
	}
	if sub.Auth != nil {
		t.Errorf("expected nil auth for mobile-push, got %v", sub.Auth)
	}
}

func TestUpsertExpoPushToken_Update(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// Insert
	_, err := db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[test123]")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Upsert with same token should not create a duplicate
	_, err = db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[test123]")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	// Should have only 1 subscription
	subs, err := db.ListExpoPushTokensByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("expected 1 expo subscription after upsert, got %d", len(subs))
	}
}

func TestListExpoPushTokensByUserID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// Insert one web-push and one expo
	_, err := db.UpsertPushSubscription(ctx, tx, uid, "https://push.example.com/sub1", "p1", "a1")
	if err != nil {
		t.Fatalf("upsert web-push: %v", err)
	}
	_, err = db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[test123]")
	if err != nil {
		t.Fatalf("upsert expo: %v", err)
	}

	// ListAll should return both
	all, err := db.ListPushSubscriptionsByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 total subscriptions, got %d", len(all))
	}

	// ListExpo should return only the expo one
	expo, err := db.ListExpoPushTokensByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list expo: %v", err)
	}
	if len(expo) != 1 {
		t.Errorf("expected 1 expo subscription, got %d", len(expo))
	}
	if expo[0].Channel != db.PushChannelMobilePush {
		t.Errorf("expected mobile-push channel, got %q", expo[0].Channel)
	}

	// ListWebPush should return only the web-push one
	webPush, err := db.ListWebPushSubscriptionsByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list web-push: %v", err)
	}
	if len(webPush) != 1 {
		t.Errorf("expected 1 web-push subscription, got %d", len(webPush))
	}
	if webPush[0].Channel != db.PushChannelWebPush {
		t.Errorf("expected web-push channel, got %q", webPush[0].Channel)
	}
}

func TestDeleteExpoPushToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	_, err := db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[expired]")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	err = db.DeleteExpoPushToken(ctx, tx, "ExponentPushToken[expired]")
	if err != nil {
		t.Fatalf("delete expo token: %v", err)
	}

	subs, err := db.ListExpoPushTokensByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 expo subscriptions after delete, got %d", len(subs))
	}
}

func TestExpoPushToken_CascadeDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	_, err := db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[cascade]")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM auth.users WHERE id = '"+uid+"'",
		[]string{"push_subscriptions"},
		"user_id = '"+uid+"'",
	)
}

func TestDeletePushSubscription_ExpoToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	sub, err := db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[delete_by_id]")
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

	subs, err := db.ListExpoPushTokensByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 subscriptions after delete, got %d", len(subs))
	}
}
