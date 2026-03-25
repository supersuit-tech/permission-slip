package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateDNSRecord_Execute(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/zones/zone1/dns_records/rec1" {
			t.Errorf("path = %s, want /zones/zone1/dns_records/rec1", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["content"] != "5.6.7.8" {
			t.Errorf("content = %v, want 5.6.7.8", body["content"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result": map[string]any{
				"id":      "rec1",
				"content": "5.6.7.8",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateDNSRecordAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"zone_id":"zone1","record_id":"rec1","content":"5.6.7.8"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["id"] != "rec1" {
		t.Errorf("id = %v, want rec1", data["id"])
	}
}

func TestUpdateDNSRecord_NoUpdateFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDNSRecordAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"zone_id":"z1","record_id":"r1"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for no update fields, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
