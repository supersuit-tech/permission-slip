package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

// InsertUser creates a users row and a corresponding profiles row.
// This is the base fixture needed by most tests since users→profiles is the root
// of the FK graph.
func InsertUser(t *testing.T, d db.DBTX, uid, username string) {
	t.Helper()
	mustExec(t, d, `INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'test-stub-hash')`,
		uid, uid+"@test.local")
	mustExec(t, d, `INSERT INTO profiles (id, username) VALUES ($1, $2)`, uid, username)
}
