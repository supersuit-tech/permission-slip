package providers

import "github.com/supersuit-tech/permission-slip-web/connectors"

func init() {
	connectors.RegisterBuiltInOAuthProvider("zoom")
}
