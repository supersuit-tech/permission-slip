package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListTunnels_Execute(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/accounts/acc1/cfd_tunnel" {
			t.Errorf("path = %s, want /accounts/acc1/cfd_tunnel", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result": []map[string]any{
				{"id": "tun1", "name": "my-tunnel", "status": "active"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listTunnelsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"account_id":"acc1"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	tunnels, ok := data["tunnels"].([]any)
	if !ok {
		t.Fatal("result missing tunnels array")
	}
	if len(tunnels) != 1 {
		t.Errorf("got %d tunnels, want 1", len(tunnels))
	}
}

func TestListTunnels_MissingAccountID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listTunnelsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
