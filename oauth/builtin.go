package oauth

import (
	"fmt"
	"os"
)

// BuiltInProviders returns the platform's pre-configured OAuth providers.
// Client credentials are read from environment variables; if not set, the
// providers are still registered (so manifest validation passes) but cannot
// initiate OAuth flows until BYOA credentials are supplied.
func BuiltInProviders() []Provider {
	return []Provider{
		{
			ID:           "google",
			AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			Scopes: []string{
				"openid",
				"https://www.googleapis.com/auth/userinfo.email",
			},
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "microsoft",
			AuthorizeURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			Scopes: []string{
				"openid",
				"offline_access",
				"User.Read",
			},
			ClientID:     os.Getenv("MICROSOFT_CLIENT_ID"),
			ClientSecret: os.Getenv("MICROSOFT_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
	}
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
// doesn't have built-in support for (e.g. Salesforce, HubSpot). They are
// registered without client credentials — users must supply those via BYOA.
// Returns an error if any provider fails validation.
func RegisterFromManifest(r *Registry, providers []ManifestProvider) error {
	for _, mp := range providers {
		if err := r.Register(Provider{
			ID:           mp.ID,
			AuthorizeURL: mp.AuthorizeURL,
			TokenURL:     mp.TokenURL,
			Scopes:       mp.Scopes,
			Source:       SourceManifest,
		}); err != nil {
			return fmt.Errorf("registering manifest OAuth provider %q: %w", mp.ID, err)
		}
	}
	return nil
}

// ManifestProvider mirrors the OAuth provider declaration in a connector
// manifest. This avoids a circular import between oauth/ and connectors/.
type ManifestProvider struct {
	ID           string
	AuthorizeURL string
	TokenURL     string
	Scopes       []string
}
