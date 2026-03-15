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
	tag1, err := d.Exec(ctx, `
		UPDATE approvals
		SET execution_result = NULL,
		    action = jsonb_build_object('type', action->>'type')
		WHERE executed_at IS NOT NULL
		  AND executed_at < now() - interval '30 minutes'
		  AND status IN ('approved', 'denied', 'cancelled')
		  AND (execution_result IS NOT NULL
		       OR action != jsonb_build_object('type', action->>'type'))`)
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
		return tag1.RowsAffected(), fmt.Errorf("scrub standing_approval_executions: %w", err)
	}

	return tag1.RowsAffected() + tag2.RowsAffected(), nil
}
