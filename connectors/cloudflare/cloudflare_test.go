package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCloudflareConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "cloudflare" {
		t.Errorf("ID() = %q, want %q", got, "cloudflare")
	}
}

func TestCloudflareConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"cloudflare.list_zones",
		"cloudflare.get_zone",
		"cloudflare.list_dns_records",
		"cloudflare.create_dns_record",
		"cloudflare.update_dns_record",
		"cloudflare.delete_dns_record",
		"cloudflare.list_tunnels",
		"cloudflare.create_tunnel",
		"cloudflare.delete_tunnel",
		"cloudflare.get_tunnel",
		"cloudflare.list_tunnel_configs",
		"cloudflare.update_tunnel_config",
		"cloudflare.check_domain",
		"cloudflare.update_domain_settings",
		"cloudflare.purge_cache",
	}
	for _, at := range want {
		if _, ok := actions[at]; !ok {
			t.Errorf("Actions() missing %q", at)
		}
	}
	if len(actions) != len(want) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(want))
	}
}

func TestCloudflareConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "abc123"}),
			wantErr: false,
		},
		{
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": ""}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("ValidateCredentials() returned %T, want *connectors.ValidationError", err)
			}
		})
	}
}

func TestCloudflareConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "cloudflare" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "cloudflare")
	}
	if m.Name != "Cloudflare" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Cloudflare")
	}
	if len(m.Actions) != 15 {
		t.Fatalf("Manifest().Actions has %d items, want 15", len(m.Actions))
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestCloudflareConnector_ActionsMatchManifest(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	manifest := c.Manifest()

	manifestTypes := make(map[string]bool, len(manifest.Actions))
	for _, a := range manifest.Actions {
		manifestTypes[a.ActionType] = true
	}

	for actionType := range actions {
		if !manifestTypes[actionType] {
			t.Errorf("Actions() has %q but Manifest() does not", actionType)
		}
	}
	for _, a := range manifest.Actions {
		if _, ok := actions[a.ActionType]; !ok {
			t.Errorf("Manifest() has %q but Actions() does not", a.ActionType)
		}
	}
}

func TestCloudflareConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*CloudflareConnector)(nil)
	var _ connectors.ManifestProvider = (*CloudflareConnector)(nil)
}

func TestCloudflareConnector_DoJSONUnwrapsEnvelope(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-api-token-123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-api-token-123")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result": map[string]any{
				"id":   "zone123",
				"name": "example.com",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)

	var resp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err := conn.doGet(t.Context(), validCreds(), "/zones/zone123", &resp)
	if err != nil {
		t.Fatalf("doGet() unexpected error: %v", err)
	}
	if resp.ID != "zone123" {
		t.Errorf("ID = %q, want %q", resp.ID, "zone123")
	}
	if resp.Name != "example.com" {
		t.Errorf("Name = %q, want %q", resp.Name, "example.com")
	}
}

func TestCloudflareConnector_ErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		headers    map[string]string
		checkErr   func(t *testing.T, err error)
	}{
		{
			name:       "401 returns AuthError",
			statusCode: 401,
			body:       `{"success":false,"errors":[{"code":9109,"message":"Invalid access token"}]}`,
			checkErr: func(t *testing.T, err error) {
				if !connectors.IsAuthError(err) {
					t.Errorf("expected AuthError, got %T: %v", err, err)
				}
			},
		},
		{
			name:       "429 returns RateLimitError",
			statusCode: 429,
			body:       `{"success":false,"errors":[{"code":10000,"message":"Rate limit"}]}`,
			headers:    map[string]string{"Retry-After": "60"},
			checkErr: func(t *testing.T, err error) {
				if !connectors.IsRateLimitError(err) {
					t.Errorf("expected RateLimitError, got %T: %v", err, err)
				}
			},
		},
		{
			name:       "400 returns ValidationError",
			statusCode: 400,
			body:       `{"success":false,"errors":[{"code":1004,"message":"Invalid request"}]}`,
			checkErr: func(t *testing.T, err error) {
				if !connectors.IsValidationError(err) {
					t.Errorf("expected ValidationError, got %T: %v", err, err)
				}
			},
		},
		{
			name:       "500 returns ExternalError",
			statusCode: 500,
			body:       `{"success":false,"errors":[{"code":10000,"message":"Internal server error"}]}`,
			checkErr: func(t *testing.T, err error) {
				if !connectors.IsExternalError(err) {
					t.Errorf("expected ExternalError, got %T: %v", err, err)
				}
			},
		},
		{
			name:       "multiple errors joined",
			statusCode: 400,
			body:       `{"success":false,"errors":[{"code":1,"message":"Error one"},{"code":2,"message":"Error two"}]}`,
			checkErr: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "Error one") || !strings.Contains(err.Error(), "Error two") {
					t.Errorf("expected both errors in message, got: %s", err.Error())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			conn := newForTest(srv.Client(), srv.URL)
			var resp struct{}
			err := conn.doGet(t.Context(), validCreds(), "/test", &resp)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			tt.checkErr(t, err)
		})
	}
}
