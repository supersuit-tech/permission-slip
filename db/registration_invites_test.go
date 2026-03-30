package db_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestRegistrationInvitesSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "registration_invites", []string{
		"id", "user_id", "invite_code_hash", "status",
		"verification_attempts", "expires_at", "consumed_at", "created_at",
	})
}

func TestRegistrationInvitesIndex(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireIndex(t, tx, "registration_invites", "idx_registration_invites_user_status")
}

func TestRegistrationInvitesCascadeDeleteOnProfileDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertRegistrationInvite(t, tx, testhelper.GenerateID(t, "ri_"), uid)

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM profiles WHERE id = '"+uid+"'",
		[]string{"registration_invites"},
		"user_id = '"+uid+"'",
	)
}

func TestRegistrationInvitesStatusCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	base := testhelper.GenerateID(t, "ri_")

	// Test valid values: active, consumed, expired
	// Test multiple invalid values: pending, revoked, empty string
	validValues := []string{"active", "consumed", "expired"}
	invalidValues := []string{"pending", "revoked", ""}

	testhelper.RequireCheckValues(t, tx, "status",
		validValues, invalidValues[0],
		func(value string, i int) error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
				 VALUES ($1, $2, $3, $4, now() + interval '1 hour')`,
				fmt.Sprintf("%s_s_%d", base, i), uid, fmt.Sprintf("hash_%s_%d", base, i), value)
			return err
		})

	// Also verify "revoked" and empty string are rejected
	for i, invalid := range invalidValues[1:] {
		err := testhelper.WithSavepoint(t, tx, func() error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
				 VALUES ($1, $2, $3, $4, now() + interval '1 hour')`,
				fmt.Sprintf("%s_inv_%d", base, i), uid, fmt.Sprintf("hash_%s_inv_%d", base, i), invalid)
			return err
		})
		if err == nil {
			t.Errorf("expected CHECK constraint violation for invalid status %q, but insert succeeded", invalid)
		}
	}
}

func TestRegistrationInvitesVerificationAttemptsCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	riID := testhelper.GenerateID(t, "ri_")

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertRegistrationInvite(t, tx, riID, uid)

	// Setting verification_attempts to a negative value should fail
	err := testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(context.Background(),
			`UPDATE registration_invites SET verification_attempts = -1 WHERE id = $1`, riID)
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for negative verification_attempts, but update succeeded")
	}
}

func TestRegistrationInvitesIDLengthCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	hashBase := testhelper.GenerateID(t, "hash_")

	// ID with exactly 255 characters should succeed (unique prefix + padding)
	prefix := testhelper.GenerateID(t, "")
	id255 := prefix + strings.Repeat("a", 255-len(prefix))
	_, err := tx.Exec(context.Background(),
		`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
		 VALUES ($1, $2, $3, 'active', now() + interval '1 hour')`,
		id255, uid, hashBase+"_255")
	if err != nil {
		t.Errorf("id with 255 chars was rejected: %v", err)
	}

	// ID with 256 characters should fail
	prefix2 := testhelper.GenerateID(t, "")
	id256 := prefix2 + strings.Repeat("b", 256-len(prefix2))
	err = testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(context.Background(),
			`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
			 VALUES ($1, $2, $3, 'active', now() + interval '1 hour')`,
			id256, uid, hashBase+"_256")
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for id with 256 chars, but insert succeeded")
	}
}

func TestRegistrationInvitesInviteCodeHashLengthCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	idBase := testhelper.GenerateID(t, "ri_")

	// Hash with exactly 128 characters should succeed (unique prefix + padding)
	prefix := testhelper.GenerateID(t, "")
	hash128 := prefix + strings.Repeat("a", 128-len(prefix))
	_, err := tx.Exec(context.Background(),
		`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
		 VALUES ($1, $2, $3, 'active', now() + interval '1 hour')`,
		idBase+"_ok", uid, hash128)
	if err != nil {
		t.Errorf("invite_code_hash with 128 chars was rejected: %v", err)
	}

	// Hash with 129 characters should fail
	prefix2 := testhelper.GenerateID(t, "")
	hash129 := prefix2 + strings.Repeat("b", 129-len(prefix2))
	err = testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(context.Background(),
			`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
			 VALUES ($1, $2, $3, 'active', now() + interval '1 hour')`,
			idBase+"_fail", uid, hash129)
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for invite_code_hash with 129 chars, but insert succeeded")
	}
}

func TestRegistrationInvitesInviteCodeHashUnique(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	hash := testhelper.GenerateID(t, "hash_")
	idA := testhelper.GenerateID(t, "ri_")
	idB := testhelper.GenerateID(t, "ri_")

	testhelper.RequireUniqueViolation(t, tx, "invite_code_hash",
		func() error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
				 VALUES ($1, $2, $3, 'active', now() + interval '1 hour')`,
				idA, uid, hash)
			return err
		},
		func() error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
				 VALUES ($1, $2, $3, 'active', now() + interval '1 hour')`,
				idB, uid, hash)
			return err
		})
}

func TestRegistrationInvitesPgCronJob(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequirePgCronJob(t, tx, "cleanup_expired_invites")
}

