package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// InsertCredential creates a credential with no label for the given user.
// The user must already exist via InsertUser.
func InsertCredential(t *testing.T, d db.DBTX, credID, userID, service string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO credentials (id, user_id, service, vault_secret_id) VALUES ($1, $2, $3, '00000000-0000-0000-0000-000000000099')`,
		credID, userID, service)
}

// InsertCredentialWithLabel creates a credential with a specific label.
// The user must already exist via InsertUser.
func InsertCredentialWithLabel(t *testing.T, d db.DBTX, credID, userID, service, label string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO credentials (id, user_id, service, label, vault_secret_id) VALUES ($1, $2, $3, $4, '00000000-0000-0000-0000-000000000099')`,
		credID, userID, service, label)
}

// InsertCredentialWithVaultSecretID creates a credential with a specific vault_secret_id.
// Use this when the test needs to pair the credential with a mock vault secret.
func InsertCredentialWithVaultSecretID(t *testing.T, d db.DBTX, credID, userID, service, vaultSecretID string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO credentials (id, user_id, service, vault_secret_id) VALUES ($1, $2, $3, $4)`,
		credID, userID, service, vaultSecretID)
}
