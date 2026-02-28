package db_test

import (
	"errors"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestConsumeToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := t.Context()

	jti := testhelper.GenerateID(t, "tok_")

	// First consumption should succeed.
	if err := db.ConsumeToken(ctx, tx, jti); err != nil {
		t.Fatalf("first ConsumeToken: unexpected error: %v", err)
	}

	// Second consumption (replay) should return already_consumed.
	err := db.ConsumeToken(ctx, tx, jti)
	if err == nil {
		t.Fatal("second ConsumeToken: expected error, got nil")
	}
	var ctErr *db.ConsumedTokenError
	if !errors.As(err, &ctErr) {
		t.Fatalf("second ConsumeToken: expected ConsumedTokenError, got %T: %v", err, err)
	}
	if ctErr.Code != db.ConsumedTokenErrAlreadyConsumed {
		t.Errorf("expected code %q, got %q", db.ConsumedTokenErrAlreadyConsumed, ctErr.Code)
	}
}

func TestConsumeToken_DifferentJTIs(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := t.Context()

	jti1 := testhelper.GenerateID(t, "tok_")
	jti2 := testhelper.GenerateID(t, "tok_")

	// Two different JTIs should both succeed.
	if err := db.ConsumeToken(ctx, tx, jti1); err != nil {
		t.Fatalf("ConsumeToken jti1: %v", err)
	}
	if err := db.ConsumeToken(ctx, tx, jti2); err != nil {
		t.Fatalf("ConsumeToken jti2: %v", err)
	}
}

func TestIsTokenConsumed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := t.Context()

	jti := testhelper.GenerateID(t, "tok_")

	// Before consumption, should return false.
	consumed, err := db.IsTokenConsumed(ctx, tx, jti)
	if err != nil {
		t.Fatalf("IsTokenConsumed before insert: %v", err)
	}
	if consumed {
		t.Error("expected false before consumption")
	}

	// Consume the token.
	if err := db.ConsumeToken(ctx, tx, jti); err != nil {
		t.Fatalf("ConsumeToken: %v", err)
	}

	// After consumption, should return true.
	consumed, err = db.IsTokenConsumed(ctx, tx, jti)
	if err != nil {
		t.Fatalf("IsTokenConsumed after insert: %v", err)
	}
	if !consumed {
		t.Error("expected true after consumption")
	}
}

func TestIsTokenConsumed_NonExistentJTI(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := t.Context()

	consumed, err := db.IsTokenConsumed(ctx, tx, "nonexistent_jti_12345")
	if err != nil {
		t.Fatalf("IsTokenConsumed: %v", err)
	}
	if consumed {
		t.Error("expected false for non-existent JTI")
	}
}
