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
	// Use per-status conditions so each status checks its own resolution timestamp.
	// For approved: require executed_at IS NOT NULL to avoid scrubbing in-flight executions.
	// For denied/cancelled: use denied_at/cancelled_at respectively.
	tag1, err := d.Exec(ctx, `
		UPDATE approvals
		SET execution_result = NULL,
		    action = action - 'parameters'
		WHERE (execution_result IS NOT NULL
		       OR action ? 'parameters')
		  AND (
		    (status = 'approved'   AND executed_at  IS NOT NULL AND executed_at  < now() - interval '30 minutes')
		    OR (status = 'denied'    AND denied_at    IS NOT NULL AND denied_at    < now() - interval '30 minutes')
		    OR (status = 'cancelled' AND cancelled_at IS NOT NULL AND cancelled_at < now() - interval '30 minutes')
		  )`)
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
