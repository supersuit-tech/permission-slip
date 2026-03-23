package providers

import "github.com/supersuit-tech/permission-slip-web/connectors"

func init() {
	if connectors.IsBuiltInConnectorDisabled("quickbooks") {
		return
	}
	connectors.RegisterBuiltInOAuthProvider("quickbooks")
}
