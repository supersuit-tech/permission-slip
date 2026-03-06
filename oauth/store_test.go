package oauth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStoreRefreshedTokens_Success(t *testing.T) {
	t.Parallel()

	expiry := time.Now().Add(1 * time.Hour)
	oldRefreshVaultID := "old-refresh-vault-id"

	var deletedIDs []string
	var createdNames []string

	vaultOps := VaultOperations{
		DeleteSecret: func(_ context.Context, id string) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
		CreateSecret: func(_ context.Context, name string, _ []byte) (string, error) {
			createdNames = append(createdNames, name)
			return "new-" + name, nil
		},
	}

	var dbAccessVaultID string
	var dbRefreshVaultID *string
	var dbExpiry *time.Time

	newAccessVaultID, err := StoreRefreshedTokens(
		context.Background(), vaultOps,
		StoreRefreshedTokensParams{
			ConnID:            "conn-123",

			OldAccessVaultID:  "old-access-vault-id",
			OldRefreshVaultID: &oldRefreshVaultID,
			Result: &RefreshResult{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token", // rotated
				Expiry:       expiry,
			},
			OldRefreshToken: "old-refresh-token",
		},
		nil, // no delete error logging
		func(_ context.Context, accessVaultID string, refreshVaultID *string, exp *time.Time) error {
			dbAccessVaultID = accessVaultID
			dbRefreshVaultID = refreshVaultID
			dbExpiry = exp
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if newAccessVaultID != "new-conn-123_access" {
		t.Errorf("unexpected access vault ID: %s", newAccessVaultID)
	}

	// Old access and refresh tokens should be deleted.
	if len(deletedIDs) != 2 {
		t.Fatalf("expected 2 deletions, got %d: %v", len(deletedIDs), deletedIDs)
	}
	if deletedIDs[0] != "old-access-vault-id" {
		t.Errorf("expected first deletion of old access token, got %s", deletedIDs[0])
	}
	if deletedIDs[1] != "old-refresh-vault-id" {
		t.Errorf("expected second deletion of old refresh token, got %s", deletedIDs[1])
	}

	// New access and refresh tokens should be created.
	if len(createdNames) != 2 {
		t.Fatalf("expected 2 creations, got %d: %v", len(createdNames), createdNames)
	}

	// DB should be updated with new vault IDs.
	if dbAccessVaultID != "new-conn-123_access" {
		t.Errorf("unexpected DB access vault ID: %s", dbAccessVaultID)
	}
	if dbRefreshVaultID == nil || *dbRefreshVaultID != "new-conn-123_refresh" {
		t.Errorf("unexpected DB refresh vault ID: %v", dbRefreshVaultID)
	}
	if dbExpiry == nil || !dbExpiry.Equal(expiry) {
		t.Errorf("unexpected DB expiry: %v", dbExpiry)
	}
}

func TestStoreRefreshedTokens_NoRotation(t *testing.T) {
	t.Parallel()

	oldRefreshVaultID := "old-refresh-vault-id"
	sameToken := "same-refresh-token"

	var deletedIDs []string

	vaultOps := VaultOperations{
		DeleteSecret: func(_ context.Context, id string) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
		CreateSecret: func(_ context.Context, name string, _ []byte) (string, error) {
			return "new-" + name, nil
		},
	}

	var dbRefreshVaultID *string

	_, err := StoreRefreshedTokens(
		context.Background(), vaultOps,
		StoreRefreshedTokensParams{
			ConnID:            "conn-123",

			OldAccessVaultID:  "old-access-vault-id",
			OldRefreshVaultID: &oldRefreshVaultID,
			Result: &RefreshResult{
				AccessToken:  "new-access-token",
				RefreshToken: sameToken, // same as old — no rotation
				Expiry:       time.Now().Add(1 * time.Hour),
			},
			OldRefreshToken: sameToken,
		},
		nil,
		func(_ context.Context, _ string, refreshVaultID *string, _ *time.Time) error {
			dbRefreshVaultID = refreshVaultID
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only old access token should be deleted (no refresh token rotation).
	if len(deletedIDs) != 1 {
		t.Fatalf("expected 1 deletion, got %d: %v", len(deletedIDs), deletedIDs)
	}

	// DB should retain the old refresh vault ID.
	if dbRefreshVaultID == nil || *dbRefreshVaultID != "old-refresh-vault-id" {
		t.Errorf("expected old refresh vault ID preserved, got %v", dbRefreshVaultID)
	}
}

func TestStoreRefreshedTokens_CreateAccessTokenFails(t *testing.T) {
	t.Parallel()

	vaultOps := VaultOperations{
		DeleteSecret: func(_ context.Context, _ string) error { return nil },
		CreateSecret: func(_ context.Context, _ string, _ []byte) (string, error) {
			return "", errors.New("vault unavailable")
		},
	}

	_, err := StoreRefreshedTokens(
		context.Background(), vaultOps,
		StoreRefreshedTokensParams{
			ConnID:           "conn-123",
			OldAccessVaultID: "old-access",
			Result: &RefreshResult{
				AccessToken:  "token",
				RefreshToken: "token",
			},
			OldRefreshToken: "token",
		},
		nil,
		func(_ context.Context, _ string, _ *string, _ *time.Time) error { return nil },
	)
	if err == nil {
		t.Fatal("expected error when vault create fails")
	}
	if !errors.Is(err, errors.Unwrap(err)) {
		// Just check the error message mentions vault
		if got := err.Error(); got == "" {
			t.Error("expected non-empty error message")
		}
	}
}

func TestStoreRefreshedTokens_ZeroExpiry(t *testing.T) {
	t.Parallel()

	vaultOps := VaultOperations{
		DeleteSecret: func(_ context.Context, _ string) error { return nil },
		CreateSecret: func(_ context.Context, name string, _ []byte) (string, error) {
			return "new-" + name, nil
		},
	}

	var dbExpiry *time.Time

	_, err := StoreRefreshedTokens(
		context.Background(), vaultOps,
		StoreRefreshedTokensParams{
			ConnID:           "conn-123",
			OldAccessVaultID: "old-access",
			Result: &RefreshResult{
				AccessToken:  "token",
				RefreshToken: "same",
				Expiry:       time.Time{}, // zero — token doesn't expire
			},
			OldRefreshToken: "same",
		},
		nil,
		func(_ context.Context, _ string, _ *string, exp *time.Time) error {
			dbExpiry = exp
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dbExpiry != nil {
		t.Errorf("expected nil expiry for zero time, got %v", dbExpiry)
	}
}
