package providers

import "github.com/supersuit-tech/permission-slip/connectors"

func init() {
	connectors.RegisterBuiltInOAuthProvider("discord")
}
