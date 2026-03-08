package protonmail

import (
	"os"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func init() {
	// Proton Mail connector depends on a local Proton Mail Bridge daemon and
	// is not cloud-safe. Only register when explicitly enabled.
	v := os.Getenv("ENABLE_PROTONMAIL_CONNECTOR")
	if strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") {
		connectors.RegisterBuiltIn(New())
	}
}
