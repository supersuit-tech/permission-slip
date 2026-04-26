package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// OAuth connection status values. Must match the CHECK constraint on
// oauth_connections.status in the migration.
const (
	OAuthStatusActive      = "active"
	OAuthStatusNeedsReauth = "needs_reauth"
	OAuthStatusRevoked     = "revoked"
)

// OAuthConnection represents a row from the oauth_connections table.
type OAuthConnection struct {
	ID                  string
	UserID              string
	Provider            string
	AccessTokenVaultID  string
	RefreshTokenVaultID *string
	Scopes              []string
	TokenExpiry         *time.Time
	Status              string
	ExtraData           json.RawMessage // provider-specific data (e.g. Salesforce instance_url)
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// CreateOAuthConnectionParams holds the parameters for inserting a new OAuth connection.
type CreateOAuthConnectionParams struct {
	ID                  string
	UserID              string
	Provider            string
	AccessTokenVaultID  string
	RefreshTokenVaultID *string
	Scopes              []string
	TokenExpiry         *time.Time
	ExtraData           json.RawMessage // optional provider-specific data
}

// OAuthConnectionError represents a domain-specific error from OAuth connection operations.
type OAuthConnectionError struct {
	Code    OAuthConnectionErrCode
	Message string
}

func (e *OAuthConnectionError) Error() string { return e.Message }

// OAuthConnectionErrCode enumerates OAuth connection error codes.
type OAuthConnectionErrCode int

const (
	OAuthConnectionErrNotFound OAuthConnectionErrCode = iota
)

// oauthConnectionColumns is the shared SELECT column list for oauth_connections queries.
const oauthConnectionColumns = `id, user_id, provider, access_token_vault_id, refresh_token_vault_id,
		       scopes, token_expiry, status, extra_data, created_at, updated_at`

// scanOAuthConnection scans an OAuthConnection from a row scanner (pgx.Row or pgx.Rows).
func scanOAuthConnection(scan func(dest ...any) error) (OAuthConnection, error) {
	var c OAuthConnection
	err := scan(&c.ID, &c.UserID, &c.Provider, &c.AccessTokenVaultID,
		&c.RefreshTokenVaultID, &c.Scopes, &c.TokenExpiry, &c.Status,
		&c.ExtraData, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// ListOAuthConnectionsByUser returns all OAuth connections for the given user,
// ordered by created_at descending (newest first). Tokens are never returned.
func ListOAuthConnectionsByUser(ctx context.Context, db DBTX, userID string) ([]OAuthConnection, error) {
	rows, err := db.Query(ctx, `
		SELECT `+oauthConnectionColumns+`
		FROM oauth_connections
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []OAuthConnection
	for rows.Next() {
		c, err := scanOAuthConnection(rows.Scan)
		if err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

// GetOAuthConnectionByProvider returns the most recent OAuth connection for a given user and
// provider (ORDER BY created_at DESC, id DESC LIMIT 1). When multiple connections exist for the same
// user+provider, ordering is deterministic even if created_at ties. Returns nil if no connection exists.
func GetOAuthConnectionByProvider(ctx context.Context, db DBTX, userID, provider string) (*OAuthConnection, error) {
	row := db.QueryRow(ctx, `
		SELECT `+oauthConnectionColumns+`
		FROM oauth_connections
		WHERE user_id = $1 AND provider = $2
		ORDER BY created_at DESC, id DESC
		LIMIT 1`,
		userID, provider,
	)
	c, err := scanOAuthConnection(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ReloadOAuthConnectionIfConcurrentRefreshSucceeded re-reads the connection row after a failed
// token refresh. If another path refreshed successfully in the meantime, the access-token vault ID
// and/or token_expiry typically change while status stays active. We only treat the row as a
// concurrent success when status is still active and at least one refresh-specific signal changed
// — not on updated_at alone (another path may have flipped to needs_reauth and also bumped updated_at).
//
// Returns (freshConn, true) when the re-read shows an active connection that looks successfully
// refreshed vs. the caller's snapshot; the caller should use freshConn and skip flipping to
// needs_reauth. On false, the caller may proceed to mark needs_reauth.
// Returns (nil, false, nil) if the row disappeared.
func ReloadOAuthConnectionIfConcurrentRefreshSucceeded(ctx context.Context, db DBTX, connID, userID string, lastKnownAccessVaultID string, lastKnownTokenExpiry *time.Time) (*OAuthConnection, bool, error) {
	fresh, err := GetOAuthConnectionByID(ctx, db, connID)
	if err != nil {
		return nil, false, err
	}
	if fresh == nil || fresh.UserID != userID {
		return nil, false, nil
	}
	if fresh.Status != OAuthStatusActive {
		return fresh, false, nil
	}
	accessRotated := fresh.AccessTokenVaultID != "" && fresh.AccessTokenVaultID != lastKnownAccessVaultID
	expiryAdvanced := lastKnownTokenExpiry != nil && fresh.TokenExpiry != nil && fresh.TokenExpiry.After(*lastKnownTokenExpiry)
	expiryBackfilled := lastKnownTokenExpiry == nil && fresh.TokenExpiry != nil
	if accessRotated || expiryAdvanced || expiryBackfilled {
		return fresh, true, nil
	}
	return fresh, false, nil
}

// CreateOAuthConnection inserts a new OAuth connection row.
func CreateOAuthConnection(ctx context.Context, db DBTX, p CreateOAuthConnectionParams) (*OAuthConnection, error) {
	row := db.QueryRow(ctx, `
		INSERT INTO oauth_connections (id, user_id, provider, access_token_vault_id, refresh_token_vault_id, scopes, token_expiry, extra_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+oauthConnectionColumns,
		p.ID, p.UserID, p.Provider, p.AccessTokenVaultID, p.RefreshTokenVaultID, p.Scopes, p.TokenExpiry, p.ExtraData,
	)
	c, err := scanOAuthConnection(row.Scan)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetOAuthConnectionByID returns a single OAuth connection by its ID.
// Returns nil if not found. Useful for token refresh and status update flows.
func GetOAuthConnectionByID(ctx context.Context, db DBTX, id string) (*OAuthConnection, error) {
	row := db.QueryRow(ctx, `
		SELECT `+oauthConnectionColumns+`
		FROM oauth_connections
		WHERE id = $1`, id)
	c, err := scanOAuthConnection(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateOAuthConnectionTokens updates the vault secret IDs and expiry for a connection
// after a successful token refresh. The userID parameter ensures the caller
// owns the connection (defense-in-depth against horizontal privilege escalation).
func UpdateOAuthConnectionTokens(ctx context.Context, db DBTX, id, userID string, accessTokenVaultID string, refreshTokenVaultID *string, tokenExpiry *time.Time) error {
	result, err := db.Exec(ctx, `
		UPDATE oauth_connections
		SET access_token_vault_id = $2,
		    refresh_token_vault_id = $3,
		    token_expiry = $4,
		    status = $5,
		    updated_at = now()
		WHERE id = $1 AND user_id = $6`,
		id, accessTokenVaultID, refreshTokenVaultID, tokenExpiry, OAuthStatusActive, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return &OAuthConnectionError{Code: OAuthConnectionErrNotFound, Message: "OAuth connection not found"}
	}
	return nil
}

// validOAuthStatuses is the set of valid OAuth connection statuses.
// Must match the CHECK constraint on oauth_connections.status.
var validOAuthStatuses = map[string]bool{
	OAuthStatusActive:      true,
	OAuthStatusNeedsReauth: true,
	OAuthStatusRevoked:     true,
}

// UpdateOAuthConnectionStatus updates the status of an OAuth connection.
// Returns an error if the status value is not one of the valid constants.
// The userID parameter ensures the caller owns the connection.
func UpdateOAuthConnectionStatus(ctx context.Context, db DBTX, id, userID, status string) error {
	if !validOAuthStatuses[status] {
		return fmt.Errorf("invalid OAuth connection status %q", status)
	}
	result, err := db.Exec(ctx, `
		UPDATE oauth_connections
		SET status = $2, updated_at = now()
		WHERE id = $1 AND user_id = $3`,
		id, status, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return &OAuthConnectionError{Code: OAuthConnectionErrNotFound, Message: "OAuth connection not found"}
	}
	return nil
}

// DeleteOAuthConnectionResult holds the result of an OAuth connection deletion.
type DeleteOAuthConnectionResult struct {
	AccessTokenVaultID  string
	RefreshTokenVaultID *string
}

// DeleteOAuthConnectionByID deletes a single OAuth connection by its ID.
// The userID parameter ensures the caller owns the connection (defense-in-depth).
// Returns the vault secret IDs so the caller can delete the vault secrets too.
func DeleteOAuthConnectionByID(ctx context.Context, db DBTX, userID, connectionID string) (*DeleteOAuthConnectionResult, error) {
	var result DeleteOAuthConnectionResult
	err := db.QueryRow(ctx, `
		DELETE FROM oauth_connections
		WHERE id = $1 AND user_id = $2
		RETURNING access_token_vault_id, refresh_token_vault_id`,
		connectionID, userID,
	).Scan(&result.AccessTokenVaultID, &result.RefreshTokenVaultID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &OAuthConnectionError{Code: OAuthConnectionErrNotFound, Message: "OAuth connection not found"}
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListExpiringOAuthConnections returns all active OAuth connections whose
// token_expiry is within the given horizon from now. Used by the background
// refresh job to proactively refresh tokens before they expire.
// Only returns connections that have a refresh token (refresh_token_vault_id IS NOT NULL).
func ListExpiringOAuthConnections(ctx context.Context, db DBTX, horizon time.Duration) ([]OAuthConnection, error) {
	rows, err := db.Query(ctx, `
		SELECT `+oauthConnectionColumns+`
		FROM oauth_connections
		WHERE status = $1
		  AND refresh_token_vault_id IS NOT NULL
		  AND token_expiry IS NOT NULL
		  AND token_expiry <= now() + $2::interval
		ORDER BY token_expiry ASC`,
		OAuthStatusActive, fmt.Sprintf("%d seconds", int(horizon.Seconds())),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []OAuthConnection
	for rows.Next() {
		c, err := scanOAuthConnection(rows.Scan)
		if err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

// GetRequiredCredentialByActionType returns a single required credential for the
// connector that owns the given action type.
//
// Deprecated: Use GetRequiredCredentialsByActionType (plural) instead, which
// returns all credential types and lets the caller handle fallback logic.
// This function is retained for backward compatibility.
func GetRequiredCredentialByActionType(ctx context.Context, db DBTX, actionType string) (*RequiredCredential, error) {
	var rc RequiredCredential
	var fieldsRaw []byte
	err := db.QueryRow(ctx, `
		SELECT crc.service, crc.auth_type, crc.instructions_url, crc.oauth_provider, crc.oauth_scopes, COALESCE(crc.credential_fields, '[]'::jsonb)
		FROM connector_actions ca
		JOIN connector_required_credentials crc ON crc.connector_id = ca.connector_id
		WHERE ca.action_type = $1
		ORDER BY CASE WHEN crc.auth_type = 'oauth2' THEN 0 ELSE 1 END, crc.service
		LIMIT 1`,
		actionType,
	).Scan(&rc.Service, &rc.AuthType, &rc.InstructionsURL, &rc.OAuthProvider, &rc.OAuthScopes, &fieldsRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(fieldsRaw) > 0 && string(fieldsRaw) != "[]" && string(fieldsRaw) != "null" {
		if err := json.Unmarshal(fieldsRaw, &rc.CredentialFields); err != nil {
			return nil, fmt.Errorf("unmarshal credential_fields: %w", err)
		}
	}
	return &rc, nil
}

// GetRequiredCredentialsByActionType returns all required credentials for the
// connector that owns the given action type, ordered by preference (oauth2
// first, then static). This is the authoritative function for execution-time
// credential resolution — use it over GetRequiredCredentialByActionType for
// new code. The execution layer uses the ordering to try OAuth first and
// fall back to static credentials when the user hasn't connected via OAuth.
func GetRequiredCredentialsByActionType(ctx context.Context, db DBTX, actionType string) ([]RequiredCredential, error) {
	rows, err := db.Query(ctx, `
		SELECT crc.service, crc.auth_type, crc.instructions_url, crc.oauth_provider, crc.oauth_scopes, COALESCE(crc.credential_fields, '[]'::jsonb)
		FROM connector_actions ca
		JOIN connector_required_credentials crc ON crc.connector_id = ca.connector_id
		WHERE ca.action_type = $1
		ORDER BY CASE WHEN crc.auth_type = 'oauth2' THEN 0 ELSE 1 END, crc.service`,
		actionType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []RequiredCredential
	for rows.Next() {
		var rc RequiredCredential
		var fieldsRaw []byte
		if err := rows.Scan(&rc.Service, &rc.AuthType, &rc.InstructionsURL, &rc.OAuthProvider, &rc.OAuthScopes, &fieldsRaw); err != nil {
			return nil, err
		}
		if len(fieldsRaw) > 0 && string(fieldsRaw) != "[]" && string(fieldsRaw) != "null" {
			if err := json.Unmarshal(fieldsRaw, &rc.CredentialFields); err != nil {
				return nil, fmt.Errorf("unmarshal credential_fields: %w", err)
			}
		}
		creds = append(creds, rc)
	}
	return creds, rows.Err()
}
