package testhelper

import (
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// InsertApproval creates a pending approval for the given agent and approver.
// The agent and approver must already exist via InsertUser and InsertAgent.
func InsertApproval(t *testing.T, d db.DBTX, approvalID string, agentID int64, approverID string) {
	t.Helper()
	InsertApprovalWithStatus(t, d, approvalID, agentID, approverID, "pending")
}

// InsertApprovalWithStatus creates an approval with the given status.
// The agent and approver must already exist via InsertUser and InsertAgent.
func InsertApprovalWithStatus(t *testing.T, d db.DBTX, approvalID string, agentID int64, approverID, status string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at)
		 VALUES ($1, $2, $3, '{"type":"test"}', '{"description":"test"}', $4, now() + interval '1 hour')`,
		approvalID, agentID, approverID, status)
}

// InsertApprovalWithCreatedAt creates a pending approval with an explicit created_at.
// Useful for testing time-windowed queries like request_count_30d.
// The agent and approver must already exist via InsertUser and InsertAgent.
func InsertApprovalWithCreatedAt(t *testing.T, d db.DBTX, approvalID string, agentID int64, approverID string, createdAt time.Time) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, created_at)
		 VALUES ($1, $2, $3, '{"type":"test"}', '{"description":"test"}', 'pending', $4::timestamptz + interval '1 hour', $4)`,
		approvalID, agentID, approverID, createdAt)
}

// InsertApprovalWithExpiresAt creates a pending approval with an explicit expires_at.
// Useful for testing expiration logic.
// The agent and approver must already exist via InsertUser and InsertAgent.
func InsertApprovalWithExpiresAt(t *testing.T, d db.DBTX, approvalID string, agentID int64, approverID string, expiresAt time.Time) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at)
		 VALUES ($1, $2, $3, '{"type":"test"}', '{"description":"test"}', 'pending', $4)`,
		approvalID, agentID, approverID, expiresAt)
}

// resolvedAtColumn maps an approval status to its resolution timestamp column.
func resolvedAtColumn(t *testing.T, status string) string {
	t.Helper()
	switch status {
	case "approved":
		return "approved_at"
	case "denied":
		return "denied_at"
	case "cancelled":
		return "cancelled_at"
	default:
		t.Fatalf("unsupported resolved status %q (use InsertApproval for pending)", status)
		return ""
	}
}

// InsertResolvedApproval creates an approval with the given status and its
// corresponding resolution timestamp set (approved_at, denied_at, or cancelled_at).
// Unlike InsertApprovalWithStatus, this produces realistic data suitable for
// audit trail queries that rely on those timestamps.
func InsertResolvedApproval(t *testing.T, d db.DBTX, approvalID string, agentID int64, approverID, status string) {
	t.Helper()
	tsColumn := resolvedAtColumn(t, status)
	mustExec(t, d,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, `+tsColumn+`)
		 VALUES ($1, $2, $3, '{"type":"test.action","version":"1","parameters":{"to":"alice@example.com"}}', '{"description":"test action"}', $4, now() + interval '1 hour', now())`,
		approvalID, agentID, approverID, status)
}

// InsertResolvedApprovalAt creates a resolved approval with an explicit resolution timestamp.
// Useful for testing audit trail ordering.
func InsertResolvedApprovalAt(t *testing.T, d db.DBTX, approvalID string, agentID int64, approverID, status string, resolvedAt time.Time) {
	t.Helper()
	tsColumn := resolvedAtColumn(t, status)
	mustExec(t, d,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, `+tsColumn+`)
		 VALUES ($1, $2, $3, '{"type":"test.action","version":"1","parameters":{"to":"alice@example.com"}}', '{"description":"test action"}', $4, $5::timestamptz + interval '1 hour', $5)`,
		approvalID, agentID, approverID, status, resolvedAt)
}

// InsertRequestID creates a request_id for the given agent and approver.
// The agent must already exist via InsertAgent.
func InsertRequestID(t *testing.T, d db.DBTX, requestID string, agentID int64, approverID string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO request_ids (request_id, agent_id, approver_id) VALUES ($1, $2, $3)`,
		requestID, agentID, approverID)
}
