package providers

import "github.com/supersuit-tech/permission-slip/connectors"

func init() {
	if connectors.IsBuiltInConnectorDisabled("kroger") {
		return
	}
	connectors.RegisterBuiltInOAuthProvider("kroger")
}
