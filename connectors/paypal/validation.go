package paypal

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/supersuit-tech/permission-slip/connectors"
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

// pathSegment returns url.PathEscape(id) after validation, for safe path construction.
func pathSegment(field, id string) (string, error) {
	if err := validatePayPalPathID(field, id); err != nil {
		return "", err
	}
	return url.PathEscape(strings.TrimSpace(id)), nil
}
