package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestNotificationPreferencesSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "notification_preferences", []string{"user_id", "channel", "enabled"})
}

func TestNotificationPreferencesCascadeDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	err := db.UpsertNotificationPreference(ctx, tx, uid, "email", true)
	if err != nil {
		t.Fatalf("upsert preference: %v", err)
	}

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM auth.users WHERE id = '"+uid+"'",
		[]string{"notification_preferences"},
		"user_id = '"+uid+"'",
	)
}

func TestNotificationPreferencesChannelConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	err := db.UpsertNotificationPreference(ctx, tx, uid, "carrier-pigeon", true)
	if err == nil {
		t.Fatal("expected error for invalid channel, got nil")
	}
}

func TestIsNotificationChannelEnabled_Default(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	enabled, err := db.IsNotificationChannelEnabled(ctx, tx, uid, "email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("missing preference should default to enabled")
	}
}

func TestIsNotificationChannelEnabled_Disabled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	if err := db.UpsertNotificationPreference(ctx, tx, uid, "sms", false); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	enabled, err := db.IsNotificationChannelEnabled(ctx, tx, uid, "sms")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected sms to be disabled")
	}
}

func TestIsNotificationChannelEnabled_Enabled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	if err := db.UpsertNotificationPreference(ctx, tx, uid, "web-push", true); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	enabled, err := db.IsNotificationChannelEnabled(ctx, tx, uid, "web-push")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected web-push to be enabled")
	}
}

func TestUpsertNotificationPreference_Toggle(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// Insert as enabled.
	if err := db.UpsertNotificationPreference(ctx, tx, uid, "email", true); err != nil {
		t.Fatalf("upsert enabled: %v", err)
	}
	enabled, _ := db.IsNotificationChannelEnabled(ctx, tx, uid, "email")
	if !enabled {
		t.Fatal("expected enabled after first upsert")
	}

	// Upsert to disabled.
	if err := db.UpsertNotificationPreference(ctx, tx, uid, "email", false); err != nil {
		t.Fatalf("upsert disabled: %v", err)
	}
	enabled, _ = db.IsNotificationChannelEnabled(ctx, tx, uid, "email")
	if enabled {
		t.Fatal("expected disabled after second upsert")
	}
}

func TestGetNotificationPreferences(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	if err := db.UpsertNotificationPreference(ctx, tx, uid, "email", true); err != nil {
		t.Fatalf("upsert email: %v", err)
	}
	if err := db.UpsertNotificationPreference(ctx, tx, uid, "sms", false); err != nil {
		t.Fatalf("upsert sms: %v", err)
	}

	prefs, err := db.GetNotificationPreferences(ctx, tx, uid)
	if err != nil {
		t.Fatalf("get preferences: %v", err)
	}
	if len(prefs) != 2 {
		t.Fatalf("expected 2 preferences, got %d", len(prefs))
	}

	prefMap := make(map[string]bool)
	for _, p := range prefs {
		prefMap[p.Channel] = p.Enabled
	}
	if !prefMap["email"] {
		t.Error("email should be enabled")
	}
	if prefMap["sms"] {
		t.Error("sms should be disabled")
	}
}

func TestGetNotificationPreferences_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	prefs, err := db.GetNotificationPreferences(context.Background(), tx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prefs) != 0 {
		t.Errorf("expected 0 preferences, got %d", len(prefs))
	}
}

func TestUpsertNotificationPreference_MobilePush(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// mobile-push should be accepted by the CHECK constraint.
	if err := db.UpsertNotificationPreference(ctx, tx, uid, "mobile-push", true); err != nil {
		t.Fatalf("upsert mobile-push: %v", err)
	}

	enabled, err := db.IsNotificationChannelEnabled(ctx, tx, uid, "mobile-push")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected mobile-push to be enabled")
	}
}
