package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// ── Phase 2: Expiration & Isolation (DB-level) ─────────────────────────────

// TestInviteExpirationBoundary creates an invite with a short TTL and verifies
// that ConsumeInvite respects the expires_at > now() check. We backdate
// expires_at to just past expiration and confirm the invite cannot be consumed.
func TestInviteExpirationBoundary(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// Create invite with a normal TTL.
	riID := testhelper.GenerateID(t, "ri_")
	hash := testhelper.GenerateID(t, "hash_")
	_, err := db.CreateRegistrationInvite(ctx, tx, riID, uid, hash, 900)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	// Backdate expires_at to 1 second in the past — simulates "just expired".
	testhelper.MustExec(t, tx,
		`UPDATE registration_invites SET expires_at = now() - interval '1 second' WHERE id = $1`, riID)

	// ConsumeInvite should return nil (no match) because expires_at <= now().
	invite, err := db.ConsumeInvite(ctx, tx, hash)
	if err != nil {
		t.Fatalf("consume invite: unexpected error: %v", err)
	}
	if invite != nil {
		t.Error("expected nil (expired invite should not be consumed), got non-nil")
	}

	// Verify via LookupInviteByCodeHash that the invite still exists but is expired.
	existing, err := db.LookupInviteByCodeHash(ctx, tx, hash)
	if err != nil {
		t.Fatalf("lookup invite: %v", err)
	}
	if existing == nil {
		t.Fatal("expected to find the invite via lookup")
	}
	if existing.Status != "active" {
		t.Errorf("expected status 'active' (not yet consumed), got %q", existing.Status)
	}

	// Also verify that an invite right at the boundary (expires_at = now()) is rejected.
	// Set expires_at to exactly now() — the check is expires_at > now(), so equal should fail.
	riID2 := testhelper.GenerateID(t, "ri_")
	hash2 := testhelper.GenerateID(t, "hash_")
	_, err = db.CreateRegistrationInvite(ctx, tx, riID2, uid, hash2, 900)
	if err != nil {
		t.Fatalf("create invite 2: %v", err)
	}
	testhelper.MustExec(t, tx,
		`UPDATE registration_invites SET expires_at = now() WHERE id = $1`, riID2)

	invite2, err := db.ConsumeInvite(ctx, tx, hash2)
	if err != nil {
		t.Fatalf("consume invite 2: unexpected error: %v", err)
	}
	if invite2 != nil {
		t.Error("expected nil when expires_at = now() (boundary: > not >=), got non-nil")
	}

	// Finally, confirm that a non-expired invite CAN still be consumed.
	riID3 := testhelper.GenerateID(t, "ri_")
	hash3 := testhelper.GenerateID(t, "hash_")
	_, err = db.CreateRegistrationInvite(ctx, tx, riID3, uid, hash3, 900)
	if err != nil {
		t.Fatalf("create invite 3: %v", err)
	}
	invite3, err := db.ConsumeInvite(ctx, tx, hash3)
	if err != nil {
		t.Fatalf("consume invite 3: %v", err)
	}
	if invite3 == nil {
		t.Error("expected non-nil (valid invite should be consumed)")
	}
}

// TestAgentRegistrationTTLBoundary creates a pending agent with a TTL, then
// backdates expires_at to simulate expiration and verifies that
// VerifyAgentConfirmationCode returns ErrRegistrationExpired.
func TestAgentRegistrationTTLBoundary(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	agent, err := db.InsertPendingAgent(ctx, tx,
		uid, "ssh-ed25519 AAAA_ttl_boundary", "AA1-BB2", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent: %v", err)
	}

	// Verify works before expiration.
	// First, check that the agent is pending and not expired.
	fetched, err := db.GetAgentByIDUnscoped(ctx, tx, agent.AgentID)
	if err != nil {
		t.Fatalf("GetAgentByIDUnscoped: %v", err)
	}
	if fetched.Status != "pending" {
		t.Fatalf("expected pending, got %q", fetched.Status)
	}
	if fetched.ExpiresAt == nil {
		t.Fatal("expected non-nil expires_at")
	}

	// Backdate expires_at to 1 second in the past.
	testhelper.MustExec(t, tx,
		`UPDATE agents SET expires_at = now() - interval '1 second' WHERE agent_id = $1`, agent.AgentID)

	// Verification should fail with ErrRegistrationExpired.
	_, err = db.VerifyAgentConfirmationCode(ctx, tx, agent.AgentID, "AA1BB2")
	if err != db.ErrRegistrationExpired {
		t.Errorf("expected ErrRegistrationExpired, got %v", err)
	}

	// Also test boundary: expires_at = now() should fail (check is > not >=).
	testhelper.MustExec(t, tx,
		`UPDATE agents SET expires_at = now(), verification_attempts = 0 WHERE agent_id = $1`, agent.AgentID)

	_, err = db.VerifyAgentConfirmationCode(ctx, tx, agent.AgentID, "AA1BB2")
	if err != db.ErrRegistrationExpired {
		t.Errorf("at boundary (expires_at = now()): expected ErrRegistrationExpired, got %v", err)
	}

	// Verify that a non-expired agent CAN still be verified.
	agent2, err := db.InsertPendingAgent(ctx, tx,
		uid, "ssh-ed25519 AAAA_ttl_boundary_2", "CC3-DD4", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent 2: %v", err)
	}
	registered, err := db.VerifyAgentConfirmationCode(ctx, tx, agent2.AgentID, "CC3DD4")
	if err != nil {
		t.Fatalf("verification of non-expired agent should succeed: %v", err)
	}
	if registered.Status != "registered" {
		t.Errorf("expected status 'registered', got %q", registered.Status)
	}
}

