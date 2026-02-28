package vault

import (
	"context"
	"testing"
)

func TestMockVaultStore_CreateAndRead(t *testing.T) {
	t.Parallel()
	m := NewMockVaultStore()
	ctx := context.Background()

	secret := []byte(`{"api_key":"sk_test_123"}`)
	id, err := m.CreateSecret(ctx, nil, "cred_test", secret)
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	got, err := m.ReadSecret(ctx, nil, id)
	if err != nil {
		t.Fatalf("ReadSecret: %v", err)
	}
	if string(got) != string(secret) {
		t.Errorf("ReadSecret = %q, want %q", got, secret)
	}
}

func TestMockVaultStore_ReadMissing(t *testing.T) {
	t.Parallel()
	m := NewMockVaultStore()
	ctx := context.Background()

	_, err := m.ReadSecret(ctx, nil, "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
}

func TestMockVaultStore_Delete(t *testing.T) {
	t.Parallel()
	m := NewMockVaultStore()
	ctx := context.Background()

	id, err := m.CreateSecret(ctx, nil, "cred_del", []byte("secret"))
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}

	if !m.HasSecret(id) {
		t.Fatal("expected secret to exist after create")
	}

	if err := m.DeleteSecret(ctx, nil, id); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}

	if m.HasSecret(id) {
		t.Fatal("expected secret to be gone after delete")
	}

	// Deleting again should be idempotent.
	if err := m.DeleteSecret(ctx, nil, id); err != nil {
		t.Fatalf("DeleteSecret (idempotent): %v", err)
	}
}

func TestMockVaultStore_SecretCount(t *testing.T) {
	t.Parallel()
	m := NewMockVaultStore()
	ctx := context.Background()

	if m.SecretCount() != 0 {
		t.Fatalf("expected 0 secrets, got %d", m.SecretCount())
	}

	if _, err := m.CreateSecret(ctx, nil, "a", []byte("1")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.CreateSecret(ctx, nil, "b", []byte("2")); err != nil {
		t.Fatal(err)
	}

	if m.SecretCount() != 2 {
		t.Fatalf("expected 2 secrets, got %d", m.SecretCount())
	}
}

func TestMockVaultStore_IsolatesData(t *testing.T) {
	t.Parallel()
	m := NewMockVaultStore()
	ctx := context.Background()

	original := []byte("original-secret")
	id, err := m.CreateSecret(ctx, nil, "iso", original)
	if err != nil {
		t.Fatal(err)
	}

	// Mutate the original slice — stored data should be unaffected.
	original[0] = 'X'

	got, err := m.ReadSecret(ctx, nil, id)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] == 'X' {
		t.Error("stored data was aliased with input slice")
	}

	// Mutate the returned slice — stored data should be unaffected.
	got[0] = 'Y'
	got2, _ := m.ReadSecret(ctx, nil, id)
	if got2[0] == 'Y' {
		t.Error("stored data was aliased with returned slice")
	}
}
