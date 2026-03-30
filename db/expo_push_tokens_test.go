package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestExpoPushTokensSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "expo_push_tokens", []string{
		"id", "user_id", "token", "created_at",
	})
}

func TestExpoPushTokensCascadeDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	_, err := db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[test123]")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM auth.users WHERE id = '"+uid+"'",
		[]string{"expo_push_tokens"},
		"user_id = '"+uid+"'",
	)
}

func TestUpsertExpoPushToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()
	tok, err := db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[abc123]")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if tok.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if tok.Token != "ExponentPushToken[abc123]" {
		t.Errorf("expected token, got %q", tok.Token)
	}
	if tok.UserID != uid {
		t.Errorf("expected user_id %q, got %q", uid, tok.UserID)
	}
}

func TestUpsertExpoPushToken_Idempotent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// Insert
	_, err := db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[abc123]")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Upsert same token again — should not create a duplicate
	_, err = db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[abc123]")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	// Should have only 1 token
	tokens, err := db.ListExpoPushTokensByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tokens) != 1 {
		t.Errorf("expected 1 token after duplicate upsert, got %d", len(tokens))
	}
}

func TestListExpoPushTokensByUserID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// Empty initially
	tokens, err := db.ListExpoPushTokensByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}

	// Insert two
	_, err = db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[device1]")
	if err != nil {
		t.Fatalf("upsert 1: %v", err)
	}
	_, err = db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[device2]")
	if err != nil {
		t.Fatalf("upsert 2: %v", err)
	}

	tokens, err = db.ListExpoPushTokensByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
	}
}

func TestDeleteExpoPushToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	tok, err := db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[device1]")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	deleted, err := db.DeleteExpoPushToken(ctx, tx, uid, tok.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}

	// Should be gone
	tokens, err := db.ListExpoPushTokensByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens after delete, got %d", len(tokens))
	}
}

func TestDeleteExpoPushToken_WrongUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u_"+uid2[:8])

	ctx := context.Background()

	tok, err := db.UpsertExpoPushToken(ctx, tx, uid1, "ExponentPushToken[device1]")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// user2 shouldn't be able to delete user1's token
	deleted, err := db.DeleteExpoPushToken(ctx, tx, uid2, tok.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted {
		t.Error("expected deleted=false for wrong user")
	}
}

func TestDeleteExpoPushTokenByToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	_, err := db.UpsertExpoPushToken(ctx, tx, uid, "ExponentPushToken[invalid]")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	err = db.DeleteExpoPushTokenByToken(ctx, tx, "ExponentPushToken[invalid]")
	if err != nil {
		t.Fatalf("delete by token: %v", err)
	}

	tokens, err := db.ListExpoPushTokensByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens after token delete, got %d", len(tokens))
	}
}
