package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

// InsertRegistrationInvite creates an active registration invite for the given user.
// The user must already exist via InsertUser.
func InsertRegistrationInvite(t *testing.T, d db.DBTX, id, userID string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
		 VALUES ($1, $2, $3, 'active', now() + interval '1 hour')`,
		id, userID, "testhash_"+id)
}
