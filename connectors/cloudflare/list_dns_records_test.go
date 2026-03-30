package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListDNSRecords_Execute(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/zones/zone1/dns_records" {
			t.Errorf("path = %s, want /zones/zone1/dns_records", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "A" {
			t.Errorf("query type = %s, want A", r.URL.Query().Get("type"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result": []any{
				map[string]any{"id": "rec1", "type": "A", "name": "example.com"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listDNSRecordsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"zone_id":"zone1","type":"A"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	records, ok := data["dns_records"].([]any)
	if !ok || len(records) != 1 {
		t.Errorf("expected 1 record, got %v", data["dns_records"])
	}
}
