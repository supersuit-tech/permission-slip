package providers

import "github.com/supersuit-tech/permission-slip/connectors"

func init() {
	if connectors.IsBuiltInConnectorDisabled("quickbooks") {
		return
	}
	connectors.RegisterBuiltInOAuthProvider("quickbooks")
}
