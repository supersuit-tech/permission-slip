package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCheckDomain_Execute(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/accounts/acc1/registrar/domains/example.com" {
			t.Errorf("path = %s, want /accounts/acc1/registrar/domains/example.com", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result": map[string]any{
				"domain_name": "example.com",
				"available":   false,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &checkDomainAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"account_id":"acc1","domain":"example.com"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["domain_name"] != "example.com" {
		t.Errorf("domain_name = %v, want example.com", data["domain_name"])
	}
}
