package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// InsertUser creates an auth.users row and a corresponding profiles row.
// This is the base fixture needed by most tests since profiles is the root
// of the FK graph.
func InsertUser(t *testing.T, d db.DBTX, uid, username string) {
	t.Helper()
	mustExec(t, d, `INSERT INTO auth.users (id) VALUES ($1)`, uid)
	mustExec(t, d, `INSERT INTO profiles (id, username) VALUES ($1, $2)`, uid, username)
}
