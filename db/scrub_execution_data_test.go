package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestScrubSensitiveExecutionData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ScrubsOldApprovalExecutionData", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		approvalID := testhelper.GenerateID(t, "appr_")

		// Insert a resolved approval with execution data older than 30 minutes.
		testhelper.InsertResolvedApproval(t, tx, approvalID, agentID, uid, "approved")
		testhelper.MustExec(t, tx,
			`UPDATE approvals SET
				execution_status = 'success',
				execution_result = '{"data":"sensitive"}',
				executed_at = now() - interval '1 hour'
			 WHERE approval_id = $1`, approvalID)

		scrubbed, err := db.ScrubSensitiveExecutionData(ctx, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scrubbed != 1 {
			t.Errorf("expected 1 scrubbed row, got %d", scrubbed)
		}

		// Verify execution_result is NULL and action is type-only.
		var execResult *string
		var actionType string
		err = tx.QueryRow(ctx,
			`SELECT execution_result::text, action->>'type' FROM approvals WHERE approval_id = $1`,
			approvalID).Scan(&execResult, &actionType)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		if execResult != nil {
			t.Errorf("expected NULL execution_result, got %s", *execResult)
		}
		if actionType != "test.action" {
			t.Errorf("expected action type 'test.action', got %q", actionType)
		}

		// Verify parameters were stripped but other keys preserved.
		var actionParams *string
		var actionVersion *string
		err = tx.QueryRow(ctx,
			`SELECT action->>'parameters', action->>'version' FROM approvals WHERE approval_id = $1`,
			approvalID).Scan(&actionParams, &actionVersion)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		if actionParams != nil {
			t.Errorf("expected action parameters to be stripped, got %s", *actionParams)
		}
		if actionVersion == nil || *actionVersion != "1" {
			t.Errorf("expected action version '1' to be preserved, got %v", actionVersion)
		}
	})

	t.Run("PreservesRecentApprovalData", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		approvalID := testhelper.GenerateID(t, "appr_")

		// Insert a resolved approval with execution data only 10 minutes old.
		testhelper.InsertResolvedApproval(t, tx, approvalID, agentID, uid, "approved")
		testhelper.MustExec(t, tx,
			`UPDATE approvals SET
				execution_status = 'success',
				execution_result = '{"data":"sensitive"}',
				executed_at = now() - interval '10 minutes'
			 WHERE approval_id = $1`, approvalID)

		scrubbed, err := db.ScrubSensitiveExecutionData(ctx, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scrubbed != 0 {
			t.Errorf("expected 0 scrubbed rows, got %d", scrubbed)
		}

		// Verify execution_result is still present.
		var execResult *string
		err = tx.QueryRow(ctx,
			`SELECT execution_result::text FROM approvals WHERE approval_id = $1`,
			approvalID).Scan(&execResult)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		if execResult == nil {
			t.Error("expected execution_result to be preserved, got NULL")
		}
	})

	t.Run("PreservesPendingApprovalAction", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		approvalID := testhelper.GenerateID(t, "appr_")

		// Insert a pending approval — should NOT be scrubbed regardless of age.
		testhelper.InsertApproval(t, tx, approvalID, agentID, uid)

		scrubbed, err := db.ScrubSensitiveExecutionData(ctx, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scrubbed != 0 {
			t.Errorf("expected 0 scrubbed rows for pending approval, got %d", scrubbed)
		}
	})

	t.Run("ScrubsOldStandingApprovalExecutionParams", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		saID := testhelper.GenerateID(t, "sa_")
		testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)
		testhelper.InsertStandingApprovalExecutionWithParams(t, tx, saID, []byte(`{"key":"sensitive_value"}`))

		// Backdate the execution to 1 hour ago.
		testhelper.MustExec(t, tx,
			`UPDATE standing_approval_executions SET executed_at = now() - interval '1 hour'
			 WHERE standing_approval_id = $1`, saID)

		scrubbed, err := db.ScrubSensitiveExecutionData(ctx, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scrubbed != 1 {
			t.Errorf("expected 1 scrubbed row, got %d", scrubbed)
		}

		// Verify parameters are NULL.
		var params *string
		err = tx.QueryRow(ctx,
			`SELECT parameters::text FROM standing_approval_executions WHERE standing_approval_id = $1`,
			saID).Scan(&params)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		if params != nil {
			t.Errorf("expected NULL parameters, got %s", *params)
		}
	})

	t.Run("PreservesRecentStandingApprovalExecutionParams", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		saID := testhelper.GenerateID(t, "sa_")
		testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)
		testhelper.InsertStandingApprovalExecutionWithParams(t, tx, saID, []byte(`{"key":"sensitive_value"}`))

		// Execution is recent (default now()) — should not be scrubbed.
		scrubbed, err := db.ScrubSensitiveExecutionData(ctx, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scrubbed != 0 {
			t.Errorf("expected 0 scrubbed rows, got %d", scrubbed)
		}
	})

	t.Run("IdempotentOnAlreadyScrubbed", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		approvalID := testhelper.GenerateID(t, "appr_")

		// Insert and scrub once.
		testhelper.InsertResolvedApproval(t, tx, approvalID, agentID, uid, "approved")
		testhelper.MustExec(t, tx,
			`UPDATE approvals SET
				execution_status = 'success',
				execution_result = '{"data":"sensitive"}',
				executed_at = now() - interval '1 hour'
			 WHERE approval_id = $1`, approvalID)

		_, err := db.ScrubSensitiveExecutionData(ctx, tx)
		if err != nil {
			t.Fatalf("first scrub error: %v", err)
		}

		// Second scrub should find nothing to do.
		scrubbed, err := db.ScrubSensitiveExecutionData(ctx, tx)
		if err != nil {
			t.Fatalf("second scrub error: %v", err)
		}
		if scrubbed != 0 {
			t.Errorf("expected 0 rows on idempotent re-scrub, got %d", scrubbed)
		}
	})

	t.Run("PreservesExecutionStatus", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		approvalID := testhelper.GenerateID(t, "appr_")

		testhelper.InsertResolvedApproval(t, tx, approvalID, agentID, uid, "denied")
		testhelper.MustExec(t, tx,
			`UPDATE approvals SET
				execution_status = 'error',
				execution_result = '{"error":"something failed"}',
				executed_at = now() - interval '2 hours'
			 WHERE approval_id = $1`, approvalID)

		_, err := db.ScrubSensitiveExecutionData(ctx, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// execution_status and executed_at should remain intact.
		var execStatus string
		var executedAt *string
		err = tx.QueryRow(ctx,
			`SELECT execution_status, executed_at::text FROM approvals WHERE approval_id = $1`,
			approvalID).Scan(&execStatus, &executedAt)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		if execStatus != "error" {
			t.Errorf("expected execution_status 'error', got %q", execStatus)
		}
		if executedAt == nil {
			t.Error("expected executed_at to be preserved, got NULL")
		}
	})
}

