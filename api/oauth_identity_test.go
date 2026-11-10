package api

import (
	"encoding/json"
	"testing"
)

func TestOAuthConnectionIdentityKey(t *testing.T) {
	t.Parallel()
	mustJSON := func(m map[string]string) json.RawMessage {
		t.Helper()
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		return b
	}

	tests := []struct {
		provider string
		extra    json.RawMessage
		want     string
	}{
		{"google", mustJSON(map[string]string{"email": "User@Example.com"}), "email:user@example.com"},
		{"google", mustJSON(map[string]string{"display_name": "Jane", "email": "jane@example.com"}), "email:jane@example.com"},
		{"shopify", mustJSON(map[string]string{"shop_domain": "MyStore.myshopify.com", "email": "a@b.com"}), "shop:mystore.myshopify.com"},
		{"zendesk", mustJSON(map[string]string{"subdomain": "MyCo"}), "subdomain:myco"},
		{"slack", mustJSON(map[string]string{"email": "u@example.com", "team_name": "Acme"}), "slack_team:acme"},
		{"github", mustJSON(map[string]string{"display_name": "octocat"}), "github_login:octocat"},
		{"google", nil, ""},
	}

	for _, tc := range tests {
		got := oauthConnectionIdentityKey(tc.provider, tc.extra)
		if got != tc.want {
			t.Errorf("provider %q extra %s: got %q want %q", tc.provider, string(tc.extra), got, tc.want)
		}
	}
}
