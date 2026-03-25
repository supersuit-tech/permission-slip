package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteTunnel_Execute(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/accounts/acc1/cfd_tunnel/tun1" {
			t.Errorf("path = %s, want /accounts/acc1/cfd_tunnel/tun1", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result":  map[string]any{"id": "tun1"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteTunnelAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"account_id":"acc1","tunnel_id":"tun1"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["status"] != "deleted" {
		t.Errorf("status = %v, want deleted", data["status"])
	}
}
