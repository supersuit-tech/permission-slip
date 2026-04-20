package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestNotificationTypePreferencesSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "notification_type_preferences", []string{"user_id", "notification_type", "enabled"})
}

func TestNotificationTypePreferencesCascadeDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	err := db.UpsertNotificationTypePreference(ctx, tx, uid, db.NotificationTypeStandingExecution, true)
	if err != nil {
		t.Fatalf("upsert preference: %v", err)
	}

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM auth.users WHERE id = '"+uid+"'",
		[]string{"notification_type_preferences"},
		"user_id = '"+uid+"'",
	)
}

func TestNotificationTypePreferencesTypeConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	err := db.UpsertNotificationTypePreference(ctx, tx, uid, "unknown_type", true)
	if err == nil {
		t.Fatal("expected error for invalid notification type, got nil")
	}
}

func TestIsNotificationTypeEnabled_Default(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	enabled, err := db.IsNotificationTypeEnabled(ctx, tx, uid, db.NotificationTypeStandingExecution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("missing preference should default to enabled")
	}
}

func TestIsNotificationTypeEnabled_Disabled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	if err := db.UpsertNotificationTypePreference(ctx, tx, uid, db.NotificationTypeStandingExecution, false); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	enabled, err := db.IsNotificationTypeEnabled(ctx, tx, uid, db.NotificationTypeStandingExecution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected standing_execution to be disabled")
	}
}

func TestGetNotificationTypePreferences(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	if err := db.UpsertNotificationTypePreference(ctx, tx, uid, db.NotificationTypeStandingExecution, false); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	prefs, err := db.GetNotificationTypePreferences(ctx, tx, uid)
	if err != nil {
		t.Fatalf("get preferences: %v", err)
	}
	if len(prefs) != 1 {
		t.Fatalf("expected 1 preference, got %d", len(prefs))
	}
	if prefs[0].NotificationType != db.NotificationTypeStandingExecution || prefs[0].Enabled {
		t.Errorf("unexpected preference: %+v", prefs[0])
	}
}
