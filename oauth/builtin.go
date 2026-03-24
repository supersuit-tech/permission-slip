package oauth

import (
	"fmt"
	"strings"
	"sync"
)

var (
	builtInMu        sync.Mutex
	builtInFactories []func() Provider
	builtInIDs       = make(map[string]bool)
)

// RegisterBuiltIn registers a provider factory as a built-in platform provider.
// It is called from init() functions in the oauth/providers package. Passing a
// factory (rather than a Provider value) ensures that environment variable
// lookups for client credentials happen at BuiltInProviders()/
// NewRegistryWithBuiltIns() call time — after the application has had a chance
// to load .env files — rather than at package init time.
//
// The factory is called once at registration time to validate required fields
// (ID, AuthorizeURL, TokenURL). If any are missing, if URLs do not use HTTPS,
// or if the ID is a duplicate, RegisterBuiltIn panics immediately.
func RegisterBuiltIn(factory func() Provider) {
	p := factory() // call once to validate required fields
	if p.ID == "" {
		panic("oauth.RegisterBuiltIn: provider ID is required")
	}
	if p.AuthorizeURL == "" {
		panic(fmt.Sprintf("oauth.RegisterBuiltIn: provider %q is missing AuthorizeURL", p.ID))
	}
	if !strings.HasPrefix(p.AuthorizeURL, "https://") {
		panic(fmt.Sprintf("oauth.RegisterBuiltIn: provider %q AuthorizeURL must use https, got %q", p.ID, p.AuthorizeURL))
	}
	if p.TokenURL == "" {
		panic(fmt.Sprintf("oauth.RegisterBuiltIn: provider %q is missing TokenURL", p.ID))
	}
	if !strings.HasPrefix(p.TokenURL, "https://") {
		panic(fmt.Sprintf("oauth.RegisterBuiltIn: provider %q TokenURL must use https, got %q", p.ID, p.TokenURL))
	}
	builtInMu.Lock()
	defer builtInMu.Unlock()
	if builtInIDs[p.ID] {
		panic(fmt.Sprintf("oauth.RegisterBuiltIn: duplicate built-in provider %q", p.ID))
	}
	builtInIDs[p.ID] = true
	builtInFactories = append(builtInFactories, factory)
}

// BuiltInProviders returns the platform's pre-configured OAuth providers.
// Providers are registered via init() in the oauth/providers package, which
// must be blank-imported by the binary entrypoint or test setup.
//
// Each call re-invokes the provider factories so that client credentials are
// read from environment variables at call time. This ensures .env files loaded
// after package init (e.g. via godotenv in main) are reflected correctly.
func BuiltInProviders() []Provider {
	// Snapshot the factory slice while holding the lock, then invoke the
	// factories without the lock. This avoids holding a mutex while calling
	// external code, which could deadlock if a factory ever calls back into
	// this package.
	builtInMu.Lock()
	factories := make([]func() Provider, len(builtInFactories))
	copy(factories, builtInFactories)
	builtInMu.Unlock()

	out := make([]Provider, len(factories))
	for i, f := range factories {
		out[i] = f()
	}
	return out
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
// credentials — operators must supply those via environment variables.
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
