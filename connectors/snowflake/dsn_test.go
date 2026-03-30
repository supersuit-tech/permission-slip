package snowflake

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/snowflakedb/gosnowflake"
	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestBuildSnowflakeConfig_passwordOnly(t *testing.T) {
	t.Parallel()
	dsn := "user:pass@org-account/db/schema?warehouse=wh"
	cfg, err := buildSnowflakeConfig(dsn, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.User != "user" || cfg.Password != "pass" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestBuildSnowflakeConfig_rejectsAuthenticatorWithPEM(t *testing.T) {
	t.Parallel()
	dsn := "u@a/db?authenticator=oauth"
	_, err := buildSnowflakeConfig(dsn, "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----")
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestBuildSnowflakeConfig_missingConnectionString(t *testing.T) {
	t.Parallel()
	_, err := buildSnowflakeConfig("", "")
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestBuildSnowflakeConfig_JWTUsesPrivateKeyField(t *testing.T) {
	t.Parallel()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	base := "svcuser@myorg-myacct/db/schema?warehouse=COMPUTE_WH"
	cfg, err := buildSnowflakeConfig(base, pemStr)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Authenticator != gosnowflake.AuthTypeJwt {
		t.Fatalf("authenticator = %v, want JWT", cfg.Authenticator)
	}
	if cfg.PrivateKey == nil {
		t.Fatal("PrivateKey not set")
	}
}
