package oauth

import (
	"sort"
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
	r.Register(p)

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
	r.Register(Provider{ID: "a", Source: SourceBuiltIn})
	r.Register(Provider{ID: "b", Source: SourceBuiltIn})

	list := r.List()
	if len(list) != 2 {
		t.Fatalf("len(List) = %d, want 2", len(list))
	}

	ids := make([]string, len(list))
	for i, p := range list {
		ids[i] = p.ID
	}
	sort.Strings(ids)
	if ids[0] != "a" || ids[1] != "b" {
		t.Errorf("IDs = %v, want [a b]", ids)
	}
}

func TestRegistry_IDs(t *testing.T) {
	r := NewRegistry()
	r.Register(Provider{ID: "x", Source: SourceBuiltIn})
	r.Register(Provider{ID: "y", Source: SourceBuiltIn})

	ids := r.IDs()
	sort.Strings(ids)
	if len(ids) != 2 || ids[0] != "x" || ids[1] != "y" {
		t.Errorf("IDs = %v, want [x y]", ids)
	}
}

func TestRegistry_Remove(t *testing.T) {
	r := NewRegistry()
	r.Register(Provider{ID: "removeme", Source: SourceBuiltIn})

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

func TestRegistry_SourcePriority(t *testing.T) {
	t.Run("manifest does not override BYOA", func(t *testing.T) {
		r := NewRegistry()
		r.Register(Provider{ID: "p", ClientID: "byoa-id", Source: SourceBYOA})
		r.Register(Provider{ID: "p", ClientID: "manifest-id", Source: SourceManifest})

		got, _ := r.Get("p")
		if got.ClientID != "byoa-id" {
			t.Errorf("ClientID = %q, want %q (BYOA should win)", got.ClientID, "byoa-id")
		}
	})

	t.Run("built-in does not override manifest", func(t *testing.T) {
		r := NewRegistry()
		r.Register(Provider{ID: "p", AuthorizeURL: "https://manifest.com/auth", Source: SourceManifest})
		r.Register(Provider{ID: "p", AuthorizeURL: "https://builtin.com/auth", Source: SourceBuiltIn})

		got, _ := r.Get("p")
		if got.AuthorizeURL != "https://manifest.com/auth" {
			t.Errorf("AuthorizeURL = %q, want manifest URL", got.AuthorizeURL)
		}
	})

	t.Run("manifest overrides built-in", func(t *testing.T) {
		r := NewRegistry()
		r.Register(Provider{ID: "p", AuthorizeURL: "https://builtin.com/auth", Source: SourceBuiltIn})
		r.Register(Provider{ID: "p", AuthorizeURL: "https://manifest.com/auth", Source: SourceManifest})

		got, _ := r.Get("p")
		if got.AuthorizeURL != "https://manifest.com/auth" {
			t.Errorf("AuthorizeURL = %q, want manifest URL", got.AuthorizeURL)
		}
	})

	t.Run("BYOA overrides everything", func(t *testing.T) {
		r := NewRegistry()
		r.Register(Provider{ID: "p", Source: SourceBuiltIn})
		r.Register(Provider{ID: "p", Source: SourceManifest})
		r.Register(Provider{ID: "p", ClientID: "user-app", Source: SourceBYOA})

		got, _ := r.Get("p")
		if got.ClientID != "user-app" {
			t.Errorf("ClientID = %q, want %q", got.ClientID, "user-app")
		}
	})

	t.Run("same source replaces", func(t *testing.T) {
		r := NewRegistry()
		r.Register(Provider{ID: "p", ClientID: "first", Source: SourceBuiltIn})
		r.Register(Provider{ID: "p", ClientID: "second", Source: SourceBuiltIn})

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
