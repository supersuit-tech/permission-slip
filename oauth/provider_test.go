package oauth

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if ids := r.IDs(); len(ids) != 0 {
		t.Errorf("new registry should be empty, got %d providers", len(ids))
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	p := Provider{
		ID:           "test",
		AuthorizeURL: "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		Source:       SourceBuiltIn,
	}
	if err := r.Register(p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := r.Get("test")
	if !ok {
		t.Fatal("expected provider to be found")
	}
	if got.ID != "test" {
		t.Errorf("ID = %q, want %q", got.ID, "test")
	}
	if got.AuthorizeURL != "https://example.com/auth" {
		t.Errorf("AuthorizeURL = %q", got.AuthorizeURL)
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected provider not found")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	mustRegister(t, r, Provider{ID: "b", Source: SourceBuiltIn})
	mustRegister(t, r, Provider{ID: "a", Source: SourceBuiltIn})

	list := r.List()
	if len(list) != 2 {
		t.Fatalf("len(List) = %d, want 2", len(list))
	}

	// List should be sorted by ID.
	if list[0].ID != "a" || list[1].ID != "b" {
		t.Errorf("List not sorted: got [%s, %s], want [a, b]", list[0].ID, list[1].ID)
	}
}

func TestRegistry_IDs(t *testing.T) {
	r := NewRegistry()
	mustRegister(t, r, Provider{ID: "y", Source: SourceBuiltIn})
	mustRegister(t, r, Provider{ID: "x", Source: SourceBuiltIn})

	ids := r.IDs()
	if len(ids) != 2 || ids[0] != "x" || ids[1] != "y" {
		t.Errorf("IDs = %v, want [x y]", ids)
	}
}

func TestRegistry_Len(t *testing.T) {
	r := NewRegistry()
	if r.Len() != 0 {
		t.Errorf("Len() = %d, want 0", r.Len())
	}
	mustRegister(t, r, Provider{ID: "a", Source: SourceBuiltIn})
	mustRegister(t, r, Provider{ID: "b", Source: SourceBuiltIn})
	if r.Len() != 2 {
		t.Errorf("Len() = %d, want 2", r.Len())
	}
}

func TestRegistry_Remove(t *testing.T) {
	r := NewRegistry()
	mustRegister(t, r, Provider{ID: "removeme", Source: SourceBuiltIn})

	if err := r.Remove("removeme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := r.Get("removeme"); ok {
		t.Error("provider should have been removed")
	}
}

func TestRegistry_RemoveNotFound(t *testing.T) {
	r := NewRegistry()
	if err := r.Remove("nonexistent"); err == nil {
		t.Error("expected error for removing nonexistent provider")
	}
}

func TestRegistry_RegisterValidation(t *testing.T) {
	r := NewRegistry()

	t.Run("empty ID", func(t *testing.T) {
		if err := r.Register(Provider{ID: "", Source: SourceBuiltIn}); err == nil {
			t.Error("expected error for empty ID")
		}
	})

	t.Run("invalid ID with uppercase", func(t *testing.T) {
		if err := r.Register(Provider{ID: "Google", Source: SourceBuiltIn}); err == nil {
			t.Error("expected error for uppercase ID")
		}
	})

	t.Run("invalid ID with spaces", func(t *testing.T) {
		if err := r.Register(Provider{ID: "my provider", Source: SourceBuiltIn}); err == nil {
			t.Error("expected error for ID with spaces")
		}
	})

	t.Run("valid ID with hyphens and underscores", func(t *testing.T) {
		if err := r.Register(Provider{ID: "my-oauth_provider", Source: SourceBuiltIn}); err != nil {
			t.Errorf("unexpected error for valid ID: %v", err)
		}
	})
}

func TestRegistry_SourcePriority(t *testing.T) {
	t.Run("built-in does not override manifest", func(t *testing.T) {
		r := NewRegistry()
		mustRegister(t, r, Provider{ID: "p", AuthorizeURL: "https://manifest.com/auth", Source: SourceManifest})
		mustRegister(t, r, Provider{ID: "p", AuthorizeURL: "https://builtin.com/auth", Source: SourceBuiltIn})

		got, _ := r.Get("p")
		if got.AuthorizeURL != "https://manifest.com/auth" {
			t.Errorf("AuthorizeURL = %q, want manifest URL", got.AuthorizeURL)
		}
	})

	t.Run("manifest overrides built-in", func(t *testing.T) {
		r := NewRegistry()
		mustRegister(t, r, Provider{ID: "p", AuthorizeURL: "https://builtin.com/auth", Source: SourceBuiltIn})
		mustRegister(t, r, Provider{ID: "p", AuthorizeURL: "https://manifest.com/auth", Source: SourceManifest})

		got, _ := r.Get("p")
		if got.AuthorizeURL != "https://manifest.com/auth" {
			t.Errorf("AuthorizeURL = %q, want manifest URL", got.AuthorizeURL)
		}
	})

	t.Run("same source replaces", func(t *testing.T) {
		r := NewRegistry()
		mustRegister(t, r, Provider{ID: "p", ClientID: "first", Source: SourceBuiltIn})
		mustRegister(t, r, Provider{ID: "p", ClientID: "second", Source: SourceBuiltIn})

		got, _ := r.Get("p")
		if got.ClientID != "second" {
			t.Errorf("ClientID = %q, want %q (same source should replace)", got.ClientID, "second")
		}
	})
}

func TestTokenSet_IsExpired(t *testing.T) {
	t.Run("zero expiry never expired", func(t *testing.T) {
		ts := TokenSet{}
		if ts.IsExpired() {
			t.Error("zero expiry should not be expired")
		}
	})

	t.Run("future expiry not expired", func(t *testing.T) {
		ts := TokenSet{Expiry: time.Now().Add(1 * time.Hour)}
		if ts.IsExpired() {
			t.Error("future expiry should not be expired")
		}
	})

	t.Run("past expiry is expired", func(t *testing.T) {
		ts := TokenSet{Expiry: time.Now().Add(-1 * time.Hour)}
		if !ts.IsExpired() {
			t.Error("past expiry should be expired")
		}
	})

	t.Run("within 5-minute buffer is expired", func(t *testing.T) {
		ts := TokenSet{Expiry: time.Now().Add(3 * time.Minute)}
		if !ts.IsExpired() {
			t.Error("expiry within 5-minute buffer should be expired")
		}
	})

	t.Run("beyond 5-minute buffer is not expired", func(t *testing.T) {
		ts := TokenSet{Expiry: time.Now().Add(10 * time.Minute)}
		if ts.IsExpired() {
			t.Error("expiry beyond 5-minute buffer should not be expired")
		}
	})
}

func TestProvider_HasClientCredentials(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		secret   string
		expected bool
	}{
		{"both set", "id", "secret", true},
		{"no id", "", "secret", false},
		{"no secret", "id", "", false},
		{"neither set", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Provider{ClientID: tt.id, ClientSecret: tt.secret}
			if got := p.HasClientCredentials(); got != tt.expected {
				t.Errorf("HasClientCredentials() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRegistry_IDsSorted(t *testing.T) {
	r := NewRegistry()
	mustRegister(t, r, Provider{ID: "zebra", Source: SourceBuiltIn})
	mustRegister(t, r, Provider{ID: "alpha", Source: SourceBuiltIn})
	mustRegister(t, r, Provider{ID: "middle", Source: SourceBuiltIn})

	ids := r.IDs()
	if !sort.StringsAreSorted(ids) {
		t.Errorf("IDs() not sorted: %v", ids)
	}
}

func TestProvider_StringRedactsSecret(t *testing.T) {
	p := Provider{
		ID:           "google",
		ClientID:     "my-client-id",
		ClientSecret: "super-secret-value",
		Source:       SourceBuiltIn,
	}
	s := p.String()
	if strings.Contains(s, "super-secret-value") {
		t.Errorf("String() exposed ClientSecret: %s", s)
	}
	if !strings.Contains(s, "google") {
		t.Errorf("String() should contain ID: %s", s)
	}

	// Also test GoString (used by %#v)
	gs := fmt.Sprintf("%#v", p)
	if strings.Contains(gs, "super-secret-value") {
		t.Errorf("GoString() exposed ClientSecret: %s", gs)
	}
}

func TestProvider_MarshalJSON_RedactsSecret(t *testing.T) {
	p := Provider{
		ID:           "google",
		ClientID:     "my-client-id",
		ClientSecret: "super-secret-value",
		Source:       SourceBuiltIn,
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	s := string(data)
	if strings.Contains(s, "super-secret-value") {
		t.Errorf("MarshalJSON exposed ClientSecret: %s", s)
	}
	if !strings.Contains(s, "[REDACTED]") {
		t.Errorf("MarshalJSON should contain [REDACTED]: %s", s)
	}
	// ClientID should still be visible.
	if !strings.Contains(s, "my-client-id") {
		t.Errorf("MarshalJSON should contain ClientID: %s", s)
	}
}

func TestProvider_MarshalJSON_NoSecretOmitted(t *testing.T) {
	p := Provider{
		ID:     "google",
		Source: SourceBuiltIn,
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if strings.Contains(string(data), "REDACTED") {
		t.Errorf("MarshalJSON should not include REDACTED when no secret: %s", string(data))
	}
}

func TestProvider_MarshalJSON_IncludesPKCE(t *testing.T) {
	p := Provider{
		ID:     "dropbox",
		PKCE:   true,
		Source: SourceBuiltIn,
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(m["pkce"]) != "true" {
		t.Errorf("expected pkce true in JSON, got %s", string(m["pkce"]))
	}
}

func TestTokenSet_StringRedactsTokens(t *testing.T) {
	ts := TokenSet{
		AccessToken:  "secret-access-token",
		RefreshToken: "secret-refresh-token",
		Scopes:       []string{"openid"},
	}
	s := ts.String()
	if strings.Contains(s, "secret-access-token") {
		t.Errorf("String() exposed AccessToken: %s", s)
	}
	if strings.Contains(s, "secret-refresh-token") {
		t.Errorf("String() exposed RefreshToken: %s", s)
	}
}

func TestTokenSet_MarshalJSON_Redacted(t *testing.T) {
	ts := TokenSet{
		AccessToken:  "secret-access-token",
		RefreshToken: "secret-refresh-token",
	}
	data, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if strings.Contains(string(data), "secret-access-token") {
		t.Errorf("MarshalJSON exposed AccessToken: %s", string(data))
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	mustRegister(t, r, Provider{ID: "base", Source: SourceBuiltIn, Scopes: []string{"read"}})

	// Run concurrent reads and writes to verify thread safety.
	// With -race flag, the race detector will catch any data races.
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 100; j++ {
				_ = r.List()
				_, _ = r.Get("base")
				_ = r.IDs()
				_ = r.Len()
			}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRegistry_GetReturnsCopy(t *testing.T) {
	r := NewRegistry()
	mustRegister(t, r, Provider{
		ID:     "test",
		Source: SourceBuiltIn,
		Scopes: []string{"openid", "email"},
	})

	// Get a provider and mutate its scopes.
	got, _ := r.Get("test")
	got.Scopes[0] = "MUTATED"

	// The registry's copy should be unaffected.
	original, _ := r.Get("test")
	if original.Scopes[0] == "MUTATED" {
		t.Error("Get() returned a reference to the registry's data — mutations leaked back")
	}
}

func TestRegistry_ListReturnsCopies(t *testing.T) {
	r := NewRegistry()
	mustRegister(t, r, Provider{
		ID:     "test",
		Source: SourceBuiltIn,
		Scopes: []string{"openid"},
	})

	list := r.List()
	list[0].Scopes[0] = "MUTATED"

	// The registry's copy should be unaffected.
	original, _ := r.Get("test")
	if original.Scopes[0] == "MUTATED" {
		t.Error("List() returned references to the registry's data — mutations leaked back")
	}
}

func TestRegistry_RegisterCopiesScopes(t *testing.T) {
	r := NewRegistry()
	scopes := []string{"openid", "email"}
	mustRegister(t, r, Provider{
		ID:     "test",
		Source: SourceBuiltIn,
		Scopes: scopes,
	})

	// Mutate the original slice.
	scopes[0] = "MUTATED"

	// The registry's copy should be unaffected.
	got, _ := r.Get("test")
	if got.Scopes[0] == "MUTATED" {
		t.Error("Register() did not deep-copy Scopes — caller mutation affected registry")
	}
}

// mustRegister is a test helper that registers a provider and fails the test on error.
func mustRegister(t *testing.T, r *Registry, p Provider) {
	t.Helper()
	if err := r.Register(p); err != nil {
		t.Fatalf("Register(%q) failed: %v", p.ID, err)
	}
}
