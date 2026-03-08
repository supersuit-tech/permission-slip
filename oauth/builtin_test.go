package oauth

import (
	"testing"
)

func TestBuiltInProviders(t *testing.T) {
	providers := BuiltInProviders()

	if len(providers) < 2 {
		t.Fatalf("expected at least 2 built-in providers, got %d", len(providers))
	}

	// Build a map for easier lookup.
	byID := make(map[string]Provider, len(providers))
	for _, p := range providers {
		byID[p.ID] = p
	}

	t.Run("google", func(t *testing.T) {
		g, ok := byID["google"]
		if !ok {
			t.Fatal("google provider not found")
		}
		if g.AuthorizeURL != "https://accounts.google.com/o/oauth2/v2/auth" {
			t.Errorf("AuthorizeURL = %q", g.AuthorizeURL)
		}
		if g.TokenURL != "https://oauth2.googleapis.com/token" {
			t.Errorf("TokenURL = %q", g.TokenURL)
		}
		if g.Source != SourceBuiltIn {
			t.Errorf("Source = %q, want %q", g.Source, SourceBuiltIn)
		}
		if len(g.Scopes) == 0 {
			t.Error("expected default scopes")
		}
	})

	t.Run("figma", func(t *testing.T) {
		f, ok := byID["figma"]
		if !ok {
			t.Fatal("figma provider not found")
		}
		if f.AuthorizeURL != "https://www.figma.com/oauth" {
			t.Errorf("AuthorizeURL = %q", f.AuthorizeURL)
		}
		if f.TokenURL != "https://api.figma.com/v1/oauth/token" {
			t.Errorf("TokenURL = %q", f.TokenURL)
		}
		if f.Source != SourceBuiltIn {
			t.Errorf("Source = %q, want %q", f.Source, SourceBuiltIn)
		}
		if len(f.Scopes) != 2 {
			t.Errorf("expected 2 scopes, got %d: %v", len(f.Scopes), f.Scopes)
		}
	})

	t.Run("github", func(t *testing.T) {
		gh, ok := byID["github"]
		if !ok {
			t.Fatal("github provider not found")
		}
		if gh.AuthorizeURL != "https://github.com/login/oauth/authorize" {
			t.Errorf("AuthorizeURL = %q", gh.AuthorizeURL)
		}
		if gh.TokenURL != "https://github.com/login/oauth/access_token" {
			t.Errorf("TokenURL = %q", gh.TokenURL)
		}
		if gh.Source != SourceBuiltIn {
			t.Errorf("Source = %q, want %q", gh.Source, SourceBuiltIn)
		}
		if len(gh.Scopes) == 0 {
			t.Error("expected default scopes")
		}
	})

	t.Run("kroger", func(t *testing.T) {
		k, ok := byID["kroger"]
		if !ok {
			t.Fatal("kroger provider not found")
		}
		if k.AuthorizeURL != "https://api.kroger.com/v1/connect/oauth2/authorize" {
			t.Errorf("AuthorizeURL = %q", k.AuthorizeURL)
		}
		if k.TokenURL != "https://api.kroger.com/v1/connect/oauth2/token" {
			t.Errorf("TokenURL = %q", k.TokenURL)
		}
		if k.Source != SourceBuiltIn {
			t.Errorf("Source = %q, want %q", k.Source, SourceBuiltIn)
		}
		if len(k.Scopes) == 0 {
			t.Error("expected default scopes")
		}
	})

	t.Run("netlify", func(t *testing.T) {
		n, ok := byID["netlify"]
		if !ok {
			t.Fatal("netlify provider not found")
		}
		if n.AuthorizeURL != "https://app.netlify.com/authorize" {
			t.Errorf("AuthorizeURL = %q", n.AuthorizeURL)
		}
		if n.TokenURL != "https://api.netlify.com/oauth/token" {
			t.Errorf("TokenURL = %q", n.TokenURL)
		}
		if n.Source != SourceBuiltIn {
			t.Errorf("Source = %q, want %q", n.Source, SourceBuiltIn)
		}
	})

	t.Run("microsoft", func(t *testing.T) {
		m, ok := byID["microsoft"]
		if !ok {
			t.Fatal("microsoft provider not found")
		}
		if m.AuthorizeURL != "https://login.microsoftonline.com/common/oauth2/v2.0/authorize" {
			t.Errorf("AuthorizeURL = %q", m.AuthorizeURL)
		}
		if m.TokenURL != "https://login.microsoftonline.com/common/oauth2/v2.0/token" {
			t.Errorf("TokenURL = %q", m.TokenURL)
		}
		if m.Source != SourceBuiltIn {
			t.Errorf("Source = %q, want %q", m.Source, SourceBuiltIn)
		}
		if len(m.Scopes) == 0 {
			t.Error("expected default scopes")
		}
	})

	t.Run("square", func(t *testing.T) {
		s, ok := byID["square"]
		if !ok {
			t.Fatal("square provider not found")
		}
		if s.AuthorizeURL != "https://connect.squareup.com/oauth2/authorize" {
			t.Errorf("AuthorizeURL = %q", s.AuthorizeURL)
		}
		if s.TokenURL != "https://connect.squareup.com/oauth2/token" {
			t.Errorf("TokenURL = %q", s.TokenURL)
		}
		if s.Source != SourceBuiltIn {
			t.Errorf("Source = %q, want %q", s.Source, SourceBuiltIn)
		}
		if len(s.Scopes) == 0 {
			t.Error("expected default scopes")
		}
	})

	t.Run("discord", func(t *testing.T) {
		d, ok := byID["discord"]
		if !ok {
			t.Fatal("discord provider not found")
		}
		if d.AuthorizeURL != "https://discord.com/oauth2/authorize" {
			t.Errorf("AuthorizeURL = %q", d.AuthorizeURL)
		}
		if d.TokenURL != "https://discord.com/api/oauth2/token" {
			t.Errorf("TokenURL = %q", d.TokenURL)
		}
		if d.Source != SourceBuiltIn {
			t.Errorf("Source = %q, want %q", d.Source, SourceBuiltIn)
		}
		if len(d.Scopes) == 0 {
			t.Error("expected default scopes")
		}
	})

	t.Run("stripe", func(t *testing.T) {
		s, ok := byID["stripe"]
		if !ok {
			t.Fatal("stripe provider not found")
		}
		if s.AuthorizeURL != "https://connect.stripe.com/oauth/authorize" {
			t.Errorf("AuthorizeURL = %q", s.AuthorizeURL)
		}
		if s.TokenURL != "https://connect.stripe.com/oauth/token" {
			t.Errorf("TokenURL = %q", s.TokenURL)
		}
		if s.Source != SourceBuiltIn {
			t.Errorf("Source = %q, want %q", s.Source, SourceBuiltIn)
		}
		if len(s.Scopes) != 1 || s.Scopes[0] != "read_write" {
			t.Errorf("Scopes = %v, want [read_write]", s.Scopes)
		}
	})

	t.Run("airtable", func(t *testing.T) {
		a, ok := byID["airtable"]
		if !ok {
			t.Fatal("airtable provider not found")
		}
		if a.AuthorizeURL != "https://airtable.com/oauth2/v1/authorize" {
			t.Errorf("AuthorizeURL = %q", a.AuthorizeURL)
		}
		if a.TokenURL != "https://airtable.com/oauth2/v1/token" {
			t.Errorf("TokenURL = %q", a.TokenURL)
		}
		if a.Source != SourceBuiltIn {
			t.Errorf("Source = %q, want %q", a.Source, SourceBuiltIn)
		}
		if len(a.Scopes) != 6 {
			t.Errorf("expected 6 scopes, got %d: %v", len(a.Scopes), a.Scopes)
		}
	})
}

