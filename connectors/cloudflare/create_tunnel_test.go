package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateTunnel_Execute(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/accounts/acc1/cfd_tunnel" {
			t.Errorf("path = %s, want /accounts/acc1/cfd_tunnel", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "my-tunnel" {
			t.Errorf("name = %v, want my-tunnel", body["name"])
		}
		if body["tunnel_secret"] == nil || body["tunnel_secret"] == "" {
			t.Error("tunnel_secret should be present")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result": map[string]any{
				"id":   "tun1",
				"name": "my-tunnel",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createTunnelAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"account_id":"acc1","name":"my-tunnel","tunnel_secret":"dGVzdHNlY3JldDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI="}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["id"] != "tun1" {
		t.Errorf("id = %v, want tun1", data["id"])
	}
}

func TestCreateTunnel_InvalidSecret(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTunnelAction{conn: conn}

	tests := []struct {
		name   string
		params string
	}{
		{"not base64", `{"account_id":"acc1","name":"t","tunnel_secret":"not-valid-base64!!!"}`},
		{"wrong length", `{"account_id":"acc1","name":"t","tunnel_secret":"dG9vc2hvcnQ="}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestCreateTunnel_AutoGeneratesSecret(t *testing.T) {
	t.Parallel()

	var receivedSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if s, ok := body["tunnel_secret"].(string); ok {
			receivedSecret = s
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result":  map[string]any{"id": "tun1", "name": "t"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createTunnelAction{conn: conn}

	// No tunnel_secret provided — should auto-generate
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"account_id":"acc1","name":"t"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if receivedSecret == "" {
		t.Error("expected auto-generated tunnel_secret, got empty")
	}
	// Base64-encoded 32 bytes = 44 characters
	if len(receivedSecret) != 44 {
		t.Errorf("auto-generated secret length = %d, want 44", len(receivedSecret))
	}
}
