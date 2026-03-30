package testhelper

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/supersuit-tech/permission-slip/db"
)

// PoolUser holds the result of SetupPoolUser — a user with an agent created
// directly on a connection pool (not inside a test transaction). Use this for
// tests that need real transaction boundaries (e.g., concurrent or dedup tests).
//
// All created rows are automatically cleaned up when the test completes.
type PoolUser struct {
	Pool    *pgxpool.Pool
	UserID  string
	AgentID int64
}

// SetupPoolUser creates a user + registered agent on a real pool (not a test
// transaction). This is needed for tests that must exercise real transaction
// boundaries — e.g., concurrent upserts, dedup via unique-constraint
// violations, or any scenario where a single enclosing transaction would
// interfere.
//
// All rows are cleaned up via t.Cleanup in reverse-FK order.
func SetupPoolUser(t *testing.T, prefix string, publicKey string) PoolUser {
	t.Helper()
	pool := SetupPool(t)
	ctx := context.Background()

	uid := GenerateUID(t)
	if _, err := pool.Exec(ctx,
		`INSERT INTO auth.users (id) VALUES ($1)`, uid); err != nil {
		t.Fatalf("SetupPoolUser: insert auth.users: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO profiles (id, username) VALUES ($1, $2)`, uid, prefix+"_"+uid[:8]); err != nil {
		t.Fatalf("SetupPoolUser: insert profiles: %v", err)
	}

	agentID := InsertAgentWithPublicKey(t, pool, uid, "registered", publicKey)

	t.Cleanup(func() {
		bg := context.Background()
		// Delete in reverse-FK order to avoid constraint violations.
		pool.Exec(bg, `DELETE FROM request_ids WHERE agent_id = $1`, agentID)
		pool.Exec(bg, `DELETE FROM standing_approval_executions WHERE standing_approval_id IN (SELECT standing_approval_id FROM standing_approvals WHERE agent_id = $1)`, agentID)
		pool.Exec(bg, `DELETE FROM standing_approvals WHERE agent_id = $1`, agentID)
		pool.Exec(bg, `DELETE FROM approvals WHERE agent_id = $1`, agentID)
		pool.Exec(bg, `DELETE FROM audit_events WHERE user_id = $1`, uid)
		pool.Exec(bg, `DELETE FROM usage_periods WHERE user_id = $1`, uid)
		pool.Exec(bg, `DELETE FROM subscriptions WHERE user_id = $1`, uid)
		pool.Exec(bg, `DELETE FROM agents WHERE agent_id = $1`, agentID)
		pool.Exec(bg, `DELETE FROM profiles WHERE id = $1`, uid)
		pool.Exec(bg, `DELETE FROM auth.users WHERE id = $1`, uid)
	})

	return PoolUser{Pool: pool, UserID: uid, AgentID: agentID}
}

// DBTX returns the pool as a db.DBTX interface, convenient for passing to
// functions that accept either a pool or transaction.
func (pu PoolUser) DBTX() db.DBTX {
	return pu.Pool
}
