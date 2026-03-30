package plaid

import "github.com/supersuit-tech/permission-slip/connectors"

func init() {
	connectors.RegisterBuiltIn(New())
}
