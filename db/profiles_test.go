package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestProfilesSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "profiles", []string{"id", "username", "email", "phone", "marketing_opt_in", "created_at"})
}

func TestProfilesUsernameUnique(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireConstraintExists(t, tx, "profiles", "profiles_username_key", "UNIQUE")
}

func TestProfileCascadeDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM auth.users WHERE id = '"+uid+"'",
		[]string{"profiles"},
		"id = '"+uid+"'",
	)
}

func TestGetProfileByUserID_Found(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	profile, err := db.GetProfileByUserID(context.Background(), tx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile, got nil")
	}
	if profile.ID != uid {
		t.Errorf("expected ID %q, got %q", uid, profile.ID)
	}
	if profile.Username != "u_"+uid[:8] {
		t.Errorf("expected username %q, got %q", "u_"+uid[:8], profile.Username)
	}
	if profile.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestGetProfileByUserID_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	profile, err := db.GetProfileByUserID(context.Background(), tx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile != nil {
		t.Errorf("expected nil profile, got %+v", profile)
	}
}

func TestGetProfileByUserID_NullContactFields(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	profile, err := db.GetProfileByUserID(context.Background(), tx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Email != nil {
		t.Errorf("expected nil email, got %q", *profile.Email)
	}
	if profile.Phone != nil {
		t.Errorf("expected nil phone, got %q", *profile.Phone)
	}
}

func TestUpdateProfileFields(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	email := "alice@example.com"
	phone := "+15551234567"

	// Set both fields.
	err := db.UpdateProfileFields(ctx, tx, uid, &email, &phone, nil)
	if err != nil {
		t.Fatalf("update profile fields: %v", err)
	}

	profile, err := db.GetProfileByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.Email == nil || *profile.Email != email {
		t.Errorf("expected email %q, got %v", email, profile.Email)
	}
	if profile.Phone == nil || *profile.Phone != phone {
		t.Errorf("expected phone %q, got %v", phone, profile.Phone)
	}

	// Clear both fields.
	err = db.UpdateProfileFields(ctx, tx, uid, nil, nil, nil)
	if err != nil {
		t.Fatalf("clear profile fields: %v", err)
	}
	profile, err = db.GetProfileByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("get profile after clear: %v", err)
	}
	if profile.Email != nil {
		t.Errorf("expected nil email after clear, got %q", *profile.Email)
	}
	if profile.Phone != nil {
		t.Errorf("expected nil phone after clear, got %q", *profile.Phone)
	}
}

func TestProfileEmailFormatConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	badEmail := "not-an-email"
	err := db.UpdateProfileFields(ctx, tx, uid, &badEmail, nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid email format, got nil")
	}
}

func TestProfilePhoneE164Constraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	badPhone := "555-1234"
	err := db.UpdateProfileFields(ctx, tx, uid, nil, &badPhone, nil)
	if err == nil {
		t.Fatal("expected error for invalid phone format, got nil")
	}
}

func TestUpdateProfileFields_MarketingOptIn(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// Default should be false.
	profile, err := db.GetProfileByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.MarketingOptIn {
		t.Error("expected marketing_opt_in to default to false")
	}

	// Set to true.
	optIn := true
	err = db.UpdateProfileFields(ctx, tx, uid, nil, nil, &optIn)
	if err != nil {
		t.Fatalf("update marketing opt-in: %v", err)
	}
	profile, err = db.GetProfileByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("re-fetch: %v", err)
	}
	if !profile.MarketingOptIn {
		t.Error("expected marketing_opt_in to be true")
	}

	// Nil should leave it unchanged.
	err = db.UpdateProfileFields(ctx, tx, uid, nil, nil, nil)
	if err != nil {
		t.Fatalf("update with nil: %v", err)
	}
	profile, err = db.GetProfileByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("re-fetch after nil: %v", err)
	}
	if !profile.MarketingOptIn {
		t.Error("expected marketing_opt_in to remain true when nil passed")
	}
}
