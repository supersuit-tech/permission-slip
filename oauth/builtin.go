package oauth

import (
	"fmt"
	"sync"
)

var (
	builtInMu        sync.Mutex
	builtInProviders []Provider
	builtInIDs       = make(map[string]bool)
)

// RegisterBuiltIn registers a provider as a built-in platform provider.
// It is called from init() functions in the oauth/providers package.
// Panics immediately on duplicate IDs or missing required fields (ID,
// AuthorizeURL, TokenURL) to surface programming errors at startup rather
// than silently accumulating bad state.
func RegisterBuiltIn(p Provider) {
	if p.ID == "" {
		panic("oauth.RegisterBuiltIn: provider ID is required")
	}
	if p.AuthorizeURL == "" {
		panic(fmt.Sprintf("oauth.RegisterBuiltIn: provider %q is missing AuthorizeURL", p.ID))
	}
	if p.TokenURL == "" {
		panic(fmt.Sprintf("oauth.RegisterBuiltIn: provider %q is missing TokenURL", p.ID))
	}
	builtInMu.Lock()
	defer builtInMu.Unlock()
	if builtInIDs[p.ID] {
		panic(fmt.Sprintf("oauth.RegisterBuiltIn: duplicate built-in provider %q", p.ID))
	}
	builtInIDs[p.ID] = true
	builtInProviders = append(builtInProviders, p)
}

// BuiltInProviders returns the platform's pre-configured OAuth providers.
// Providers are registered via init() in the oauth/providers package, which
// must be blank-imported by the binary entrypoint or test setup.
// Client credentials are read from environment variables; if not set, the
// providers are still registered (so manifest validation passes) but cannot
// initiate OAuth flows until BYOA credentials are supplied.
func BuiltInProviders() []Provider {
	builtInMu.Lock()
	defer builtInMu.Unlock()
	cp := make([]Provider, len(builtInProviders))
	copy(cp, builtInProviders)
	return cp
}

// NewRegistryWithBuiltIns creates a new provider registry pre-populated with
// the platform's built-in providers. Panics if a built-in provider has an
// invalid ID (programming error).
func NewRegistryWithBuiltIns() *Registry {
	r := NewRegistry()
	for _, p := range BuiltInProviders() {
		if err := r.Register(p); err != nil {
			panic(fmt.Sprintf("failed to register built-in OAuth provider %q: %v", p.ID, err))
		}
	}
	return r
}

// RegisterFromManifest registers providers declared in a connector manifest's
// oauth_providers section. These are external providers that the platform
// doesn't have built-in support for. They are registered without client
// credentials — users must supply those via BYOA.
// Returns an error if any provider fails validation.
func RegisterFromManifest(r *Registry, providers []ManifestProvider) error {
	for _, mp := range providers {
		if err := r.Register(Provider{
			ID:              mp.ID,
			AuthorizeURL:    mp.AuthorizeURL,
			TokenURL:        mp.TokenURL,
			Scopes:          mp.Scopes,
			AuthorizeParams: mp.AuthorizeParams,
			Source:          SourceManifest,
		}); err != nil {
			return fmt.Errorf("registering manifest OAuth provider %q: %w", mp.ID, err)
		}
	}
	return nil
}

// ManifestProvider mirrors the OAuth provider declaration in a connector
// manifest. This avoids a circular import between oauth/ and connectors/.
type ManifestProvider struct {
	ID              string
	AuthorizeURL    string
	TokenURL        string
	Scopes          []string
	AuthorizeParams map[string]string
}
