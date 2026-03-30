package cloudflare

import (
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
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

// requirePathParam checks that a parameter is non-empty and safe for use in
// URL paths. Use this for required parameters that appear in API path segments.
func requirePathParam(name, value string) error {
	if value == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", name)}
	}
	return validatePathParam(name, value)
}

// requireParam checks that a parameter is non-empty.
// Use this for required parameters that do NOT appear in URL paths.
func requireParam(name, value string) error {
	if value == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", name)}
	}
	return nil
}
