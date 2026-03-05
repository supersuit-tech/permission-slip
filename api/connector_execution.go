package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

// executeConnectorAction looks up the action in the connector registry,
// fetches and decrypts the user's credentials, and executes the action.
// Returns (nil, nil) if no connector is registered for the action type
// (graceful degradation during the transition period).
func executeConnectorAction(ctx context.Context, deps *Deps, userID, actionType string, parameters json.RawMessage) (*connectors.ActionResult, error) {
	if deps.Connectors == nil {
		return nil, nil
	}

	action, ok := deps.Connectors.GetAction(actionType)
	if !ok {
		return nil, nil
	}

	// Check auth_type for this connector's required credentials.
	reqCred, err := db.GetRequiredCredentialByActionType(ctx, deps.DB, actionType)
	if err != nil {
		return nil, fmt.Errorf("look up required credential: %w", err)
	}

	var creds connectors.Credentials
	if reqCred != nil && reqCred.AuthType == "oauth2" {
		// OAuth2 path: resolve access token from oauth_connections.
		creds, err = resolveOAuthCredentials(ctx, deps, userID, reqCred)
		if err != nil {
			return nil, err
		}
	} else {
		// Static credential path (api_key, basic, custom): existing behavior.
		creds, err = resolveStaticCredentials(ctx, deps, userID, actionType)
		if err != nil {
			return nil, err
		}
	}

	// Validate credentials before executing the action.
	connectorID := strings.SplitN(actionType, ".", 2)[0]
	if conn, ok := deps.Connectors.Get(connectorID); ok {
		if err := conn.ValidateCredentials(ctx, creds); err != nil {
			return nil, err
		}
	}

	result, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  actionType,
		Parameters:  parameters,
		Credentials: creds,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// resolveStaticCredentials fetches and decrypts static credentials (api_key, basic, custom)
// for the given action type.
func resolveStaticCredentials(ctx context.Context, deps *Deps, userID, actionType string) (connectors.Credentials, error) {
	services, err := db.GetRequiredServicesByActionType(ctx, deps.DB, actionType)
	if err != nil {
		return connectors.Credentials{}, fmt.Errorf("look up required services: %w", err)
	}

	var zero connectors.Credentials
	credMap := make(map[string]string, len(services))
	for _, service := range services {
		if deps.Vault == nil {
			return zero, fmt.Errorf("credential vault is not configured but connector requires service %q", service)
		}
		decrypted, err := db.GetDecryptedCredentials(ctx, deps.DB, deps.Vault.ReadSecret, userID, service, nil)
		if err != nil {
			var credErr *db.CredentialError
			if errors.As(err, &credErr) && credErr.Code == db.CredentialErrNotFound {
				return zero, &connectors.ValidationError{
					Message: fmt.Sprintf("no credentials stored for service %q", service),
				}
			}
			return zero, fmt.Errorf("decrypt credentials for service %q: %w", service, err)
		}
		// Flatten decrypted JSON map into string values for the Credentials type.
		for k, v := range decrypted {
			switch vv := v.(type) {
			case string:
				credMap[k] = vv
			default:
				b, err := json.Marshal(v)
				if err != nil {
					return zero, fmt.Errorf("marshal credential %q for service %q: %w", k, service, err)
				}
				credMap[k] = string(b)
			}
		}
	}

	return connectors.NewCredentials(credMap), nil
}

// resolveOAuthCredentials looks up the user's OAuth connection for the required
// provider, refreshes the token if it's expired or near-expiry, and returns
// credentials containing the valid access token.
func resolveOAuthCredentials(ctx context.Context, deps *Deps, userID string, reqCred *db.RequiredCredential) (connectors.Credentials, error) {
	var zero connectors.Credentials
	if deps.Vault == nil {
		return zero, fmt.Errorf("credential vault is not configured but connector requires OAuth credentials")
	}
	if deps.OAuthProviders == nil {
		return zero, fmt.Errorf("OAuth provider registry is not configured")
	}

	providerID := ""
	if reqCred.OAuthProvider != nil {
		providerID = *reqCred.OAuthProvider
	}
	if providerID == "" {
		return zero, &connectors.ValidationError{
			Message: "connector requires OAuth but no oauth_provider is configured",
		}
	}

	// Look up the user's OAuth connection for this provider.
	conn, err := db.GetOAuthConnectionByProvider(ctx, deps.DB, userID, providerID)
	if err != nil {
		return zero, fmt.Errorf("look up OAuth connection for provider %q: %w", providerID, err)
	}
	if conn == nil {
		return zero, &connectors.OAuthRefreshError{
			Provider: providerID,
			Message:  fmt.Sprintf("no OAuth connection for provider %q — user must connect via Settings", providerID),
		}
	}

	// Check connection status.
	if conn.Status == db.OAuthStatusNeedsReauth || conn.Status == db.OAuthStatusRevoked {
		return zero, &connectors.OAuthRefreshError{
			Provider: providerID,
			Message:  fmt.Sprintf("OAuth connection for %q has status %q — user must re-authorize", providerID, conn.Status),
		}
	}

	// Check if the token needs refreshing (expired or within 5-minute buffer).
	if conn.TokenExpiry != nil && time.Now().After(conn.TokenExpiry.Add(-5*time.Minute)) {
		if err := refreshOAuthConnection(ctx, deps, conn, providerID); err != nil {
			return zero, err
		}
	}

	// Read the (possibly refreshed) access token from the vault.
	accessTokenBytes, err := deps.Vault.ReadSecret(ctx, deps.DB, conn.AccessTokenVaultID)
	if err != nil {
		return zero, fmt.Errorf("read access token from vault: %w", err)
	}

	return connectors.NewCredentials(map[string]string{
		"access_token": string(accessTokenBytes),
	}), nil
}

// refreshOAuthConnection performs a token refresh for the given OAuth connection.
// On success, it updates the vault secrets and DB row. On failure, it marks
// the connection as needs_reauth and returns an OAuthRefreshError.
func refreshOAuthConnection(ctx context.Context, deps *Deps, conn *db.OAuthConnection, providerID string) error {
	provider, ok := deps.OAuthProviders.Get(providerID)
	if !ok {
		return &connectors.OAuthRefreshError{
			Provider: providerID,
			Message:  fmt.Sprintf("OAuth provider %q is not registered", providerID),
		}
	}

	if conn.RefreshTokenVaultID == nil {
		// No refresh token — can't refresh. Mark as needs_reauth.
		if statusErr := db.UpdateOAuthConnectionStatus(ctx, deps.DB, conn.ID, conn.UserID, db.OAuthStatusNeedsReauth); statusErr != nil {
			log.Printf("failed to update OAuth connection status to needs_reauth: %v", statusErr)
		}
		return &connectors.OAuthRefreshError{
			Provider: providerID,
			Message:  "token expired and no refresh token available — user must re-authorize",
		}
	}

	// Read the refresh token from the vault.
	refreshTokenBytes, err := deps.Vault.ReadSecret(ctx, deps.DB, *conn.RefreshTokenVaultID)
	if err != nil {
		return fmt.Errorf("read refresh token from vault: %w", err)
	}

	// Perform the refresh.
	result, err := oauth.RefreshTokens(ctx, provider, string(refreshTokenBytes))
	if err != nil {
		// Refresh failed (token revoked, expired, etc.). Mark as needs_reauth.
		if statusErr := db.UpdateOAuthConnectionStatus(ctx, deps.DB, conn.ID, conn.UserID, db.OAuthStatusNeedsReauth); statusErr != nil {
			log.Printf("failed to update OAuth connection status to needs_reauth: %v", statusErr)
		}
		return &connectors.OAuthRefreshError{
			Provider: providerID,
			Message:  fmt.Sprintf("token refresh failed — user must re-authorize: %v", err),
		}
	}

	// Store the new access token in the vault (update in place by deleting
	// old and creating new, since vault doesn't have an update operation).
	tx, owned, txErr := db.BeginOrContinue(ctx, deps.DB)
	if txErr != nil {
		return fmt.Errorf("begin tx for token refresh: %w", txErr)
	}
	if owned {
		defer db.RollbackTx(ctx, tx) //nolint:errcheck // best-effort cleanup
	}

	// Delete old access token and create new one.
	_ = deps.Vault.DeleteSecret(ctx, tx, conn.AccessTokenVaultID)
	newAccessVaultID, err := deps.Vault.CreateSecret(ctx, tx, conn.ID+"_access", []byte(result.AccessToken))
	if err != nil {
		return fmt.Errorf("vault create refreshed access token: %w", err)
	}

	// Handle refresh token rotation.
	var newRefreshVaultID *string
	if result.RefreshToken != string(refreshTokenBytes) {
		// Provider rotated the refresh token — store the new one.
		_ = deps.Vault.DeleteSecret(ctx, tx, *conn.RefreshTokenVaultID)
		id, err := deps.Vault.CreateSecret(ctx, tx, conn.ID+"_refresh", []byte(result.RefreshToken))
		if err != nil {
			return fmt.Errorf("vault create rotated refresh token: %w", err)
		}
		newRefreshVaultID = &id
	} else {
		newRefreshVaultID = conn.RefreshTokenVaultID
	}

	var tokenExpiry *time.Time
	if !result.Expiry.IsZero() {
		tokenExpiry = &result.Expiry
	}

	if err := db.UpdateOAuthConnectionTokens(ctx, tx, conn.ID, conn.UserID, newAccessVaultID, newRefreshVaultID, tokenExpiry); err != nil {
		return fmt.Errorf("update OAuth connection tokens: %w", err)
	}

	if owned {
		if err := db.CommitTx(ctx, tx); err != nil {
			return fmt.Errorf("commit token refresh: %w", err)
		}
	}

	// Update the in-memory connection so the caller reads the new vault ID.
	conn.AccessTokenVaultID = newAccessVaultID

	return nil
}
