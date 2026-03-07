package airtable

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateRecords_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/appABC123/Tasks" {
			t.Errorf("expected path /appABC123/Tasks, got %s", r.URL.Path)
		}

		var body updateRecordsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(body.Records) != 1 {
			t.Errorf("expected 1 record, got %d", len(body.Records))
		}
		if body.Records[0].ID != "recXYZ789" {
			t.Errorf("expected record ID 'recXYZ789', got %q", body.Records[0].ID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"records": []map[string]any{
				{
					"id":          "recXYZ789",
					"createdTime": "2024-01-15T10:30:00.000Z",
					"fields":      map[string]any{"Name": "Updated Task", "Status": "Done"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateRecordsAction{conn: conn}

	params, _ := json.Marshal(updateRecordsParams{
		BaseID: "appABC123",
		Table:  "Tasks",
		Records: []updateRecordInput{
			{ID: "recXYZ789", Fields: map[string]any{"Status": "Done"}},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.update_records",
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

func TestUpdateRecords_InvalidRecordID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecordsAction{conn: conn}

	params, _ := json.Marshal(updateRecordsParams{
		BaseID: "appABC123",
		Table:  "Tasks",
		Records: []updateRecordInput{
			{ID: "bad123", Fields: map[string]any{"Status": "Done"}},
		},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.update_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid record ID")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateRecords_TooManyRecords(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecordsAction{conn: conn}

	records := make([]updateRecordInput, 11)
	for i := range records {
		records[i] = updateRecordInput{ID: "recABC0000", Fields: map[string]any{"x": 1}}
	}

	params, _ := json.Marshal(updateRecordsParams{
		BaseID:  "appABC123",
		Table:   "Tasks",
		Records: records,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.update_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for too many records")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateRecords_MissingFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecordsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"base_id": "appABC123",
		"table":   "Tasks",
		"records": []map[string]any{{"id": "recXYZ789"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.update_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateRecords_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecordsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.update_records",
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
