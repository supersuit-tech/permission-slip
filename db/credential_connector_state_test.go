package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestProtonmailUIDValidityRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertCredential(t, tx, credID, uid, "protonmail")

	_, known, err := db.GetProtonmailUIDValidity(ctx, tx, credID, "INBOX")
	if err != nil {
		t.Fatalf("GetProtonmailUIDValidity: %v", err)
	}
	if known {
		t.Fatal("expected no stored UIDVALIDITY initially")
	}

	if err := db.SetProtonmailUIDValidity(ctx, tx, credID, "INBOX", 12345); err != nil {
		t.Fatalf("SetProtonmailUIDValidity: %v", err)
	}

	got, known, err := db.GetProtonmailUIDValidity(ctx, tx, credID, "INBOX")
	if err != nil {
		t.Fatalf("GetProtonmailUIDValidity after set: %v", err)
	}
	if !known || got != 12345 {
		t.Fatalf("got validity %d (known=%v), want 12345", got, known)
	}

	if err := db.SetProtonmailUIDValidity(ctx, tx, credID, "Archive", 99); err != nil {
		t.Fatalf("SetProtonmailUIDValidity Archive: %v", err)
	}
	got, known, err = db.GetProtonmailUIDValidity(ctx, tx, credID, "INBOX")
	if err != nil || !known || got != 12345 {
		t.Fatalf("INBOX validity should be unchanged, got %d (known=%v) err=%v", got, known, err)
	}
}

func TestProtonmailHealthRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertCredential(t, tx, credID, uid, "protonmail")

	health, err := db.GetProtonmailHealth(ctx, tx, credID)
	if err != nil {
		t.Fatalf("GetProtonmailHealth: %v", err)
	}
	if health != nil {
		t.Fatalf("expected no health initially, got %+v", health)
	}

	checkedAt, err := time.Parse(time.RFC3339, "2026-06-09T12:00:00Z")
	if err != nil {
		t.Fatalf("time.Parse: %v", err)
	}
	if err := db.SetProtonmailHealth(ctx, tx, credID, db.ProtonmailHealthState{
		Status:    db.ProtonmailHealthOK,
		CheckedAt: checkedAt,
	}); err != nil {
		t.Fatalf("SetProtonmailHealth: %v", err)
	}

	health, err = db.GetProtonmailHealth(ctx, tx, credID)
	if err != nil {
		t.Fatalf("GetProtonmailHealth after set: %v", err)
	}
	if health == nil || health.Status != db.ProtonmailHealthOK {
		t.Fatalf("got %+v, want status ok", health)
	}

	if err := db.SetProtonmailUIDValidity(ctx, tx, credID, "INBOX", 42); err != nil {
		t.Fatalf("SetProtonmailUIDValidity: %v", err)
	}
	health, err = db.GetProtonmailHealth(ctx, tx, credID)
	if err != nil || health == nil || health.Status != db.ProtonmailHealthOK {
		t.Fatalf("health should survive UIDVALIDITY update, got %+v err=%v", health, err)
	}
}
