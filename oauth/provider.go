// Package oauth implements the OAuth 2.0 provider registry for Permission Slip.
// It manages provider configurations (endpoints, scopes, client credentials)
// and provides a thread-safe registry that merges built-in providers,
// connector-manifest-declared providers, and user BYOA configurations.
package oauth

import (
	"fmt"
	"sync"
	"time"
)

// Provider holds the configuration for an OAuth 2.0 provider. Built-in
// providers (Google, Microsoft) ship with endpoints and default scopes
// pre-filled; connector-declared providers are populated from manifests;
// BYOA providers get client credentials from user configuration.
type Provider struct {
	// ID is a unique identifier for the provider (e.g. "google", "microsoft",
	// "salesforce"). Must be lowercase alphanumeric with hyphens/underscores.
	ID string

	// AuthorizeURL is the provider's OAuth 2.0 authorization endpoint.
	AuthorizeURL string

	// TokenURL is the provider's OAuth 2.0 token endpoint.
	TokenURL string

	// Scopes are the default OAuth scopes requested during authorization.
	// Connector manifests may request additional scopes beyond these defaults.
	Scopes []string

	// ClientID is the OAuth application's client ID. For built-in providers
	// this comes from server config (env vars). For BYOA it comes from user
	// configuration stored in the database.
	ClientID string

	// ClientSecret is the OAuth application's client secret. Same sourcing
	// rules as ClientID.
	ClientSecret string

	// Source indicates where the provider configuration originated.
	Source ProviderSource
}

// ProviderSource indicates the origin of a provider configuration.
type ProviderSource string

const (
	// SourceBuiltIn indicates a provider shipped with the platform.
	SourceBuiltIn ProviderSource = "built_in"

	// SourceManifest indicates a provider declared in a connector manifest.
	SourceManifest ProviderSource = "manifest"

	// SourceBYOA indicates a provider configured by the user (Bring Your Own App).
	SourceBYOA ProviderSource = "byoa"
)

// TokenSet holds the tokens returned by an OAuth 2.0 token exchange or refresh.
type TokenSet struct {
	// AccessToken is the token used to authenticate API requests.
	AccessToken string

	// RefreshToken is used to obtain a new access token when the current one
	// expires. May be empty if the provider doesn't issue refresh tokens.
	RefreshToken string

	// Expiry is the time at which the access token expires. A zero value
	// means the token does not expire (or expiry is unknown).
	Expiry time.Time

	// Scopes are the scopes granted by the provider. May differ from the
	// scopes requested if the user denied some during consent.
	Scopes []string
}

// IsExpired reports whether the access token has expired. It uses a 5-minute
// buffer to allow pre-emptive refresh before execution.
func (ts TokenSet) IsExpired() bool {
	if ts.Expiry.IsZero() {
		return false
	}
	return time.Now().After(ts.Expiry.Add(-5 * time.Minute))
}

// HasClientCredentials reports whether the provider has both a client ID and
// client secret configured. Providers without credentials can be registered
// (from manifests) but cannot initiate OAuth flows until BYOA credentials are
// supplied.
func (p Provider) HasClientCredentials() bool {
	return p.ClientID != "" && p.ClientSecret != ""
}

// Registry is a thread-safe registry of OAuth providers. It is populated at
// startup from built-in configs, connector manifests, and user BYOA settings.
// The registry is read-heavy and write-rare (writes happen at startup and when
// BYOA configs change).
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds or replaces a provider in the registry. If a provider with the
// same ID already exists, it is replaced only if the new source has equal or
// higher priority: BYOA > Manifest > BuiltIn.
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.providers[p.ID]
	if exists && sourcePriority(existing.Source) > sourcePriority(p.Source) {
		return
	}
	r.providers[p.ID] = p
}

// Get returns the provider with the given ID. The second return value is false
// if the provider is not registered.
func (r *Registry) Get(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

// List returns all registered providers. The returned slice is a snapshot;
// subsequent mutations to the registry are not reflected.
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}

// IDs returns the IDs of all registered providers.
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	return ids
}

// Remove deletes a provider from the registry. Returns an error if the
// provider does not exist.
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.providers[id]; !ok {
		return fmt.Errorf("oauth provider %q not found", id)
	}
	delete(r.providers, id)
	return nil
}

// sourcePriority returns a numeric priority for a ProviderSource. Higher values
// take precedence when merging providers with the same ID.
func sourcePriority(s ProviderSource) int {
	switch s {
	case SourceBuiltIn:
		return 1
	case SourceManifest:
		return 2
	case SourceBYOA:
		return 3
	default:
		return 0
	}
}
