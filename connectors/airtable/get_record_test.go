package airtable

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetRecord_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/appABC123/Tasks/recXYZ789" {
			t.Errorf("expected path /appABC123/Tasks/recXYZ789, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":          "recXYZ789",
			"createdTime": "2024-01-15T10:30:00.000Z",
			"fields":      map[string]any{"Name": "Task 1", "Status": "Active"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getRecordAction{conn: conn}

	params, _ := json.Marshal(getRecordParams{
		BaseID:   "appABC123",
		Table:    "Tasks",
		RecordID: "recXYZ789",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.get_record",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data recordSummary
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.ID != "recXYZ789" {
		t.Errorf("expected record ID 'recXYZ789', got %q", data.ID)
	}
	name, ok := data.Fields["Name"]
	if !ok {
		t.Fatal("expected field 'Name' in response")
	}
	if name != "Task 1" {
		t.Errorf("expected Name 'Task 1', got %q", name)
	}
}

func TestGetRecord_MissingRecordID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getRecordAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"base_id": "appABC123",
		"table":   "Tasks",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.get_record",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing record_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetRecord_InvalidRecordID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getRecordAction{conn: conn}

	params, _ := json.Marshal(getRecordParams{
		BaseID:   "appABC123",
		Table:    "Tasks",
		RecordID: "bad123",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.get_record",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid record_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetRecord_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "NOT_FOUND",
				"message": "Could not find record",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getRecordAction{conn: conn}

	params, _ := json.Marshal(getRecordParams{
		BaseID:   "appABC123",
		Table:    "Tasks",
		RecordID: "recNONEXIST",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.get_record",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetRecord_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getRecordAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.get_record",
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
