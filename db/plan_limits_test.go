package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// --- CountRegisteredAgentsByUser ---

func TestCountRegisteredAgentsByUser_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "count_agents_empty")

	count, err := db.CountRegisteredAgentsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("CountRegisteredAgentsByUser: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestCountRegisteredAgentsByUser_CountsRegisteredAndPending(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "count_agents_mix")

	// 1 registered agent
	testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	// 1 pending (not expired) agent
	testhelper.InsertAgent(t, tx, uid)
	// 1 deactivated agent (should NOT count)
	testhelper.InsertAgentWithStatus(t, tx, uid, "deactivated")

	count, err := db.CountRegisteredAgentsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("CountRegisteredAgentsByUser: %v", err)
	}
	// pending without expires_at counts (expires_at IS NULL → counted)
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestCountRegisteredAgentsByUser_ExcludesExpiredPending(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "count_agents_expired")

	// 1 registered agent
	testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	// 1 expired pending agent (expires_at in the past)
	testhelper.MustExec(t, tx,
		`INSERT INTO agents (public_key, approver_id, status, expires_at)
		 VALUES ('pk', $1, 'pending', now() - interval '1 hour')`, uid)

	count, err := db.CountRegisteredAgentsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("CountRegisteredAgentsByUser: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 (only registered), got %d", count)
	}
}

// --- CountActiveStandingApprovalsByUser ---

func TestCountActiveStandingApprovalsByUser_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "count_sa_empty")

	count, err := db.CountActiveStandingApprovalsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("CountActiveStandingApprovalsByUser: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestCountActiveStandingApprovalsByUser_CountsOnlyActive(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "count_sa_mix")

	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	// 2 active standing approvals
	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	// 1 revoked (should NOT count)
	testhelper.InsertStandingApprovalWithStatus(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, "revoked")
	// 1 expired (should NOT count)
	testhelper.InsertStandingApprovalWithStatus(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, "expired")

	count, err := db.CountActiveStandingApprovalsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("CountActiveStandingApprovalsByUser: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

// --- CountCredentialsByUser ---

func TestCountCredentialsByUser_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "count_cred_empty")

	count, err := db.CountCredentialsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("CountCredentialsByUser: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestCountCredentialsByUser_CountsAll(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "count_cred_all")

	testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid, "github")
	testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid, "stripe")
	testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid, "slack")

	count, err := db.CountCredentialsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("CountCredentialsByUser: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestCountCredentialsByUser_DoesNotCountOtherUsers(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "count_cred_user1")
	testhelper.InsertUser(t, tx, uid2, "count_cred_user2")

	testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid1, "github")
	testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid2, "stripe")

	count, err := db.CountCredentialsByUser(ctx, tx, uid1)
	if err != nil {
		t.Fatalf("CountCredentialsByUser: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}
