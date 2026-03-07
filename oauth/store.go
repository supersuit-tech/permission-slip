package oauth

import (
	"context"
	"fmt"
	"time"
)

// StoreRefreshedTokensParams holds the inputs for persisting refreshed OAuth tokens.
type StoreRefreshedTokensParams struct {
	ConnID            string         // OAuth connection ID (used for vault secret naming).
	OldAccessVaultID  string         // Current access token vault secret ID.
	OldRefreshVaultID *string        // Current refresh token vault secret ID (nil if none).
	Result            *RefreshResult // The new tokens from the provider.
	OldRefreshToken   string         // The old refresh token value (for rotation detection).
}

// VaultOperations defines the vault operations needed to store refreshed tokens.
// Callers provide these as closures bound to the current transaction.
type VaultOperations struct {
	DeleteSecret func(ctx context.Context, id string) error
	CreateSecret func(ctx context.Context, name string, secret []byte) (string, error)
}

// StoreRefreshedTokens persists refreshed OAuth tokens in the vault and updates
// the database row. It handles refresh token rotation (storing the new refresh
// token when the provider rotates it) and old secret cleanup.
//
// The updateDB callback should update the oauth_connections row with the new
// vault IDs and expiry within the same transaction used for vault operations.
//
// Returns the new access token vault ID so callers can update in-memory state.
func StoreRefreshedTokens(
	ctx context.Context,
	vault VaultOperations,
	p StoreRefreshedTokensParams,
	logDeleteErr func(what, vaultID string, err error),
	updateDB func(ctx context.Context, accessVaultID string, refreshVaultID *string, expiry *time.Time) error,
) (newAccessVaultID string, err error) {
	// Delete old access token (best-effort) and create the new one.
	if delErr := vault.DeleteSecret(ctx, p.OldAccessVaultID); delErr != nil && logDeleteErr != nil {
		logDeleteErr("access token", p.OldAccessVaultID, delErr)
	}
	newAccessVaultID, err = vault.CreateSecret(ctx, p.ConnID+"_access", []byte(p.Result.AccessToken))
	if err != nil {
		return "", fmt.Errorf("vault create access token: %w", err)
	}

	// Handle refresh token rotation.
	var newRefreshVaultID *string
	if p.Result.RefreshToken != p.OldRefreshToken {
		// Provider rotated the refresh token — store the new one.
		if p.OldRefreshVaultID != nil {
			if delErr := vault.DeleteSecret(ctx, *p.OldRefreshVaultID); delErr != nil && logDeleteErr != nil {
				logDeleteErr("refresh token", *p.OldRefreshVaultID, delErr)
			}
		}
		id, createErr := vault.CreateSecret(ctx, p.ConnID+"_refresh", []byte(p.Result.RefreshToken))
		if createErr != nil {
			return "", fmt.Errorf("vault create refresh token: %w", createErr)
		}
		newRefreshVaultID = &id
	} else {
		newRefreshVaultID = p.OldRefreshVaultID
	}

	var tokenExpiry *time.Time
	if !p.Result.Expiry.IsZero() {
		tokenExpiry = &p.Result.Expiry
	}

	if err := updateDB(ctx, newAccessVaultID, newRefreshVaultID, tokenExpiry); err != nil {
		return "", fmt.Errorf("update connection tokens: %w", err)
	}

	return newAccessVaultID, nil
}
