// Package oauth implements the OAuth 2.0 provider registry for Permission Slip.
// It manages provider configurations (endpoints, scopes, client credentials)
// and provides a thread-safe registry that merges built-in providers and
// connector-manifest-declared providers.
//
// Provider sources have a defined priority: Manifest > BuiltIn.
// When multiple sources register a provider with the same ID, the higher-priority
// source wins.
//
// Security: Provider and TokenSet implement fmt.Stringer, fmt.GoStringer, and
// json.Marshaler to redact secrets (ClientSecret, AccessToken, RefreshToken).
// Code that needs actual secret values must access struct fields directly.
//
// Thread safety: Registry methods are safe for concurrent use. Returned Provider
// values are deep copies — mutations do not affect the registry.
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
// pre-filled; connector-declared providers are populated from manifests.
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

	// ClientID is the OAuth application's client ID, sourced from server
	// config (environment variables).
	ClientID string

	// ClientSecret is the OAuth application's client secret, sourced from
	// server config (environment variables).
	ClientSecret string

	// AuthorizeParams are extra query parameters appended to the authorization
	// URL. Some providers require additional params beyond the standard OAuth 2.0
	// set (e.g. Atlassian needs audience=api.atlassian.com for 3LO, Slack needs
	// comma-separated scopes via a "scope" override).
	AuthorizeParams map[string]string

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

// TokenExpiryBuffer is the time before actual expiry at which a token is
// considered expired. This allows pre-emptive refresh so actions don't fail
// mid-execution due to an expired token. Used by both the background refresh
// job and synchronous execution-time refresh.
const TokenExpiryBuffer = 5 * time.Minute

// IsExpired reports whether the access token has expired or is within the
// pre-emptive refresh buffer (5 minutes before actual expiry).
func (ts TokenSet) IsExpired() bool {
	if ts.Expiry.IsZero() {
		return false
	}
	return time.Now().After(ts.Expiry.Add(-TokenExpiryBuffer))
}

// HasClientCredentials reports whether the provider has both a client ID and
// client secret configured. Providers without credentials can be registered
// (from manifests) but cannot initiate OAuth flows.
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
// startup from built-in configs and connector manifests. The registry is
// read-heavy and write-rare (writes happen at startup).
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// ProviderIDPattern matches valid provider IDs: lowercase alphanumeric, hyphens,
// and underscores. Mirrors the connector manifest ID pattern for consistency.
// Exported so API handlers can validate path parameters before use.
var ProviderIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)

// Register adds or replaces a provider in the registry. If a provider with the
// same ID already exists, it is replaced only if the new source has equal or
// higher priority: Manifest > BuiltIn.
//
// Returns an error if the provider ID is empty or invalid.
func (r *Registry) Register(p Provider) error {
	if p.ID == "" {
		return fmt.Errorf("oauth provider ID is required")
	}
	if !ProviderIDPattern.MatchString(p.ID) {
		return fmt.Errorf("oauth provider ID %q must match %s", p.ID, ProviderIDPattern.String())
	}
	// Deep-copy slices and maps so the caller cannot mutate the registry's
	// stored data after registration.
	p.Scopes = copyStrings(p.Scopes)
	p.AuthorizeParams = copyStringMap(p.AuthorizeParams)

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.providers[p.ID]
	if exists && sourcePriority(existing.Source) > sourcePriority(p.Source) {
		return nil
	}

	r.providers[p.ID] = p
	return nil
}

// Get returns a copy of the provider with the given ID. The second return
// value is false if the provider is not registered. The returned Provider
// has its own copy of slice fields — mutations do not affect the registry.
func (r *Registry) Get(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	if ok {
		p.Scopes = copyStrings(p.Scopes)
		p.AuthorizeParams = copyStringMap(p.AuthorizeParams)
	}
	return p, ok
}

// List returns all registered providers sorted by ID. The returned slice is a
// deep snapshot — mutations to the returned providers do not affect the
// registry. Sorting ensures deterministic output in logs and API responses.
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		p.Scopes = copyStrings(p.Scopes)
		p.AuthorizeParams = copyStringMap(p.AuthorizeParams)
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

// copyStrings returns a deep copy of a string slice. Returns nil if the input
// is nil, preserving the distinction between nil and empty slices.
func copyStrings(s []string) []string {
	if s == nil {
		return nil
	}
	cp := make([]string, len(s))
	copy(cp, s)
	return cp
}

// copyStringMap returns a deep copy of a string map. Returns nil if the input
// is nil, preserving the distinction between nil and empty maps.
func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

// sourcePriority returns a numeric priority for a ProviderSource. Higher values
// take precedence when merging providers with the same ID.
func sourcePriority(s ProviderSource) int {
	switch s {
	case SourceBuiltIn:
		return 1
	case SourceManifest:
		return 2
	default:
		return 0
	}
}
