package testhelper

import (
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// InsertOAuthConnection creates an OAuth connection for the given user and provider.
// Uses placeholder vault IDs. The user must already exist via InsertUser.
func InsertOAuthConnection(t *testing.T, d db.DBTX, connID, userID, provider string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO oauth_connections (id, user_id, provider, access_token_vault_id, scopes, status)
		 VALUES ($1, $2, $3, '00000000-0000-0000-0000-000000000001', '{}', 'active')`,
		connID, userID, provider)
}

// OAuthConnectionOpts holds optional fields for InsertOAuthConnectionFull.
type OAuthConnectionOpts struct {
	AccessTokenVaultID  string
	RefreshTokenVaultID *string
	Scopes              []string
	TokenExpiry         *time.Time
	Status              string
	// ExtraData is optional JSON for oauth_connections.extra_data (nil = NULL).
	ExtraData []byte
}

// InsertOAuthConnectionFull creates an OAuth connection with full control over all fields.
func InsertOAuthConnectionFull(t *testing.T, d db.DBTX, connID, userID, provider string, opts OAuthConnectionOpts) {
	t.Helper()
	status := opts.Status
	if status == "" {
		status = "active"
	}
	accessVaultID := opts.AccessTokenVaultID
	if accessVaultID == "" {
		accessVaultID = "00000000-0000-0000-0000-000000000001"
	}
	mustExec(t, d,
		`INSERT INTO oauth_connections (id, user_id, provider, access_token_vault_id, refresh_token_vault_id, scopes, token_expiry, status, extra_data)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		connID, userID, provider, accessVaultID, opts.RefreshTokenVaultID, opts.Scopes, opts.TokenExpiry, status, opts.ExtraData)
}

// InsertConnectorRequiredCredentialOAuth creates an OAuth-type required credential entry.
// The connector must already exist via InsertConnector.
func InsertConnectorRequiredCredentialOAuth(t *testing.T, d db.DBTX, connectorID, service, oauthProvider string, oauthScopes []string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO connector_required_credentials (connector_id, service, auth_type, oauth_provider, oauth_scopes)
		 VALUES ($1, $2, 'oauth2', $3, $4)`,
		connectorID, service, oauthProvider, oauthScopes)
}
