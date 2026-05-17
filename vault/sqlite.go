package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/supersuit-tech/permission-slip/db"
)

// SQLiteVault stores connector OAuth secrets and other credential payloads in
// the app SQLite database, encrypted with AES-256-GCM using a master key from
// SECRET_ENCRYPTION_KEY (32 random bytes, standard base64).
type SQLiteVault struct {
	masterKey [32]byte
}

// NewSQLiteVault returns a vault that encrypts with the given 32-byte AES-256 key.
func NewSQLiteVault(masterKey []byte) (*SQLiteVault, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("master key must be exactly 32 bytes for AES-256, got %d", len(masterKey))
	}
	var v SQLiteVault
	copy(v.masterKey[:], masterKey)
	return &v, nil
}

// NewSQLiteVaultFromEnv builds a SQLiteVault using SECRET_ENCRYPTION_KEY.
func NewSQLiteVaultFromEnv() (*SQLiteVault, error) {
	key, err := LoadSecretEncryptionKeyFromEnv()
	if err != nil {
		return nil, err
	}
	return NewSQLiteVault(key)
}

// LoadSecretEncryptionKeyFromEnv reads and validates SECRET_ENCRYPTION_KEY.
func LoadSecretEncryptionKeyFromEnv() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("SECRET_ENCRYPTION_KEY"))
	if raw == "" {
		return nil, fmt.Errorf("SECRET_ENCRYPTION_KEY is not set")
	}
	return ParseSecretEncryptionKey(raw)
}

// ParseSecretEncryptionKey decodes a base64-encoded 32-byte AES-256 key.
func ParseSecretEncryptionKey(base64Key string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(base64Key))
	if err != nil {
		return nil, fmt.Errorf("SECRET_ENCRYPTION_KEY is not valid base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("SECRET_ENCRYPTION_KEY must decode to exactly 32 bytes (use: openssl rand -base64 32), got %d", len(key))
	}
	return key, nil
}

func (v *SQLiteVault) gcm() (cipher.AEAD, error) {
	block, err := aes.NewCipher(v.masterKey[:])
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	return gcm, nil
}

// CreateSecret encrypts secret with AES-256-GCM and inserts into vault_secrets.
func (v *SQLiteVault) CreateSecret(ctx context.Context, tx db.DBTX, name string, secret []byte) (string, error) {
	gcm, err := v.gcm()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, secret, nil)

	id := uuid.New().String()
	_, err = tx.Exec(ctx,
		`INSERT INTO vault_secrets (id, name, nonce, ciphertext) VALUES ($1, $2, $3, $4)`,
		id, name, nonce, ciphertext,
	)
	if err != nil {
		return "", fmt.Errorf("vault insert: %w", err)
	}
	return id, nil
}

// ReadSecret loads ciphertext and nonce for id and decrypts with AES-256-GCM.
func (v *SQLiteVault) ReadSecret(ctx context.Context, tx db.DBTX, secretID string) ([]byte, error) {
	gcm, err := v.gcm()
	if err != nil {
		return nil, err
	}
	row := tx.QueryRow(ctx,
		`SELECT nonce, ciphertext FROM vault_secrets WHERE id = $1`,
		secretID,
	)
	var nonce, ciphertext []byte
	if err := row.Scan(&nonce, &ciphertext); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vault secret %s not found", secretID)
		}
		return nil, fmt.Errorf("vault select: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("vault secret %s: invalid nonce length %d", secretID, len(nonce))
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("vault decrypt: %w", err)
	}
	return plain, nil
}

// DeleteSecret removes a row from vault_secrets. Idempotent.
func (v *SQLiteVault) DeleteSecret(ctx context.Context, tx db.DBTX, secretID string) error {
	_, err := tx.Exec(ctx, `DELETE FROM vault_secrets WHERE id = $1`, secretID)
	if err != nil {
		return fmt.Errorf("vault delete: %w", err)
	}
	return nil
}
