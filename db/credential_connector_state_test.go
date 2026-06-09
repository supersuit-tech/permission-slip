package db_test

import (
	"context"
	"testing"

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
