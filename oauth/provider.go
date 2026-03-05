// Package oauth implements the OAuth 2.0 provider registry for Permission Slip.
// It manages provider configurations (endpoints, scopes, client credentials)
// and provides a thread-safe registry that merges built-in providers,
// connector-manifest-declared providers, and user BYOA configurations.
package oauth

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
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

// String returns a human-readable representation with secrets redacted.
// This prevents accidental exposure of ClientSecret in logs or error messages.
func (p Provider) String() string {
	creds := "none"
	if p.HasClientCredentials() {
		creds = "configured"
	}
	return fmt.Sprintf("Provider{ID: %q, Source: %q, Credentials: %s}", p.ID, p.Source, creds)
}

// GoString redacts secrets in %#v formatting.
func (p Provider) GoString() string {
	return p.String()
}

// MarshalJSON serializes the provider with the ClientSecret redacted. This
// prevents accidental exposure if a Provider value is included in an API
// response, log entry, or error report. Code that needs the actual secret
// should access the ClientSecret field directly.
func (p Provider) MarshalJSON() ([]byte, error) {
	type safeProvider struct {
		ID           string         `json:"id"`
		AuthorizeURL string         `json:"authorize_url"`
		TokenURL     string         `json:"token_url"`
		Scopes       []string       `json:"scopes,omitempty"`
		ClientID     string         `json:"client_id,omitempty"`
		ClientSecret string         `json:"client_secret,omitempty"`
		Source       ProviderSource `json:"source"`
	}
	safe := safeProvider{
		ID:           p.ID,
		AuthorizeURL: p.AuthorizeURL,
		TokenURL:     p.TokenURL,
		Scopes:       p.Scopes,
		ClientID:     p.ClientID,
		Source:       p.Source,
	}
	if p.ClientSecret != "" {
		safe.ClientSecret = "[REDACTED]"
	}
	return json.Marshal(safe)
}

// String returns a redacted representation of the token set. This prevents
// accidental exposure of access/refresh tokens in logs or error messages.
func (ts TokenSet) String() string {
	return fmt.Sprintf("TokenSet{Scopes: %v, Expiry: %v, HasRefresh: %v}",
		ts.Scopes, ts.Expiry, ts.RefreshToken != "")
}

// GoString redacts tokens in %#v formatting.
func (ts TokenSet) GoString() string {
	return ts.String()
}

// MarshalJSON redacts token values to prevent accidental serialization of
// access/refresh tokens.
func (ts TokenSet) MarshalJSON() ([]byte, error) {
	return []byte(`"[REDACTED]"`), nil
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

// providerIDPattern matches valid provider IDs: lowercase alphanumeric, hyphens,
// and underscores. Mirrors the connector manifest ID pattern for consistency.
var providerIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)

// Register adds or replaces a provider in the registry. If a provider with the
// same ID already exists, it is replaced only if the new source has equal or
// higher priority: BYOA > Manifest > BuiltIn.
//
// When a BYOA provider overrides an existing provider, the client credentials
// from the BYOA registration are merged into the existing provider's endpoint
// and scope configuration. This means BYOA users only need to supply their
// client ID and secret — they don't need to re-specify authorize/token URLs
// or default scopes that were already set by the built-in or manifest source.
//
// Returns an error if the provider ID is empty or invalid.
func (r *Registry) Register(p Provider) error {
	if p.ID == "" {
		return fmt.Errorf("oauth provider ID is required")
	}
	if !providerIDPattern.MatchString(p.ID) {
		return fmt.Errorf("oauth provider ID %q must match %s", p.ID, providerIDPattern.String())
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.providers[p.ID]
	if exists && sourcePriority(existing.Source) > sourcePriority(p.Source) {
		return nil
	}

	// When BYOA overrides an existing provider, merge client credentials into
	// the existing config so users don't need to re-specify endpoints/scopes.
	if exists && p.Source == SourceBYOA {
		merged := existing
		merged.ClientID = p.ClientID
		merged.ClientSecret = p.ClientSecret
		merged.Source = SourceBYOA
		// Allow BYOA to override endpoints/scopes only if explicitly provided.
		if p.AuthorizeURL != "" {
			merged.AuthorizeURL = p.AuthorizeURL
		}
		if p.TokenURL != "" {
			merged.TokenURL = p.TokenURL
		}
		if len(p.Scopes) > 0 {
			merged.Scopes = p.Scopes
		}
		r.providers[p.ID] = merged
		return nil
	}

	r.providers[p.ID] = p
	return nil
}

// Get returns the provider with the given ID. The second return value is false
// if the provider is not registered.
func (r *Registry) Get(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

// List returns all registered providers sorted by ID. The returned slice is a
// snapshot; subsequent mutations to the registry are not reflected. Sorting
// ensures deterministic output in logs and API responses.
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// IDs returns the IDs of all registered providers in sorted order.
// Sorting ensures deterministic output in logs and validation.
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Len returns the number of registered providers without allocating a slice.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
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