func TestScrubSensitiveExecutionData_SQLFunction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ScrubsViaSQL", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		approvalID := testhelper.GenerateID(t, "appr_")

		testhelper.InsertResolvedApproval(t, tx, approvalID, agentID, uid, "approved")
		testhelper.MustExec(t, tx,
			`UPDATE approvals SET
				execution_status = 'success',
				execution_result = '{"data":"sensitive"}',
				executed_at = now() - interval '1 hour'
			 WHERE approval_id = $1`, approvalID)

		// Call the SQL function directly.
		_, err := tx.Exec(ctx, "SELECT scrub_sensitive_execution_data()")
		if err != nil {
			t.Fatalf("SQL function error: %v", err)
		}

		// Verify scrubbed.
		var execResult *string
		var actionType string
		err = tx.QueryRow(ctx,
			`SELECT execution_result::text, action->>'type' FROM approvals WHERE approval_id = $1`,
			approvalID).Scan(&execResult, &actionType)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		if execResult != nil {
			t.Errorf("expected NULL execution_result, got %s", *execResult)
		}
		if actionType != "test.action" {
			t.Errorf("expected action type 'test.action', got %q", actionType)
		}
	})
}

func TestScrubSensitiveExecutionData_PgCronJob(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequirePgCronJob(t, tx, "scrub_sensitive_execution_data")
}
