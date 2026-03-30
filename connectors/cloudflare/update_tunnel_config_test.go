package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateTunnelConfig_Execute(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/accounts/acc1/cfd_tunnel/tun1/configurations" {
			t.Errorf("path = %s, want /accounts/acc1/cfd_tunnel/tun1/configurations", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["config"] == nil {
			t.Error("expected config in request body")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result": map[string]any{
				"config": map[string]any{"ingress": []any{}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateTunnelConfigAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"account_id":"acc1","tunnel_id":"tun1","config":{"ingress":[{"hostname":"example.com","service":"http://localhost:8080"},{"service":"http_status:404"}]}}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["config"] == nil {
		t.Error("expected config in result")
	}
}

func TestUpdateTunnelConfig_MissingConfig(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTunnelConfigAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"account_id":"acc1","tunnel_id":"tun1"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateTunnelConfig_NullConfig(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTunnelConfigAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"account_id":"acc1","tunnel_id":"tun1","config":null}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for null config, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
