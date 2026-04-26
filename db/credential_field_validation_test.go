package db_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

func TestValidateStaticCredentialKeys_apiKeyDefault(t *testing.T) {
	t.Parallel()
	rc := db.RequiredCredential{Service: "x", AuthType: "api_key"}
	err := db.ValidateStaticCredentialKeys(rc, map[string]any{"api_key": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ValidateStaticCredentialKeys(rc, map[string]any{"api_key": "x", "extra": "y"}); err == nil {
		t.Fatal("expected error for extra key")
	}
}

func TestValidateStaticCredentialKeys_customManifest(t *testing.T) {
	t.Parallel()
	rc := db.RequiredCredential{
		Service:  "aws",
		AuthType: "custom",
		CredentialFields: []db.CredentialFieldSpec{
			{Key: "access_key_id", Label: "Access Key ID", Secret: false, Required: true},
			{Key: "secret_access_key", Label: "Secret", Secret: true, Required: true},
			{Key: "region", Label: "Region", Secret: false, Required: true},
		},
	}
	err := db.ValidateStaticCredentialKeys(rc, map[string]any{
		"access_key_id":     "AKIAEXAMPLE",
		"secret_access_key": "s",
		"region":            "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ValidateStaticCredentialKeys(rc, map[string]any{
		"access_key_id": "AKIAEXAMPLE",
	}); err == nil {
		t.Fatal("expected error for missing keys")
	}
}

func TestValidateStaticCredentialKeys_basic(t *testing.T) {
	t.Parallel()
	rc := db.RequiredCredential{Service: "t", AuthType: "basic"}
	err := db.ValidateStaticCredentialKeys(rc, map[string]any{
		"username": "u",
		"password": "p",
	})
	if err != nil {
		t.Fatal(err)
	}
}
