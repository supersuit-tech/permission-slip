package snowflake

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// composeDSN merges vault credentials into a Snowflake DSN for gosnowflake.
// When private_key_pem is set, JWT auth (SNOWFLAKE_JWT) is appended unless the
// connection string already sets authenticator=.
func composeDSN(connectionString, privateKeyPEM string) (string, error) {
	dsn := strings.TrimSpace(connectionString)
	if dsn == "" {
		return "", &connectors.ValidationError{Message: "missing credential: connection_string"}
	}
	pkPEM := strings.TrimSpace(privateKeyPEM)
	if pkPEM == "" {
		return dsn, nil
	}

	lower := strings.ToLower(dsn)
	if strings.Contains(lower, "authenticator=") {
		return dsn, nil
	}

	pk, err := parseRSAPrivateKeyPEM(pkPEM)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalPKCS8PrivateKey(pk)
	if err != nil {
		return "", &connectors.ValidationError{Message: fmt.Sprintf("encoding private key: %v", err)}
	}
	enc := base64.URLEncoding.EncodeToString(der)
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "authenticator=SNOWFLAKE_JWT&privateKey=" + enc, nil
}

func parseRSAPrivateKeyPEM(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, &connectors.ValidationError{Message: "private_key_pem is not valid PEM"}
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key2, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("parse private key: %v", err)}
		}
		return key2, nil
	}
	rk, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, &connectors.ValidationError{Message: "private key must be RSA"}
	}
	return rk, nil
}
