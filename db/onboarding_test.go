package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestCreateProfile_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	profile, err := db.CreateProfile(context.Background(), tx, uid, "newuser", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile, got nil")
	}
	if profile.ID != uid {
		t.Errorf("expected id %q, got %q", uid, profile.ID)
	}
	if profile.Username != "newuser" {
		t.Errorf("expected username %q, got %q", "newuser", profile.Username)
	}
	if profile.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestCreateProfile_UsernameTaken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "taken")

	_, err := db.CreateProfile(context.Background(), tx, uid2, "taken", false)
	if err == nil {
		t.Fatal("expected error for duplicate username, got nil")
	}

	var onboardErr *db.OnboardingError
	if !isOnboardingErr(err, &onboardErr) {
		t.Fatalf("expected *db.OnboardingError, got %T: %v", err, err)
	}
	if onboardErr.Code != db.OnboardingErrUsernameTaken {
		t.Errorf("expected code %q, got %q", db.OnboardingErrUsernameTaken, onboardErr.Code)
	}
}

func TestCreateProfile_Idempotent_AuthUsers(t *testing.T) {
	t.Parallel()
	// Calling CreateProfile when auth.users row already exists should not fail.
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	// Pre-insert the auth.users row (as Supabase would have done)
	testhelper.MustExec(t, tx, `INSERT INTO auth.users (id) VALUES ($1)`, uid)

	profile, err := db.CreateProfile(context.Background(), tx, uid, "preseeded", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile == nil || profile.Username != "preseeded" {
		t.Errorf("expected profile with username 'preseeded', got %+v", profile)
	}
}

func TestCreateProfile_MarketingOptIn(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	profile, err := db.CreateProfile(context.Background(), tx, uid, "marketer", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !profile.MarketingOptIn {
		t.Error("expected marketing_opt_in to be true")
	}

	// Re-fetch to confirm persistence.
	fetched, err := db.GetProfileByUserID(context.Background(), tx, uid)
	if err != nil {
		t.Fatalf("re-fetch: %v", err)
	}
	if !fetched.MarketingOptIn {
		t.Error("expected marketing_opt_in to be true after re-fetch")
	}
}

// isOnboardingErr is a helper to avoid importing errors in test file.
func isOnboardingErr(err error, target **db.OnboardingError) bool {
	if e, ok := err.(*db.OnboardingError); ok {
		*target = e
		return true
	}
	return false
}