func TestCreateRegistrationInvite_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	riID := testhelper.GenerateID(t, "ri_")
	hash := testhelper.GenerateID(t, "hash_")

	ri, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, hash, 900)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ri.ID != riID {
		t.Errorf("expected ID %q, got %q", riID, ri.ID)
	}
	if ri.UserID != uid {
		t.Errorf("expected UserID %q, got %q", uid, ri.UserID)
	}
	if ri.InviteCodeHash != hash {
		t.Errorf("expected InviteCodeHash %q, got %q", hash, ri.InviteCodeHash)
	}
	if ri.Status != "active" {
		t.Errorf("expected status 'active', got %q", ri.Status)
	}
	if ri.VerificationAttempts != 0 {
		t.Errorf("expected 0 verification_attempts, got %d", ri.VerificationAttempts)
	}
	if ri.ConsumedAt != nil {
		t.Errorf("expected nil consumed_at, got %v", ri.ConsumedAt)
	}
	if ri.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}

	expectedExpiry := ri.CreatedAt.Add(900 * time.Second)
	if diff := ri.ExpiresAt.Sub(expectedExpiry); diff < -2*time.Second || diff > 2*time.Second {
		t.Errorf("expires_at not ~900s after created_at: created=%v, expires=%v", ri.CreatedAt, ri.ExpiresAt)
	}
}

func TestCreateRegistrationInvite_DuplicateID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	riID := testhelper.GenerateID(t, "ri_")

	_, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, testhelper.GenerateID(t, "hash_"), 900)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	err = testhelper.WithSavepoint(t, tx, func() error {
		_, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, testhelper.GenerateID(t, "hash_"), 900)
		return err
	})
	if err == nil {
		t.Error("expected error on duplicate ID, got nil")
	}
}

func TestCreateRegistrationInvite_DuplicateHash(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	hash := testhelper.GenerateID(t, "hash_")

	_, err := db.CreateRegistrationInvite(context.Background(), tx, testhelper.GenerateID(t, "ri_"), uid, hash, 900)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	err = testhelper.WithSavepoint(t, tx, func() error {
		_, err := db.CreateRegistrationInvite(context.Background(), tx, testhelper.GenerateID(t, "ri_"), uid, hash, 900)
		return err
	})
	if err == nil {
		t.Error("expected error on duplicate invite_code_hash, got nil")
	}
}

func TestCountRecentInvitesByUser_CountsWithinWindow(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// No invites yet — count should be 0.
	count, err := db.CountRecentInvitesByUser(context.Background(), tx, uid, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	// Create 3 invites.
	for i := 0; i < 3; i++ {
		_, err := db.CreateRegistrationInvite(context.Background(), tx,
			testhelper.GenerateID(t, "ri_"), uid, testhelper.GenerateID(t, "hash_"), 900)
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	count, err = db.CountRecentInvitesByUser(context.Background(), tx, uid, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestCountRecentInvitesByUser_ExcludesOutsideWindow(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Insert an invite, then backdate it to 2 hours ago.
	riID := testhelper.GenerateID(t, "ri_")
	_, err := db.CreateRegistrationInvite(context.Background(), tx,
		riID, uid, testhelper.GenerateID(t, "hash_"), 900)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err = tx.Exec(context.Background(),
		`UPDATE registration_invites SET created_at = now() - interval '2 hours' WHERE id = $1`, riID)
	if err != nil {
		t.Fatalf("backdate: %v", err)
	}

	// With a 1-hour window, the backdated invite should not be counted.
	count, err := db.CountRecentInvitesByUser(context.Background(), tx, uid, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 (invite is outside window), got %d", count)
	}
}

func TestCountRecentInvitesByUser_FiltersByUserID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	userA := testhelper.GenerateUID(t)
	userB := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, userA, "u_"+userA[:8])
	testhelper.InsertUser(t, tx, userB, "u_"+userB[:8])

	// Create 2 invites for user A and 1 for user B.
	for i := 0; i < 2; i++ {
		_, err := db.CreateRegistrationInvite(context.Background(), tx,
			testhelper.GenerateID(t, "ri_"), userA, testhelper.GenerateID(t, "hash_"), 900)
		if err != nil {
			t.Fatalf("insert userA %d: %v", i, err)
		}
	}
	_, err := db.CreateRegistrationInvite(context.Background(), tx,
		testhelper.GenerateID(t, "ri_"), userB, testhelper.GenerateID(t, "hash_"), 900)
	if err != nil {
		t.Fatalf("insert userB: %v", err)
	}

	countA, err := db.CountRecentInvitesByUser(context.Background(), tx, userA, time.Hour)
	if err != nil {
		t.Fatalf("count userA: %v", err)
	}
	if countA != 2 {
		t.Errorf("userA: expected 2, got %d", countA)
	}

	countB, err := db.CountRecentInvitesByUser(context.Background(), tx, userB, time.Hour)
	if err != nil {
		t.Fatalf("count userB: %v", err)
	}
	if countB != 1 {
		t.Errorf("userB: expected 1, got %d", countB)
	}
}
