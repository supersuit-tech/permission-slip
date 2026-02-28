package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestConsumeInvite_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	riID := testhelper.GenerateID(t, "ri_")
	hash := testhelper.GenerateID(t, "hash_")

	_, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, hash, 900)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	invite, err := db.ConsumeInvite(context.Background(), tx, hash)
	if err != nil {
		t.Fatalf("consume invite: %v", err)
	}
	if invite == nil {
		t.Fatal("expected non-nil invite")
	}
	if invite.ID != riID {
		t.Errorf("expected ID %q, got %q", riID, invite.ID)
	}
	if invite.UserID != uid {
		t.Errorf("expected UserID %q, got %q", uid, invite.UserID)
	}
	if invite.Status != "consumed" {
		t.Errorf("expected status 'consumed', got %q", invite.Status)
	}
	if invite.ConsumedAt == nil {
		t.Error("expected non-nil consumed_at")
	}
}

func TestConsumeInvite_AlreadyConsumed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	riID := testhelper.GenerateID(t, "ri_")
	hash := testhelper.GenerateID(t, "hash_")

	_, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, hash, 900)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	// First consume should succeed.
	invite, err := db.ConsumeInvite(context.Background(), tx, hash)
	if err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if invite == nil {
		t.Fatal("expected non-nil invite on first consume")
	}

	// Second consume should return nil (already consumed).
	invite2, err := db.ConsumeInvite(context.Background(), tx, hash)
	if err != nil {
		t.Fatalf("second consume: %v", err)
	}
	if invite2 != nil {
		t.Error("expected nil invite on second consume (already consumed)")
	}
}

func TestConsumeInvite_Expired(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	riID := testhelper.GenerateID(t, "ri_")
	hash := testhelper.GenerateID(t, "hash_")

	_, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, hash, 900)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	// Backdate expires_at to the past.
	testhelper.MustExec(t, tx, `UPDATE registration_invites SET expires_at = now() - interval '1 hour' WHERE id = $1`, riID)

	invite, err := db.ConsumeInvite(context.Background(), tx, hash)
	if err != nil {
		t.Fatalf("consume expired invite: %v", err)
	}
	if invite != nil {
		t.Error("expected nil invite for expired code")
	}
}

func TestConsumeInvite_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	invite, err := db.ConsumeInvite(context.Background(), tx, "nonexistent_hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invite != nil {
		t.Error("expected nil invite for nonexistent code")
	}
}

func TestLookupInviteByCodeHash_Found(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	riID := testhelper.GenerateID(t, "ri_")
	hash := testhelper.GenerateID(t, "hash_")

	_, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, hash, 900)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	invite, err := db.LookupInviteByCodeHash(context.Background(), tx, hash)
	if err != nil {
		t.Fatalf("lookup invite: %v", err)
	}
	if invite == nil {
		t.Fatal("expected non-nil invite")
	}
	if invite.ID != riID {
		t.Errorf("expected ID %q, got %q", riID, invite.ID)
	}
}

func TestLookupInviteByCodeHash_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	invite, err := db.LookupInviteByCodeHash(context.Background(), tx, "nonexistent_hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invite != nil {
		t.Error("expected nil invite for nonexistent hash")
	}
}
