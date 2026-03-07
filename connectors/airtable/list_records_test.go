package airtable

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListRecords_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/appABC123/Tasks" {
			t.Errorf("expected path /appABC123/Tasks, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"records": []map[string]any{
				{
					"id":          "recXYZ789",
					"createdTime": "2024-01-15T10:30:00.000Z",
					"fields":      map[string]any{"Name": "Task 1", "Status": "Active"},
				},
			},
			"offset": "itrNext123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listRecordsAction{conn: conn}

	params, _ := json.Marshal(listRecordsParams{
		BaseID: "appABC123",
		Table:  "Tasks",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_records",
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
	if data.Offset != "itrNext123" {
		t.Errorf("expected offset 'itrNext123', got %q", data.Offset)
	}
}

func TestListRecords_WithFilter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		formula := r.URL.Query().Get("filterByFormula")
		if formula != "{Status} = 'Active'" {
			t.Errorf("expected formula, got %q", formula)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"records": []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listRecordsAction{conn: conn}

	params, _ := json.Marshal(listRecordsParams{
		BaseID:          "appABC123",
		Table:           "Tasks",
		FilterByFormula: "{Status} = 'Active'",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListRecords_MissingBaseID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listRecordsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"table": "Tasks"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_records",
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

func TestListRecords_MissingTable(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listRecordsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"base_id": "appABC123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing table")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListRecords_InvalidBaseID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listRecordsAction{conn: conn}

	params, _ := json.Marshal(listRecordsParams{BaseID: "bad123", Table: "Tasks"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid base_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListRecords_InvalidPageSize(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listRecordsAction{conn: conn}

	params, _ := json.Marshal(listRecordsParams{BaseID: "appABC123", Table: "Tasks", PageSize: 200})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid page_size")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListRecords_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "NOT_FOUND",
				"message": "Could not find table",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listRecordsAction{conn: conn}

	params, _ := json.Marshal(listRecordsParams{BaseID: "appABC123", Table: "Nonexistent"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for API error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestListRecords_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listRecordsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_records",
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
