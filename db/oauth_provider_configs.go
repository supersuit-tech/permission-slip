package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OAuthProviderConfig represents a row from the oauth_provider_configs table.
// This stores user-provided ("bring your own app") OAuth client credentials.
//
// The actual client ID and secret are encrypted in Supabase Vault — this
// struct only holds the vault secret IDs (references), not plaintext values.
// The table has a unique constraint on (user_id, provider) to prevent
// duplicate configs.
type OAuthProviderConfig struct {
	ID                   string
	UserID               string
	Provider             string
	ClientIDVaultID      string
	ClientSecretVaultID  string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// CreateOAuthProviderConfigParams holds the parameters for inserting a new provider config.
type CreateOAuthProviderConfigParams struct {
	ID                   string
	UserID               string
	Provider             string
	ClientIDVaultID      string
	ClientSecretVaultID  string
}

// OAuthProviderConfigError represents a domain-specific error from provider config operations.
type OAuthProviderConfigError struct {
	Code    OAuthProviderConfigErrCode
	Message string
}

func (e *OAuthProviderConfigError) Error() string { return e.Message }

// OAuthProviderConfigErrCode enumerates provider config error codes.
type OAuthProviderConfigErrCode int

const (
	OAuthProviderConfigErrNotFound  OAuthProviderConfigErrCode = iota
	OAuthProviderConfigErrDuplicate                            // unique (user_id, provider) violation
)

// ListAllOAuthProviderConfigs returns all OAuth provider configs across all users,
// ordered by creation time ascending. Used at startup to reload BYOA credentials
// into the in-memory provider registry.
func ListAllOAuthProviderConfigs(ctx context.Context, db DBTX) ([]OAuthProviderConfig, error) {
	rows, err := db.Query(ctx, `
		SELECT id, user_id, provider, client_id_vault_id, client_secret_vault_id, created_at, updated_at
		FROM oauth_provider_configs
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []OAuthProviderConfig
	for rows.Next() {
		var c OAuthProviderConfig
		if err := rows.Scan(&c.ID, &c.UserID, &c.Provider, &c.ClientIDVaultID, &c.ClientSecretVaultID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// ListOAuthProviderConfigsByUser returns all OAuth provider configs for the given user.
func ListOAuthProviderConfigsByUser(ctx context.Context, db DBTX, userID string) ([]OAuthProviderConfig, error) {
	rows, err := db.Query(ctx, `
		SELECT id, user_id, provider, client_id_vault_id, client_secret_vault_id, created_at, updated_at
		FROM oauth_provider_configs
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []OAuthProviderConfig
	for rows.Next() {
		var c OAuthProviderConfig
		if err := rows.Scan(&c.ID, &c.UserID, &c.Provider, &c.ClientIDVaultID, &c.ClientSecretVaultID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// GetOAuthProviderConfig returns the provider config for a given user and provider.
// Returns nil if no config exists.
func GetOAuthProviderConfig(ctx context.Context, db DBTX, userID, provider string) (*OAuthProviderConfig, error) {
	var c OAuthProviderConfig
	err := db.QueryRow(ctx, `
		SELECT id, user_id, provider, client_id_vault_id, client_secret_vault_id, created_at, updated_at
		FROM oauth_provider_configs
		WHERE user_id = $1 AND provider = $2`,
		userID, provider,
	).Scan(&c.ID, &c.UserID, &c.Provider, &c.ClientIDVaultID, &c.ClientSecretVaultID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateOAuthProviderConfig inserts a new provider config row.
func CreateOAuthProviderConfig(ctx context.Context, db DBTX, p CreateOAuthProviderConfigParams) (*OAuthProviderConfig, error) {
	var c OAuthProviderConfig
	err := db.QueryRow(ctx, `
		INSERT INTO oauth_provider_configs (id, user_id, provider, client_id_vault_id, client_secret_vault_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, provider, client_id_vault_id, client_secret_vault_id, created_at, updated_at`,
		p.ID, p.UserID, p.Provider, p.ClientIDVaultID, p.ClientSecretVaultID,
	).Scan(&c.ID, &c.UserID, &c.Provider, &c.ClientIDVaultID, &c.ClientSecretVaultID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgCodeUniqueViolation {
			return nil, &OAuthProviderConfigError{Code: OAuthProviderConfigErrDuplicate, Message: "OAuth provider config already exists for this provider"}
		}
		return nil, err
	}
	return &c, nil
}

// UpdateOAuthProviderConfig updates the vault secret IDs for an existing provider
// config and bumps the updated_at timestamp. Used for credential rotation.
func UpdateOAuthProviderConfig(ctx context.Context, db DBTX, userID, provider, clientIDVaultID, clientSecretVaultID string) (*OAuthProviderConfig, error) {
	var c OAuthProviderConfig
	err := db.QueryRow(ctx, `
		UPDATE oauth_provider_configs
		SET client_id_vault_id = $3, client_secret_vault_id = $4, updated_at = now()
		WHERE user_id = $1 AND provider = $2
		RETURNING id, user_id, provider, client_id_vault_id, client_secret_vault_id, created_at, updated_at`,
		userID, provider, clientIDVaultID, clientSecretVaultID,
	).Scan(&c.ID, &c.UserID, &c.Provider, &c.ClientIDVaultID, &c.ClientSecretVaultID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &OAuthProviderConfigError{Code: OAuthProviderConfigErrNotFound, Message: "OAuth provider config not found"}
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// DeleteOAuthProviderConfigResult holds the result of a provider config deletion.
type DeleteOAuthProviderConfigResult struct {
	ClientIDVaultID     string
	ClientSecretVaultID string
}

// DeleteOAuthProviderConfig deletes a provider config by user and provider.
// Returns the vault secret IDs so the caller can delete the vault secrets too.
func DeleteOAuthProviderConfig(ctx context.Context, db DBTX, userID, provider string) (*DeleteOAuthProviderConfigResult, error) {
	var result DeleteOAuthProviderConfigResult
	err := db.QueryRow(ctx, `
		DELETE FROM oauth_provider_configs
		WHERE user_id = $1 AND provider = $2
		RETURNING client_id_vault_id, client_secret_vault_id`,
		userID, provider,
	).Scan(&result.ClientIDVaultID, &result.ClientSecretVaultID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &OAuthProviderConfigError{Code: OAuthProviderConfigErrNotFound, Message: "OAuth provider config not found"}
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}
