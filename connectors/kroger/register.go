package kroger

import "github.com/supersuit-tech/permission-slip-web/connectors"

func init() {
	connectors.RegisterBuiltIn(New())
}
