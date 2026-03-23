package providers

import "github.com/supersuit-tech/permission-slip-web/connectors"

func init() {
	if connectors.IsBuiltInConnectorDisabled("salesforce") {
		return
	}
	connectors.RegisterBuiltInOAuthProvider("salesforce")
}