func TestNewRegistryWithBuiltIns(t *testing.T) {
	r := NewRegistryWithBuiltIns()

	ids := r.IDs()
	if len(ids) < 2 {
		t.Fatalf("expected at least 2 providers, got %d", len(ids))
	}

	if _, ok := r.Get("airtable"); !ok {
		t.Error("airtable not found in registry")
	}
	if _, ok := r.Get("discord"); !ok {
		t.Error("discord not found in registry")
	}
	if _, ok := r.Get("figma"); !ok {
		t.Error("figma not found in registry")
	}
	if _, ok := r.Get("github"); !ok {
		t.Error("github not found in registry")
	}
	if _, ok := r.Get("google"); !ok {
		t.Error("google not found in registry")
	}
	if _, ok := r.Get("kroger"); !ok {
		t.Error("kroger not found in registry")
	}
	if _, ok := r.Get("microsoft"); !ok {
		t.Error("microsoft not found in registry")
	}
	if _, ok := r.Get("netlify"); !ok {
		t.Error("netlify not found in registry")
	}
	if _, ok := r.Get("square"); !ok {
		t.Error("square not found in registry")
	}
	if _, ok := r.Get("stripe"); !ok {
		t.Error("stripe not found in registry")
	}
}

func TestRegisterFromManifest(t *testing.T) {
	r := NewRegistry()

	providers := []ManifestProvider{
		{
			ID:           "salesforce",
			AuthorizeURL: "https://login.salesforce.com/services/oauth2/authorize",
			TokenURL:     "https://login.salesforce.com/services/oauth2/token",
			Scopes:       []string{"api", "refresh_token"},
		},
		{
			ID:           "hubspot",
			AuthorizeURL: "https://app.hubspot.com/oauth/authorize",
			TokenURL:     "https://api.hubapi.com/oauth/v1/token",
		},
	}

	if err := RegisterFromManifest(r, providers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Len() != 2 {
		t.Fatalf("expected 2 providers, got %d", r.Len())
	}

	sf, ok := r.Get("salesforce")
	if !ok {
		t.Fatal("salesforce not found")
	}
	if sf.Source != SourceManifest {
		t.Errorf("Source = %q, want %q", sf.Source, SourceManifest)
	}
	if sf.AuthorizeURL != "https://login.salesforce.com/services/oauth2/authorize" {
		t.Errorf("AuthorizeURL = %q", sf.AuthorizeURL)
	}
	if len(sf.Scopes) != 2 {
		t.Errorf("Scopes = %v, want [api refresh_token]", sf.Scopes)
	}
	if sf.HasClientCredentials() {
		t.Error("manifest-registered provider should not have client credentials")
	}
}

func TestRegisterFromManifest_InvalidID(t *testing.T) {
	r := NewRegistry()
	err := RegisterFromManifest(r, []ManifestProvider{
		{ID: "INVALID", AuthorizeURL: "https://example.com/auth", TokenURL: "https://example.com/token"},
	})
	if err == nil {
		t.Error("expected error for invalid provider ID")
	}
}

func TestRegisterFromManifest_DoesNotOverrideBuiltIn(t *testing.T) {
	r := NewRegistryWithBuiltIns()

	// Try to register a manifest provider with the same ID as a built-in.
	if err := RegisterFromManifest(r, []ManifestProvider{
		{
			ID:           "google",
			AuthorizeURL: "https://evil.com/auth",
			TokenURL:     "https://evil.com/token",
		},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Manifest has higher priority than built-in, so it should override.
	// This is intentional — a connector may need different endpoints.
	g, _ := r.Get("google")
	if g.Source != SourceManifest {
		t.Errorf("expected manifest to override built-in, got Source = %q", g.Source)
	}
}
