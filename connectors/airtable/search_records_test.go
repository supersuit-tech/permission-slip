package airtable

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSearchRecords_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		formula := r.URL.Query().Get("filterByFormula")
		if formula != "SEARCH('John', {Name})" {
			t.Errorf("expected formula, got %q", formula)
		}
		maxRecords := r.URL.Query().Get("maxRecords")
		if maxRecords != "50" {
			t.Errorf("expected maxRecords '50', got %q", maxRecords)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"records": []map[string]any{
				{
					"id":          "recXYZ789",
					"createdTime": "2024-01-15T10:30:00.000Z",
					"fields":      map[string]any{"Name": "John Doe", "Email": "john@example.com"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchRecordsAction{conn: conn}

	params, _ := json.Marshal(searchRecordsParams{
		BaseID:     "appABC123",
		Table:      "Contacts",
		Formula:    "SEARCH('John', {Name})",
		MaxRecords: 50,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.search_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listRecordsResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(data.Records))
	}
	if data.Records[0].ID != "recXYZ789" {
		t.Errorf("expected record ID 'recXYZ789', got %q", data.Records[0].ID)
	}
}

func TestSearchRecords_DefaultMaxRecords(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		maxRecords := r.URL.Query().Get("maxRecords")
		if maxRecords != "100" {
			t.Errorf("expected default maxRecords '100', got %q", maxRecords)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"records": []map[string]any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchRecordsAction{conn: conn}

	params, _ := json.Marshal(searchRecordsParams{
		BaseID:  "appABC123",
		Table:   "Contacts",
		Formula: "{Status} = 'Active'",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.search_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchRecords_MissingFormula(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchRecordsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"base_id": "appABC123",
		"table":   "Contacts",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.search_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing formula")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchRecords_MissingBaseID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchRecordsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"table":   "Contacts",
		"formula": "{Status} = 'Active'",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.search_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing base_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchRecords_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchRecordsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.search_records",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
