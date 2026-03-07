package airtable

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteRecords_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/appABC123/Tasks" {
			t.Errorf("expected path /appABC123/Tasks, got %s", r.URL.Path)
		}

		// Verify record IDs in query params
		records := r.URL.Query()["records[]"]
		if len(records) != 2 {
			t.Fatalf("expected 2 record IDs in query, got %d", len(records))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"records": []map[string]any{
				{"id": "recXYZ789", "deleted": true},
				{"id": "recABC456", "deleted": true},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteRecordsAction{conn: conn}

	params, _ := json.Marshal(deleteRecordsParams{
		BaseID:    "appABC123",
		Table:     "Tasks",
		RecordIDs: []string{"recXYZ789", "recABC456"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.delete_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data deleteRecordsResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Deleted) != 2 {
		t.Fatalf("expected 2 deleted records, got %d", len(data.Deleted))
	}
	if !data.Deleted[0].Deleted {
		t.Error("expected first record to be deleted")
	}
}

func TestDeleteRecords_MissingRecordIDs(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteRecordsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"base_id": "appABC123",
		"table":   "Tasks",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.delete_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing record_ids")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDeleteRecords_InvalidRecordID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteRecordsAction{conn: conn}

	params, _ := json.Marshal(deleteRecordsParams{
		BaseID:    "appABC123",
		Table:     "Tasks",
		RecordIDs: []string{"recGOOD123", "bad456"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.delete_records",
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

func TestDeleteRecords_TooManyRecords(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteRecordsAction{conn: conn}

	ids := make([]string, 11)
	for i := range ids {
		ids[i] = "recABC0000"
	}

	params, _ := json.Marshal(deleteRecordsParams{
		BaseID:    "appABC123",
		Table:     "Tasks",
		RecordIDs: ids,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.delete_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for too many record IDs")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDeleteRecords_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteRecordsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.delete_records",
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
