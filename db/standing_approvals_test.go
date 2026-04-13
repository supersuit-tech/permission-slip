package db_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

const standingApprovalDBTestConnector = "standing_approval_db_test"

func insertTestStandingApprovalRow(t *testing.T, tx db.DBTX, agentID int64, uid, standingApprovalID, actionType, status string, startsAt time.Time, expiresAt *time.Time, maxExec *int) error {
	t.Helper()
	ctx := context.Background()
	_, _ = tx.Exec(ctx,
		`INSERT INTO connectors (id, name) VALUES ($1, $1) ON CONFLICT (id) DO NOTHING`,
		standingApprovalDBTestConnector)
	_, _ = tx.Exec(ctx,
		`INSERT INTO connector_actions (connector_id, action_type, name) VALUES ($1, $2, $2) ON CONFLICT (connector_id, action_type) DO NOTHING`,
		standingApprovalDBTestConnector, actionType)
	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, standingApprovalDBTestConnector, actionType)
	t.Cleanup(func() {
		_, _ = tx.Exec(ctx, `DELETE FROM action_configurations WHERE id = $1`, configID)
	})
	_, err := tx.Exec(ctx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, source_action_configuration_id, max_executions, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		standingApprovalID, agentID, uid, actionType, status, configID, maxExec, startsAt, expiresAt)
	return err
}

func TestStandingApprovalsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "standing_approvals", []string{
		"standing_approval_id", "agent_id", "user_id", "action_type",
		"action_version", "constraints", "status", "max_executions",
		"execution_count", "starts_at", "expires_at", "created_at",
		"revoked_at", "expired_at", "exhausted_at",
	})
}

func TestStandingApprovalsIndex(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireIndex(t, tx, "standing_approvals", "idx_standing_approvals_agent_action_status")
	testhelper.RequireIndex(t, tx, "standing_approvals", "idx_standing_approvals_source_config_active")
}

func TestStandingApprovalsCascadeDeleteOnAgentDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)

	testhelper.RequireCascadeDeletes(t, tx,
		fmt.Sprintf("DELETE FROM agents WHERE agent_id = %d", agentID),
		[]string{"standing_approvals"},
		fmt.Sprintf("agent_id = %d", agentID),
	)
}

func TestStandingApprovalsCascadeDeleteOnProfileDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM profiles WHERE id = '"+uid+"'",
		[]string{"standing_approvals"},
		"user_id = '"+uid+"'",
	)
}

func TestStandingApprovalsStatusCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	base := testhelper.GenerateID(t, "sa_")
	testhelper.RequireCheckValues(t, tx, "status",
		[]string{"active", "expired", "revoked", "exhausted"}, "invalid",
		func(value string, i int) error {
			start := time.Now().UTC()
			exp := start.Add(30 * 24 * time.Hour)
			return insertTestStandingApprovalRow(t, tx, agentID, uid, fmt.Sprintf("%s_%d", base, i), "test.action", value, start, &exp, nil)
		})
}

func TestStandingApprovalsExpiryConstraints(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	base := testhelper.GenerateID(t, "sa_")
	start2026 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// NULL expires_at (no expiry) should succeed
	err := insertTestStandingApprovalRow(t, tx, agentID, uid, base+"_null_ok", "test.action", "active", start2026, nil, nil)
	if err != nil {
		t.Errorf("NULL expires_at was rejected: %v", err)
	}

	// 365 days should succeed (no max duration limit)
	exp365 := start2026.AddDate(1, 0, 0)
	err = insertTestStandingApprovalRow(t, tx, agentID, uid, base+"_365d_ok", "test.action", "active", start2026, &exp365, nil)
	if err != nil {
		t.Errorf("365 days was rejected: %v", err)
	}

	// expires_at before starts_at should still fail
	err = testhelper.WithSavepoint(t, tx, func() error {
		startMar := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		expFeb := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		return insertTestStandingApprovalRow(t, tx, agentID, uid, base+"_backward", "test.action", "active", startMar, &expFeb, nil)
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for expires_at before starts_at, but insert succeeded")
	}
}

func TestStandingApprovalsExecutionCountCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	// Setting execution_count to a negative value should fail
	err := testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(context.Background(),
			`UPDATE standing_approvals SET execution_count = -1 WHERE standing_approval_id = $1`, saID)
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for negative execution_count, but update succeeded")
	}
}

// ── standing_approval_executions table ───────────────────────────────────────

func TestStandingApprovalExecutionsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "standing_approval_executions", []string{
		"id", "standing_approval_id",
		"parameters", "executed_at",
	})
}

func TestStandingApprovalExecutionsIndexes(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireIndex(t, tx, "standing_approval_executions", "idx_sa_executions_sa_id")
}

func TestStandingApprovalExecutionsCascadeOnStandingApprovalDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)
	testhelper.InsertStandingApprovalExecution(t, tx, saID)

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM standing_approvals WHERE standing_approval_id = '"+saID+"'",
		[]string{"standing_approval_executions"},
		"standing_approval_id = '"+saID+"'",
	)
}

func TestStandingApprovalExecutionsCascadeOnProfileDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)
	testhelper.InsertStandingApprovalExecution(t, tx, saID)

	// Cascade: profiles → standing_approvals → standing_approval_executions
	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM profiles WHERE id = '"+uid+"'",
		[]string{"standing_approval_executions"},
		"standing_approval_id = '"+saID+"'",
	)
}

func TestStandingApprovalsMaxExecutionsCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	base := testhelper.GenerateID(t, "sa_")
	start := time.Now().UTC()
	exp := start.Add(30 * 24 * time.Hour)
	one := 1
	zero := 0

	// max_executions = NULL (unlimited) should succeed
	err := insertTestStandingApprovalRow(t, tx, agentID, uid, base+"_null", "test.action", "active", start, &exp, nil)
	if err != nil {
		t.Errorf("NULL max_executions was rejected: %v", err)
	}

	// max_executions = 1 should succeed
	err = insertTestStandingApprovalRow(t, tx, agentID, uid, base+"_1", "test.action", "active", start, &exp, &one)
	if err != nil {
		t.Errorf("max_executions=1 was rejected: %v", err)
	}

	// max_executions = 0 should fail (must be > 0)
	err = testhelper.WithSavepoint(t, tx, func() error {
		return insertTestStandingApprovalRow(t, tx, agentID, uid, base+"_0", "test.action", "active", start, &exp, &zero)
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for max_executions=0, but insert succeeded")
	}
}
