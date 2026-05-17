package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestCreateProfile_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.MustExec(t, tx, `INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'x')`,
		uid, uid+"@example.com")

	profile, err := db.CreateProfile(context.Background(), tx, uid, "newuser", "newuser@example.com", false)
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
	if profile.Email == nil || *profile.Email != "newuser@example.com" {
		t.Errorf("expected email %q, got %v", "newuser@example.com", profile.Email)
	}
}

func TestCreateProfile_UsernameTaken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "taken")
	// uid2 needs a users row but no profile yet so CreateProfile can attempt insert.
	testhelper.MustExec(t, tx, `INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'x')`,
		uid2, uid2+"@example.com")

	_, err := db.CreateProfile(context.Background(), tx, uid2, "taken", "taken@example.com", false)
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

// TestCreateProfile_NewUser verifies that CreateProfile works when the users
// row already exists (the normal flow: auth signup creates users row, then
// the onboarding endpoint calls CreateProfile).
func TestCreateProfile_NewUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.MustExec(t, tx, `INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'x')`,
		uid, "preseeded@example.com")

	profile, err := db.CreateProfile(context.Background(), tx, uid, "preseeded", "preseeded@example.com", false)
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
	testhelper.MustExec(t, tx, `INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'x')`,
		uid, uid+"@example.com")

	profile, err := db.CreateProfile(context.Background(), tx, uid, "marketer", "marketer@example.com", true)
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

func TestCreateProfile_SMSDisabledByDefault(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.MustExec(t, tx, `INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'x')`,
		uid, uid+"@example.com")

	_, err := db.CreateProfile(context.Background(), tx, uid, "smsuser", "smsuser@example.com", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	enabled, err := db.IsNotificationChannelEnabled(context.Background(), tx, uid, "sms")
	if err != nil {
		t.Fatalf("IsNotificationChannelEnabled: %v", err)
	}
	if enabled {
		t.Error("expected SMS notifications to be disabled for new users")
	}
}

func TestCreateProfile_EmptyEmail(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.MustExec(t, tx, `INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'x')`,
		uid, uid+"@example.com")

	profile, err := db.CreateProfile(context.Background(), tx, uid, "noemailer", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Email != nil {
		t.Errorf("expected nil email, got %v", profile.Email)
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
