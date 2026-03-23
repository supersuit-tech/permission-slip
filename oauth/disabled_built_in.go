package oauth

import "github.com/supersuit-tech/permission-slip-web/connectors"

// IsBuiltInOAuthProviderDisabled reports whether the built-in OAuth provider
// with this id is turned off alongside its connector (marker file at
// connectors/<id>/disabled, embedded in the binary with the connectors package).
func IsBuiltInOAuthProviderDisabled(id string) bool {
	return connectors.IsBuiltInConnectorDisabled(id)
}
