package testhelper

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

func standingApprovalFixtureConnectorID(t *testing.T) string {
	t.Helper()
	// Unique per test so ON CONFLICT cannot leak a shared connector row into the
	// long-lived CI DB (shared pool + transaction rollback).
	return "sa_fixture_c_" + hex.EncodeToString(mustRandBytes(t, 8))
}

// ensureStandingApprovalFixtureConnector creates connector and action rows for tests.
func ensureStandingApprovalFixtureConnector(t *testing.T, d db.DBTX, actionType string) string {
	t.Helper()
	connectorID := standingApprovalFixtureConnectorID(t)
	mustExec(t, d,
		`INSERT INTO connectors (id, name) VALUES ($1, $1)`,
		connectorID)
	mustExec(t, d,
		`INSERT INTO connector_actions (connector_id, action_type, name) VALUES ($1, $2, $2)`,
		connectorID, actionType)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = d.Exec(ctx, `DELETE FROM connector_actions WHERE connector_id = $1`, connectorID)
		_, _ = d.Exec(ctx, `DELETE FROM connectors WHERE id = $1`, connectorID)
	})
	return connectorID
}

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
	actionType := "test.action"
	ensureStandingApprovalFixtureConnector(t, d, actionType)
	name := "Test standing approval"
	mustExec(t, d,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, name, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+30 days'))`,
		saID, agentID, userID, actionType, status, name)
}

// InsertStandingApprovalWithCreatedAt creates an active standing approval with an explicit created_at timestamp.
// Useful for pagination tests that need deterministic ordering.
func InsertStandingApprovalWithCreatedAt(t *testing.T, d db.DBTX, saID string, agentID int64, userID string, createdAt time.Time) {
	t.Helper()
	actionType := "test.action"
	ensureStandingApprovalFixtureConnector(t, d, actionType)
	name := "Test standing approval"
	expiresAt := createdAt.Add(30 * 24 * time.Hour)
	mustExec(t, d,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, name, starts_at, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, 'active', $5, $6, $7, $8)`,
		saID, agentID, userID, actionType, name,
		db.TimestampForSQLite(createdAt),
		db.TimestampForSQLite(expiresAt),
		db.TimestampForSQLite(createdAt),
	)
}

// InsertStandingApprovalWithActionType creates an active standing approval with a specific action_type.
// The agent and user must already exist via InsertUser and InsertAgent.
func InsertStandingApprovalWithActionType(t *testing.T, d db.DBTX, saID string, agentID int64, userID, actionType string) {
	t.Helper()
	ensureStandingApprovalFixtureConnector(t, d, actionType)
	name := "Test standing approval"
	mustExec(t, d,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, name, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, 'active', $5, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+30 days'))`,
		saID, agentID, userID, actionType, name)
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
}

// StandingApprovalOpts holds optional fields for InsertStandingApprovalFull.
type StandingApprovalOpts struct {
	ActionType          string
	Status              string
	Constraints         []byte
	Name                *string
	Description         *string
	ConnectorInstanceID *string
	StartsAt            time.Time
	ExpiresAt           time.Time
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
	name := opts.Name
	if name == nil {
		defaultName := "Test standing approval"
		name = &defaultName
	}
	ensureStandingApprovalFixtureConnector(t, d, opts.ActionType)
	startsStr := opts.StartsAt.UTC().Format("2006-01-02T15:04:05.000000Z")
	expiresStr := opts.ExpiresAt.UTC().Format("2006-01-02T15:04:05.000000Z")
	mustExec(t, d,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, constraints, name, description, connector_instance_id, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		saID, agentID, userID, opts.ActionType, opts.Status, opts.Constraints, name, opts.Description, opts.ConnectorInstanceID, startsStr, expiresStr)
}

// RequireStandingApprovalExecutionCount asserts the number of rows in
// standing_approval_executions for the given standing approval.
func RequireStandingApprovalExecutionCount(t *testing.T, d db.DBTX, standingApprovalID string, want int) {
	t.Helper()
	ctx := context.Background()
	var n int
	if err := d.QueryRow(ctx,
		`SELECT COUNT(*) FROM standing_approval_executions WHERE standing_approval_id = $1`,
		standingApprovalID,
	).Scan(&n); err != nil {
		t.Fatalf("count executions for %s: %v", standingApprovalID, err)
	}
	if n != want {
		t.Fatalf("expected %d standing_approval_executions rows for %s, got %d", want, standingApprovalID, n)
	}
}
