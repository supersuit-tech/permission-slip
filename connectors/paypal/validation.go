package paypal

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validatePayPalPathID ensures id is safe to embed in a URL path segment.
func validatePayPalPathID(field, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", field)}
	}
	if len(id) > 128 {
		return &connectors.ValidationError{Message: fmt.Sprintf("%s is too long (max 128 characters)", field)}
	}
	if strings.ContainsAny(id, "/?#%\\") {
		return &connectors.ValidationError{Message: fmt.Sprintf("%s contains invalid characters", field)}
	}
	for _, r := range id {
		if unicode.IsSpace(r) {
			return &connectors.ValidationError{Message: fmt.Sprintf("%s must not contain whitespace", field)}
		}
	}
	return nil
}
