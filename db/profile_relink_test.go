package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

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
