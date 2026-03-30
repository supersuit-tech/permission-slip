package quickbooks

import "github.com/supersuit-tech/permission-slip/connectors"

func init() {
	if connectors.IsBuiltInConnectorDisabled("quickbooks") {
		return
	}
	connectors.RegisterBuiltIn(New())
}
