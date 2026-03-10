package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestFindProfileByAuthEmail_Found(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	email := "relink_found_" + uid[:8] + "@example.com"

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.MustExec(t, tx, `UPDATE auth.users SET email = $1 WHERE id = $2`, email, uid)

	profile, err := db.FindProfileByAuthEmail(context.Background(), tx, email)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile, got nil")
	}
	if profile.ID != uid {
		t.Errorf("expected ID %q, got %q", uid, profile.ID)
	}
}

func TestFindProfileByAuthEmail_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	profile, err := db.FindProfileByAuthEmail(context.Background(), tx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile != nil {
		t.Errorf("expected nil profile, got %+v", profile)
	}
}

func TestFindProfileByAuthEmail_EmptyEmail(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	profile, err := db.FindProfileByAuthEmail(context.Background(), tx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile != nil {
		t.Errorf("expected nil profile for empty email, got %+v", profile)
	}
}

func TestRelinkProfile(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	oldUID := testhelper.GenerateUID(t)
	newUID := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, oldUID, "relink_"+oldUID[:8])

	ctx := context.Background()

	err := db.RelinkProfile(ctx, tx, oldUID, newUID)
	if err != nil {
		t.Fatalf("RelinkProfile: %v", err)
	}

	// Old ID should no longer have a profile.
	old, err := db.GetProfileByUserID(ctx, tx, oldUID)
	if err != nil {
		t.Fatalf("GetProfileByUserID(old): %v", err)
	}
	if old != nil {
		t.Errorf("expected nil profile for old ID, got %+v", old)
	}

	// New ID should have the profile with the same username.
	relinked, err := db.GetProfileByUserID(ctx, tx, newUID)
	if err != nil {
		t.Fatalf("GetProfileByUserID(new): %v", err)
	}
	if relinked == nil {
		t.Fatal("expected profile for new ID, got nil")
	}
	if relinked.Username != "relink_"+oldUID[:8] {
		t.Errorf("expected username %q, got %q", "relink_"+oldUID[:8], relinked.Username)
	}
}

func TestRelinkProfile_CascadesToChildTables(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	oldUID := testhelper.GenerateUID(t)
	newUID := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, oldUID, "cascade_"+oldUID[:8])

	ctx := context.Background()

	// Insert a credential under the old user.
	credID := testhelper.GenerateID(t, "cred_")
	vaultID := testhelper.GenerateUID(t)
	testhelper.MustExec(t, tx,
		`INSERT INTO credentials (id, user_id, service, vault_secret_id) VALUES ($1, $2, 'test-svc', $3)`,
		credID, oldUID, vaultID,
	)

	err := db.RelinkProfile(ctx, tx, oldUID, newUID)
	if err != nil {
		t.Fatalf("RelinkProfile: %v", err)
	}

	// Credential should now reference the new user ID.
	var credUserID string
	err = tx.QueryRow(ctx,
		`SELECT user_id FROM credentials WHERE service = 'test-svc' AND user_id = $1`,
		newUID,
	).Scan(&credUserID)
	if err != nil {
		t.Fatalf("credential not cascaded: %v", err)
	}
	if credUserID != newUID {
		t.Errorf("expected credential user_id %q, got %q", newUID, credUserID)
	}
}

func TestRelinkProfile_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	oldUID := testhelper.GenerateUID(t)
	newUID := testhelper.GenerateUID(t)

	err := db.RelinkProfile(context.Background(), tx, oldUID, newUID)
	if err == nil {
		t.Fatal("expected error for non-existent profile, got nil")
	}
}

func TestFindProfileByUsername_Found(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	username := "finduser_" + uid[:8]

	testhelper.InsertUser(t, tx, uid, username)

	profile, err := db.FindProfileByUsername(context.Background(), tx, username)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile, got nil")
	}
	if profile.ID != uid {
		t.Errorf("expected ID %q, got %q", uid, profile.ID)
	}
	if profile.Username != username {
		t.Errorf("expected username %q, got %q", username, profile.Username)
	}
}

func TestFindProfileByUsername_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	profile, err := db.FindProfileByUsername(context.Background(), tx, "nonexistent_user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile != nil {
		t.Errorf("expected nil profile, got %+v", profile)
	}
}
