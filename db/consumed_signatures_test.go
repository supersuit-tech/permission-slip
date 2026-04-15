package db_test

import (
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// TestConsumeSignature_FirstInsertSucceeds verifies the happy path: a fresh
// signature hash is recorded and inserted=true is returned.
func TestConsumeSignature_FirstInsertSucceeds(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") // 32 bytes
	expiresAt := time.Now().Add(10 * time.Minute)

	inserted, err := db.ConsumeSignature(t.Context(), tx, hash, 42, expiresAt)
	if err != nil {
		t.Fatalf("ConsumeSignature: %v", err)
	}
	if !inserted {
		t.Fatal("expected inserted=true for first insert, got false")
	}
}

// TestConsumeSignature_DuplicateRejected verifies the replay-prevention core:
// inserting the same signature hash twice returns inserted=false on the second
// call. The caller treats this as a replay attempt.
func TestConsumeSignature_DuplicateRejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	hash := []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	expiresAt := time.Now().Add(10 * time.Minute)

	inserted, err := db.ConsumeSignature(t.Context(), tx, hash, 1, expiresAt)
	if err != nil {
		t.Fatalf("first ConsumeSignature: %v", err)
	}
	if !inserted {
		t.Fatal("expected first insert to succeed")
	}

	inserted, err = db.ConsumeSignature(t.Context(), tx, hash, 1, expiresAt)
	if err != nil {
		t.Fatalf("second ConsumeSignature: %v", err)
	}
	if inserted {
		t.Fatal("expected duplicate insert to return inserted=false, got true")
	}
}

// TestConsumeSignature_DistinctHashes verifies that different signature hashes
// do not collide — the primary key is on signature_hash only, so two agents
// (or the same agent with different signatures) coexist freely.
func TestConsumeSignature_DistinctHashes(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	hashA := []byte("cccccccccccccccccccccccccccccccc")
	hashB := []byte("dddddddddddddddddddddddddddddddd")
	expiresAt := time.Now().Add(10 * time.Minute)

	insertedA, err := db.ConsumeSignature(t.Context(), tx, hashA, 1, expiresAt)
	if err != nil {
		t.Fatalf("ConsumeSignature A: %v", err)
	}
	insertedB, err := db.ConsumeSignature(t.Context(), tx, hashB, 2, expiresAt)
	if err != nil {
		t.Fatalf("ConsumeSignature B: %v", err)
	}
	if !insertedA || !insertedB {
		t.Fatalf("expected both distinct hashes to insert successfully, got A=%v B=%v", insertedA, insertedB)
	}
}

// TestConsumeSignature_ZeroAgentIDForRegistration verifies that agent_id=0 is
// accepted, which is the placeholder used for pre-registration requests where
// no agent row exists yet.
func TestConsumeSignature_ZeroAgentIDForRegistration(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	hash := []byte("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	expiresAt := time.Now().Add(10 * time.Minute)

	inserted, err := db.ConsumeSignature(t.Context(), tx, hash, 0, expiresAt)
	if err != nil {
		t.Fatalf("ConsumeSignature: %v", err)
	}
	if !inserted {
		t.Fatal("expected insert with agent_id=0 to succeed")
	}
}

// TestCleanupExpiredConsumedSignatures_DeletesExpiredPreservesFuture verifies
// that the cleanup query removes only rows whose expires_at has already passed
// and leaves future rows intact.
func TestCleanupExpiredConsumedSignatures_DeletesExpiredPreservesFuture(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	expiredHashes := [][]byte{
		[]byte("ffffffffffffffffffffffffffffffff"),
		[]byte("gggggggggggggggggggggggggggggggg"),
	}
	futureHashes := [][]byte{
		[]byte("hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh"),
	}

	past := time.Now().Add(-10 * time.Minute)
	future := time.Now().Add(10 * time.Minute)

	for _, h := range expiredHashes {
		if _, err := db.ConsumeSignature(t.Context(), tx, h, 1, past); err != nil {
			t.Fatalf("seed expired: %v", err)
		}
	}
	for _, h := range futureHashes {
		if _, err := db.ConsumeSignature(t.Context(), tx, h, 1, future); err != nil {
			t.Fatalf("seed future: %v", err)
		}
	}

	deleted, err := db.CleanupExpiredConsumedSignatures(t.Context(), tx)
	if err != nil {
		t.Fatalf("CleanupExpiredConsumedSignatures: %v", err)
	}
	if deleted != int64(len(expiredHashes)) {
		t.Errorf("expected %d rows deleted, got %d", len(expiredHashes), deleted)
	}

	// After cleanup, the future hash should still collide (row survives).
	inserted, err := db.ConsumeSignature(t.Context(), tx, futureHashes[0], 1, future)
	if err != nil {
		t.Fatalf("re-consume future: %v", err)
	}
	if inserted {
		t.Error("expected future-expiry row to survive cleanup (re-insert should hit conflict)")
	}

	// The expired hashes should now be insertable again (row gone).
	for _, h := range expiredHashes {
		inserted, err := db.ConsumeSignature(t.Context(), tx, h, 1, future)
		if err != nil {
			t.Fatalf("re-consume expired: %v", err)
		}
		if !inserted {
			t.Errorf("expected expired row to be gone after cleanup (re-insert should succeed)")
		}
	}
}

// TestCleanupExpiredConsumedSignatures_EmptyTable verifies the no-op case
// returns zero deleted without error.
func TestCleanupExpiredConsumedSignatures_EmptyTable(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deleted, err := db.CleanupExpiredConsumedSignatures(t.Context(), tx)
	if err != nil {
		t.Fatalf("CleanupExpiredConsumedSignatures: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 rows deleted from empty table, got %d", deleted)
	}
}
