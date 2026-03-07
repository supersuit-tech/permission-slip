package aws

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with valid AWS credentials for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_key_id":     "AKIAIOSFODNN7EXAMPLE",
		"secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	})
}