// TestLockoutVsExpirationPrecedence makes 5 failed verification attempts, then
// lets the TTL expire, and verifies which error takes precedence. The lockout
// check (verification_attempts >= 5) should take precedence since the atomic
// UPDATE fails on both conditions; the diagnosis step checks status, then
// expiration, then lockout — but lockout is checked last. This test documents
// the actual behavior.
func TestLockoutVsExpirationPrecedence(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	agent, err := db.InsertPendingAgent(ctx, tx,
		uid, "ssh-ed25519 AAAA_lockout_exp", "EE5-FF6", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent: %v", err)
	}

	// Make 5 failed attempts to trigger lockout.
	for i := 0; i < 5; i++ {
		_, verifyErr := db.VerifyAgentConfirmationCode(ctx, tx, agent.AgentID, "ZZZZZZ")
		if verifyErr != db.ErrInvalidConfirmation {
			t.Fatalf("attempt %d: expected ErrInvalidConfirmation, got %v", i+1, verifyErr)
		}
	}

	// Verify lockout is in effect.
	_, err = db.VerifyAgentConfirmationCode(ctx, tx, agent.AgentID, "EE5FF6")
	if err != db.ErrVerificationLocked {
		t.Fatalf("expected ErrVerificationLocked after 5 attempts, got %v", err)
	}

	// Now also expire the TTL.
	testhelper.MustExec(t, tx,
		`UPDATE agents SET expires_at = now() - interval '1 hour' WHERE agent_id = $1`, agent.AgentID)

	// Both conditions are now true: locked out AND expired.
	// The diagnosis function (diagnosePendingAgent) checks:
	//   1. status != 'pending' → ErrAgentNotPending
	//   2. expires_at < now() → ErrRegistrationExpired
	//   3. attempts >= 5 → ErrVerificationLocked
	// Since the agent is still 'pending' and expired, ErrRegistrationExpired
	// should take precedence over ErrVerificationLocked.
	_, err = db.VerifyAgentConfirmationCode(ctx, tx, agent.AgentID, "EE5FF6")
	if err != db.ErrRegistrationExpired {
		t.Errorf("expected ErrRegistrationExpired (expiration takes precedence over lockout), got %v", err)
	}
}

// TestConfirmationCodeIsolation verifies that agent A's confirmation code
// cannot be used to verify agent B. Codes are agent-specific, not global.
func TestConfirmationCodeIsolation(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	// Create two pending agents with different confirmation codes.
	agentA, err := db.InsertPendingAgent(ctx, tx,
		uid, "ssh-ed25519 AAAA_iso_a", "AA1-BB2", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent A: %v", err)
	}
	agentB, err := db.InsertPendingAgent(ctx, tx,
		uid, "ssh-ed25519 AAAA_iso_b", "CC3-DD4", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent B: %v", err)
	}

	// Try to verify agent B using agent A's code — should fail.
	_, err = db.VerifyAgentConfirmationCode(ctx, tx, agentB.AgentID, "AA1BB2")
	if err != db.ErrInvalidConfirmation {
		t.Errorf("using agent A's code on agent B: expected ErrInvalidConfirmation, got %v", err)
	}

	// Try to verify agent A using agent B's code — should fail.
	_, err = db.VerifyAgentConfirmationCode(ctx, tx, agentA.AgentID, "CC3DD4")
	if err != db.ErrInvalidConfirmation {
		t.Errorf("using agent B's code on agent A: expected ErrInvalidConfirmation, got %v", err)
	}

	// Verify each agent with its own code — should succeed.
	regA, err := db.VerifyAgentConfirmationCode(ctx, tx, agentA.AgentID, "AA1BB2")
	if err != nil {
		t.Fatalf("verify agent A with own code: %v", err)
	}
	if regA.Status != "registered" {
		t.Errorf("agent A: expected 'registered', got %q", regA.Status)
	}

	regB, err := db.VerifyAgentConfirmationCode(ctx, tx, agentB.AgentID, "CC3DD4")
	if err != nil {
		t.Fatalf("verify agent B with own code: %v", err)
	}
	if regB.Status != "registered" {
		t.Errorf("agent B: expected 'registered', got %q", regB.Status)
	}
}
