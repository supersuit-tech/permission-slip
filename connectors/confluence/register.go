package confluence

import "github.com/supersuit-tech/permission-slip-web/connectors"

func init() {
	connectors.RegisterBuiltIn(New())
}
