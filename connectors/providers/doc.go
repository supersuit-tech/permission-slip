// Package providers registers all built-in OAuth provider IDs with the
// connectors package. Import it with a blank identifier to activate
// registration:
//
//	import _ "github.com/supersuit-tech/permission-slip/connectors/providers"
//
// Each provider has its own file. To add a new built-in provider, create a
// new file following this template:
//
//	package providers
//
//	import "github.com/supersuit-tech/permission-slip/connectors"
//
//	func init() {
//	    connectors.RegisterBuiltInOAuthProvider("myprovider")
//	}
//
// The provider ID must match the corresponding ID registered in the
// oauth/providers package.
package providers
