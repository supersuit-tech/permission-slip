package connectors

import "sync"

var (
	builtInOAuthMu        sync.Mutex
	builtInOAuthProviders = make(map[string]bool)
)

// RegisterBuiltInOAuthProvider marks an OAuth provider ID as a built-in
// platform provider. It is called from init() functions in the
// connectors/providers package, which must be blank-imported by the binary
// entrypoint or test setup.
//
// Panics if id is empty to catch registration mistakes at startup rather than
// silently allowing an invalid provider into the registry.
func RegisterBuiltInOAuthProvider(id string) {
	if id == "" {
		panic("connectors.RegisterBuiltInOAuthProvider: provider ID must not be empty")
	}
	builtInOAuthMu.Lock()
	defer builtInOAuthMu.Unlock()
	builtInOAuthProviders[id] = true
}

// BuiltInOAuthProviderIDs returns all registered built-in OAuth provider IDs.
// Order is not guaranteed. Intended for testing and diagnostics.
func BuiltInOAuthProviderIDs() []string {
	builtInOAuthMu.Lock()
	defer builtInOAuthMu.Unlock()
	ids := make([]string, 0, len(builtInOAuthProviders))
	for id := range builtInOAuthProviders {
		ids = append(ids, id)
	}
	return ids
}
