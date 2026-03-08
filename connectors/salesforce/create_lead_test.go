package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateLead_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/services/data/v62.0/sobjects/Lead/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["LastName"] != "Doe" {
			t.Errorf("expected LastName 'Doe', got %v", body["LastName"])
		}
		if body["Company"] != "Acme" {
			t.Errorf("expected Company 'Acme', got %v", body["Company"])
		}
		if body["Email"] != "john@acme.com" {
			t.Errorf("expected Email, got %v", body["Email"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sfCreateResponse{ID: "00Qxx0000001abc", Success: true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createLeadAction{conn: conn}

	params, _ := json.Marshal(createLeadParams{
		LastName:  "Doe",
		Company:   "Acme",
		FirstName: "John",
		Email:     "john@acme.com",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_lead",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "00Qxx0000001abc" {
		t.Errorf("expected id '00Qxx0000001abc', got %v", data["id"])
	}
	if data["success"] != true {
		t.Errorf("expected success true, got %v", data["success"])
	}
	if data["record_url"] != "https://myorg.salesforce.com/00Qxx0000001abc" {
		t.Errorf("expected record_url, got %v", data["record_url"])
	}
}

func TestCreateLead_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createLeadAction{conn: conn}

	tests := []struct {
		name   string
		params map[string]any
	}{
		{"missing last_name", map[string]any{"company": "Acme"}},
		{"missing company", map[string]any{"last_name": "Doe"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := json.Marshal(tt.params)
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "salesforce.create_lead",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error for missing required field")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}

func TestCreateLead_InvalidEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createLeadAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"last_name": "Doe",
		"company":   "Acme",
		"email":     "not-an-email",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_lead",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
