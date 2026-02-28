package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Credential represents the metadata of a stored credential (secrets are never returned).
type Credential struct {
	ID        string
	UserID    string
	Service   string
	Label     *string
	CreatedAt time.Time
}

// CreateCredentialParams holds the parameters for inserting a new credential row.
type CreateCredentialParams struct {
	ID            string
	UserID        string
	Service       string
	Label         *string
	VaultSecretID string // UUID of the secret stored in the vault
}

// CredentialError represents a domain-specific error from credential operations.
type CredentialError struct {
	Code    CredentialErrCode
	Message string
}

func (e *CredentialError) Error() string { return e.Message }

// CredentialErrCode enumerates credential-specific error codes.
type CredentialErrCode int

const (
	CredentialErrNotFound  CredentialErrCode = iota
	CredentialErrDuplicate                   // unique (user_id, service, label) violation
)

// ListCredentialsByUser returns all credential metadata for the given user,
// ordered by created_at descending (newest first).
func ListCredentialsByUser(ctx context.Context, db DBTX, userID string) ([]Credential, error) {
	rows, err := db.Query(ctx, `
		SELECT id, user_id, service, label, created_at
		FROM credentials
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []Credential
	for rows.Next() {
		var c Credential
		if err := rows.Scan(&c.ID, &c.UserID, &c.Service, &c.Label, &c.CreatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

// CreateCredential inserts a new credential row.
// Returns a *CredentialError with code CredentialErrDuplicate if the (user_id, service, label)
// unique constraint is violated.
func CreateCredential(ctx context.Context, db DBTX, p CreateCredentialParams) (*Credential, error) {
	var c Credential
	err := db.QueryRow(ctx, `
		INSERT INTO credentials (id, user_id, service, label, vault_secret_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, service, label, created_at`,
		p.ID, p.UserID, p.Service, p.Label, p.VaultSecretID,
	).Scan(&c.ID, &c.UserID, &c.Service, &c.Label, &c.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgCodeUniqueViolation {
			return nil, &CredentialError{Code: CredentialErrDuplicate, Message: "Credentials already stored for this service with this label"}
		}
		return nil, err
	}
	return &c, nil
}

// DeleteCredentialResult holds the result of a credential deletion.
type DeleteCredentialResult struct {
	DeletedAt     time.Time
	VaultSecretID string // UUID of the vault secret that should also be deleted
}

// DeleteCredential deletes a credential by ID, scoped to the given user.
// Returns the vault_secret_id so the caller can delete the vault secret too.
// Returns a *CredentialError with code CredentialErrNotFound if no matching row exists.
func DeleteCredential(ctx context.Context, db DBTX, credID, userID string) (*DeleteCredentialResult, error) {
	var result DeleteCredentialResult
	err := db.QueryRow(ctx, `
		DELETE FROM credentials
		WHERE id = $1 AND user_id = $2
		RETURNING now(), vault_secret_id`, credID, userID,
	).Scan(&result.DeletedAt, &result.VaultSecretID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &CredentialError{Code: CredentialErrNotFound, Message: "Credential not found"}
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CredentialBelongsToUser returns true if a credential with the given ID
// exists and belongs to the specified user.
func CredentialBelongsToUser(ctx context.Context, db DBTX, credID, userID string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM credentials WHERE id = $1 AND user_id = $2)`,
		credID, userID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetVaultSecretID looks up the vault_secret_id for a credential by user and service.
// This is used for decrypting credentials at action execution time.
func GetVaultSecretID(ctx context.Context, db DBTX, userID, service string, label *string) (string, error) {
	var vaultSecretID string
	var err error
	if label != nil {
		err = db.QueryRow(ctx, `
			SELECT vault_secret_id FROM credentials
			WHERE user_id = $1 AND service = $2 AND label = $3`,
			userID, service, *label,
		).Scan(&vaultSecretID)
	} else {
		err = db.QueryRow(ctx, `
			SELECT vault_secret_id FROM credentials
			WHERE user_id = $1 AND service = $2 AND label IS NULL`,
			userID, service,
		).Scan(&vaultSecretID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "", &CredentialError{Code: CredentialErrNotFound, Message: "Credential not found"}
	}
	if err != nil {
		return "", err
	}
	return vaultSecretID, nil
}

// GetDecryptedCredentials retrieves and decrypts the credential secret for a
// given user and service. This is designed to be called by the connector engine
// at action execution time.
//
// The VaultStore is passed as an interface to keep the db package independent
// of the vault package. Callers provide a function that reads from the vault.
func GetDecryptedCredentials(
	ctx context.Context,
	d DBTX,
	readSecret func(ctx context.Context, tx DBTX, secretID string) ([]byte, error),
	userID, service string,
	label *string,
) (map[string]any, error) {
	vaultSecretID, err := GetVaultSecretID(ctx, d, userID, service, label)
	if err != nil {
		return nil, err
	}

	raw, err := readSecret(ctx, d, vaultSecretID)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}

	var creds map[string]any
	if err := json.Unmarshal(raw, &creds); err != nil {
		return nil, fmt.Errorf("unmarshal decrypted credentials: %w", err)
	}
	return creds, nil
}