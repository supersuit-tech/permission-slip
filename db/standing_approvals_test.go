package db_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

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
			_, err := tx.Exec(context.Background(),
				`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, starts_at, expires_at)
				 VALUES ($1, $2, $3, 'test.action', $4, now(), now() + interval '30 days')`,
				fmt.Sprintf("%s_%d", base, i), agentID, uid, value)
			return err
		})
}

func TestStandingApprovalsExpiryConstraints(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	base := testhelper.GenerateID(t, "sa_")

	// NULL expires_at (no expiry) should succeed
	_, err := tx.Exec(ctx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, starts_at, expires_at)
		 VALUES ($1, $2, $3, 'test.action', 'active', '2026-01-01 00:00:00+00', NULL)`,
		base+"_null_ok", agentID, uid)
	if err != nil {
		t.Errorf("NULL expires_at was rejected: %v", err)
	}

	// 365 days should succeed (no max duration limit)
	_, err = tx.Exec(ctx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, starts_at, expires_at)
		 VALUES ($1, $2, $3, 'test.action', 'active', '2026-01-01 00:00:00+00', '2027-01-01 00:00:00+00')`,
		base+"_365d_ok", agentID, uid)
	if err != nil {
		t.Errorf("365 days was rejected: %v", err)
	}

	// expires_at before starts_at should still fail
	err = testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(ctx,
			`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, starts_at, expires_at)
			 VALUES ($1, $2, $3, 'test.action', 'active', '2026-03-01 00:00:00+00', '2026-02-01 00:00:00+00')`,
			base+"_backward", agentID, uid)
		return err
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

	// max_executions = NULL (unlimited) should succeed
	_, err := tx.Exec(ctx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, max_executions, starts_at, expires_at)
		 VALUES ($1, $2, $3, 'test.action', 'active', NULL, now(), now() + interval '30 days')`,
		base+"_null", agentID, uid)
	if err != nil {
		t.Errorf("NULL max_executions was rejected: %v", err)
	}

	// max_executions = 1 should succeed
	_, err = tx.Exec(ctx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, max_executions, starts_at, expires_at)
		 VALUES ($1, $2, $3, 'test.action', 'active', 1, now(), now() + interval '30 days')`,
		base+"_1", agentID, uid)
	if err != nil {
		t.Errorf("max_executions=1 was rejected: %v", err)
	}

	// max_executions = 0 should fail (must be > 0)
	err = testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(ctx,
			`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, max_executions, starts_at, expires_at)
			 VALUES ($1, $2, $3, 'test.action', 'active', 0, now(), now() + interval '30 days')`,
			base+"_0", agentID, uid)
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for max_executions=0, but insert succeeded")
	}
}
