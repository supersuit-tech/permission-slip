// Package vault defines the VaultStore interface for encrypting, storing,
// and retrieving credential secrets. In production this is backed by Supabase
// Vault (server-side AES-256-GCM encryption); in tests it is replaced by an
// in-memory mock.
package vault

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// VaultStore manages encrypted secrets in the credential vault.
//
// All methods accept a db.DBTX so that vault operations participate in the
// caller's database transaction, ensuring atomicity between the credential
// row and the vault secret.
type VaultStore interface {
	// CreateSecret encrypts and stores a secret, returning its UUID.
	// The name is a human-readable identifier (e.g. "cred_abc123") used
	// for auditing; the secret is the raw bytes to encrypt.
	CreateSecret(ctx context.Context, tx db.DBTX, name string, secret []byte) (string, error)

	// ReadSecret retrieves and decrypts a secret by its vault UUID.
	ReadSecret(ctx context.Context, tx db.DBTX, secretID string) ([]byte, error)

	// DeleteSecret removes a secret from the vault. Implementations should
	// be idempotent — deleting a non-existent secret is not an error.
	DeleteSecret(ctx context.Context, tx db.DBTX, secretID string) error
}
