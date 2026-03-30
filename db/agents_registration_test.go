package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestInsertPendingAgent_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	agent, err := db.InsertPendingAgent(context.Background(), tx,
		uid, "ssh-ed25519 AAAA_test_key", "XK7-M9P", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent: %v", err)
	}
	if agent.AgentID <= 0 {
		t.Errorf("expected positive agent_id, got %d", agent.AgentID)
	}
	if agent.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", agent.Status)
	}
	if agent.PublicKey != "ssh-ed25519 AAAA_test_key" {
		t.Errorf("expected public key 'ssh-ed25519 AAAA_test_key', got %q", agent.PublicKey)
	}
	if agent.ApproverID != uid {
		t.Errorf("expected approver_id %q, got %q", uid, agent.ApproverID)
	}
	if agent.ConfirmationCode == nil || *agent.ConfirmationCode != "XK7-M9P" {
		t.Errorf("expected confirmation_code 'XK7-M9P', got %v", agent.ConfirmationCode)
	}
	if agent.ExpiresAt == nil {
		t.Error("expected non-nil expires_at")
	}
	if agent.RegisteredAt != nil {
		t.Error("expected nil registered_at for pending agent")
	}
}

func TestInsertPendingAgent_WithMetadata(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	metadata := []byte(`{"name":"test-agent","version":"1.0.0"}`)
	agent, err := db.InsertPendingAgent(context.Background(), tx,
		uid, "ssh-ed25519 AAAA_test_key_2", "AB2-CD3", 300, metadata)
	if err != nil {
		t.Fatalf("InsertPendingAgent: %v", err)
	}
	if len(agent.Metadata) == 0 {
		t.Error("expected non-empty metadata")
	}
}

func TestVerifyAgentConfirmationCode_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	agent, err := db.InsertPendingAgent(context.Background(), tx,
		uid, "ssh-ed25519 AAAA_test_key_3", "XX3-YY4", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent: %v", err)
	}

	// Submit normalized code (uppercase, no hyphens) — matches stored "XX3-YY4".
	registered, err := db.VerifyAgentConfirmationCode(context.Background(), tx, agent.AgentID, "XX3YY4")
	if err != nil {
		t.Fatalf("VerifyAgentConfirmationCode: %v", err)
	}
	if registered == nil {
		t.Fatal("expected non-nil registered agent")
	}
	if registered.Status != "registered" {
		t.Errorf("expected status 'registered', got %q", registered.Status)
	}
	if registered.RegisteredAt == nil {
		t.Error("expected non-nil registered_at")
	}
	if registered.ConfirmationCode != nil {
		t.Error("expected nil confirmation_code after registration")
	}
}

func TestVerifyAgentConfirmationCode_WrongCode(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	agent, err := db.InsertPendingAgent(context.Background(), tx,
		uid, "ssh-ed25519 AAAA_test_key_4", "AA2-BB3", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent: %v", err)
	}

	result, err := db.VerifyAgentConfirmationCode(context.Background(), tx, agent.AgentID, "ZZZZZZ")
	if err == nil {
		t.Fatal("expected error for wrong code")
	}
	if err != db.ErrInvalidConfirmation {
		t.Errorf("expected ErrInvalidConfirmation, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result even on failure")
	}
	if result.VerificationAttempts != 1 {
		t.Errorf("expected 1 verification attempt, got %d", result.VerificationAttempts)
	}
}

func TestVerifyAgentConfirmationCode_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	result, err := db.VerifyAgentConfirmationCode(context.Background(), tx, 999999, "ABCDEF")
	if err != nil {
		t.Fatalf("expected nil error for not found, got %v", err)
	}
	if result != nil {
		t.Error("expected nil result for nonexistent agent")
	}
}

func TestVerifyAgentConfirmationCode_Lockout(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	agent, err := db.InsertPendingAgent(context.Background(), tx,
		uid, "ssh-ed25519 AAAA_test_key_5", "CC4-DD5", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent: %v", err)
	}

	// Make 5 failed attempts.
	for i := 0; i < 5; i++ {
		_, verifyErr := db.VerifyAgentConfirmationCode(context.Background(), tx, agent.AgentID, "ZZZZZZ")
		if verifyErr != db.ErrInvalidConfirmation {
			t.Fatalf("attempt %d: expected ErrInvalidConfirmation, got %v", i+1, verifyErr)
		}
	}

	// 6th attempt should return ErrVerificationLocked.
	_, err = db.VerifyAgentConfirmationCode(context.Background(), tx, agent.AgentID, "CC4DD5")
	if err != db.ErrVerificationLocked {
		t.Errorf("expected ErrVerificationLocked, got %v", err)
	}
}

func TestVerifyAgentConfirmationCode_Expired(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	agent, err := db.InsertPendingAgent(context.Background(), tx,
		uid, "ssh-ed25519 AAAA_test_key_6", "EE5-FF6", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent: %v", err)
	}

	// Backdate expires_at to the past.
	testhelper.MustExec(t, tx, `UPDATE agents SET expires_at = now() - interval '1 hour' WHERE agent_id = $1`, agent.AgentID)

	_, err = db.VerifyAgentConfirmationCode(context.Background(), tx, agent.AgentID, "EE5FF6")
	if err != db.ErrRegistrationExpired {
		t.Errorf("expected ErrRegistrationExpired, got %v", err)
	}
}

func TestVerifyAgentConfirmationCode_AlreadyRegistered(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	agent, err := db.InsertPendingAgent(context.Background(), tx,
		uid, "ssh-ed25519 AAAA_test_key_7", "GG6-HH7", 300, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent: %v", err)
	}

	// First verify should succeed.
	_, err = db.VerifyAgentConfirmationCode(context.Background(), tx, agent.AgentID, "GG6HH7")
	if err != nil {
		t.Fatalf("first verify: %v", err)
	}

	// Second verify should return ErrAgentNotPending.
	_, err = db.VerifyAgentConfirmationCode(context.Background(), tx, agent.AgentID, "GG6HH7")
	if err != db.ErrAgentNotPending {
		t.Errorf("expected ErrAgentNotPending, got %v", err)
	}
}
