package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateDNSRecord_Execute(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/zones/zone1/dns_records" {
			t.Errorf("path = %s, want /zones/zone1/dns_records", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["type"] != "A" {
			t.Errorf("type = %v, want A", body["type"])
		}
		if body["name"] != "test.example.com" {
			t.Errorf("name = %v, want test.example.com", body["name"])
		}
		if body["content"] != "1.2.3.4" {
			t.Errorf("content = %v, want 1.2.3.4", body["content"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result": map[string]any{
				"id":      "rec1",
				"type":    "A",
				"name":    "test.example.com",
				"content": "1.2.3.4",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createDNSRecordAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"zone_id":"zone1","type":"A","name":"test.example.com","content":"1.2.3.4"}`),
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

func TestCreateDNSRecord_MissingRequired(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDNSRecordAction{conn: conn}

	tests := []struct {
		name   string
		params string
	}{
		{"missing zone_id", `{"type":"A","name":"test","content":"1.2.3.4"}`},
		{"missing type", `{"zone_id":"z1","name":"test","content":"1.2.3.4"}`},
		{"missing name", `{"zone_id":"z1","type":"A","content":"1.2.3.4"}`},
		{"missing content", `{"zone_id":"z1","type":"A","name":"test"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
