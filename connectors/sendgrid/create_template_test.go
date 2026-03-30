package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateTemplate_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/templates" {
			t.Errorf("got %s %s, want POST /templates", r.Method, r.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}
		if body["name"] != "Welcome Email" {
			t.Errorf("name = %q, want Welcome Email", body["name"])
		}
		if body["generation"] != "dynamic" {
			t.Errorf("generation = %q, want dynamic", body["generation"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "d-abc123",
			"name":       "Welcome Email",
			"generation": "dynamic",
			"updated_at": "2026-03-07T12:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.create_template"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.create_template",
		Parameters:  json.RawMessage(`{"name":"Welcome Email"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["template_id"] != "d-abc123" {
		t.Errorf("template_id = %v, want d-abc123", data["template_id"])
	}
	if data["generation"] != "dynamic" {
		t.Errorf("generation = %v, want dynamic", data["generation"])
	}
}

func TestCreateTemplate_LegacyGeneration(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}
		if body["generation"] != "legacy" {
			t.Errorf("generation = %q, want legacy", body["generation"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "tpl_legacy",
			"name":       "Old Template",
			"generation": "legacy",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.create_template"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.create_template",
		Parameters:  json.RawMessage(`{"name":"Old Template","generation":"legacy"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateTemplate_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.create_template"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing name", params: `{}`},
		{name: "empty name", params: `{"name":""}`},
		{name: "invalid generation", params: `{"name":"Test","generation":"invalid"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "sendgrid.create_template",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
