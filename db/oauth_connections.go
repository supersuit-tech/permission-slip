package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OAuthConnection represents a row from the oauth_connections table.
type OAuthConnection struct {
	ID                   string
	UserID               string
	Provider             string
	AccessTokenVaultID   string
	RefreshTokenVaultID  *string
	Scopes               []string
	TokenExpiry          *time.Time
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// CreateOAuthConnectionParams holds the parameters for inserting a new OAuth connection.
type CreateOAuthConnectionParams struct {
	ID                   string
	UserID               string
	Provider             string
	AccessTokenVaultID   string
	RefreshTokenVaultID  *string
	Scopes               []string
	TokenExpiry          *time.Time
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
	OAuthConnectionErrNotFound  OAuthConnectionErrCode = iota
	OAuthConnectionErrDuplicate                        // unique (user_id, provider) violation
)

// ListOAuthConnectionsByUser returns all OAuth connections for the given user,
// ordered by created_at descending (newest first). Tokens are never returned.
func ListOAuthConnectionsByUser(ctx context.Context, db DBTX, userID string) ([]OAuthConnection, error) {
	rows, err := db.Query(ctx, `
		SELECT id, user_id, provider, access_token_vault_id, refresh_token_vault_id,
		       scopes, token_expiry, status, created_at, updated_at
		FROM oauth_connections
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []OAuthConnection
	for rows.Next() {
		var c OAuthConnection
		if err := rows.Scan(&c.ID, &c.UserID, &c.Provider, &c.AccessTokenVaultID,
			&c.RefreshTokenVaultID, &c.Scopes, &c.TokenExpiry, &c.Status,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

// GetOAuthConnectionByProvider returns the OAuth connection for a given user and provider.
// Returns nil if no connection exists.
func GetOAuthConnectionByProvider(ctx context.Context, db DBTX, userID, provider string) (*OAuthConnection, error) {
	var c OAuthConnection
	err := db.QueryRow(ctx, `
		SELECT id, user_id, provider, access_token_vault_id, refresh_token_vault_id,
		       scopes, token_expiry, status, created_at, updated_at
		FROM oauth_connections
		WHERE user_id = $1 AND provider = $2`,
		userID, provider,
	).Scan(&c.ID, &c.UserID, &c.Provider, &c.AccessTokenVaultID,
		&c.RefreshTokenVaultID, &c.Scopes, &c.TokenExpiry, &c.Status,
		&c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateOAuthConnection inserts a new OAuth connection row.
// Returns a *OAuthConnectionError with code OAuthConnectionErrDuplicate if
// the (user_id, provider) unique constraint is violated.
func CreateOAuthConnection(ctx context.Context, db DBTX, p CreateOAuthConnectionParams) (*OAuthConnection, error) {
	var c OAuthConnection
	err := db.QueryRow(ctx, `
		INSERT INTO oauth_connections (id, user_id, provider, access_token_vault_id, refresh_token_vault_id, scopes, token_expiry)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, provider, access_token_vault_id, refresh_token_vault_id,
		          scopes, token_expiry, status, created_at, updated_at`,
		p.ID, p.UserID, p.Provider, p.AccessTokenVaultID, p.RefreshTokenVaultID, p.Scopes, p.TokenExpiry,
	).Scan(&c.ID, &c.UserID, &c.Provider, &c.AccessTokenVaultID,
		&c.RefreshTokenVaultID, &c.Scopes, &c.TokenExpiry, &c.Status,
		&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgCodeUniqueViolation {
			return nil, &OAuthConnectionError{Code: OAuthConnectionErrDuplicate, Message: "OAuth connection already exists for this provider"}
		}
		return nil, err
	}
	return &c, nil
}

// UpdateOAuthConnectionTokens updates the vault secret IDs and expiry for a connection
// after a successful token refresh.
func UpdateOAuthConnectionTokens(ctx context.Context, db DBTX, id string, accessTokenVaultID string, refreshTokenVaultID *string, tokenExpiry *time.Time) error {
	result, err := db.Exec(ctx, `
		UPDATE oauth_connections
		SET access_token_vault_id = $2,
		    refresh_token_vault_id = $3,
		    token_expiry = $4,
		    status = 'active',
		    updated_at = now()
		WHERE id = $1`,
		id, accessTokenVaultID, refreshTokenVaultID, tokenExpiry)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return &OAuthConnectionError{Code: OAuthConnectionErrNotFound, Message: "OAuth connection not found"}
	}
	return nil
}

// UpdateOAuthConnectionStatus updates the status of an OAuth connection.
func UpdateOAuthConnectionStatus(ctx context.Context, db DBTX, id, status string) error {
	result, err := db.Exec(ctx, `
		UPDATE oauth_connections
		SET status = $2, updated_at = now()
		WHERE id = $1`,
		id, status)
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

// DeleteOAuthConnection deletes an OAuth connection by user and provider.
// Returns the vault secret IDs so the caller can delete the vault secrets too.
func DeleteOAuthConnection(ctx context.Context, db DBTX, userID, provider string) (*DeleteOAuthConnectionResult, error) {
	var result DeleteOAuthConnectionResult
	err := db.QueryRow(ctx, `
		DELETE FROM oauth_connections
		WHERE user_id = $1 AND provider = $2
		RETURNING access_token_vault_id, refresh_token_vault_id`,
		userID, provider,
	).Scan(&result.AccessTokenVaultID, &result.RefreshTokenVaultID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &OAuthConnectionError{Code: OAuthConnectionErrNotFound, Message: "OAuth connection not found"}
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRequiredCredentialByActionType returns the required credential for the connector
// that owns the given action type. Used to determine auth_type at execution time.
func GetRequiredCredentialByActionType(ctx context.Context, db DBTX, actionType string) (*RequiredCredential, error) {
	var rc RequiredCredential
	err := db.QueryRow(ctx, `
		SELECT crc.service, crc.auth_type, crc.instructions_url, crc.oauth_provider, crc.oauth_scopes
		FROM connector_actions ca
		JOIN connector_required_credentials crc ON crc.connector_id = ca.connector_id
		WHERE ca.action_type = $1
		LIMIT 1`,
		actionType,
	).Scan(&rc.Service, &rc.AuthType, &rc.InstructionsURL, &rc.OAuthProvider, &rc.OAuthScopes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rc, nil
}
