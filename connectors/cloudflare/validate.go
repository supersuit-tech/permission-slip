package cloudflare

import (
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validatePathParam rejects values containing characters that could alter URL
// path structure when interpolated into API paths.
func validatePathParam(name, value string) error {
	if strings.ContainsAny(value, "/?#%") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid %s: contains reserved URL characters", name),
		}
	}
	return nil
}
