package kroger

import "github.com/supersuit-tech/permission-slip-web/connectors"

func init() {
	if connectors.IsBuiltInConnectorDisabled("kroger") {
		return
	}
	connectors.RegisterBuiltIn(New())
}
