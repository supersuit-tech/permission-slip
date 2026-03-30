package vault

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/supersuit-tech/permission-slip/db"
)

// MockVaultStore is an in-memory VaultStore for tests. Secrets are stored as
// plaintext in a map — no encryption is performed.
//
// This must never be used in production. The application startup validates
// that a real vault is configured in non-test environments.
type MockVaultStore struct {
	mu      sync.Mutex
	secrets map[string][]byte
}

// NewMockVaultStore returns a MockVaultStore ready for use in tests.
func NewMockVaultStore() *MockVaultStore {
	return &MockVaultStore{
		secrets: make(map[string][]byte),
	}
}

// CreateSecret stores the secret in memory and returns a random UUID.
// The tx parameter is accepted for interface compatibility but unused.
func (m *MockVaultStore) CreateSecret(_ context.Context, _ db.DBTX, _ string, secret []byte) (string, error) {
	id, err := randomUUID()
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Store a copy to avoid aliasing.
	stored := make([]byte, len(secret))
	copy(stored, secret)
	m.secrets[id] = stored

	return id, nil
}

// ReadSecret returns the plaintext secret for the given ID.
func (m *MockVaultStore) ReadSecret(_ context.Context, _ db.DBTX, secretID string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, ok := m.secrets[secretID]
	if !ok {
		return nil, fmt.Errorf("vault secret %s not found", secretID)
	}

	// Return a copy.
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

// DeleteSecret removes a secret from the in-memory store. Idempotent.
func (m *MockVaultStore) DeleteSecret(_ context.Context, _ db.DBTX, secretID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.secrets, secretID)
	return nil
}

// SecretCount returns the number of secrets currently stored (test helper).
func (m *MockVaultStore) SecretCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.secrets)
}

// HasSecret reports whether a secret with the given ID exists (test helper).
func (m *MockVaultStore) HasSecret(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.secrets[id]
	return ok
}

// SeedSecretForTest stores plaintext at an explicit secret ID. Use when OAuth
// fixtures reference a fixed access_token_vault_id (tests only).
func (m *MockVaultStore) SeedSecretForTest(secretID string, secret []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := make([]byte, len(secret))
	copy(stored, secret)
	m.secrets[secretID] = stored
}

// randomUUID generates a v4 UUID string.
func randomUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate UUID: %w", err)
	}
	// Set version 4 and variant bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
