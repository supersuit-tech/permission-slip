package db

import (
	"context"
	"fmt"
)

// ScrubSensitiveExecutionData nullifies sensitive execution data that is older
// than 30 minutes. This covers three columns:
//
//   - approvals.execution_result → NULL
//   - approvals.action → {"type":"<original>"} (parameters stripped, type preserved)
//   - standing_approval_executions.parameters → NULL
//
// Only resolved approvals (approved/denied/cancelled) are scrubbed; pending
// approvals keep their action parameters so approvers can still review them.
// The function is idempotent — already-scrubbed rows are skipped via WHERE clauses.
//
// Returns the total number of rows updated across both tables.
func ScrubSensitiveExecutionData(ctx context.Context, d DBTX) (int64, error) {
	// Scrub approvals: NULL out execution_result, strip action to type-only.
	// Use COALESCE to pick the relevant resolution timestamp for each status:
	// executed_at for approved+executed, denied_at for denied, cancelled_at for cancelled.
	// Without this, denied/cancelled approvals (which have NULL executed_at) are never scrubbed.
	tag1, err := d.Exec(ctx, `
		UPDATE approvals
		SET execution_result = NULL,
		    action = action - 'parameters'
		WHERE status IN ('approved', 'denied', 'cancelled')
		  AND COALESCE(executed_at, approved_at, denied_at, cancelled_at)
		      < now() - interval '30 minutes'
		  AND (execution_result IS NOT NULL
		       OR action ? 'parameters')`)
	if err != nil {
		return 0, fmt.Errorf("scrub approvals: %w", err)
	}

	// Scrub standing_approval_executions: NULL out parameters.
	tag2, err := d.Exec(ctx, `
		UPDATE standing_approval_executions
		SET parameters = NULL
		WHERE executed_at < now() - interval '30 minutes'
		  AND parameters IS NOT NULL`)
	if err != nil {
		return 0, fmt.Errorf("scrub standing_approval_executions: %w", err)
	}

	return tag1.RowsAffected() + tag2.RowsAffected(), nil
}
