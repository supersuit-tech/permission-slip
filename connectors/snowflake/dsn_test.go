package snowflake

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestComposeDSN_passwordOnly(t *testing.T) {
	t.Parallel()
	dsn := "user:pass@org-account/db/schema?warehouse=wh"
	out, err := composeDSN(dsn, "")
	if err != nil {
		t.Fatal(err)
	}
	if out != dsn {
		t.Fatalf("got %q want %q", out, dsn)
	}
}

func TestComposeDSN_respectsExistingAuthenticator(t *testing.T) {
	t.Parallel()
	dsn := "u@a/db?authenticator=oauth"
	out, err := composeDSN(dsn, "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----")
	if err != nil {
		t.Fatal(err)
	}
	if out != dsn {
		t.Fatalf("should not modify DSN when authenticator set: got %q", out)
	}
}

func TestComposeDSN_missingConnectionString(t *testing.T) {
	t.Parallel()
	_, err := composeDSN("", "")
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestParseRSAPrivateKeyPEM_invalid(t *testing.T) {
	t.Parallel()
	_, err := parseRSAPrivateKeyPEM("not pem")
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestComposeDSN_appendsJWTParams(t *testing.T) {
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
	out, err := composeDSN(base, pemStr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "authenticator=SNOWFLAKE_JWT") {
		t.Fatalf("missing JWT authenticator in %q", out)
	}
	if !strings.Contains(out, "privateKey=") {
		t.Fatalf("missing privateKey in %q", out)
	}
}
