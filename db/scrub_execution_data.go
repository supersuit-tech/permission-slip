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
		    action = json_remove(action, '$.parameters')
		WHERE (execution_result IS NOT NULL
		       OR json_type(action, '$.parameters') IS NOT NULL)
		  AND (
		    (status = 'approved'   AND executed_at  IS NOT NULL AND executed_at  < strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-30 minutes'))
		    OR (status = 'denied'    AND denied_at    IS NOT NULL AND denied_at    < strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-30 minutes'))
		    OR (status = 'cancelled' AND cancelled_at IS NOT NULL AND cancelled_at < strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-30 minutes'))
		  )`)
	if err != nil {
		return 0, fmt.Errorf("scrub approvals: %w", err)
	}

	// Scrub standing_approval_executions: NULL out parameters.
	tag2, err := d.Exec(ctx, `
		UPDATE standing_approval_executions
		SET parameters = NULL
		WHERE executed_at < strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-30 minutes')
		  AND parameters IS NOT NULL`)
	if err != nil {
		return 0, fmt.Errorf("scrub standing_approval_executions: %w", err)
	}

	return RowsAffected(tag1) + RowsAffected(tag2), nil
}
