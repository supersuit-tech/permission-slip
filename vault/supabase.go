package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/supersuit-tech/permission-slip/db"
)

// SupabaseVaultStore stores secrets using the Supabase Vault Postgres extension
// (pgsql_vault / supabase_vault). Secrets are encrypted server-side with
// AES-256-GCM using the vault's secret key.
//
// All operations use the caller-provided DBTX so they participate in the
// same transaction as the credential row insert/delete.
type SupabaseVaultStore struct{}

// NewSupabaseVaultStore returns a SupabaseVaultStore.
func NewSupabaseVaultStore() *SupabaseVaultStore {
	return &SupabaseVaultStore{}
}

// CreateSecret encrypts and stores a secret via vault.create_secret().
// Returns the UUID assigned by the vault.
func (s *SupabaseVaultStore) CreateSecret(ctx context.Context, tx db.DBTX, name string, secret []byte) (string, error) {
	var id string
	err := tx.QueryRow(ctx,
		`SELECT vault.create_secret($1, $2)`,
		string(secret), name,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("vault.create_secret: %w", err)
	}
	return id, nil
}

// ReadSecret retrieves and decrypts a secret from vault.decrypted_secrets.
func (s *SupabaseVaultStore) ReadSecret(ctx context.Context, tx db.DBTX, secretID string) ([]byte, error) {
	var decrypted string
	err := tx.QueryRow(ctx,
		`SELECT decrypted_secret FROM vault.decrypted_secrets WHERE id = $1`,
		secretID,
	).Scan(&decrypted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("vault secret %s not found", secretID)
		}
		return nil, fmt.Errorf("vault.read_secret: %w", err)
	}
	return []byte(decrypted), nil
}

// DeleteSecret removes a secret from the vault. Idempotent — does not error
// if the secret does not exist.
func (s *SupabaseVaultStore) DeleteSecret(ctx context.Context, tx db.DBTX, secretID string) error {
	_, err := tx.Exec(ctx,
		`DELETE FROM vault.secrets WHERE id = $1`,
		secretID,
	)
	if err != nil {
		return fmt.Errorf("vault.delete_secret: %w", err)
	}
	return nil
}
