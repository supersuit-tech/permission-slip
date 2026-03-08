package connectors

import "sync"

var (
	builtInOAuthMu        sync.Mutex
	builtInOAuthProviders = make(map[string]bool)
)

// RegisterBuiltInOAuthProvider marks an OAuth provider ID as a built-in
// platform provider. It is called from init() functions in the
// oauth/providers package, which must be blank-imported by the binary
// entrypoint or test setup.
func RegisterBuiltInOAuthProvider(id string) {
	builtInOAuthMu.Lock()
	defer builtInOAuthMu.Unlock()
	builtInOAuthProviders[id] = true
}
