package vault

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"path/filepath"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

func testMasterKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		t.Fatal(err)
	}
	return k
}

func setupVaultTestDB(t *testing.T) (*db.Pool, func()) {
	t.Helper()
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "vault_test.db")
	if err := db.Migrate(ctx, path); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.Connect(ctx, path)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return pool, func() { _ = pool.Close() }
}

func TestSQLiteVault_RoundTrip(t *testing.T) {
	pool, cleanup := setupVaultTestDB(t)
	defer cleanup()

	v, err := NewSQLiteVault(testMasterKey(t))
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	plain := []byte(`{"token":"oauth-secret"}`)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	id, err := v.CreateSecret(ctx, tx, "cred_test", plain)
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	got, err := v.ReadSecret(ctx, pool, id)
	if err != nil {
		t.Fatalf("ReadSecret: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("plaintext mismatch")
	}

	tx2, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.DeleteSecret(ctx, tx2, id); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatal(err)
	}

	_, err = v.ReadSecret(ctx, pool, id)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestSQLiteVault_TamperedCiphertext(t *testing.T) {
	pool, cleanup := setupVaultTestDB(t)
	defer cleanup()

	v, err := NewSQLiteVault(testMasterKey(t))
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	id, err := v.CreateSecret(ctx, tx, "tamper", []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	var ct []byte
	row := pool.QueryRow(ctx, `SELECT ciphertext FROM vault_secrets WHERE id = $1`, id)
	if err := row.Scan(&ct); err != nil {
		t.Fatal(err)
	}
	if len(ct) == 0 {
		t.Fatal("empty ciphertext")
	}
	ct[len(ct)-1] ^= 0xFF
	if _, err := pool.Exec(ctx, `UPDATE vault_secrets SET ciphertext = $1 WHERE id = $2`, ct, id); err != nil {
		t.Fatal(err)
	}

	_, err = v.ReadSecret(ctx, pool, id)
	if err == nil {
		t.Fatal("expected decrypt failure on tampered ciphertext")
	}
}

func TestSQLiteVault_WrongKey(t *testing.T) {
	pool, cleanup := setupVaultTestDB(t)
	defer cleanup()

	k1 := testMasterKey(t)
	k2 := testMasterKey(t)
	v1, _ := NewSQLiteVault(k1)
	v2, _ := NewSQLiteVault(k2)
	ctx := t.Context()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	id, err := v1.CreateSecret(ctx, tx, "wrongkey", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	_, err = v2.ReadSecret(ctx, pool, id)
	if err == nil {
		t.Fatal("expected decrypt failure with wrong master key")
	}
}

func TestParseSecretEncryptionKey(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	enc := base64.StdEncoding.EncodeToString(key)
	got, err := ParseSecretEncryptionKey(enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, key) {
		t.Fatal("round-trip mismatch")
	}
	if _, err := ParseSecretEncryptionKey("!!!"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if _, err := ParseSecretEncryptionKey(base64.StdEncoding.EncodeToString(make([]byte, 31))); err == nil {
		t.Fatal("expected error for wrong decoded length")
	}
}

func TestNewSQLiteVault_KeyLength(t *testing.T) {
	if _, err := NewSQLiteVault([]byte{1, 2, 3}); err == nil {
		t.Fatal("expected error for short key")
	}
}
