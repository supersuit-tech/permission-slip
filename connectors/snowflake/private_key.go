package snowflake

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

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
