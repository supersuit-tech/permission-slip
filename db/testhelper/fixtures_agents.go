package testhelper

import (
	"context"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// InsertUserWithAgent creates a user (profile + auth.users) and an agent
// owned by that user in 'pending' status. Returns the auto-generated agent_id.
func InsertUserWithAgent(t *testing.T, d db.DBTX, uid, username string) int64 {
	t.Helper()
	InsertUser(t, d, uid, username)
	return InsertAgent(t, d, uid)
}

// InsertAgent creates an agent in 'pending' status with a placeholder public key.
// The approver (uid) must already exist via InsertUser. Returns the auto-generated agent_id.
func InsertAgent(t *testing.T, d db.DBTX, approverID string) int64 {
	t.Helper()
	return InsertAgentWithStatus(t, d, approverID, "pending")
}

// InsertAgentWithStatus creates an agent with the given status.
// The approver must already exist via InsertUser. Returns the auto-generated agent_id.
func InsertAgentWithStatus(t *testing.T, d db.DBTX, approverID, status string) int64 {
	t.Helper()
	var agentID int64
	err := d.QueryRow(context.Background(),
		`INSERT INTO agents (public_key, approver_id, status) VALUES ('pk', $1, $2) RETURNING agent_id`,
		approverID, status).Scan(&agentID)
	if err != nil {
		t.Fatalf("InsertAgentWithStatus: %v", err)
	}
	return agentID
}

// InsertAgentWithCreatedAt creates a pending agent with an explicit created_at timestamp.
// Useful for pagination tests that need deterministic ordering.
// The approver must already exist via InsertUser. Returns the auto-generated agent_id.
func InsertAgentWithCreatedAt(t *testing.T, d db.DBTX, approverID string, createdAt time.Time) int64 {
	t.Helper()
	var agentID int64
	err := d.QueryRow(context.Background(),
		`INSERT INTO agents (public_key, approver_id, status, created_at) VALUES ('pk', $1, 'pending', $2) RETURNING agent_id`,
		approverID, createdAt).Scan(&agentID)
	if err != nil {
		t.Fatalf("InsertAgentWithCreatedAt: %v", err)
	}
	return agentID
}

// InsertAgentWithPublicKey creates an agent with the given status and a real
// public key (e.g. an OpenSSH-format Ed25519 key). Use this when testing
// agent signature authentication.
func InsertAgentWithPublicKey(t *testing.T, d db.DBTX, approverID, status, publicKey string) int64 {
	t.Helper()
	var agentID int64
	err := d.QueryRow(context.Background(),
		`INSERT INTO agents (public_key, approver_id, status, registered_at)
		 VALUES ($1, $2, $3, CASE WHEN $3 = 'registered' THEN now() ELSE NULL END)
		 RETURNING agent_id`,
		publicKey, approverID, status).Scan(&agentID)
	if err != nil {
		t.Fatalf("InsertAgentWithPublicKey: %v", err)
	}
	return agentID
}

// SetAgentLastActiveAt sets the last_active_at timestamp on an existing agent.
func SetAgentLastActiveAt(t *testing.T, d db.DBTX, agentID int64, approverID string, lastActiveAt time.Time) {
	t.Helper()
	mustExec(t, d,
		`UPDATE agents SET last_active_at = $1 WHERE agent_id = $2 AND approver_id = $3`,
		lastActiveAt, agentID, approverID)
}
