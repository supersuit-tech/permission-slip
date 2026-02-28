package testhelper

import (
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// InsertStandingApproval creates an active standing approval for the given agent and user.
// The agent and user must already exist via InsertUser and InsertAgent.
func InsertStandingApproval(t *testing.T, d db.DBTX, saID string, agentID int64, userID string) {
	t.Helper()
	InsertStandingApprovalWithStatus(t, d, saID, agentID, userID, "active")
}

// InsertStandingApprovalWithStatus creates a standing approval with the given status.
// The agent and user must already exist via InsertUser and InsertAgent.
func InsertStandingApprovalWithStatus(t *testing.T, d db.DBTX, saID string, agentID int64, userID, status string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, starts_at, expires_at)
		 VALUES ($1, $2, $3, 'test.action', $4, now(), now() + interval '30 days')`,
		saID, agentID, userID, status)
}

// InsertStandingApprovalWithCreatedAt creates an active standing approval with an explicit created_at timestamp.
// Useful for pagination tests that need deterministic ordering.
func InsertStandingApprovalWithCreatedAt(t *testing.T, d db.DBTX, saID string, agentID int64, userID string, createdAt time.Time) {
	t.Helper()
	expiresAt := createdAt.Add(30 * 24 * time.Hour)
	mustExec(t, d,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, starts_at, expires_at, created_at)
		 VALUES ($1, $2, $3, 'test.action', 'active', $4, $5, $4)`,
		saID, agentID, userID, createdAt, expiresAt)
}

// InsertStandingApprovalWithActionType creates an active standing approval with a specific action_type.
// The agent and user must already exist via InsertUser and InsertAgent.
func InsertStandingApprovalWithActionType(t *testing.T, d db.DBTX, saID string, agentID int64, userID, actionType string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, 'active', now(), now() + interval '30 days')`,
		saID, agentID, userID, actionType)
}

// InsertStandingApprovalExecution records a standing approval execution with no parameters.
func InsertStandingApprovalExecution(t *testing.T, d db.DBTX, saID string) {
	t.Helper()
	InsertStandingApprovalExecutionWithParams(t, d, saID, nil)
}

// InsertStandingApprovalExecutionWithParams records a standing approval execution with the given parameters.
func InsertStandingApprovalExecutionWithParams(t *testing.T, d db.DBTX, saID string, params []byte) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO standing_approval_executions (standing_approval_id, parameters)
		 VALUES ($1, $2)`,
		saID, params)
	mustExec(t, d,
		`UPDATE standing_approvals
		 SET execution_count = execution_count + 1
		 WHERE standing_approval_id = $1`,
		saID)
}

// StandingApprovalOpts holds optional fields for InsertStandingApprovalFull.
type StandingApprovalOpts struct {
	ActionType    string
	Status        string
	Constraints   []byte
	MaxExecutions *int
	StartsAt      time.Time
	ExpiresAt     time.Time
}

// InsertStandingApprovalFull creates a standing approval with full control over all fields.
func InsertStandingApprovalFull(t *testing.T, d db.DBTX, saID string, agentID int64, userID string, opts StandingApprovalOpts) {
	t.Helper()
	if opts.ActionType == "" {
		opts.ActionType = "test.action"
	}
	if opts.Status == "" {
		opts.Status = "active"
	}
	if opts.StartsAt.IsZero() {
		opts.StartsAt = time.Now().Add(-time.Hour)
	}
	if opts.ExpiresAt.IsZero() {
		opts.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	}
	mustExec(t, d,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, constraints, max_executions, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		saID, agentID, userID, opts.ActionType, opts.Status, opts.Constraints, opts.MaxExecutions, opts.StartsAt, opts.ExpiresAt)
}
