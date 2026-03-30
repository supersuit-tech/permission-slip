package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestCredentialsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "credentials", []string{
		"id", "user_id", "service", "label", "vault_secret_id", "created_at",
	})
}

func TestCredentialsIndex(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireIndex(t, tx, "credentials", "idx_credentials_user_service")
}

func TestCredentialsCascadeDeleteOnProfileDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid, "gmail")

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM profiles WHERE id = '"+uid+"'",
		[]string{"credentials"},
		"user_id = '"+uid+"'",
	)
}

func TestCredentialsUniqueConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	credA := testhelper.GenerateID(t, "cred_")
	credB := testhelper.GenerateID(t, "cred_")
	credC := testhelper.GenerateID(t, "cred_")
	credD := testhelper.GenerateID(t, "cred_")

	// Duplicate (user_id, service, label) should fail
	testhelper.RequireUniqueViolation(t, tx, "(user_id, service, label)",
		func() error {
			testhelper.InsertCredentialWithLabel(t, tx, credA, uid, "gmail", "personal")
			return nil
		},
		func() error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO credentials (id, user_id, service, label, vault_secret_id)
				 VALUES ($1, $2, 'gmail', 'personal', '00000000-0000-0000-0000-000000000099')`,
				credB, uid)
			return err
		})

	// Different label should succeed
	testhelper.InsertCredentialWithLabel(t, tx, credC, uid, "gmail", "work")

	// Different service should succeed
	testhelper.InsertCredentialWithLabel(t, tx, credD, uid, "stripe", "personal")
}

func TestCredentialsUniqueConstraintNullLabel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	credA := testhelper.GenerateID(t, "cred_")
	credB := testhelper.GenerateID(t, "cred_")

	testhelper.RequireUniqueViolation(t, tx, "(user_id, service, NULL label)",
		func() error {
			testhelper.InsertCredential(t, tx, credA, uid, "gmail")
			return nil
		},
		func() error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO credentials (id, user_id, service, vault_secret_id)
				 VALUES ($1, $2, 'gmail', '00000000-0000-0000-0000-000000000099')`,
				credB, uid)
			return err
		})
}

func TestDeleteCredential_ReturnsVaultSecretID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, "github")

	result, err := db.DeleteCredential(context.Background(), tx, credID, uid)
	if err != nil {
		t.Fatalf("DeleteCredential: %v", err)
	}
	if result.VaultSecretID != "00000000-0000-0000-0000-000000000099" {
		t.Errorf("expected vault_secret_id '00000000-0000-0000-0000-000000000099', got %q", result.VaultSecretID)
	}
	if result.DeletedAt.IsZero() {
		t.Error("expected non-zero deleted_at")
	}
}

func TestDeleteCredential_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	_, err := db.DeleteCredential(context.Background(), tx, "cred_nonexistent", uid)
	if err == nil {
		t.Fatal("expected error for non-existent credential")
	}
	credErr, ok := err.(*db.CredentialError)
	if !ok {
		t.Fatalf("expected *CredentialError, got %T", err)
	}
	if credErr.Code != db.CredentialErrNotFound {
		t.Errorf("expected CredentialErrNotFound, got %v", credErr.Code)
	}
}

func TestGetVaultSecretID_WithLabel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithLabel(t, tx, credID, uid, "github", "work")

	label := "work"
	vaultID, err := db.GetVaultSecretID(context.Background(), tx, uid, "github", &label)
	if err != nil {
		t.Fatalf("GetVaultSecretID: %v", err)
	}
	if vaultID != "00000000-0000-0000-0000-000000000099" {
		t.Errorf("expected '00000000-0000-0000-0000-000000000099', got %q", vaultID)
	}
}

func TestGetVaultSecretID_WithoutLabel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, "slack")

	vaultID, err := db.GetVaultSecretID(context.Background(), tx, uid, "slack", nil)
	if err != nil {
		t.Fatalf("GetVaultSecretID: %v", err)
	}
	if vaultID != "00000000-0000-0000-0000-000000000099" {
		t.Errorf("expected '00000000-0000-0000-0000-000000000099', got %q", vaultID)
	}
}

func TestGetVaultSecretID_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	_, err := db.GetVaultSecretID(context.Background(), tx, uid, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for non-existent credential")
	}
	credErr, ok := err.(*db.CredentialError)
	if !ok {
		t.Fatalf("expected *CredentialError, got %T", err)
	}
	if credErr.Code != db.CredentialErrNotFound {
		t.Errorf("expected CredentialErrNotFound, got %v", credErr.Code)
	}
}

func TestGetDecryptedCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, "github")

	// Mock vault reader that returns JSON for the placeholder UUID.
	mockReader := func(_ context.Context, _ db.DBTX, secretID string) ([]byte, error) {
		if secretID == "00000000-0000-0000-0000-000000000099" {
			return []byte(`{"api_key":"ghp_secret_123","org":"myorg"}`), nil
		}
		return nil, &db.CredentialError{Code: db.CredentialErrNotFound, Message: "not found"}
	}

	creds, err := db.GetDecryptedCredentials(context.Background(), tx, mockReader, uid, "github", nil)
	if err != nil {
		t.Fatalf("GetDecryptedCredentials: %v", err)
	}
	if creds["api_key"] != "ghp_secret_123" {
		t.Errorf("expected api_key 'ghp_secret_123', got %v", creds["api_key"])
	}
	if creds["org"] != "myorg" {
		t.Errorf("expected org 'myorg', got %v", creds["org"])
	}
}
