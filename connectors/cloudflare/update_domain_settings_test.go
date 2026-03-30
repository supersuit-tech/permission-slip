package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateDomainSettings_Execute(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/accounts/acc1/registrar/domains/example.com" {
			t.Errorf("path = %s, want /accounts/acc1/registrar/domains/example.com", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["auto_renew"] != true {
			t.Errorf("auto_renew = %v, want true", body["auto_renew"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"errors":  []any{},
			"result": map[string]any{
				"domain_name": "example.com",
				"auto_renew":  true,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateDomainSettingsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"account_id":"acc1","domain":"example.com","auto_renew":true}`),
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

func TestUpdateDomainSettings_NoSettingsProvided(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDomainSettingsAction{conn: conn}

	// No auto_renew in params — should require at least one setting
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		Parameters:  json.RawMessage(`{"account_id":"acc1","domain":"example.com"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for no settings provided, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateDomainSettings_MissingRequired(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDomainSettingsAction{conn: conn}

	tests := []struct {
		name   string
		params string
	}{
		{"missing account_id", `{"domain":"example.com"}`},
		{"missing domain", `{"account_id":"acc1"}`},
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
