package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestValidatePathParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid hex id", "abc123def456", false},
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"contains slash", "abc/def", true},
		{"contains question mark", "abc?key=val", true},
		{"contains hash", "abc#frag", true},
		{"contains percent", "abc%2F", true},
		{"path traversal", "../../etc/passwd", true},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validatePathParam("test_param", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePathParam(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if err != nil && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}

func TestRequirePathParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "abc123", false},
		{"empty", "", true},
		{"slash", "a/b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := requirePathParam("test", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("requirePathParam(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestRequireParam(t *testing.T) {
	t.Parallel()

	if err := requireParam("x", "val"); err != nil {
		t.Errorf("requireParam(non-empty) unexpected error: %v", err)
	}
	if err := requireParam("x", ""); err == nil {
		t.Error("requireParam(empty) expected error, got nil")
	}
}

func TestPathParamValidation_RejectsInjection(t *testing.T) {
	t.Parallel()

	conn := New()
	actions := conn.Actions()

	// For each action that uses path params, verify injection is rejected.
	tests := []struct {
		name   string
		action string
		params string
	}{
		{"get_zone slash", "cloudflare.get_zone", `{"zone_id":"zone1/../../accounts"}`},
		{"list_dns_records slash", "cloudflare.list_dns_records", `{"zone_id":"z1/x"}`},
		{"create_dns_record slash", "cloudflare.create_dns_record", `{"zone_id":"z1/x","type":"A","name":"t","content":"1.2.3.4"}`},
		{"update_dns_record zone slash", "cloudflare.update_dns_record", `{"zone_id":"z/x","record_id":"r1"}`},
		{"update_dns_record record slash", "cloudflare.update_dns_record", `{"zone_id":"z1","record_id":"r/x"}`},
		{"delete_dns_record slash", "cloudflare.delete_dns_record", `{"zone_id":"z1","record_id":"r/x"}`},
		{"list_tunnels slash", "cloudflare.list_tunnels", `{"account_id":"a/x"}`},
		{"create_tunnel slash", "cloudflare.create_tunnel", `{"account_id":"a/x","name":"t"}`},
		{"get_tunnel slash", "cloudflare.get_tunnel", `{"account_id":"a1","tunnel_id":"t/x"}`},
		{"delete_tunnel slash", "cloudflare.delete_tunnel", `{"account_id":"a1","tunnel_id":"t/x"}`},
		{"list_tunnel_configs slash", "cloudflare.list_tunnel_configs", `{"account_id":"a1","tunnel_id":"t/x"}`},
		{"update_tunnel_config slash", "cloudflare.update_tunnel_config", `{"account_id":"a/x","tunnel_id":"t1","config":{"ingress":[]}}`},
		{"check_domain slash", "cloudflare.check_domain", `{"account_id":"a1","domain":"d/x"}`},
		{"update_domain_settings slash", "cloudflare.update_domain_settings", `{"account_id":"a/x","domain":"d1"}`},
		{"purge_cache slash", "cloudflare.purge_cache", `{"zone_id":"z/x","purge_everything":true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			action := actions[tt.action]
			if action == nil {
				t.Fatalf("action %q not found", tt.action)
			}
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected validation error for path injection, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestDoJSON_EnvelopeSuccessFalse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"errors": []any{
				map[string]any{"code": 1003, "message": "Invalid or missing zone id"},
			},
			"result": nil,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)

	var resp json.RawMessage
	err := conn.doGet(t.Context(), validCreds(), "/zones/bad", &resp)
	if err == nil {
		t.Fatal("expected error for success:false envelope, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestDoJSON_EnvelopeSuccessFalse_EmptyErrors(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"errors":  []any{},
			"result":  nil,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)

	var resp json.RawMessage
	err := conn.doGet(t.Context(), validCreds(), "/zones/bad", &resp)
	if err == nil {
		t.Fatal("expected error for success:false with empty errors, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestDoJSON_EnvelopeSuccessFalse_NilRespBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"errors": []any{
				map[string]any{"code": 1003, "message": "Delete failed"},
			},
			"result": nil,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)

	// nil respBody simulates a DELETE action
	err := conn.doJSON(t.Context(), validCreds(), http.MethodDelete, "/test", nil, nil)
	if err == nil {
		t.Fatal("expected error for success:false with nil respBody, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
