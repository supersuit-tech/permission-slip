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
	testhelper.RequireColumns(t, tx, "profiles", []string{"id", "username", "email", "phone", "created_at"})
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

func TestUpdateProfileContactFields(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	email := "alice@example.com"
	phone := "+15551234567"

	// Set both fields.
	err := db.UpdateProfileContactFields(ctx, tx, uid, &email, &phone)
	if err != nil {
		t.Fatalf("update contact fields: %v", err)
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
	err = db.UpdateProfileContactFields(ctx, tx, uid, nil, nil)
	if err != nil {
		t.Fatalf("clear contact fields: %v", err)
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
	err := db.UpdateProfileContactFields(ctx, tx, uid, &badEmail, nil)
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
	err := db.UpdateProfileContactFields(ctx, tx, uid, nil, &badPhone)
	if err == nil {
		t.Fatal("expected error for invalid phone format, got nil")
	}
}
