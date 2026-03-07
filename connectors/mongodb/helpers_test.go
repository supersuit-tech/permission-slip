package mongodb

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with a valid connection URI for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"connection_uri": "mongodb://localhost:27017/testdb",
	})
}
