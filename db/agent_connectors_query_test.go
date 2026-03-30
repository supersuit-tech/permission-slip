package db_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// ── ListAgentConnectors ─────────────────────────────────────────────────────

func TestListAgentConnectors_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connectors, err := db.ListAgentConnectors(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("ListAgentConnectors: %v", err)
	}
	if len(connectors) != 0 {
		t.Errorf("expected 0 connectors, got %d", len(connectors))
	}
}

func TestListAgentConnectors_WithEnabledConnectors(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "test_svc", "api_key")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	connectors, err := db.ListAgentConnectors(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("ListAgentConnectors: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(connectors))
	}

	ac := connectors[0]
	if ac.ID != connID {
		t.Errorf("expected id %q, got %q", connID, ac.ID)
	}
	if len(ac.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(ac.Actions))
	}
	if len(ac.RequiredCredentials) != 1 {
		t.Errorf("expected 1 required credential, got %d", len(ac.RequiredCredentials))
	}
	if ac.EnabledAt.IsZero() {
		t.Error("expected enabled_at to be set")
	}
}

func TestListAgentConnectors_ScopedToApprover(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agentID1 := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	agentID2 := testhelper.InsertUserWithAgent(t, tx, uid2, "u2_"+uid2[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID1, uid1, connID)
	testhelper.InsertAgentConnector(t, tx, agentID2, uid2, connID)

	// User 1 should only see their own agent's connector
	connectors, err := db.ListAgentConnectors(t.Context(), tx, agentID1, uid1)
	if err != nil {
		t.Fatalf("ListAgentConnectors: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 connector for user1, got %d", len(connectors))
	}

	// User 2 should not see user 1's agent connector
	connectors, err = db.ListAgentConnectors(t.Context(), tx, agentID1, uid2)
	if err != nil {
		t.Fatalf("ListAgentConnectors: %v", err)
	}
	if len(connectors) != 0 {
		t.Errorf("expected 0 connectors for user2 on user1's agent, got %d", len(connectors))
	}
}

// ── EnableAgentConnector ────────────────────────────────────────────────────

func TestEnableAgentConnector_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)

	row, err := db.EnableAgentConnector(t.Context(), tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("EnableAgentConnector: %v", err)
	}
	if row == nil {
		t.Fatal("expected non-nil result")
	}
	if row.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, row.AgentID)
	}
	if row.ConnectorID != connID {
		t.Errorf("expected connector_id %q, got %q", connID, row.ConnectorID)
	}
	if row.EnabledAt.IsZero() {
		t.Error("expected enabled_at to be set")
	}
}

func TestEnableAgentConnector_Idempotent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)

	row1, err := db.EnableAgentConnector(t.Context(), tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("first EnableAgentConnector: %v", err)
	}

	row2, err := db.EnableAgentConnector(t.Context(), tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("second EnableAgentConnector: %v", err)
	}

	// Should return the same enabled_at (idempotent, no update).
	if !row1.EnabledAt.Equal(row2.EnabledAt) {
		t.Errorf("expected same enabled_at on idempotent enable, got %v and %v", row1.EnabledAt, row2.EnabledAt)
	}
}

// ── DisableAgentConnector ───────────────────────────────────────────────────

func TestDisableAgentConnector_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	result, err := db.DisableAgentConnector(t.Context(), tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("DisableAgentConnector: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, result.AgentID)
	}
	if result.ConnectorID != connID {
		t.Errorf("expected connector_id %q, got %q", connID, result.ConnectorID)
	}
	if result.RevokedStandingApprovals != 0 {
		t.Errorf("expected 0 revoked standing approvals, got %d", result.RevokedStandingApprovals)
	}

	// Verify it was actually deleted
	connectors, err := db.ListAgentConnectors(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("ListAgentConnectors after disable: %v", err)
	}
	if len(connectors) != 0 {
		t.Errorf("expected 0 connectors after disable, got %d", len(connectors))
	}
}

func TestDisableAgentConnector_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	result, err := db.DisableAgentConnector(t.Context(), tx, agentID, uid, "nonexistent")
	if err != nil {
		t.Fatalf("DisableAgentConnector: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for nonexistent connector")
	}
}

func TestDisableAgentConnector_RevokesStandingApprovals(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action1", "Action 1")
	testhelper.InsertConnectorAction(t, tx, connID, "test.action2", "Action 2")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	// Create active standing approvals for connector actions
	sa1 := testhelper.GenerateID(t, "sa_")
	sa2 := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalWithActionType(t, tx, sa1, agentID, uid, "test.action1")
	testhelper.InsertStandingApprovalWithActionType(t, tx, sa2, agentID, uid, "test.action2")

	// Also create a standing approval for a different action (should not be revoked)
	sa3 := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalWithActionType(t, tx, sa3, agentID, uid, "other.action")

	result, err := db.DisableAgentConnector(t.Context(), tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("DisableAgentConnector: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.RevokedStandingApprovals != 2 {
		t.Errorf("expected 2 revoked standing approvals, got %d", result.RevokedStandingApprovals)
	}

	// Verify the other standing approval is still active
	var status string
	err = tx.QueryRow(t.Context(),
		`SELECT status FROM standing_approvals WHERE standing_approval_id = $1`, sa3).Scan(&status)
	if err != nil {
		t.Fatalf("query standing approval: %v", err)
	}
	if status != "active" {
		t.Errorf("expected unrelated standing approval to still be active, got %q", status)
	}
}
